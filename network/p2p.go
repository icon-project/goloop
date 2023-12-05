package network

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

const (
	LoggerFieldKeySubModule = "sub"
)

const (
	DefaultTransportNet         = "tcp4"
	DefaultDialTimeout          = 5 * time.Second
	DefaultReceiveQueueSize     = 4096
	DefaultPacketBufferSize     = 4096 //bufio.defaultBufSize=4096
	DefaultPacketPayloadMax     = 1024 * 1024
	DefaultPacketPoolNumBucket  = 20
	DefaultPacketPoolBucketLen  = 500
	DefaultDiscoveryPeriod      = 2 * time.Second
	DefaultSeedPeriod           = 3 * time.Second
	DefaultAlternateSendPeriod  = 1 * time.Second
	DefaultSendTimeout          = 5 * time.Second
	DefaultSendQueueMaxPriority = 7
	DefaultSendQueueSize        = 1000
	DefaultEventQueueSize       = 100
	DefaultPeerSendQueueSize    = 1000
	DefaultPeerPoolExpireSecond = 5
	DefaultParentsLimit         = 1
	DefaultUnclesLimit          = 1
	DefaultChildrenLimit        = 10
	DefaultNephewsLimit         = 10
	DefaultOthersLimit          = 10
	DefaultPacketRewriteLimit   = 10
	DefaultPacketRewriteDelay   = 100 * time.Millisecond
	DefaultRttAccuracy          = 10 * time.Millisecond
	DefaultRttLogTimeout        = 1 * time.Second
	DefaultRttLogThreshold      = 1 * time.Second
	DefaultFailureNodeMin       = 2
	DefaultSelectiveFloodingAdd = 2
	DefaultSimplePeerIDSize     = 4
	DefaultDuplicatedPeerTime   = 1 * time.Second
	DefaultMaxRetryClose        = 10
	AttrP2PConnectionRequest    = "P2PConnectionRequest"
	AttrP2PLegacy               = "P2PLegacy"
	AttrSupportDefaultProtocols = "SupportDefaultProtocols"
	AttrSRHeight                = "SeedRequestHeight"
	DefaultQueryElementLength   = 200
	DefaultDiffMaxHold          = 10
)

var (
	p2pProtoControl     = module.ProtoP2P
	p2pControlProtocols = []module.ProtocolInfo{p2pProtoControl, p2pProtoControlV1}
)

var (
	p2pProtoQueryReq  = module.ProtocolInfo(0x0700)
	p2pProtoQueryResp = module.ProtocolInfo(0x0800)
	p2pProtoConnReq   = module.ProtocolInfo(0x0900)
	p2pProtoConnResp  = module.ProtocolInfo(0x0A00)
	p2pProtoRttReq    = module.ProtocolInfo(0x0B00)
	p2pProtoRttResp   = module.ProtocolInfo(0x0C00)
)

type PeerToPeer struct {
	*peerHandler
	channel         string
	sendQueue       *WeightQueue
	alternateQueue  Queue
	onPacketCbFuncs map[uint16]packetCbFunc
	onEventCbFuncs  map[string]map[uint16]eventCbFunc
	packetPool      *PacketPool
	dialer          *Dialer

	self *Peer

	rh   *rttHandler
	rr   *roleResolver
	pm   *peerManager
	as   *addressSyncer
	qh   *queryHandler
	qhv1 *queryHandlerV1

	//monitor
	mtr *metric.NetworkMetric

	stopCh chan bool
	run    bool
	mtx    sync.RWMutex
}

type eventCbFunc func(evt string, p *Peer)

const (
	p2pEventJoin       = "join"
	p2pEventLeave      = "leave"
	p2pEventDuplicate  = "duplicate"
	p2pEventNotAllowed = "not allowed"
)

type messageCodec interface {
	encode(interface{}) []byte
	decode([]byte, interface{}) error
}

func newPeerToPeer(channel string, self *Peer, d *Dialer, sm *SeedManager, mtr *metric.NetworkMetric, l log.Logger) *PeerToPeer {
	rh := newRTTHandler(l)
	p2p := &PeerToPeer{
		peerHandler: newPeerHandler(
			self.ID(),
			l.WithFields(log.Fields{LoggerFieldKeySubModule: "p2p"})),
		channel:         channel,
		sendQueue:       NewWeightQueue(DefaultSendQueueSize, DefaultSendQueueMaxPriority+1),
		alternateQueue:  NewQueue(DefaultSendQueueSize),
		onPacketCbFuncs: make(map[uint16]packetCbFunc),
		onEventCbFuncs:  make(map[string]map[uint16]eventCbFunc),
		packetPool:      NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen),
		dialer:          d,
		//
		self: self,
		//
		rh: rh,
		//
		mtr: mtr,
	}
	p2p.rr = newRoleResolver(self, p2p.onAllowedPeerIDSetUpdate, l)
	p2p.pm = newPeerManager(p2p, self, p2p.onEvent, p2p._onClose, l)
	p2p.as = newAddressSyncer(d, p2p.pm, l)
	p2p.qh = newQueryHandler(p2p, self, p2p.pm, p2p.rr, p2p.as, rh, l)
	p2p.qhv1 = newQueryHandlerV1(p2p, self, p2p.pm, p2p.rr, p2p.as, rh, p2p.sm, l)
	return p2p
}

