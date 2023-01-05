package network

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	DefaultReceiveQueueSize     = 1000
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
	DefaultSelectiveFloodingAdd = 1
	DefaultSimplePeerIDSize     = 4
	DefaultDuplicatedPeerTime   = 1 * time.Second
	DefaultMaxRetryClose        = 10
	AttrP2PConnectionRequest    = "P2PConnectionRequest"
	AttrP2PLegacy               = "P2PLegacy"
	AttrSupportDefaultProtocols = "SupportDefaultProtocols"
	DefaultQueryElementLength   = 200
)

var (
	p2pProtoControl     = module.ProtoP2P
	p2pControlProtocols = []module.ProtocolInfo{p2pProtoControl}
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

	//Topology with Connected Peers
	self       *Peer
	m          map[PeerConnectionType]*PeerSet
	transiting *PeerSet
	reject     *PeerSet
	connMtx    sync.RWMutex

	//NetAddresses  //if value of map is duplicated, then old will be removed.
	trustSeeds *NetAddressSet //map[DialNetAddress]NetAddress
	seeds      *NetAddressSet //map[NetAddress]PeerID
	roots      *NetAddressSet //map[NetAddress]PeerID //Only for seed and root

	//managed PeerId
	allowedRoots *PeerIDSet
	allowedSeeds *PeerIDSet
	allowedPeers *PeerIDSet

	//connection limit
	cLimit    map[PeerConnectionType]int
	cLimitMtx sync.RWMutex

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

func newPeerToPeer(channel string, self *Peer, d *Dialer, mtr *metric.NetworkMetric, l log.Logger) *PeerToPeer {
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
		self:       self,
		m:          make(map[PeerConnectionType]*PeerSet),
		transiting: NewPeerSet(),
		reject:     NewPeerSet(),
		//
		trustSeeds: NewNetAddressSet(),
		seeds:      NewNetAddressSet(),
		roots:      NewNetAddressSet(),
		//
		allowedRoots: NewPeerIDSet(),
		allowedSeeds: NewPeerIDSet(),
		allowedPeers: NewPeerIDSet(),
		//
		cLimit: make(map[PeerConnectionType]int),
		//
		mtr: mtr,
	}
	for connType := p2pConnTypeNone; connType < p2pConnTypeReserved; connType++ {
		p2p.m[connType] = NewPeerSet()
	}
	p2p.allowedRoots.onUpdate = func(s *PeerIDSet) {
		p2p.onAllowedPeerIDSetUpdate(s, p2pRoleRoot)
	}
	p2p.allowedSeeds.onUpdate = func(s *PeerIDSet) {
		p2p.onAllowedPeerIDSetUpdate(s, p2pRoleSeed)
	}
	p2p.allowedPeers.onUpdate = func(s *PeerIDSet) {
		p2p.onAllowedPeerIDSetUpdate(s, p2pRoleNone)
	}
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
			ps := p2p.findPeers(nil)
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
	if !p2p.allowedPeers.IsEmpty() && !p2p.allowedPeers.Contains(p.ID()) {
		p2p.onEvent(p2pEventNotAllowed, p)
		p.CloseByError(fmt.Errorf("onPeer not allowed connection"))
		return
	}
	if p2p.isTrustSeed(p) {
		p2p.trustSeeds.SetAndRemoveByData(p.DialNetAddress(), string(p.NetAddress()))
	}
	if dp := p2p.getPeer(p.ID(), false); dp != nil {
		p2p.onEvent(p2pEventDuplicate, p)

		//'b' is higher (ex : 'b' > 'a'), disconnect lower.outgoing
		higher := strings.Compare(p2p.ID().String(), p.ID().String()) > 0
		diff := p.timestamp.Sub(dp.timestamp)

		if diff < DefaultDuplicatedPeerTime && dp.In() != p.In() && higher == p.In() {
			//close new which is lower outgoing
			p.CloseByError(ErrDuplicatedPeer)
			p2p.logger.Infoln("Already exists connected Peer, close new", p, diff)
			return
		}
		//close old
		dp.CloseByError(ErrDuplicatedPeer)
		p2p.logger.Infoln("Already exists connected Peer, close old", dp, diff)
	}
	if p2p.addPeer(p) && !p.In() {
		p2p.sendQuery(p)
	}
}