func (p2p *PeerToPeer) IsStarted() bool {
	defer p2p.mtx.RUnlock()
	p2p.mtx.RLock()

	return p2p.run
}

func (p2p *PeerToPeer) Start() {
	defer p2p.mtx.Unlock()
	p2p.mtx.Lock()

	if p2p.run {
		return
	}
	p2p.run = true
	p2p.stopCh = make(chan bool)

	go p2p.sendRoutine()
	go p2p.alternateSendRoutine()
	go p2p.discoverRoutine()
}

func (p2p *PeerToPeer) Stop() {
	defer p2p.mtx.Unlock()
	p2p.mtx.Lock()

	if !p2p.run {
		return
	}
	p2p.logger.Debugln("Stop", "try close p2p.stopCh")
	close(p2p.stopCh)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
	Loop:
		for i := 0; i < DefaultMaxRetryClose; i++ {
			ps := p2p.pm.findPeers(nil)
			p2p.logger.Debugln("Stop", "try close Peers", len(ps))
			for _, p := range ps {
				if !p.IsClosed() {
					p.Close("stopCh")
				}
			}
			if len(ps) < 1 {
				break Loop
			}
			time.Sleep(time.Second)
		}
		wg.Done()
	}()
	p2p.logger.Debugln("Stop", "wait peer Closing")
	wg.Wait()

	p2p.run = false
	p2p.logger.Debugln("Stop", "Done")
}

func (p2p *PeerToPeer) supportedProtocols() []module.ProtocolInfo {
	return p2pControlProtocols
}

func (p2p *PeerToPeer) dial(na NetAddress) error {
	if err := p2p.dialer.Dial(string(na)); err != nil {
		if err == ErrAlreadyDialing {
			p2p.logger.Infoln("Dial ignore", na, err)
			return nil
		}
		p2p.logger.Infoln("Dial fail", na, err)
		return err
	}
	return nil
}

func (p2p *PeerToPeer) setCbFunc(pi module.ProtocolInfo, pktFunc packetCbFunc,
	evtFunc eventCbFunc, evts ...string) {
	k := pi.Uint16()
	if _, ok := p2p.onPacketCbFuncs[k]; ok {
		p2p.logger.Infoln("overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[k] = pktFunc
	for _, evt := range evts {
		p2p.setEventCbFunc(evt, k, evtFunc)
	}
}

func (p2p *PeerToPeer) unsetCbFunc(pi module.ProtocolInfo) {
	k := pi.Uint16()
	if _, ok := p2p.onPacketCbFuncs[k]; ok {
		p2p.unsetEventCbFunc(k)
		delete(p2p.onPacketCbFuncs, k)
	}
}

func (p2p *PeerToPeer) setEventCbFunc(evt string, k uint16, evtFunc eventCbFunc) {
	m := p2p.onEventCbFuncs[evt]
	if m == nil {
		m = make(map[uint16]eventCbFunc)
		p2p.onEventCbFuncs[evt] = m
	}
	m[k] = evtFunc
}

func (p2p *PeerToPeer) unsetEventCbFunc(k uint16) {
	for _, m := range p2p.onEventCbFuncs {
		if _, ok := m[k]; ok {
			delete(m, k)
		}
	}
}

//callback from PeerDispatcher.onPeer
func (p2p *PeerToPeer) onPeer(p *Peer) {
	p2p.logger.Debugln("onPeer", p)
	if !p2p.rr.isAllowed(p2pRoleNone, p.ID()) {
		p2p.onEvent(p2pEventNotAllowed, p)
		p.CloseByError(fmt.Errorf("onPeer not allowed connection"))
		return
	}
	p2p.rr.onPeer(p)
	if p2p.pm.onPeer(p) && !p.In() {
		p2p.sendQuery(p)
	}
}

func (p2p *PeerToPeer) onClose(p *Peer) {
	p2p.logger.Debugln("onClose", p.CloseInfo(), p)
	p2p.pm.onClose(p)
}

func (p2p *PeerToPeer) onEvent(evt string, p *Peer) {
	//if !p2p.IsStarted() {
	//	return
	//}
	p2p.logger.Traceln("onEvent", evt, p)
	if m, ok := p2p.onEventCbFuncs[evt]; ok {
		for k, cbFunc := range m {
			pp := PeerPredicates.Protocol(module.ProtocolInfo(k))
			if pp(p) {
				cbFunc(evt, p)
			}
		}
	}
}

func (p2p *PeerToPeer) onFailure(err error, pkt *Packet, c *Counter) {
	//if !p2p.IsStarted() {
	//	return
	//}
	p2p.logger.Debugln("onFailure", err, pkt, c)
}

func (p2p *PeerToPeer) _onClose(p *Peer, removed bool) {
	p2p.as.removeData(p)
	p2p.rr.onClose(p)
	if removed {
		//clearPeerQueue
		p.WaitClose()
		for ctx := p.q.Pop(); ctx != nil; ctx = p.q.Pop() {
			c := ctx.Value(p2pContextKeyCounter).(*Counter)
			c.increaseClose()
			if atomic.LoadInt32(&c.fixed) == 1 && c.Close() == c.enqueue {
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				p2p.onFailure(ErrNotAvailable, pkt, c)
			}
		}
	}
}

//callback from Peer.receiveRoutine
func (p2p *PeerToPeer) onPacket(pkt *Packet, p *Peer) {
	//if !p2p.IsStarted() {
	//	return
	//}
	pp := PeerPredicates.Protocol(pkt.protocol)
	if !pp(p) {
		p.CloseByError(ErrNotRegisteredProtocol)
		return
	}
	if pkt.protocol.ID() == module.ProtoP2P.ID() {
		switch pkt.protocol {
		case p2pProtoControl:
			handled := p2p.pm.onPacket(pkt, p)
			if !handled {
				handled = p2p.qh.onPacket(pkt, p)
			}
			if !handled {
				p.CloseByError(ErrNotRegisteredProtocol)
			}
		case p2pProtoControlV1:
			handled := p2p.pm.onPacket(pkt, p)
			if !handled {
				handled = p2p.qhv1.onPacket(pkt, p)
			}
			if !handled {
				p.CloseByError(ErrNotRegisteredProtocol)
			}
		default:
			//cannot be reached
			p2p.logger.Infoln("onPacket", "Close, not supported p2p control protocol", pkt.protocol, pkt.subProtocol)
			p.CloseByError(ErrNotRegisteredProtocol)
			return
		}
	} else {
		if p.ConnType() == p2pConnTypeNone {
			p2p.logger.Infoln("onPacket", "Drop, undetermined PeerConnectionType", pkt.protocol, pkt.subProtocol)
			return
		}

		if p2p.ID().Equal(pkt.src) {
			p2p.logger.Infoln("onPacket", "Drop, Invalid self-src", pkt.src, pkt.protocol, pkt.subProtocol)
			return
		}

		isSourcePeer := p.ID().Equal(pkt.src)
		isOneHop := pkt.ttl != 0 || pkt.dest == p2pDestPeer
		if isOneHop && !isSourcePeer {
			p2p.logger.Infoln("onPacket", "Drop, Invalid 1hop-src:", pkt.src, ",expected:", p.ID(), pkt.protocol, pkt.subProtocol)
			return
		}

		isBroadcast := pkt.dest == p2pDestAny && pkt.ttl == 0
		if isBroadcast && isSourcePeer && !p.HasRole(p2pRoleRoot) {
			p2p.logger.Infoln("onPacket", "Drop, Not authorized", p.ID(), pkt.protocol, pkt.subProtocol)
			return
		}

		if cbFunc := p2p.onPacketCbFuncs[pkt.protocol.Uint16()]; cbFunc != nil {
			if isOneHop {
				cbFunc(pkt, p)
			} else {
				if !p2p.packetPool.PutWith(pkt, func(packet *Packet) {
					cbFunc(packet, p)
				}) {
					p2p.logger.Traceln("onPacket", "Drop, Duplicated by footer", pkt.protocol, pkt.subProtocol, pkt.hashOfPacket, p.ID())
				}
			}
		} else {
			//cannot be reached
			p2p.logger.Infoln("onPacket", "Close, not exists callback function", p.ID(), pkt.protocol, pkt.subProtocol)
			p.CloseByError(ErrNotRegisteredProtocol)
		}
	}
}

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) {
	rr := p2p.rr.resolveRole(r, p2p.ID(), false)
	if rr != r {
		msg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, r)
		p2p.logger.Debugln("setRole", msg)
	}
	if !p2p.self.EqualsRole(rr) {
		p2p.self.setRole(rr)
		p2p.as.applyPeerRole(p2p.self)
	}
}

func (p2p *PeerToPeer) updateAllowed(version int64, r PeerRoleFlag, peers ...module.PeerID) {
	p2p.rr.updateAllowed(version, r, peers...)
}