func (p2p *PeerToPeer) onClose(p *Peer) {
	p2p.logger.Debugln("onClose", p.CloseInfo(), p)
	if p2p.removePeer(p) {
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

func (p2p *PeerToPeer) onEvent(evt string, p *Peer) {
	//if !p2p.IsStarted() {
	//	return
	//}
	p2p.logger.Traceln("onEvent", evt, p)
	if m, ok := p2p.onEventCbFuncs[evt]; ok {
		for k, cbFunc := range m {
			if p.ProtocolInfos().Exists(module.ProtocolInfo(k)) {
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

func (p2p *PeerToPeer) removePeer(p *Peer) bool {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	if p.HasRole(p2pRoleRoot) {
		p2p.roots.RemoveData(p.NetAddress())
	}
	if p.HasRole(p2pRoleSeed) {
		p2p.seeds.RemoveData(p.NetAddress())
	}
	if p2p.isTrustSeed(p) {
		p2p.trustSeeds.RemoveData(p.DialNetAddress())
	}
	ok := p2p._removePeer(p)
	if ok && p.ConnType() != p2pConnTypeNone {
		p2p.onEvent(p2pEventLeave, p)
	}
	return ok
}

//callback from Peer.receiveRoutine
func (p2p *PeerToPeer) onPacket(pkt *Packet, p *Peer) {
	//if !p2p.IsStarted() {
	//	return
	//}
	if !p.ProtocolInfos().Exists(pkt.protocol) {
		p.CloseByError(ErrNotRegisteredProtocol)
		return
	}
	if pkt.protocol.ID() == p2pProtoControl.ID() {
		switch pkt.protocol {
		case p2pProtoControl:
			switch pkt.subProtocol {
			case p2pProtoQueryReq: //roots, seeds, children
				p2p.handleQuery(pkt, p)
			case p2pProtoQueryResp:
				p2p.handleQueryResult(pkt, p)
			case p2pProtoRttReq: //roots, seeds, children
				p2p.handleRttRequest(pkt, p)
			case p2pProtoRttResp:
				p2p.handleRttResponse(pkt, p)
			case p2pProtoConnReq:
				p2p.handleP2PConnectionRequest(pkt, p)
			case p2pProtoConnResp:
				p2p.handleP2PConnectionResponse(pkt, p)
			default:
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
			if isOneHop || p2p.packetPool.Put(pkt) {
				cbFunc(pkt, p)
			} else {
				p2p.logger.Traceln("onPacket", "Drop, Duplicated by footer", pkt.protocol, pkt.subProtocol, pkt.hashOfPacket, p.ID())
			}
		} else {
			//cannot be reached
			p2p.logger.Infoln("onPacket", "Close, not exists callback function", p.ID(), pkt.protocol, pkt.subProtocol)
			p.CloseByError(ErrNotRegisteredProtocol)
		}
	}
}

type QueryMessage struct {
	Role PeerRoleFlag
}

type QueryResultMessage struct {
	Role     PeerRoleFlag
	Seeds    []NetAddress
	Roots    []NetAddress
	Children []NetAddress
	Nephews  []NetAddress
	Message  string
}

type RttMessage struct {
	Last    time.Duration
	Average time.Duration
}

func (p2p *PeerToPeer) applyPeerRole(p *Peer) {
	r := p.Role()
	if r.Has(p2pRoleSeed) {
		c, o := p2p.seeds.SetAndRemoveByData(p.NetAddress(), p.ID().String())
		if o != "" {
			p2p.logger.Debugln("applyPeerRole", "addSeed", "updated NetAddress old:", o, ", now:", p.NetAddress(), ",peerID:", p.ID())
		}
		if c != "" {
			p2p.logger.Infoln("applyPeerRole", "addSeed", "conflict NetAddress", p.NetAddress(), "removed:", c, ",now:", p.ID())
		}
	} else {
		p2p.seeds.Remove(p.NetAddress())
	}
	if r.Has(p2pRoleRoot) {
		c, o := p2p.roots.SetAndRemoveByData(p.NetAddress(), p.ID().String())
		if o != "" {
			p2p.logger.Debugln("applyPeerRole", "addRoot", "updated NetAddress old:", o, ", now:", p.NetAddress(), ",peerID:", p.ID())
		}
		if c != "" {
			p2p.logger.Infoln("applyPeerRole", "addRoot", "conflict NetAddress", p.NetAddress(), "removed:", c, ",now:", p.ID())
		}
	} else {
		p2p.roots.Remove(p.NetAddress())
	}
}

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) {
	rr := p2p.resolveRole(r, p2p.ID(), false)
	if rr != r {
		msg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, r)
		p2p.logger.Debugln("setRole", msg)
	}
	if !p2p.self.EqualsRole(rr) {
		p2p.self.setRole(rr)
		p2p.applyPeerRole(p2p.self)
	}
}

func (p2p *PeerToPeer) getAllowed(role module.Role) *PeerIDSet {
	switch role {
	case module.ROLE_VALIDATOR:
		return p2p.allowedRoots
	case module.ROLE_SEED:
		return p2p.allowedSeeds
	case module.ROLE_NORMAL:
		return p2p.allowedPeers
	default:
		return nil
	}
}

func (p2p *PeerToPeer) onAllowedPeerIDSetUpdate(s *PeerIDSet, r PeerRoleFlag) {
	switch r {
	case p2pRoleNone:
		ps := p2p.findPeers(func(p *Peer) bool {
			return !s.Contains(p.ID())
		})
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
			p2p.applyPeerRole(p)
		}
		for _, p := range p2p.findPeers(pp) {
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

func (p2p *PeerToPeer) startRtt(p *Peer) {
	p.rtt.StartWithAfterFunc(DefaultRttLogTimeout, func() {
		p2p.logger.Warnln("RTT Timeout", DefaultRttLogTimeout, p)
	})
}

func (p2p *PeerToPeer) stopRtt(p *Peer) time.Duration {
	rttLast := p.rtt.Stop()
	if rttLast >= DefaultRttLogThreshold {
		p2p.logger.Warnln("RTT Threshold", DefaultRttLogThreshold, p)
	}
	return rttLast
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.Role()}
	pkt := newPacket(p2pProtoControl, p2pProtoQueryReq, p2p.encode(m), p2p.ID())
	pkt.destPeer = p.ID()
	err := p.sendPacket(pkt)
	if err != nil {
		p2p.logger.Infoln("sendQuery", err, p)
	} else {
		p2p.startRtt(p)
		p2p.logger.Traceln("sendQuery", m, p)
	}
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := p2p.decode(pkt.payload, qm)
	if err != nil {
		p2p.logger.Infoln("handleQuery", err, p)
		return
	}
	p2p.logger.Traceln("handleQuery", qm, p)

	r := p2p.Role()
	m := &QueryResultMessage{
		Role:     r,
		Children: p2p.getNetAddresses(p2pConnTypeChildren),
		Nephews:  p2p.getNetAddresses(p2pConnTypeNephew),
	}
	rr := p2p.resolveRole(qm.Role, p.ID(), true)
	if rr != qm.Role {
		m.Message = fmt.Sprintf("not equal resolved role %d, expected %d", rr, qm.Role)
		p2p.logger.Infoln("handleQuery", m.Message, p)
	}
	p.setRecvRole(qm.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		p2p.applyPeerRole(p)
	}
	if rr.Has(p2pRoleSeed) || rr.Has(p2pRoleRoot) {
		m.Roots = p2p.roots.Array()
		m.Seeds = p2p.seeds.Array()
	} else {
		if r.Has(p2pRoleRoot) {
			p2p.logger.Infoln("handleQuery", "not allowed connection", p)
			p.Close("handleQuery not allowed connection")
			return
		}
		m.Seeds = make([]NetAddress, 0)
		for _, s := range p2p.seeds.Array() {
			if !p2p.roots.Contains(s) {
				m.Seeds = append(m.Seeds, s)
			}
		}
	}

	//prevent propagation of addresses via normal nodes
	if r == p2pRoleNone {
		m.Roots = m.Roots[:0]
		m.Seeds = m.Seeds[:0]
	}

	if len(m.Roots) > DefaultQueryElementLength {
		m.Roots = m.Roots[:DefaultQueryElementLength]
	}
	if len(m.Seeds) > DefaultQueryElementLength {
		m.Seeds = m.Seeds[:DefaultQueryElementLength]
	}

	rpkt := newPacket(p2pProtoControl, p2pProtoQueryResp, p2p.encode(m), p2p.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.logger.Infoln("handleQuery", "sendQueryResult", err, p)
	} else {
		p2p.startRtt(p)
		p2p.logger.Traceln("handleQuery", "sendQueryResult", m, p)
	}
}

func (p2p *PeerToPeer) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := p2p.decode(pkt.payload, qrm)
	if err != nil {
		p2p.logger.Infoln("handleQueryResult", err, p)
		return
	}
	p2p.stopRtt(p)
	if len(qrm.Roots) > DefaultQueryElementLength {
		p2p.logger.Infoln("handleQueryResult", "invalid Roots Length:", len(qrm.Roots), p)
		qrm.Roots = qrm.Roots[:DefaultQueryElementLength]
	}
	if len(qrm.Seeds) > DefaultQueryElementLength {
		p2p.logger.Infoln("handleQueryResult", "invalid Seeds Length:", len(qrm.Seeds), p)
		qrm.Seeds = qrm.Seeds[:DefaultQueryElementLength]
	}
	if len(qrm.Children) > DefaultQueryElementLength {
		p2p.logger.Infoln("handleQueryResult", "invalid Children Length:", len(qrm.Children), p)
		qrm.Children = qrm.Children[:DefaultQueryElementLength]
	}
	if len(qrm.Nephews) > DefaultQueryElementLength {
		p2p.logger.Infoln("handleQueryResult", "invalid Nephews Length:", len(qrm.Nephews), p)
		qrm.Nephews = qrm.Nephews[:DefaultQueryElementLength]
	}
	p2p.logger.Traceln("handleQueryResult", qrm, p)

	p.children.ClearAndAdd(qrm.Children...)
	p.nephews.ClearAndAdd(qrm.Nephews...)

	rr := p2p.resolveRole(qrm.Role, p.ID(), true)
	if rr != qrm.Role {
		msg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, qrm.Role)
		p2p.logger.Infoln("handleQueryResult", msg, p)
	}
	p.setRecvRole(qrm.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		p2p.applyPeerRole(p)
	}
	if !rr.Has(p2pRoleSeed) && !rr.Has(p2pRoleRoot) {
		if !p2p.isTrustSeed(p) {
			p2p.logger.Infoln("handleQueryResult", "invalid query, not allowed connection", p)
			p.CloseByError(fmt.Errorf("handleQueryResult invalid query, resolved role %d", rr))
			return
		}
	}

	r := p2p.Role()
	if r.Has(p2pRoleSeed) || r.Has(p2pRoleRoot) {
		roots := make([]NetAddress, 0)
		for _, na := range qrm.Roots {
			if d, ok := p2p.seeds.Data(na); !ok || len(d) == 0 {
				roots = append(roots, na)
			}
		}
		p2p.roots.Merge(roots...)
	}
	seeds := make([]NetAddress, 0)
	for _, na := range qrm.Seeds {
		if d, ok := p2p.seeds.Data(na); !ok || len(d) == 0 {
			seeds = append(seeds, na)
		}
	}
	p2p.seeds.Merge(seeds...)

	last, avg := p.rtt.Value()
	m := &RttMessage{Last: last, Average: avg}
	rpkt := newPacket(p2pProtoControl, p2pProtoRttReq, p2p.encode(m), p2p.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.logger.Infoln("handleQueryResult", "sendRttRequest", err, p)
	} else {
		p2p.logger.Traceln("handleQueryResult", "sendRttRequest", m, p)
	}
}

func (p2p *PeerToPeer) handleRttRequest(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := p2p.decode(pkt.payload, rm)
	if err != nil {
		p2p.logger.Infoln("handleRttRequest", err, p)
		return
	}
	p2p.logger.Traceln("handleRttRequest", rm, p)
	rttLast := p2p.stopRtt(p)

	df := rm.Last - rttLast
	if df > DefaultRttAccuracy {
		p2p.logger.Debugln("handleRttRequest", df, "DefaultRttAccuracy", DefaultRttAccuracy, p)
	}
	last, avg := p.rtt.Value()
	m := &RttMessage{Last: last, Average: avg}
	rpkt := newPacket(p2pProtoControl, p2pProtoRttResp, p2p.encode(m), p2p.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.logger.Infoln("handleRttRequest", "sendRttResponse", err, p)
	} else {
		p2p.logger.Traceln("handleRttRequest", "sendRttResponse", m, p)
	}
}

func (p2p *PeerToPeer) handleRttResponse(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := p2p.decode(pkt.payload, rm)
	if err != nil {
		p2p.logger.Infoln("handleRttResponse", err, p)
		return
	}
	p2p.logger.Traceln("handleRttResponse", rm, p)

	rttLast, _ := p.rtt.Value()
	df := rm.Last - rttLast
	if df > DefaultRttAccuracy {
		p2p.logger.Debugln("handleRttResponse", df, "DefaultRttAccuracy", DefaultRttAccuracy, p)
	}
}

func (p2p *PeerToPeer) sendToPeers(ctx context.Context, connTypes ...PeerConnectionType) int {
	pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
	ps := p2p.findPeers(func(p *Peer) bool {
		return p.ProtocolInfos().Exists(pkt.protocol)
	}, connTypes...)
	for _, p := range ps {
		if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
			p2p.logger.Infoln("sendToPeers", err, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
	return len(ps)
}

func (p2p *PeerToPeer) selectPeersFromFriends(pkt *Packet) ([]*Peer, []byte) {
	src := pkt.src

	ps := p2p.findPeers(func(p *Peer) bool {
		return p.ProtocolInfos().Exists(pkt.protocol)
	}, p2pConnTypeFriend)
	nr := p2p.allowedRoots.Len() - 1
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
					if pkt.ttl == byte(module.BROADCAST_NEIGHBOR) {
						if r.Has(p2pRoleRoot) {
							p2p.sendToPeers(ctx, p2pConnTypeFriend)
						}
						p2p.sendToPeers(ctx,
							p2pConnTypeParent, p2pConnTypeUncle,
							p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther)
					} else if pkt.ttl == byte(module.BROADCAST_CHILDREN) {
						if r.Has(p2pRoleRoot) {
							p2p.sendToFriends(ctx)
						}
						p2p.sendToPeers(ctx, p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther)
					} else {
						if r.Has(p2pRoleRoot) {
							p2p.sendToFriends(ctx)
						}
						p2p.sendToPeers(ctx, p2pConnTypeChildren, p2pConnTypeOther)
						c.alternate = p2p.lenPeersByProtocol(pkt.protocol, p2pConnTypeNephew)
					}
				case p2pDestRoot:
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
					} else {
						p2p.sendToPeers(ctx, p2pConnTypeParent)
						c.alternate = p2p.lenPeersByProtocol(pkt.protocol, p2pConnTypeUncle)
					}
				case p2pDestSeed:
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
						if r == p2pRoleRoot {
							p2p.sendToPeers(ctx, p2pConnTypeChildren)
							c.alternate = p2p.lenPeersByProtocol(pkt.protocol, p2pConnTypeNephew)
						}
					} else {
						p2p.sendToPeers(ctx, p2pConnTypeParent)
						c.alternate = p2p.lenPeersByProtocol(pkt.protocol, p2pConnTypeUncle)
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
		//BROADCAST_ALL && not relay && not has p2pRoleRoot
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

func (p2p *PeerToPeer) getPeer(id module.PeerID, onlyJoin bool) (p *Peer) {
	if id == nil {
		return nil
	}

	if onlyJoin {
		return p2p.findPeer(func(p *Peer) bool {
			return p.ID().Equal(id)
		}, joinPeerConnectionTypes...)
	} else {
		return p2p.findPeer(func(p *Peer) bool {
			return p.ID().Equal(id)
		})
	}
}

func (p2p *PeerToPeer) getPeerByProtocol(id module.PeerID, pi module.ProtocolInfo) (p *Peer) {
	if p = p2p.getPeer(id, true); p == nil || !p.ProtocolInfos().Exists(pi) {
		return nil
	}
	return p
}

func (p2p *PeerToPeer) getPeers() []*Peer {
	return p2p.findPeers(nil, joinPeerConnectionTypes...)
}

func (p2p *PeerToPeer) getPeersByProtocol(pi module.ProtocolInfo) []*Peer {
	return p2p.findPeers(func(p *Peer) bool {
		return p.ProtocolInfos().Exists(pi)
	}, joinPeerConnectionTypes...)
}

func (p2p *PeerToPeer) hasNetAddress(na NetAddress) bool {
	p2p.mtx.RLock()
	defer p2p.mtx.RUnlock()
	if p2p.NetAddress() == na {
		return true
	}
	for _, v := range p2p.m {
		if v.HasNetAddress(na) {
			return true
		}
	}
	return false
}

func (p2p *PeerToPeer) available(pkt *Packet) bool {
	r := p2p.Role()
	switch pkt.dest {
	case p2pDestPeer:
		if p := p2p.getPeerByProtocol(pkt.destPeer, pkt.protocol); p == nil {
			return false
		}
	case p2pDestAny:
		connTypes := []PeerConnectionType{
			p2pConnTypeChildren, p2pConnTypeNephew, p2pConnTypeOther,
		}
		if r.Has(p2pRoleRoot) {
			connTypes = append(connTypes, p2pConnTypeFriend)
		}
		if pkt.ttl == byte(module.BROADCAST_NEIGHBOR) {
			connTypes = append(connTypes, p2pConnTypeParent, p2pConnTypeUncle)
		}
		if p2p.lenPeersByProtocol(pkt.protocol, connTypes...) < 1 {
			return false
		}
	case p2pDestRoot:
		var connTypes []PeerConnectionType
		if p2p.Role().Has(p2pRoleRoot) {
			connTypes = []PeerConnectionType{p2pConnTypeFriend}
		} else {
			connTypes = []PeerConnectionType{p2pConnTypeParent, p2pConnTypeUncle}
		}
		if p2p.lenPeersByProtocol(pkt.protocol, connTypes...) < 1 {
			return false
		}
	//case p2pDestSeed:
	default:
		if p2p.lenPeersByProtocol(pkt.protocol, joinPeerConnectionTypes...) < 1 {
			return false
		}
	}
	return true
}

func (p2p *PeerToPeer) resolveRole(r PeerRoleFlag, id module.PeerID, onlyUnSet bool) PeerRoleFlag {
	if onlyUnSet {
		if r.Has(p2pRoleRoot) && !p2p.allowedRoots.IsEmpty() && !p2p.allowedRoots.Contains(id) {
			r.UnSetFlag(p2pRoleRoot)
		}
		if r.Has(p2pRoleSeed) && !p2p.allowedSeeds.IsEmpty() && !p2p.allowedSeeds.Contains(id) {
			r.UnSetFlag(p2pRoleSeed)
		}
	} else {
		if p2p.allowedRoots.Contains(id) {
			r.SetFlag(p2pRoleRoot)
		} else if r.Has(p2pRoleRoot) && !p2p.allowedSeeds.IsEmpty() {
			r.UnSetFlag(p2pRoleRoot)
		}
		if p2p.allowedSeeds.Contains(id) {
			r.SetFlag(p2pRoleSeed)
		} else if r.Has(p2pRoleSeed) && !p2p.allowedSeeds.IsEmpty() {
			r.UnSetFlag(p2pRoleSeed)
		}
	}
	return r
}

//Dial to seeds, roots, nodes and create p2p connection
func (p2p *PeerToPeer) discoverRoutine() {
	discoveryTicker := time.NewTicker(DefaultDiscoveryPeriod)
	seedTicker := time.NewTicker(DefaultSeedPeriod)
	defer func() {
		seedTicker.Stop()
		discoveryTicker.Stop()
	}()
	for na, _ := range p2p.trustSeeds.Map() {
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
				dialed := 0
				for _, s := range p2p.seeds.Array() {
					if !p2p.hasNetAddress(s) {
						p2p.logger.Debugln("discoverRoutine", "seedTicker", "dial to p2pRoleSeed", s)
						if err := p2p.dial(s); err != nil {
							p2p.seeds.Remove(s)
						} else {
							dialed++
						}
					}
				}
				if r.Has(p2pRoleSeed) || dialed == 0 {
					for na, d := range p2p.trustSeeds.Map() {
						if len(d) != 0 {
							na = NetAddress(d)
						}
						if !p2p.seeds.Contains(na) &&
							!p2p.hasNetAddress(na) {
							p2p.logger.Debugln("discoverRoutine", "seedTicker", "dial to trustSeed", na)
							p2p.dial(na)
						}
					}
				}
			} else {
				outSeeds := p2p.findPeers(func(p *Peer) bool {
					return !p.In() && p.HasRole(p2pRoleSeed) && !p.HasRole(p2pRoleRoot)
				}, p2pConnTypeNone)
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
				s := p2p.seeds
				if r == p2pRoleSeed {
					rr = p2pRoleRoot
					s = p2p.roots
				}

				for _, p := range p2p.findPeers(nil, p2pConnTypeFriend) {
					p2p.tryTransitPeerConnection(p, p2pConnTypeNone)
				}

				complete := p2p.discoverParents(rr)
				if complete {
					complete = p2p.discoverUncles(rr)
				}
				if !complete {
					for _, na := range s.Array() {
						if !p2p.hasNetAddress(na) {
							p2p.logger.Debugln("discoverRoutine", "discoveryTicker", "dial to", rr, na)
							if err := p2p.dial(na); err != nil {
								s.Remove(na)
							}
						}
					}
				}
			}
		}
	}
}

func (p2p *PeerToPeer) query(r PeerRoleFlag) (needMoreSeeds bool) {
	ps := make([]*Peer, 0)
	if r.Has(p2pRoleRoot) {
		friends := p2p.findPeers(nil, p2pConnTypeFriend)
		for _, p := range friends {
			if !p.In() {
				ps = append(ps, p)
			}
		}
		ps = append(ps, p2p.findPeers(func(p *Peer) bool {
			return !p.In()
		}, p2pConnTypeOther)...)

		numOfFailureNode := (p2p.allowedRoots.Len() - 1) / 3
		needMoreSeeds = (2*numOfFailureNode) > len(friends) || (p2p.lenPeers(p2pConnTypeChildren, p2pConnTypeNephew) < 1)

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
		ps = p2p.findPeers(nil, p2pConnTypeParent, p2pConnTypeUncle)
		if r == p2pRoleSeed {
			ps = append(ps, p2p.findPeers(func(p *Peer) bool {
				return !p.In() && p.HasRole(p2pRoleRoot)
			}, p2pConnTypeNone)...)
			ps = append(ps, p2p.findPeers(func(p *Peer) bool {
				return !p.In()
			}, p2pConnTypeOther)...)
		}
		needMoreSeeds = p2p.lenPeers(p2pConnTypeParent) < p2p.getConnectionLimit(p2pConnTypeParent) ||
			p2p.lenPeers(p2pConnTypeUncle) < p2p.getConnectionLimit(p2pConnTypeUncle)
	}

	if needMoreSeeds {
		ps = append(ps, p2p.findPeers(func(p *Peer) bool {
			return !p.In() && p.HasRole(p2pRoleSeed)
		}, p2pConnTypeNone)...)
	}
	for _, p := range ps {
		p2p.sendQuery(p)
	}
	return needMoreSeeds
}

func (p2p *PeerToPeer) discoverFriends() {
	ps := p2p.findPeers(func(p *Peer) bool {
		return !p.HasRole(p2pRoleRoot)
	}, p2pConnTypeFriend)
	for _, p := range ps {
		if p.HasRole(p2pRoleSeed) {
			if p2p.tryTransitPeerConnection(p, p2pConnTypeNone) {
				p2p.logger.Debugln("discoverFriends", "not allowed friend connection", p.id)
			}
		} else {
			p2p.logger.Debugln("discoverFriends", "not allowed connection", p.id)
			p.Close("discoverFriends not allowed connection")
		}
	}

	roots := p2p.findPeers(func(p *Peer) bool {
		return p.ConnType() != p2pConnTypeFriend && p.HasRole(p2pRoleRoot)
	}, joinPeerConnectionTypes...)
	for _, p := range roots {
		if p2p.tryTransitPeerConnection(p, p2pConnTypeFriend) {
			p2p.logger.Debugln("discoverFriends", "try p2pConnTypeFriend", p.ID(), p.ConnType())
		}
	}

	for _, na := range p2p.roots.Array() {
		if !p2p.hasNetAddress(na) {
			p2p.logger.Debugln("discoverFriends", "dial to p2pRoleRoot", na)
			if err := p2p.dial(na); err != nil {
				p2p.roots.Remove(na)
			}
		}
	}
}

func (p2p *PeerToPeer) isTrustSeed(p *Peer) bool {
	return p2p.trustSeeds.Contains(p.DialNetAddress())
}

func (p2p *PeerToPeer) setTrustSeeds(seeds []NetAddress) {
	var ss []NetAddress
	for _, s := range seeds {
		if s != p2p.NetAddress() && s.Validate() == nil {
			ss = append(ss, s)
		}
	}
	p2p.trustSeeds.ClearAndAdd(ss...)
}

func (p2p *PeerToPeer) discoverParents(pr PeerRoleFlag) (complete bool) {
	ps := p2p.findPeers(func(p *Peer) bool {
		return !p.HasRole(pr)
	}, p2pConnTypeParent)
	for _, p := range ps {
		if !(pr == p2pRoleSeed && p2p.isTrustSeed(p)) {
			p2p.logger.Debugln("discoverParents", "not allowed connection", p.id)
			p.Close("discoverParents not allowed connection")
		}
	}

	n := p2p.getConnectionLimit(p2pConnTypeParent) - p2p.lenPeers(p2pConnTypeParent)
	if n < 1 {
		p2p.logger.Traceln("discoverParents", "nothing to do")
		return true
	}

	limit := p2p.getConnectionLimit(p2pConnTypeChildren)
	var candidates []*Peer
	if pr == p2pRoleSeed {
		candidates = p2p.findPeers(func(p *Peer) bool {
			return !p.In() && (p.HasRole(pr) || p2p.isTrustSeed(p)) && p.children.Len() < limit
		}, p2pConnTypeNone, p2pConnTypeUncle)
	} else {
		candidates = p2p.findPeers(func(p *Peer) bool {
			return p.HasRole(pr) && p.children.Len() < limit
		}, p2pConnTypeNone, p2pConnTypeUncle)
	}
	try := 0
	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			avg1 := candidates[i].rtt.Avg(time.Millisecond)
			avg2 := candidates[j].rtt.Avg(time.Millisecond)
			if avg1 < avg2 {
				return true
			} else if avg1 == avg2 {
				return candidates[i].children.Len() < candidates[j].children.Len()
			}
			return false
		})
		for _, p := range candidates {
			if try == n {
				return false
			}
			if p2p.tryTransitPeerConnection(p, p2pConnTypeParent) {
				p2p.logger.Debugln("discoverParents", "try p2pConnTypeParent", p.ID(), p.ConnType())
				try++
			}
		}
	}
	if try == 0 {
		p2p.reject.Clear()
	}
	return false
}

func (p2p *PeerToPeer) discoverUncles(ur PeerRoleFlag) (complete bool) {
	ps := p2p.findPeers(func(p *Peer) bool {
		return !p.HasRole(ur)
	}, p2pConnTypeUncle)
	for _, p := range ps {
		if !(ur == p2pRoleSeed && p2p.isTrustSeed(p)) {
			p2p.logger.Debugln("discoverUncles", "not allowed connection", p.id)
			p.Close("discoverUncles not allowed connection")
		}
	}

	n := p2p.getConnectionLimit(p2pConnTypeUncle) - p2p.lenPeers(p2pConnTypeUncle)
	if n < 1 {
		p2p.logger.Traceln("discoverUncles", "nothing to do")
		return true
	}

	limit := p2p.getConnectionLimit(p2pConnTypeNephew)
	var candidates []*Peer
	if ur == p2pRoleSeed {
		candidates = p2p.findPeers(func(p *Peer) bool {
			return !p.In() && (p.HasRole(ur) || p2p.isTrustSeed(p)) && p.nephews.Len() < limit
		}, p2pConnTypeNone)
	} else {
		candidates = p2p.findPeers(func(p *Peer) bool {
			return p.HasRole(ur) && p.nephews.Len() < limit
		}, p2pConnTypeNone, p2pConnTypeUncle)
	}
	try := 0
	if len(candidates) > 0 {
		sort.Slice(candidates, func(i, j int) bool {
			avg1 := candidates[i].rtt.Avg(time.Millisecond)
			avg2 := candidates[j].rtt.Avg(time.Millisecond)
			if avg1 < avg2 {
				return true
			} else if avg1 == avg2 {
				return candidates[i].nephews.Len() < candidates[j].nephews.Len()
			}
			return false
		})
		for _, p := range candidates {
			if try == n {
				return false
			}
			if p2p.tryTransitPeerConnection(p, p2pConnTypeUncle) {
				p2p.logger.Debugln("discoverUncles", "try p2pConnTypeUncle", p.ID(), p.ConnType())
				try++
			}
		}
	}

	if try == 0 {
		p2p.reject.Clear()
	}
	return false
}

func (p2p *PeerToPeer) setConnectionLimit(connType PeerConnectionType, v int) {
	p2p.cLimitMtx.Lock()
	defer p2p.cLimitMtx.Unlock()

	if connType < p2pConnTypeNone || connType > p2pConnTypeOther {
		return
	}
	p2p.cLimit[connType] = v
}

func (p2p *PeerToPeer) getConnectionLimit(connType PeerConnectionType) int {
	p2p.cLimitMtx.RLock()
	defer p2p.cLimitMtx.RUnlock()
	v, ok := p2p.cLimit[connType]
	if !ok || v < 0 {
		switch connType {
		case p2pConnTypeParent:
			return DefaultParentsLimit
		case p2pConnTypeChildren:
			return DefaultChildrenLimit
		case p2pConnTypeUncle:
			return DefaultUnclesLimit
		case p2pConnTypeNephew:
			return DefaultNephewsLimit
		case p2pConnTypeOther:
			return DefaultOthersLimit
		default:
			v = -1
		}
	}
	return v
}

func (p2p *PeerToPeer) addPeer(p *Peer) bool {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	if p.IsClosed() {
		return false
	}
	return p2p.m[p2pConnTypeNone].Add(p)
}

func (p2p *PeerToPeer) _removePeer(p *Peer) bool {
	p2p.reject.Remove(p)
	p2p.transiting.Remove(p)

	if v, ok := p2p.m[p.ConnType()]; ok {
		return v.Remove(p)
	}
	return false
}

func (p2p *PeerToPeer) findPeer(f PeerPredicate, connTypes ...PeerConnectionType) *Peer {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	if len(connTypes) == 0 {
		for _, v := range p2p.m {
			if p := v.FindOne(f); p != nil {
				return p
			}
		}
	} else {
		for _, k := range connTypes {
			if v, ok := p2p.m[k]; ok {
				if p := v.FindOne(f); p != nil {
					return p
				}
			}
		}
	}
	return nil
}

func (p2p *PeerToPeer) _findPeers(f func(s *PeerSet) []*Peer, connTypes []PeerConnectionType) []*Peer {
	arr := make([]*Peer, 0)
	if len(connTypes) == 0 {
		for _, v := range p2p.m {
			arr = append(arr, f(v)...)
		}
	} else {
		for _, k := range connTypes {
			if v, ok := p2p.m[k]; ok {
				arr = append(arr, f(v)...)
			}
		}
	}
	return arr
}

func (p2p *PeerToPeer) findPeers(f PeerPredicate, connTypes ...PeerConnectionType) []*Peer {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	if f == nil {
		return p2p._findPeers(func(s *PeerSet) []*Peer {
			return s.Array()
		}, connTypes)
	} else {
		return p2p._findPeers(func(s *PeerSet) []*Peer {
			return s.Find(f)
		}, connTypes)
	}
}

func (p2p *PeerToPeer) _lenPeers(f func(s *PeerSet) int, connTypes []PeerConnectionType) int {
	n := 0
	if len(connTypes) == 0 {
		for _, v := range p2p.m {
			n += f(v)
		}
	} else {
		for _, k := range connTypes {
			if v, ok := p2p.m[k]; ok {
				n += f(v)
			}
		}
	}
	return n
}

func (p2p *PeerToPeer) lenPeers(connTypes ...PeerConnectionType) int {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	return p2p._lenPeers(func(s *PeerSet) int {
		return s.Len()
	}, connTypes)
}

func (p2p *PeerToPeer) lenPeersByProtocol(pi module.ProtocolInfo, connTypes ...PeerConnectionType) int {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	return p2p._lenPeers(func(s *PeerSet) int {
		return s.LenByProtocol(pi)
	}, connTypes)
}

func (p2p *PeerToPeer) getNetAddresses(connType PeerConnectionType) []NetAddress {
	p2p.mtx.RLock()
	defer p2p.mtx.RUnlock()

	return p2p.m[connType].NetAddresses()
}

func (p2p *PeerToPeer) updatePeerConnectionType(p *Peer, connType PeerConnectionType) (updated bool) {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()

	if p.IsClosed() {
		return false
	}
	from := p.ConnType()
	if connType < p2pConnTypeNone || connType > p2pConnTypeOther || connType == from {
		return
	}

	t := p2p.m[connType]
	l := p2p.getConnectionLimit(connType)
	if l < 0 || l > t.Len() {
		p2p._removePeer(p)
		if updated = t.Add(p); !updated {
			//unexpected failure
			return
		}
		if l == t.Len() {
			p2p.logger.Debugln("updatePeerConnectionType", "complete", strPeerConnectionType[connType])
			if connType == p2pConnTypeParent || connType == p2pConnTypeUncle {
				p2p.reject.Clear()
			}
		}
		p.setConnType(connType)
		if from == p2pConnTypeNone {
			p2p.onEvent(p2pEventJoin, p)
		}
		if connType == p2pConnTypeNone {
			p2p.onEvent(p2pEventLeave, p)
		}
	}
	return
}

func (p2p *PeerToPeer) transitPeer(p *Peer, remove bool) bool {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()
	if p.IsClosed() {
		return false
	}
	if remove {
		return p2p.transiting.Remove(p)
	} else {
		if !p2p.reject.Contains(p) && !p2p.transiting.Contains(p) {
			return p2p.transiting.Add(p)
		}
	}
	return false
}

func (p2p *PeerToPeer) rejectPeer(p *Peer) bool {
	p2p.connMtx.Lock()
	defer p2p.connMtx.Unlock()
	if p.IsClosed() {
		return false
	}
	return p2p.reject.Add(p)
}

func (p2p *PeerToPeer) tryTransitPeerConnection(p *Peer, connType PeerConnectionType) bool {
	switch connType {
	case p2pConnTypeNone:
		p2p.updatePeerConnectionType(p, p2pConnTypeNone)
		p2p.sendP2PConnectionRequest(p2pConnTypeNone, p)
		return true
	default:
		if p.EqualsAttr(AttrSupportDefaultProtocols, false) {
			return false
		}
		if p2p.transitPeer(p, false) {
			p.PutAttr(AttrP2PConnectionRequest, connType)
			p2p.sendP2PConnectionRequest(connType, p)
			return true
		}
	}
	return false
}

type P2PConnectionRequest struct {
	ConnType PeerConnectionType
}

type P2PConnectionResponse struct {
	ReqConnType PeerConnectionType
	ConnType    PeerConnectionType
}

func (p2p *PeerToPeer) sendP2PConnectionRequest(connType PeerConnectionType, p *Peer) {
	m := &P2PConnectionRequest{ConnType: connType}
	pkt := newPacket(p2pProtoControl, p2pProtoConnReq, p2p.encode(m), p2p.ID())
	pkt.destPeer = p.ID()
	err := p.sendPacket(pkt)
	if err != nil {
		p2p.logger.Infoln("sendP2PConnectionRequest", err, p)
	} else {
		p2p.logger.Debugln("sendP2PConnectionRequest", m, p)
	}
}
func (p2p *PeerToPeer) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := p2p.decode(pkt.payload, req)
	if err != nil {
		p2p.logger.Infoln("handleP2PConnectionRequest", err, p)
		return
	}
	p2p.logger.Debugln("handleP2PConnectionRequest", req, p)
	p.setRecvConnType(req.ConnType)
	rc := p2pConnTypeNone
	r := p2p.Role()
	notAllowed := false
	invalidReq := false
	if r.Has(p2pRoleRoot) {
		switch req.ConnType {
		case p2pConnTypeFriend:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeFriend
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeOther
			} else {
				notAllowed = true
			}
		case p2pConnTypeParent:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeOther
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeChildren
			} else {
				notAllowed = true
			}
		case p2pConnTypeUncle:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeOther
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeNephew
			} else {
				notAllowed = true
			}
		case p2pConnTypeNone:
			rc = req.ConnType
		default:
			invalidReq = true
		}
	} else if r.Has(p2pRoleSeed) {
		switch req.ConnType {
		case p2pConnTypeFriend:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeParent
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeOther
			} else {
				invalidReq = true
			}
		case p2pConnTypeParent:
			if p.HasRole(p2pRoleRoot) || p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeChildren
			}
		case p2pConnTypeUncle:
			if p.HasRole(p2pRoleRoot) || p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeNephew
			}
		case p2pConnTypeNone:
			rc = req.ConnType
		default:
			invalidReq = true
		}
	} else {
		switch req.ConnType {
		case p2pConnTypeParent:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeNone
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeChildren
			}
		case p2pConnTypeUncle:
			if p.HasRole(p2pRoleRoot) {
				rc = p2pConnTypeNone
			} else if p.HasRole(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeNephew
			}
		case p2pConnTypeNone:
			rc = req.ConnType
		default:
			invalidReq = true
		}
	}

	if notAllowed {
		p2p.logger.Infoln("handleP2PConnectionRequest", "not allowed reqConnType", req.ConnType, "from", p.ID(), p.ConnType())
	} else if invalidReq {
		p2p.logger.Infoln("handleP2PConnectionRequest", "invalid reqConnType", req.ConnType, "from", p.ID(), p.ConnType())
	} else {
		if rc != p2pConnTypeNone && !p.EqualsAttr(AttrSupportDefaultProtocols, true) {
			rc = p2pConnTypeOther
			p2p.logger.Debugln("handleP2PConnectionResponse", "not support defaultProtocols", p.ID())
		}
		switch rc {
		case p2pConnTypeParent:
			if !p2p.updatePeerConnectionType(p, p2pConnTypeParent) &&
				!p2p.updatePeerConnectionType(p, p2pConnTypeUncle) {
				p2p.logger.Infoln("handleP2PConnectionRequest",
					"ignore p2pConnTypeFriend request, already has enough upstream connections", strPeerConnectionType[rc],
					"from", p.ID(), p.ConnType())
			}
		case p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, rc)
		case p2pConnTypeChildren, p2pConnTypeNephew:
			if !p2p.updatePeerConnectionType(p, rc) {
				p2p.logger.Infoln("handleP2PConnectionRequest", "reject by limit", strPeerConnectionType[rc],
					"from", p.ID(), p.ConnType())
			}
		}
	}
	m := &P2PConnectionResponse{ReqConnType: req.ConnType, ConnType: p.ConnType()}
	if m.ConnType == p2pConnTypeOther {
		//for legacy which is not supported p2pConnTypeOther response
		if p.EqualsAttr(AttrP2PLegacy, true) {
			switch req.ConnType {
			case p2pConnTypeParent:
				m.ConnType = p2pConnTypeChildren
			case p2pConnTypeUncle:
				m.ConnType = p2pConnTypeNephew
			}
		}
	}
	rpkt := newPacket(p2pProtoControl, p2pProtoConnResp, p2p.encode(m), p2p.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.logger.Infoln("handleP2PConnectionRequest", "sendP2PConnectionResponse", err, p)
	} else {
		p2p.logger.Debugln("handleP2PConnectionRequest", "sendP2PConnectionResponse", m, p)
	}
}