func (p2p *PeerToPeer) onAllowedPeerIDSetUpdate(s *PeerIDSet, r PeerRoleFlag) {
	switch r {
	case p2pRoleNone:
		pp := func(p *Peer) bool {
			return !s.Contains(p.ID())
		}
		ps := p2p.pm.findPeers(pp)
		for _, p := range ps {
			p2p.onEvent(p2pEventNotAllowed, p)
			p.CloseByError(fmt.Errorf("onUpdate not allowed connection"))
		}
	default:
		pp := func(p *Peer) bool {
			return p.HasRole(r) != s.Contains(p.ID())
		}
		h := func(p *Peer) {
			if p.HasRole(r) {
				p.removeRole(r)
			} else {
				p.addRole(r)
			}
			p2p.as.applyPeerRole(p)
		}
		for _, p := range p2p.pm.findPeers(pp) {
			h(p)
		}
		if pp(p2p.self) {
			h(p2p.self)
		}
	}
}

func (p2p *PeerToPeer) Role() PeerRoleFlag {
	return p2p.self.Role()
}

func (p2p *PeerToPeer) HasRole(r PeerRoleFlag) bool {
	return p2p.self.HasRole(r)
}

func (p2p *PeerToPeer) EqualsRole(r PeerRoleFlag) bool {
	return p2p.self.EqualsRole(r)
}

func (p2p *PeerToPeer) ID() module.PeerID {
	return p2p.self.ID()
}

func (p2p *PeerToPeer) NetAddress() NetAddress {
	return p2p.self.NetAddress()
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	if p.ProtocolInfos().Exists(p2pProtoControlV1) {
		p2p.qhv1.sendQuery(p)
	} else {
		p2p.qh.sendQuery(p)
	}
}

func (p2p *PeerToPeer) sendToPeers(ctx context.Context, connTypes ...PeerConnectionType) int {
	pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
	pp := PeerPredicates.Protocol(pkt.protocol)
	ps := p2p.pm.findPeers(pp, connTypes...)
	for _, p := range ps {
		if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
			p2p.logger.Infoln("sendToPeers", err, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
	return len(ps)
}

func (p2p *PeerToPeer) selectPeersFromFriends(pkt *Packet) ([]*Peer, []byte) {
	src := pkt.src

	pp := PeerPredicates.Protocol(pkt.protocol)
	ps := p2p.pm.findPeers(pp, p2pConnTypeFriend)
	nr := p2p.rr.getAllowed(p2pRoleRoot).Len() - 1
	if nr < 1 {
		nr = len(ps)
	}
	f := nr / 3
	if f < DefaultFailureNodeMin {
		f = DefaultFailureNodeMin
	}
	n := f + DefaultSelectiveFloodingAdd
	tps := make([]*Peer, n)
	lps := make([]*Peer, len(ps))
	ti, li := 0, 0

	rids, _ := NewBytesSetFromBytes(pkt.ext, DefaultSimplePeerIDSize)
	tids := NewBytesSet(DefaultSimplePeerIDSize)
	for _, p := range ps {
		if src.Equal(p.ID()) {
			continue
		}
		tb := p.ID().Bytes()[:DefaultSimplePeerIDSize]
		if !rids.Contains(tb) {
			tps[ti] = p
			ti++
			tids.Add(tb)
		} else {
			lps[li] = p
			li++
		}

		if ti >= n {
			break
		}
	}
	p2p.logger.Traceln("selectPeersFromFriends", "hash:", pkt.hashOfPacket, "src:", pkt.src, "ext:", pkt.extendInfo, "rids:", rids, "tids:", tids)
	ext := tids.Bytes()
	n = n - ti
	for i := 0; i < n && i < li; i++ {
		tps[ti] = lps[i]
		ti++
	}
	return tps[:ti], ext
}

func (p2p *PeerToPeer) sendToFriends(ctx context.Context) {
	//selective (F+1) flooding with node-list
	pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
	ps, ext := p2p.selectPeersFromFriends(pkt)
	pkt.extendInfo = newPacketExtendInfo(pkt.extendInfo.hint()+1, pkt.extendInfo.len()+len(ext))
	if len(pkt.ext) > 0 {
		ext = append(pkt.ext, ext...)
	}
	pkt.footerToBytes(true)
	pkt.ext = ext[:]
	for _, p := range ps {
		if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
			p2p.logger.Infoln("sendToFriends", err, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
}

func (p2p *PeerToPeer) sendRoutine() {
Loop:
	for {
		select {
		case <-p2p.stopCh:
			p2p.logger.Debugln("sendRoutine", "stop")
			break Loop
		case <-p2p.sendQueue.Wait():
			for {
				ctx := p2p.sendQueue.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				c := ctx.Value(p2pContextKeyCounter).(*Counter)
				_ = pkt.updateHash(false)
				r := p2p.Role()
				switch pkt.dest {
				case p2pDestPeer:
					if p := p2p.getPeerByProtocol(pkt.destPeer, pkt.protocol); p != nil {
						if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
							p2p.logger.Infoln("sendToPeer", err, pkt.protocol, pkt.subProtocol, p.ID())
						}
					}
				case p2pDestAny:
					if pkt.ttl == byte(module.BroadcastNeighbor) {
						if r.Has(p2pRoleRoot) {
							p2p.sendToPeers(ctx, p2pConnTypeFriend)
						}
						p2p.sendToPeers(ctx,
							p2pConnTypeParent, p2pConnTypeUncle,
							p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther)
					} else if pkt.ttl == byte(module.BroadcastChildren) {
						if r.Has(p2pRoleRoot) {
							p2p.sendToFriends(ctx)
						}
						p2p.sendToPeers(ctx, p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther)
					} else {
						if r.Has(p2pRoleRoot) {
							p2p.sendToFriends(ctx)
						}
						p2p.sendToPeers(ctx, p2pConnTypeChildren, p2pConnTypeOther)
						c.alternate = p2p.pm.lenPeersByProtocol(pkt.protocol, p2pConnTypeNephew)
					}
				case p2pDestRoot:
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
					} else {
						p2p.sendToPeers(ctx, p2pConnTypeParent)
						c.alternate = p2p.pm.lenPeersByProtocol(pkt.protocol, p2pConnTypeUncle)
					}
				case p2pDestSeed:
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
						if r == p2pRoleRoot {
							p2p.sendToPeers(ctx, p2pConnTypeChildren)
							c.alternate = p2p.pm.lenPeersByProtocol(pkt.protocol, p2pConnTypeNephew)
						}
					} else {
						p2p.sendToPeers(ctx, p2pConnTypeParent)
						c.alternate = p2p.pm.lenPeersByProtocol(pkt.protocol, p2pConnTypeUncle)
					}
				default:
				}

				if c.alternate < 1 {
					atomic.StoreInt32(&c.fixed, 1)
					if c.peer < 1 {
						p2p.onFailure(ErrNotAvailable, pkt, c)
					} else {
						if c.enqueue < 1 {
							if c.overflow > 0 {
								p2p.onFailure(ErrQueueOverflow, pkt, c)
							} else { //if c.duplicate == c.peer
								//flooding-end by peer-history
							}
						} else {
							if c.enqueue == c.Close() {
								p2p.onFailure(ErrNotAvailable, pkt, c)
							}
						}
					}
				} else if !p2p.alternateQueue.Push(ctx) && c.enqueue < 1 {
					atomic.StoreInt32(&c.fixed, 1)
					p2p.onFailure(ErrQueueOverflow, pkt, c)
				}
			}
		}
	}
}

func (p2p *PeerToPeer) alternateSendRoutine() {
	var m = make(map[uint64]context.Context)
	sendTicker := time.NewTicker(DefaultAlternateSendPeriod)
	defer sendTicker.Stop()
Loop:
	for {
		select {
		case <-p2p.stopCh:
			p2p.logger.Debugln("alternateSendRoutine", "stop")
			break Loop
		case <-p2p.alternateQueue.Wait():
			for {
				ctx := p2p.alternateQueue.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				if preCtx, ok := m[pkt.hashOfPacket]; ok {
					c := preCtx.Value(p2pContextKeyCounter).(*Counter)
					atomic.StoreInt32(&c.fixed, 1)
					p2p.logger.Infoln("alternateSendRoutine", "ignore duplicated packet", pkt)
				}
				m[pkt.hashOfPacket] = ctx
			}
		case <-sendTicker.C:
			for _, ctx := range m {
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				c := ctx.Value(p2pContextKeyCounter).(*Counter)
				switch pkt.dest {
				case p2pDestPeer:
				case p2pDestAny:
					c.alternate = p2p.sendToPeers(ctx, p2pConnTypeNephew)
					p2p.logger.Traceln("alternateSendRoutine", "nephews", c.alternate, pkt.protocol, pkt.subProtocol)
				case p2pDestRoot:
					c.alternate = p2p.sendToPeers(ctx, p2pConnTypeUncle)
					p2p.logger.Traceln("alternateSendRoutine", "uncles", c.alternate, pkt.protocol, pkt.subProtocol)
				case p2pDestSeed:
					r := p2p.Role()
					if !r.Has(p2pRoleRoot) {
						c.alternate = p2p.sendToPeers(ctx, p2pConnTypeUncle)
					} else if r == p2pRoleRoot {
						c.alternate = p2p.sendToPeers(ctx, p2pConnTypeNephew)
					}
				default:
				}
				delete(m, pkt.hashOfPacket)

				atomic.StoreInt32(&c.fixed, 1)
				if c.peer < 1 {
					p2p.onFailure(ErrNotAvailable, pkt, c)
				} else {
					if c.enqueue < 1 {
						if c.overflow > 0 {
							p2p.onFailure(ErrQueueOverflow, pkt, c)
						} else { //if c.duplicate == c.peer
							//flooding-end by peer-history
						}
					} else {
						if c.enqueue == c.Close() {
							p2p.onFailure(ErrNotAvailable, pkt, c)
						}
					}
				}
			}
		}
	}
}

func (p2p *PeerToPeer) Send(pkt *Packet) error {
	if !p2p.IsStarted() {
		return ErrNotStarted
	}

	if pkt.src == nil {
		pkt.src = p2p.ID()
	}

	if pkt.dest == p2pDestAny && pkt.ttl == 0 &&
		p2p.ID().Equal(pkt.src) &&
		!p2p.HasRole(p2pRoleRoot) {
		//BroadcastAll && not relay && not has p2pRoleRoot
		return ErrNotAuthorized
	}

	if !p2p.available(pkt) {
		if pkt.dest == p2pDestAny && pkt.ttl == 0 &&
			p2p.EqualsRole(p2pRoleNone) {
			return nil
		}
		//p2p.logger.Infoln("Send", "Not Available", pkt.dest, pkt.protocol, pkt.subProtocol)
		return ErrNotAvailable
	}

	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, &Counter{})
	if ok := p2p.sendQueue.Push(ctx, int(pkt.protocol.ID())); !ok {
		p2p.logger.Infoln("Send", "Queue Push failure", pkt.protocol, pkt.subProtocol)
		return ErrQueueOverflow
	}
	return nil
}