func (p2p *PeerToPeer) handleP2PConnectionResponse(pkt *Packet, p *Peer) {
	resp := &P2PConnectionResponse{}
	err := p2p.decode(pkt.payload, resp)
	if err != nil {
		p2p.logger.Infoln("handleP2PConnectionResponse", err, p)
		return
	}
	p2p.logger.Debugln("handleP2PConnectionResponse", resp, p)
	p.setRecvConnType(resp.ConnType)
	if resp.ReqConnType == p2pConnTypeNone {
		return
	}
	if !p2p.transitPeer(p, true) {
		p2p.logger.Infoln("handleP2PConnectionResponse", "invalid peer", resp, p)
		return
	} else {
		if !p.EqualsAttr(AttrP2PConnectionRequest, resp.ReqConnType) {
			p2p.logger.Infoln("handleP2PConnectionResponse", "invalid ReqConnType", resp, p)
			return
		}
		p.RemoveAttr(AttrP2PConnectionRequest)
	}

	rc := p2pConnTypeNone
	r := p2p.Role()
	invalidResp := false
	rejectResp := false
	if r.Has(p2pRoleRoot) {
		switch resp.ReqConnType {
		case p2pConnTypeFriend:
			switch resp.ConnType {
			case p2pConnTypeFriend:
				rc = p2pConnTypeFriend
			case p2pConnTypeOther, p2pConnTypeNone:
				//in case of p2pConnTypeNone
				// for legacy which p2p.others managed by discovery only,
				// legacy ignore request of p2pConnTypeFriend and response p2pConnTypeNone
				if p.HasRecvRole(p2pRoleRoot) {
					rc = p2pConnTypeFriend
				} else {
					rc = p2pConnTypeOther
				}
			case p2pConnTypeParent, p2pConnTypeUncle:
				rc = p2pConnTypeOther
			default:
				invalidResp = true
			}
		default:
			invalidResp = true
		}
	} else if r.Has(p2pRoleSeed) {
		switch resp.ReqConnType {
		case p2pConnTypeParent:
			switch resp.ConnType {
			case p2pConnTypeChildren, p2pConnTypeOther:
				rc = p2pConnTypeParent
			case p2pConnTypeNephew:
				rejectResp = true
			case p2pConnTypeNone:
				rc = p2pConnTypeNone
				rejectResp = true
			default:
				rejectResp = true
			}
		case p2pConnTypeUncle:
			switch resp.ConnType {
			case p2pConnTypeNephew, p2pConnTypeOther:
				rc = p2pConnTypeUncle
			case p2pConnTypeNone:
				rc = p2pConnTypeNone
				rejectResp = true
			default:
				rejectResp = true
			}
		default:
			invalidResp = true
		}
	} else {
		switch resp.ReqConnType {
		case p2pConnTypeParent:
			switch resp.ConnType {
			case p2pConnTypeChildren:
				rc = p2pConnTypeParent
			case p2pConnTypeNephew:
				rejectResp = true
			case p2pConnTypeOther:
				rc = p2pConnTypeOther
			case p2pConnTypeNone:
				rc = p2pConnTypeNone
				rejectResp = true
			default:
				rejectResp = true
			}
		case p2pConnTypeUncle:
			switch resp.ConnType {
			case p2pConnTypeNephew:
				rc = p2pConnTypeUncle
			case p2pConnTypeOther:
				rc = p2pConnTypeOther
			case p2pConnTypeNone:
				rc = p2pConnTypeNone
				rejectResp = true
			default:
				rejectResp = true
			}
		default:
			invalidResp = true
		}
	}

	if rejectResp {
		p2p.rejectPeer(p)
		p2p.logger.Infoln("handleP2PConnectionResponse", "rejected",
			strPeerConnectionType[resp.ReqConnType], "resp", strPeerConnectionType[resp.ConnType],
			"from", p.ID(), p.ConnType())
	} else if invalidResp {
		p2p.logger.Infoln("handleP2PConnectionResponse", "invalid ReqConnType", resp,
			"from", p.ID(), p.ConnType())
	} else {
		p2p.logger.Debugln("handleP2PConnectionResponse", "resolvedConnType", strPeerConnectionType[resp.ConnType],
			"from", p.ID(), p.ConnType())
		if rc != p2pConnTypeNone && !p.EqualsAttr(AttrSupportDefaultProtocols, true) {
			rc = p2pConnTypeOther
			p2p.logger.Debugln("handleP2PConnectionResponse", "not support defaultProtocols", p.ID())
		}
		switch rc {
		case p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, rc)
		case p2pConnTypeParent:
			if !p2p.updatePeerConnectionType(p, p2pConnTypeParent) {
				p2p.logger.Debugln("handleP2PConnectionResponse", "already p2pConnTypeParent", resp,
					"from", p.ID(), p.ConnType())
				if p2p.lenPeers(p2pConnTypeUncle) < p2p.getConnectionLimit(p2pConnTypeUncle) {
					p2p.tryTransitPeerConnection(p, p2pConnTypeUncle)
				} else {
					p.Close("already has enough upstream connections")
				}
			}
		case p2pConnTypeUncle:
			if !p2p.updatePeerConnectionType(p, p2pConnTypeUncle) {
				p2p.logger.Debugln("handleP2PConnectionResponse", "already p2pConnTypeUncle", resp,
					"from", p.ID(), p.ConnType())
				if p2p.lenPeers(p2pConnTypeParent) < p2p.getConnectionLimit(p2pConnTypeParent) {
					p2p.tryTransitPeerConnection(p, p2pConnTypeParent)
				} else {
					p.Close("already has enough upstream connections")
				}
			}
		}
	}
}