type p2pContextKey string

var (
	p2pContextKeyPacket  = p2pContextKey("packet")
	p2pContextKeyPeer    = p2pContextKey("peer")
	p2pContextKeyEvent   = p2pContextKey("event")
	p2pContextKeyCounter = p2pContextKey("counter")
)

type Counter struct {
	peer      int
	alternate int
	fixed     int32 //no more change peer and alternate
	//
	enqueue   int
	duplicate int
	overflow  int
	//
	close int
	mtx   sync.RWMutex
}

func (c *Counter) String() string {
	return fmt.Sprintf("{peer:%d,alt:%d,enQ:%d,dup:%d,of:%d,close:%d}",
		c.peer, c.alternate, c.enqueue, c.duplicate, c.overflow, c.Close())
}

func (c *Counter) increaseClose() {
	defer c.mtx.Unlock()
	c.mtx.Lock()
	c.close++
}

func (c *Counter) Close() int {
	defer c.mtx.RUnlock()
	c.mtx.RLock()
	return c.close
}

func (p2p *PeerToPeer) getPeerByProtocol(id module.PeerID, pi module.ProtocolInfo) (p *Peer) {
	return p2p.pm.findPeer(PeerPredicates.IDAndProtocol(id, pi), joinPeerConnectionTypes...)
}

func (p2p *PeerToPeer) getPeers() []*Peer {
	return p2p.pm.findPeers(nil, joinPeerConnectionTypes...)
}

func (p2p *PeerToPeer) getPeersByProtocol(pi module.ProtocolInfo) []*Peer {
	return p2p.pm.findPeers(PeerPredicates.Protocol(pi), joinPeerConnectionTypes...)
}

func (p2p *PeerToPeer) available(pkt *Packet) bool {
	r := p2p.Role()
	var connTypes []PeerConnectionType
	switch pkt.dest {
	case p2pDestPeer:
		p := p2p.getPeerByProtocol(pkt.destPeer, pkt.protocol)
		return p != nil
	case p2pDestAny:
		connTypes = []PeerConnectionType{p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther}
		if r.Has(p2pRoleRoot) {
			connTypes = append(connTypes, p2pConnTypeFriend)
		}
		if pkt.ttl == byte(module.BroadcastNeighbor) {
			connTypes = append(connTypes, p2pConnTypeParent, p2pConnTypeUncle)
		}
	case p2pDestRoot:
		if r.Has(p2pRoleRoot) {
			connTypes = []PeerConnectionType{p2pConnTypeFriend}
		} else {
			connTypes = []PeerConnectionType{p2pConnTypeParent, p2pConnTypeUncle}
		}
	case p2pDestSeed:
		if r.Has(p2pRoleRoot) {
			connTypes = []PeerConnectionType{p2pConnTypeChildren, p2pConnTypeNephew}
		} else {
			connTypes = []PeerConnectionType{p2pConnTypeParent, p2pConnTypeUncle}
		}
	default:
		connTypes = joinPeerConnectionTypes[:]
	}
	if len(connTypes) == 0 {
		return false
	}
	return p2p.pm.lenPeersByProtocol(pkt.protocol, connTypes...) > 0
}

//Dial to seeds, roots, nodes and create p2p connection
func (p2p *PeerToPeer) discoverRoutine() {
	discoveryTicker := time.NewTicker(DefaultDiscoveryPeriod)
	seedTicker := time.NewTicker(DefaultSeedPeriod)
	defer func() {
		seedTicker.Stop()
		discoveryTicker.Stop()
	}()
	for na, _ := range p2p.rr.getTrustSeedsMap() {
		p2p.logger.Debugln("discoverRoutine", "initialize", "dial to trustSeed", na)
		p2p.dial(na)
	}
Loop:
	for {
		select {
		case <-p2p.stopCh:
			p2p.logger.Debugln("discoverRoutine", "stop")
			break Loop
		case <-seedTicker.C:
			r := p2p.Role()
			if p2p.query(r) {
				dialed := p2p.as.dial(p2pRoleSeed)
				if r.Has(p2pRoleSeed) || dialed == 0 {
					//dial to trustSeeds
					for na, d := range p2p.rr.getTrustSeedsMap() {
						if len(d) != 0 {
							na = NetAddress(d)
						}
						if !p2p.as.contains(p2pRoleSeed, na) &&
							!p2p.pm.hasNetAddress(na) {
							p2p.logger.Debugln("discoverRoutine", "seedTicker", "dial to trustSeed", na)
							p2p.dial(na)
						}
					}
				}
			} else {
				pp := PeerPredicates.InAndRole(false, p2pRoleSeed)
				outSeeds := p2p.pm.findPeers(pp, p2pConnTypeNone)
				for _, p := range outSeeds {
					p2p.logger.Debugln("discoverRoutine", "seedTicker", "no need outgoing p2pRoleSeed connection")
					p.Close("discoverRoutine no need outgoing p2pRoleSeed connection")
				}
			}
		case <-discoveryTicker.C:
			r := p2p.Role()
			if r.Has(p2pRoleRoot) {
				p2p.discoverFriends()
			} else {
				rr := p2pRoleSeed
				if r == p2pRoleSeed {
					rr = p2pRoleRoot
				}

				for _, p := range p2p.pm.findPeers(nil, p2pConnTypeFriend) {
					p2p.pm.tryTransitPeerConnection(p, p2pConnTypeNone)
				}

				complete := p2p.discoverParents(rr)
				if complete {
					complete = p2p.discoverUncles(rr)
				}
				if !complete {
					p2p.as.dial(rr)
				}
			}
		}
	}
}

func (p2p *PeerToPeer) query(r PeerRoleFlag) (needMoreSeeds bool) {
	ps := make([]*Peer, 0)
	if r.Has(p2pRoleRoot) {
		friends := p2p.pm.findPeers(nil, p2pConnTypeFriend)
		for _, p := range friends {
			if !p.In() {
				ps = append(ps, p)
			}
		}
		ps = append(ps, p2p.pm.findPeers(PeerPredicates.In(false), p2pConnTypeOther)...)

		numOfFailureNode := (p2p.rr.getAllowed(p2pRoleRoot).Len() - 1) / 3
		needMoreSeeds = (2*numOfFailureNode) > len(friends) || (p2p.pm.lenPeers(p2pConnTypeChildren, p2pConnTypeNephew) < 1)

		if len(ps) < numOfFailureNode {
			for _, p := range friends {
				if numOfFailureNode <= len(ps) {
					break
				}
				if p.In() {
					ps = append(ps, p)
				}
			}
		}
	} else {
		ps = p2p.pm.findPeers(nil, p2pConnTypeParent, p2pConnTypeUncle)
		if r == p2pRoleSeed {
			ps = append(ps, p2p.pm.findPeers(PeerPredicates.InAndHasRole(false, p2pRoleRoot), p2pConnTypeNone)...)
			ps = append(ps, p2p.pm.findPeers(PeerPredicates.In(false), p2pConnTypeOther)...)
		}
		needMoreSeeds = p2p.pm.getConnectionAvailable(p2pConnTypeParent) > 0 ||
			p2p.pm.getConnectionAvailable(p2pConnTypeUncle) > 0
	}

	if needMoreSeeds {
		ps = append(ps, p2p.pm.findPeers(PeerPredicates.InAndHasRole(false, p2pRoleSeed), p2pConnTypeNone)...)
	}
	for _, p := range ps {
		p2p.sendQuery(p)
	}
	return needMoreSeeds
}

func (p2p *PeerToPeer) discoverFriends() {
	hasRole := PeerPredicates.HasRole(p2pRoleRoot)
	ps := p2p.pm.findPeers(hasRole.Not(), p2pConnTypeFriend)
	for _, p := range ps {
		if p.HasRole(p2pRoleSeed) {
			if p2p.pm.tryTransitPeerConnection(p, p2pConnTypeNone) {
				p2p.logger.Debugln("discoverFriends", "not allowed friend connection", p.id)
			}
		} else {
			p2p.logger.Debugln("discoverFriends", "not allowed connection", p.id)
			p.Close("discoverFriends not allowed connection")
		}
	}

	candidates := p2p.pm.findPeers(hasRole,
		p2pConnTypeNone, p2pConnTypeParent, p2pConnTypeUncle, p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther)
	for _, p := range candidates {
		if p2p.pm.tryTransitPeerConnection(p, p2pConnTypeFriend) {
			p2p.logger.Debugln("discoverFriends", "try p2pConnTypeFriend", p.ID(), p.ConnType())
		}
	}

	p2p.as.dial(p2pRoleRoot)
}

func (p2p *PeerToPeer) setTrustSeeds(seeds []NetAddress) {
	p2p.rr.setTrustSeeds(seeds)
}

func (p2p *PeerToPeer) discoverParents(pr PeerRoleFlag) (complete bool) {
	hasRole := PeerPredicates.HasRole(pr)
	ps := p2p.pm.findPeers(hasRole.Not(), p2pConnTypeParent)
	for _, p := range ps {
		if !(pr == p2pRoleSeed && p2p.rr.isTrustSeed(p)) {
			p2p.logger.Debugln("discoverParents", "not allowed connection", p.id)
			p.Close("discoverParents not allowed connection")
		}
	}

	n := p2p.pm.getConnectionAvailable(p2pConnTypeParent)
	if n < 1 {
		p2p.logger.Traceln("discoverParents", "nothing to do")
		return true
	}

	limit := p2p.pm.getConnectionLimit(p2pConnTypeChildren)
	pp := PeerPredicate(func(p *Peer) bool {
		return p.Conns(p2pConnTypeChildren).Len() < limit
	})
	if pr == p2pRoleSeed {
		pp = pp.And(PeerPredicates.In(false)).And(hasRole.Or(p2p.rr.isTrustSeed))
	} else {
		pp = pp.And(hasRole)
	}
	candidates := p2p.pm.findPeers(pp, p2pConnTypeNone, p2pConnTypeUncle)
	try := 0
	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			avg1 := candidates[i].rtt.Avg(time.Millisecond)
			avg2 := candidates[j].rtt.Avg(time.Millisecond)
			if avg1 < avg2 {
				return true
			} else if avg1 == avg2 {
				return candidates[i].Conns(p2pConnTypeChildren).Len() < candidates[j].Conns(p2pConnTypeChildren).Len()
			}
			return false
		})
		for _, p := range candidates {
			if try == n {
				return false
			}
			if p2p.pm.tryTransitPeerConnection(p, p2pConnTypeParent) {
				p2p.logger.Debugln("discoverParents", "try p2pConnTypeParent", p.ID(), p.ConnType())
				try++
			}
		}
	}
	if try == 0 {
		p2p.pm.clearReject()
	}
	return false
}

func (p2p *PeerToPeer) discoverUncles(ur PeerRoleFlag) (complete bool) {
	hasRole := PeerPredicates.HasRole(ur)
	ps := p2p.pm.findPeers(hasRole.Not(), p2pConnTypeUncle)
	for _, p := range ps {
		if !(ur == p2pRoleSeed && p2p.rr.isTrustSeed(p)) {
			p2p.logger.Debugln("discoverUncles", "not allowed connection", p.id)
			p.Close("discoverUncles not allowed connection")
		}
	}

	n := p2p.pm.getConnectionAvailable(p2pConnTypeUncle)
	if n < 1 {
		p2p.logger.Traceln("discoverUncles", "nothing to do")
		return true
	}

	limit := p2p.pm.getConnectionLimit(p2pConnTypeNephew)
	pp := PeerPredicate(func(p *Peer) bool {
		return p.Conns(p2pConnTypeNephew).Len() < limit
	})
	if ur == p2pRoleSeed {
		pp = pp.And(PeerPredicates.In(false)).And(hasRole.Or(p2p.rr.isTrustSeed))
	} else {
		pp = pp.And(hasRole)
	}
	candidates := p2p.pm.findPeers(pp, p2pConnTypeNone)
	try := 0
	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			avg1 := candidates[i].rtt.Avg(time.Millisecond)
			avg2 := candidates[j].rtt.Avg(time.Millisecond)
			if avg1 < avg2 {
				return true
			} else if avg1 == avg2 {
				return candidates[i].Conns(p2pConnTypeNephew).Len() < candidates[j].Conns(p2pConnTypeNephew).Len()
			}
			return false
		})
		for _, p := range candidates {
			if try == n {
				return false
			}
			if p2p.pm.tryTransitPeerConnection(p, p2pConnTypeUncle) {
				p2p.logger.Debugln("discoverUncles", "try p2pConnTypeUncle", p.ID(), p.ConnType())
				try++
			}
		}
	}

	if try == 0 {
		p2p.pm.clearReject()
	}
	return false
}

func (p2p *PeerToPeer) setConnectionLimit(connType PeerConnectionType, v int) {
	p2p.pm.setConnectionLimit(connType, v)
}
