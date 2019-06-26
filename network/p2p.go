package network

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type PeerToPeer struct {
	channel          string
	sendQueue        *WeightQueue
	alternateQueue   Queue
	sendTicker       *time.Ticker
	onPacketCbFuncs  map[uint16]packetCbFunc
	onFailureCbFuncs map[uint16]failureCbFunc
	onEventCbFuncs   map[string]map[uint16]eventCbFunc
	packetPool       *PacketPool
	packetRw         *PacketReadWriter
	dialer           *Dialer

	//Topology with Connected Peers
	self       *Peer
	parent     *Peer
	children   *PeerSet
	uncles     *PeerSet
	nephews    *PeerSet
	friends    *PeerSet //Only for root, parent is nil, uncles is empty
	orphanages *PeerSet //Not joined
	pre        *PeerSet
	reject     *PeerSet
	parentMtx  sync.RWMutex

	//Discovery
	discoveryTicker *time.Ticker
	seedTicker      *time.Ticker
	duplicated      *Set

	//Addresses
	seeds *NetAddressSet
	roots *NetAddressSet //For seed, root
	//[TBD] 2hop peers of current tree for status change
	grandParent   NetAddress
	grandChildren *NetAddressSet

	//managed PeerId
	allowedRoots *PeerIDSet
	allowedSeeds *PeerIDSet
	allowedPeers *PeerIDSet

	//log
	log *logger

	//monitor
	mtr *metric.NetworkMetric

	stopCh chan bool
	run    bool
	mtx    sync.RWMutex
}

type failureCbFunc func(err error, pkt *Packet, c *Counter)

type eventCbFunc func(evt string, p *Peer)

const (
	p2pEventJoin       = "join"
	p2pEventLeave      = "leave"
	p2pEventDuplicate  = "duplicate"
	p2pEventNotAllowed = "not allowed"
)

func newPeerToPeer(channel string, self *Peer, d *Dialer, mtr *metric.NetworkMetric) *PeerToPeer {
	p2p := &PeerToPeer{
		channel:          channel,
		sendQueue:        NewWeightQueue(DefaultSendQueueSize, DefaultSendQueueMaxPriority+1),
		alternateQueue:   NewQueue(DefaultSendQueueSize),
		sendTicker:       time.NewTicker(DefaultAlternateSendPeriod),
		onPacketCbFuncs:  make(map[uint16]packetCbFunc),
		onFailureCbFuncs: make(map[uint16]failureCbFunc),
		onEventCbFuncs:   make(map[string]map[uint16]eventCbFunc),
		packetPool:       NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen),
		packetRw:         NewPacketReadWriter(),
		dialer:           d,
		//
		self:            self,
		children:        NewPeerSet(),
		uncles:          NewPeerSet(),
		nephews:         NewPeerSet(),
		friends:         NewPeerSet(),
		orphanages:      NewPeerSet(),
		pre:             NewPeerSet(),
		reject:          NewPeerSet(),
		discoveryTicker: time.NewTicker(DefaultDiscoveryPeriod),
		seedTicker:      time.NewTicker(DefaultSeedPeriod),
		duplicated:      NewSet(),
		//
		seeds:         NewNetAddressSet(),
		roots:         NewNetAddressSet(),
		grandChildren: NewNetAddressSet(),
		//
		allowedRoots: NewPeerIDSet(),
		allowedSeeds: NewPeerIDSet(),
		allowedPeers: NewPeerIDSet(),
		//
		log: newLogger("PeerToPeer", fmt.Sprintf("%s.%s", channel, hex.EncodeToString(self.id.Bytes()[:DefaultSimplePeerIDSize]))),
		//
		mtr: mtr,
	}
	p2p.allowedRoots.onUpdate = func() {
		p2p.setRoleByAllowedSet()
	}
	p2p.allowedSeeds.onUpdate = func() {
		p2p.setRoleByAllowedSet()
	}
	p2p.allowedPeers.onUpdate = func() {
		peers := p2p.getPeers(false)
		for _, p := range peers {
			if !p2p.allowedPeers.Contains(p.ID()) {
				p2p.onEvent(p2pEventNotAllowed, p)
				p.CloseByError(fmt.Errorf("onUpdate not allowed connection"))
			}
		}
	}

	p2p.log.excludes = []string{
		//"onPeer",
		//"onClose",
		"onEvent",
		"onPacket",
		//"onFailure",
		"sendQuery",
		"handleQuery",
		"sendRtt",
		"handleRtt",
		"sendRoutine",
		"alternateSendRoutine",
		//"discoverRoutine",
		"discoverParent",
		"discoverUncle",
		"discoverFriends",
		//"updatePeerConnectionType",
		"setRoleByAllowedSet",
		"selectPeersFromFriends",
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
	p2p.log.Println("Stop", "try close p2p.stopCh")
	close(p2p.stopCh)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
	Loop:
		for {
			ps := p2p.getPeers(false)
			for _, p := range ps {
				p2p.log.Println("Stop", "try Peer.Close", p)
				p.Close("stopCh")
			}
			if len(ps) < 1 {
				break Loop
			}
			time.Sleep(time.Second)
		}
		wg.Done()
	}()
	p2p.log.Println("Stop", "wait peer Closing")
	wg.Wait()

	p2p.run = false
	p2p.log.Println("Stop", "Done")
}

func (p2p *PeerToPeer) dial(na NetAddress) error {
	if err := p2p.dialer.Dial(string(na)); err != nil {
		if err == ErrAlreadyDialing {
			p2p.log.Println("Warning", "Dial ignore", na, err)
			return nil
		}
		p2p.log.Println("Warning", "Dial fail", na, err)
		return err
	}
	return nil
}

func (p2p *PeerToPeer) setCbFunc(pi module.ProtocolInfo, pktFunc packetCbFunc,
	failFunc failureCbFunc, evtFunc eventCbFunc, evts ...string) {
	k := pi.Uint16()
	if _, ok := p2p.onPacketCbFuncs[k]; ok {
		p2p.log.Println("Warning", "overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[k] = pktFunc
	p2p.onFailureCbFuncs[k] = failFunc
	for _, evt := range evts {
		p2p.setEventCbFunc(evt, k, evtFunc)
	}
}

func (p2p *PeerToPeer) unsetCbFunc(pi module.ProtocolInfo) {
	k := pi.Uint16()
	if _, ok := p2p.onPacketCbFuncs[k]; ok {
		p2p.unsetEventCbFunc(k)
		delete(p2p.onFailureCbFuncs, k)
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
	p2p.log.Println("onPeer", p)
	if !p2p.allowedPeers.IsEmpty() && !p2p.allowedPeers.Contains(p.id) {
		p2p.onEvent(p2pEventNotAllowed, p)
		p.CloseByError(fmt.Errorf("onPeer not allowed connection"))
		return
	}
	if dp := p2p.getPeer(p.id, false); dp != nil {
		if p2p.removePeer(dp) {
			p2p.onEvent(p2pEventDuplicate, p)
		}
		p2p.duplicated.Add(dp)
		if dp.incomming == p.incomming {
			dp.CloseByError(fmt.Errorf("onPeer duplicated peer"))
			p2p.log.Println("Warning", "Already exists connected Peer, close duplicated peer", dp, p.incomming)
		} else {
			dp.Close("onPeer duplicated peer")
		}
	}
	p2p.orphanages.Add(p)
	if !p.incomming {
		p2p.sendQuery(p)
	}
}

//callback from Peer.sendRoutine or Peer.receiveRoutine
func (p2p *PeerToPeer) onError(err error, p *Peer, pkt *Packet) {
	p2p.log.Println("Warning", "onError", err, p, pkt)

	//Peer.receiveRoutine
	//// bufio.Reader.Read error except {net.OpError, io.EOF, io.ErrUnexpectedEOF}
	//Peer.sendRoutine
	//// net.Conn.SetWriteDeadline error
	//// bufio.Writer.Write error
	//// bufio.Writer.Flush error

	//if p.isTemporaryError(err) {p.onError(err)}
	//else {p.CloseByError(err)}

	//if pkt == nil //readError
}

func (p2p *PeerToPeer) onClose(p *Peer) {
	p2p.log.Println("onClose", p.CloseInfo(), p)
	if p2p.removePeer(p) {
		p2p.onEvent(p2pEventLeave, p)
		<-p.close
		ctx := p.q.Last()
		if ctx == nil {
			ctx = p.q.Pop()
		}

		for ; ctx != nil; ctx = p.q.Pop() {
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
	//FIXME [TBD]if !p2p.IsStarted() return

	p2p.log.Println("onEvent", evt, p)
	if m, ok := p2p.onEventCbFuncs[evt]; ok {
		for _, cbFunc := range m {
			cbFunc(evt, p)
		}
	}
}

func (p2p *PeerToPeer) onFailure(err error, pkt *Packet, c *Counter) {
	//FIXME [TBD]if !p2p.IsStarted() return

	p2p.log.Println("onFailure", err, pkt, c)
	if cbFunc, ok := p2p.onFailureCbFuncs[pkt.protocol.Uint16()]; ok {
		cbFunc(err, pkt, c)
	}
}

func (p2p *PeerToPeer) removePeer(p *Peer) (isLeave bool) {
	if p2p.duplicated.Remove(p) {
		return
	}

	if p.compareRole(p2pRoleSeed, false) {
		p2p.removeSeed(p)
		p2p.seeds.Add(p.netAddress)
	}
	if p.compareRole(p2pRoleRoot, false) {
		p2p.removeRoot(p)
		p2p.roots.Add(p.netAddress)
	}

	isLeave = !(p.connType == p2pConnTypeNone)
	switch p.connType {
	case p2pConnTypeNone:
		p2p.orphanages.Remove(p)
	case p2pConnTypeParent:
		p2p.setParent(nil)
	case p2pConnTypeChildren:
		p2p.children.Remove(p)
	case p2pConnTypeUncle:
		p2p.uncles.Remove(p)
	case p2pConnTypeNephew:
		p2p.nephews.Remove(p)
	case p2pConnTypeFriend:
		p2p.friends.Remove(p)
	}
	p2p.pre.Remove(p)
	return
}

//callback from Peer.receiveRoutine
func (p2p *PeerToPeer) onPacket(pkt *Packet, p *Peer) {
	//FIXME if !p2p.IsStarted() return
	if pkt.protocol == PROTO_CONTOL {
		p2p.log.Println("onPacket", pkt, p)
		switch pkt.protocol {
		case PROTO_CONTOL:
			switch pkt.subProtocol {
			case PROTO_P2P_QUERY: //roots, seeds, children
				p2p.handleQuery(pkt, p)
			case PROTO_P2P_QUERY_RESULT:
				p2p.handleQueryResult(pkt, p)
			case PROTO_P2P_RTT_REQ: //roots, seeds, children
				p2p.handleRttRequest(pkt, p)
			case PROTO_P2P_RTT_RESP:
				p2p.handleRttResponse(pkt, p)
			case PROTO_P2P_CONN_REQ:
				p2p.handleP2PConnectionRequest(pkt, p)
			case PROTO_P2P_CONN_RESP:
				p2p.handleP2PConnectionResponse(pkt, p)
			}
		}
	} else {
		if p.connType == p2pConnTypeNone {
			p2p.log.Println("Warning", "onPacket", "Drop, undetermined PeerConnectionType", pkt.protocol, pkt.subProtocol)
			return
		}

		if p2p.self.id.Equal(pkt.src) {
			p2p.log.Println("Warning", "onPacket", "Drop, Invalid self-src", pkt.src, pkt.protocol, pkt.subProtocol)
			return
		}

		isSourcePeer := p.id.Equal(pkt.src)
		isOneHop := pkt.ttl == 1 || pkt.dest == p2pDestPeer
		if isOneHop && !isSourcePeer {
			p2p.log.Println("Warning", "onPacket", "Drop, Invalid 1hop-src:", pkt.src, ",expected:", p.id, pkt.protocol, pkt.subProtocol)
			return
		}

		isBroadcast := pkt.dest == p2pDestAny && pkt.ttl != 1
		if isBroadcast && isSourcePeer && !p.compareRole(p2pRoleRoot, false) {
			p2p.log.Println("Warning", "onPacket", "Drop, Not authorized", p.id, pkt.protocol, pkt.subProtocol)
			return
		}

		if cbFunc := p2p.onPacketCbFuncs[pkt.protocol.Uint16()]; cbFunc != nil {
			if isOneHop || p2p.packetPool.Put(pkt) {
				cbFunc(pkt, p)
			} else {
				p2p.log.Println("onPacket", "Drop, Duplicated by footer", pkt.protocol, pkt.subProtocol, pkt.hashOfPacket, p.id)
			}
		}
	}
}

func (p2p *PeerToPeer) encodeMsgpack(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.MP.NewEncoderBytes(&b)
	if err := enc.Encode(v); err != nil {
		log.Panicf("Fail to encode err=%+v", err)
	}
	return b
}

func (p2p *PeerToPeer) decodeMsgpack(b []byte, v interface{}) error {
	_, err := codec.MP.UnmarshalFromBytes(b, v)
	return err
}

//TODO timestamp or sequencenumber for validation (query,result pair)
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

func (p2p *PeerToPeer) addSeed(p *Peer) {
	c, o := p2p.seeds.PutByPeer(p)
	if o != "" {
		p2p.log.Println("addSeed", "updated NetAddress old:", o, ", now:", p.netAddress, ",peerID:", p.id)
	}
	if c != "" {
		p2p.log.Println("Warning", "addSeed", "conflict NetAddress", p.netAddress, "removed:", c, ",now:", p.id)
	}
}
func (p2p *PeerToPeer) removeSeed(p *Peer) {
	p2p.seeds.RemoveByPeer(p)
}
func (p2p *PeerToPeer) addRoot(p *Peer) {
	c, o := p2p.roots.PutByPeer(p)
	if o != "" {
		p2p.log.Println("addRoot", "updated NetAddress old:", o, ", now:", p.netAddress, ",peerID:", p.id)
	}
	if c != "" {
		p2p.log.Println("Warning", "addRoot", "conflict NetAddress", p.netAddress, "removed:", c, ",now:", p.id)
	}
}
func (p2p *PeerToPeer) removeRoot(p *Peer) {
	p2p.roots.RemoveByPeer(p)
}
func (p2p *PeerToPeer) applyPeerRole(p *Peer) {
	r := p.getRole()
	switch r {
	case p2pRoleNone:
		p2p.removeRoot(p)
		p2p.removeSeed(p)
	case p2pRoleSeed:
		p2p.addSeed(p)
		p2p.removeRoot(p)
	case p2pRoleRoot:
		p2p.addRoot(p)
		p2p.removeSeed(p)
	case p2pRoleRootSeed:
		p2p.addRoot(p)
		p2p.addSeed(p)
	}
}

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) bool {
	if !p2p.self.compareRole(r, true) {
		p2p.self.setRole(r)
		p2p.applyPeerRole(p2p.self)
		return true
	}
	return false
}

func (p2p *PeerToPeer) setRoleByAllowedSet() PeerRoleFlag {
	r := p2pRoleNone
	if p2p.isAllowedRole(p2pRoleRoot, p2p.self) {
		r |= p2pRoleRoot
	}
	if p2p.isAllowedRole(p2pRoleSeed, p2p.self) {
		r |= p2pRoleSeed
	}
	role := PeerRoleFlag(r)
	p2p.setRole(role)
	//TODO disconnect invalid peer case p2pRoleRoot, p2pRoleSeed
	p2p.log.Println("setRoleByAllowedSet", p2p.getRole(), p2p.allowedRoots.Len(), p2p.allowedSeeds.Len(), p2p.allowedPeers.Len())
	return role
}

func (p2p *PeerToPeer) getRole() PeerRoleFlag {
	return p2p.self.getRole()
}

func (p2p *PeerToPeer) setParent(p *Peer) {
	p2p.parentMtx.Lock()
	defer p2p.parentMtx.Unlock()

	p2p.parent = p
}

func (p2p *PeerToPeer) getParent() *Peer {
	p2p.parentMtx.RLock()
	defer p2p.parentMtx.RUnlock()

	return p2p.parent
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.getRole()}
	pkt := newPacket(PROTO_P2P_QUERY, p2p.encodeMsgpack(m), p2p.self.id)
	pkt.destPeer = p.id
	err := p.sendPacket(pkt)
	if err != nil {
		p2p.log.Println("Warning", "sendQuery", err, p)
	} else {
		p.rtt.Start()
		p2p.log.Println("sendQuery", m, p)
	}
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qm)
	if err != nil {
		p2p.log.Println("Warning", "handleQuery", err, p)
		return
	}
	p2p.log.Println("handleQuery", qm, p)
	m := &QueryResultMessage{}
	m.Role = p2p.getRole()
	if p2p.isAllowedRole(qm.Role, p) {
		p.setRole(qm.Role)
		p2p.applyPeerRole(p)
		m.Seeds = p2p.seeds.Array()
		m.Children = p2p.children.NetAddresses()
		m.Nephews = p2p.nephews.NetAddresses()
		if qm.Role.Has(p2pRoleSeed) || qm.Role.Has(p2pRoleRoot) {
			m.Roots = p2p.roots.Array()
		} else {
			if m.Role.Has(p2pRoleRoot) && !m.Role.Has(p2pRoleSeed) {
				p2p.log.Println("Warning", "handleQuery", "p2pRoleNone cannot query to p2pRoleRoot", p)
				m.Message = "not allowed to query"
				m.Seeds = nil
				m.Children = nil
				m.Nephews = nil
			}
		}
	} else {
		m.Message = "not exists allowedlist"
	}
	rpkt := newPacket(PROTO_P2P_QUERY_RESULT, p2p.encodeMsgpack(m), p2p.self.id)
	rpkt.destPeer = p.id
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.log.Println("Warning", "handleQuery", "sendQueryResult", err, p)
	} else {
		p.rtt.Start()
		p2p.log.Println("handleQuery", "sendQueryResult", m, p)
	}
}

func (p2p *PeerToPeer) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qrm)
	if err != nil {
		p2p.log.Println("Warning", "handleQueryResult", err, p)
		return
	}
	p2p.log.Println("handleQueryResult", qrm, p)
	p.rtt.Stop()
	p.children = qrm.Children
	p.nephews = len(qrm.Nephews)
	role := p2p.getRole()
	if p2p.isAllowedRole(qrm.Role, p) {
		p.setRole(qrm.Role)
		p2p.applyPeerRole(p)
		if role.Has(p2pRoleSeed) || role.Has(p2pRoleRoot) {
			p2p.seeds.Merge(qrm.Seeds...)
			p2p.roots.Merge(qrm.Roots...)
		} else {
			p2p.seeds.Merge(qrm.Seeds...)
		}
	} else {
		p2p.log.Println("handleQueryResult", "not exists allowedlist", p)
		p.CloseByError(fmt.Errorf("handleQueryResult not exists allowedlist"))
		return
	}

	m := &RttMessage{Last: p.rtt.last, Average: p.rtt.avg}
	rpkt := newPacket(PROTO_P2P_RTT_REQ, p2p.encodeMsgpack(m), p2p.self.id)
	rpkt.destPeer = p.id
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.log.Println("Warning", "handleQueryResult", "sendRttRequest", err, p)
	} else {
		p2p.log.Println("handleQueryResult", "sendRttRequest", m, p)
	}
}

func (p2p *PeerToPeer) handleRttRequest(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := p2p.decodeMsgpack(pkt.payload, rm)
	if err != nil {
		p2p.log.Println("Warning", "handleRttRequest", err, p)
		return
	}
	p2p.log.Println("handleRttRequest", rm, p)
	p.rtt.Stop()
	//p.rtt.et.Sub(pkt.timestamp)

	df := rm.Last - p.rtt.last
	if df > DefaultRttAccuracy {
		p2p.log.Println("Warning", "handleRttRequest", df, "DefaultRttAccuracy", DefaultRttAccuracy, p)
	}

	m := &RttMessage{Last: p.rtt.last, Average: p.rtt.avg}
	rpkt := newPacket(PROTO_P2P_RTT_RESP, p2p.encodeMsgpack(m), p2p.self.id)
	rpkt.destPeer = p.id
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.log.Println("Warning", "handleRttRequest", "sendRttResponse", err, p)
	} else {
		p2p.log.Println("handleRttRequest", "sendRttResponse", m, p)
	}
}

func (p2p *PeerToPeer) handleRttResponse(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := p2p.decodeMsgpack(pkt.payload, rm)
	if err != nil {
		p2p.log.Println("Warning", "handleRttResponse", err, p)
		return
	}
	p2p.log.Println("handleRttResponse", rm, p)

	df := rm.Last - p.rtt.last
	if df > DefaultRttAccuracy {
		p2p.log.Println("Warning", "handleRttResponse", df, "DefaultRttAccuracy", DefaultRttAccuracy, p)
	}
}

func (p2p *PeerToPeer) sendToPeers(ctx context.Context, peers *PeerSet) {
	for _, p := range peers.Array() {
		//p2p.packetRw.WriteTo(p.writer)
		if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			p2p.log.Println("Warning", "sendToPeers", err, pkt.protocol, pkt.subProtocol, p.id)
		}
	}
}

func (p2p *PeerToPeer) selectPeersFromFriends(pkt *Packet) ([]*Peer, []byte) {
	src := pkt.src

	ps := p2p.friends.Array()
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

	var ext []byte
	if DefaultSimplePeerIDSize >= peerIDSize {
		rids, _ := NewPeerIDSetFromBytes(pkt.ext)
		tids := NewPeerIDSet()
		for _, p := range ps {
			if src.Equal(p.id) {
				continue
			}
			if !rids.Contains(p.id) {
				tps[ti] = p
				ti++
				tids.Add(p.id)
			} else {
				lps[li] = p
				li++
			}

			if ti >= n {
				break
			}
		}
		ext = tids.Bytes()
		p2p.log.Println("selectPeersFromFriends", "hash:", pkt.hashOfPacket, "src:", pkt.src, "ext:", pkt.extendInfo, "rids:", rids, "tids:", tids)
	} else {
		rids, _ := NewBytesSetFromBytes(pkt.ext, DefaultSimplePeerIDSize)
		tids := NewBytesSet(DefaultSimplePeerIDSize)
		for _, p := range ps {
			if src.Equal(p.id) {
				continue
			}
			tb := p.id.Bytes()[:DefaultSimplePeerIDSize]
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
		ext = tids.Bytes()
		p2p.log.Println("selectPeersFromFriends", "hash:", pkt.hashOfPacket, "src:", pkt.src, "ext:", pkt.extendInfo, "rids:", rids, "tids:", tids)
	}
	n = n - ti
	for i := 0; i < n && i < li; i++ {
		tps[ti] = lps[i]
		ti++
	}
	return tps[:ti], ext
}

func (p2p *PeerToPeer) sendToFriends(ctx context.Context) {
	if UsingSelectiveFlooding { //selective (F+1) flooding with node-list
		pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
		ps, ext := p2p.selectPeersFromFriends(pkt)
		pkt.extendInfo = newPacketExtendInfo(pkt.extendInfo.hint()+1, pkt.extendInfo.len()+len(ext))
		if len(pkt.ext) > 0 {
			ext = append(pkt.ext, ext...)
		}
		pkt.footerToBytes(true)
		pkt.ext = ext[:]
		for _, p := range ps {
			//p2p.packetRw.WriteTo(p.writer)
			if err := p.send(ctx); err != nil && err != ErrDuplicatedPacket {
				p2p.log.Println("Warning", "sendToFriends", err, pkt.protocol, pkt.subProtocol, p.id)
			}
		}
	} else {
		pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
		pkt.extendInfo = newPacketExtendInfo(pkt.extendInfo.hint()+1, 0)
		pkt.footerToBytes(true)
		p2p.sendToPeers(ctx, p2p.friends)
	}
	//TODO 1-hop broadcast with previous received packet-footer
	//TODO clustered, using gateway
}

func (p2p *PeerToPeer) sendRoutine() {
Loop:
	for {
		select {
		case <-p2p.stopCh:
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
				p2p.log.Println("sendRoutine", pkt)
				// p2p.packetRw.WritePacket(pkt)
				r := p2p.getRole()
				switch pkt.dest {
				case p2pDestPeer:
					p := p2p.getPeer(pkt.destPeer, true)
					_ = p.send(ctx)
				case p2pDestAny:
					if pkt.ttl == 1 {
						if r.Has(p2pRoleRoot) {
							p2p.sendToPeers(ctx, p2p.friends)
						}
						if parent := p2p.getParent(); parent != nil {
							_ = parent.send(ctx)
						}
						p2p.sendToPeers(ctx, p2p.uncles)
						p2p.sendToPeers(ctx, p2p.children)
						p2p.sendToPeers(ctx, p2p.nephews)
					} else {
						if r.Has(p2pRoleRoot) {
							p2p.sendToFriends(ctx)
						}
						p2p.sendToPeers(ctx, p2p.children)
						c.alternate = p2p.nephews.Len()
					}
				case p2pRoleRoot: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
					} else {
						if parent := p2p.getParent(); parent != nil {
							_ = parent.send(ctx)
						}
						c.alternate = p2p.uncles.Len()
					}
				case p2pRoleSeed:
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
						if r == p2pRoleRoot {
							p2p.sendToPeers(ctx, p2p.children)
							c.alternate = p2p.nephews.Len()
						}
					} else {
						if parent := p2p.getParent(); parent != nil {
							_ = parent.send(ctx)
						}
						c.alternate = p2p.uncles.Len()
					}
				default: //p2pDestPeerGroup < dest < p2pDestPeer
					//TODO multicast Routing or Flooding
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
Loop:
	for {
		select {
		case <-p2p.stopCh:
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
					p2p.log.Println("Warning", "alternateSendRoutine", "ignore duplicated packet", pkt)
				}
				m[pkt.hashOfPacket] = ctx
			}
		case <-p2p.sendTicker.C:
			for _, ctx := range m {
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				c := ctx.Value(p2pContextKeyCounter).(*Counter)
				switch pkt.dest {
				case p2pDestPeer:
				case p2pDestAny:
					p2p.sendToPeers(ctx, p2p.nephews)
					c.alternate = p2p.nephews.Len()
					p2p.log.Println("alternateSendRoutine", "nephews", c.alternate, pkt.protocol, pkt.subProtocol)
				case p2pRoleRoot: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
					p2p.sendToPeers(ctx, p2p.uncles)
					c.alternate = p2p.uncles.Len()
					p2p.log.Println("alternateSendRoutine", "uncles", c.alternate, pkt.protocol, pkt.subProtocol)
				case p2pRoleSeed: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
					r := p2p.getRole()
					if !r.Has(p2pRoleRoot) {
						p2p.sendToPeers(ctx, p2p.uncles)
						c.alternate = p2p.uncles.Len()
					} else if r == p2pRoleRoot {
						p2p.sendToPeers(ctx, p2p.nephews)
						c.alternate = p2p.nephews.Len()
					}
				default: //p2pDestPeerGroup < dest < p2pDestPeer
				}
				delete(m, pkt.hashOfPacket)

				atomic.StoreInt32(&c.fixed, 1)
				if c.peer < 1 {
					p2p.onFailure(ErrNotAvailable, pkt, c)
				} else {
					//TODO alternate onFailure
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
		pkt.src = p2p.self.id
	}

	if pkt.dest == p2pDestAny && pkt.ttl != 1 &&
		p2p.self.id.Equal(pkt.src) &&
		!p2p.self.compareRole(p2pRoleRoot, false) {
		//BROADCAST_ALL && not relay && not has p2pRoleRoot
		return ErrNotAuthorized
	}

	if !p2p.available(pkt) {
		if pkt.dest == p2pDestAny && pkt.ttl != 1 &&
			p2p.self.compareRole(p2pRoleNone, true) {
			return nil
		}
		//p2p.log.Println("Warning", "Send", "Not Available", pkt.dest, pkt.protocol, pkt.subProtocol)
		return ErrNotAvailable
	}

	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, &Counter{})
	if ok := p2p.sendQueue.Push(ctx, int(pkt.protocol.ID())); !ok {
		p2p.log.Println("Warning", "Send", "Queue Push failure", pkt.protocol, pkt.subProtocol)
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
	p2pContextKeyError   = p2pContextKey("error")
	p2pContextKeyDone    = p2pContextKey("done")
)

//TODO data-race mutex
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

func (p2p *PeerToPeer) getPeer(id module.PeerID, onlyJoin bool) *Peer {
	if id == nil {
		return nil
	}
	if parent := p2p.getParent(); parent != nil && parent.id.Equal(id) {
		return parent
	} else if p := p2p.uncles.GetByID(id); p != nil {
		return p
	} else if p := p2p.children.GetByID(id); p != nil {
		return p
	} else if p := p2p.nephews.GetByID(id); p != nil {
		return p
	} else if p := p2p.friends.GetByID(id); p != nil {
		return p
	}
	if !onlyJoin {
		if p := p2p.orphanages.GetByID(id); p != nil {
			return p
		}
	}
	return nil
}

func (p2p *PeerToPeer) getPeers(onlyJoin bool) []*Peer {
	arr := make([]*Peer, 0)
	if parent := p2p.getParent(); parent != nil {
		arr = append(arr, parent)
	}
	arr = append(arr, p2p.uncles.Array()...)
	arr = append(arr, p2p.children.Array()...)
	arr = append(arr, p2p.nephews.Array()...)
	arr = append(arr, p2p.friends.Array()...)

	if !onlyJoin {
		arr = append(arr, p2p.orphanages.Array()...)
	}
	return arr
}

func (p2p *PeerToPeer) hasNetAddresse(na NetAddress) bool {
	parent := p2p.getParent()
	return p2p.self.netAddress == na ||
		(parent != nil && parent.netAddress == na) ||
		p2p.uncles.HasNetAddresse(na) ||
		p2p.children.HasNetAddresse(na) ||
		p2p.nephews.HasNetAddresse(na) ||
		p2p.friends.HasNetAddresse(na) ||
		p2p.orphanages.HasNetAddresse(na)
}

func (p2p *PeerToPeer) hasNetAddresseAndIncomming(na NetAddress, incomming bool) bool {
	parent := p2p.getParent()
	return p2p.self.netAddress == na ||
		(parent != nil && parent.netAddress == na) ||
		p2p.uncles.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.children.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.nephews.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.friends.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.orphanages.HasNetAddresseAndIncomming(na, incomming)
}

func (p2p *PeerToPeer) connections() map[PeerConnectionType]int {
	m := make(map[PeerConnectionType]int)
	m[p2pConnTypeParent] = 0
	if p2p.getParent() != nil {
		m[p2pConnTypeParent] = 1
	}
	m[p2pConnTypeChildren] = p2p.children.Len()
	m[p2pConnTypeUncle] = p2p.uncles.Len()
	m[p2pConnTypeNephew] = p2p.nephews.Len()
	m[p2pConnTypeFriend] = p2p.friends.Len()
	m[p2pConnTypeNone] = p2p.orphanages.Len()

	return m
}

func (p2p *PeerToPeer) available(pkt *Packet) bool {
	m := p2p.connections()

	u := m[p2pConnTypeParent]
	u += m[p2pConnTypeUncle]
	d := m[p2pConnTypeChildren]
	d += m[p2pConnTypeNephew]
	f := m[p2pConnTypeFriend]
	j := f + u + d

	switch pkt.dest {
	case p2pDestPeer:
		p := p2p.getPeer(pkt.destPeer, true)
		if p == nil {
			return false
		}
	case p2pDestAny:
		if pkt.ttl == 1 {
			if j < 1 {
				return false
			}
		} else {
			if d < 1 && f < 1 {
				return false
			}
		}
	case p2pRoleRoot: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
		if u < 1 && f < 1 {
			return false
		}
	//case p2pRoleSeed:
	default: //p2pDestPeerGroup < dest < p2pDestPeer
		//TODO using route table
		if j < 1 {
			return false
		}
	}
	return true
}

func (p2p *PeerToPeer) isAllowedRole(role PeerRoleFlag, p *Peer) bool {
	switch role {
	case p2pRoleSeed:
		//p2p.log.Println("isAllowedRole p2pRoleSeed", p2p.allowedSeeds)
		return p2p.allowedSeeds.IsEmpty() || p2p.allowedSeeds.Contains(p.id)
	case p2pRoleRoot:
		//p2p.log.Println("isAllowedRole p2pRoleRoot", p2p.allowedRoots)
		return p2p.allowedRoots.IsEmpty() || p2p.allowedRoots.Contains(p.id)
	case p2pRoleRootSeed:
		return p2p.isAllowedRole(p2pRoleRoot, p) && p2p.isAllowedRole(p2pRoleSeed, p)
	default:
		return true
	}
}

//Dial to seeds, roots, nodes and create p2p connection
func (p2p *PeerToPeer) discoverRoutine() {
Loop:
	for {
		select {
		case <-p2p.stopCh:
			break Loop
		case <-p2p.seedTicker.C:
			seeds := p2p.orphanages.GetByRoleAndIncomming(p2pRoleSeed, true, false)
			minSeed := DefaultMinSeed
			if p2p.seeds.Contains(p2p.self.netAddress) {
				minSeed++
			}
			if p2p.syncSeeds() {
				for _, s := range p2p.seeds.Array() {
					if !p2p.hasNetAddresse(s) {
						p2p.log.Println("discoverRoutine", "seedTicker", "dial to p2pRoleSeed", s)
						if err := p2p.dial(s); err != nil {
							if p2p.seeds.Len() > minSeed {
								p2p.seeds.Remove(s)
							}
						}
					}
				}
				for _, p := range seeds {
					p2p.sendQuery(p)
				}
			} else {
				for _, p := range seeds {
					p2p.log.Println("discoverRoutine", "seedTicker", "no need outgoing p2pRoleSeed connection")
					p.Close("discoverRoutine no need outgoing p2pRoleSeed connection")
				}
			}
		case <-p2p.discoveryTicker.C:
			r := p2p.getRole()
			pr := PeerRoleFlag(p2pRoleSeed)
			strRole := "p2pRoleSeed"
			s := p2p.seeds
			if r == p2pRoleSeed {
				pr = PeerRoleFlag(p2pRoleRoot)
				strRole = "p2pRoleRoot"
				s = p2p.roots
			}

			if r.Has(p2pRoleRoot) {
				p2p.discoverFriends()
			} else {
				p2p.discoverParent(pr)
				if p2p.getParent() != nil {
					p2p.discoverUncle(pr)
				}

				n := DefaultUncleLimit + 1 - p2p.uncles.Len() - p2p.pre.Len()
				if p2p.getParent() != nil {
					n--
				}
				dialed := 0
			NetAddressSetLoop:
				for _, na := range s.Array() {
					if dialed >= n {
						break NetAddressSetLoop
					}
					if !p2p.hasNetAddresseAndIncomming(na, false) {
						p2p.log.Println("discoverRoutine", "discoveryTicker", "dial to", strRole, na)
						if err := p2p.dial(na); err == nil {
							dialed++
						}
					}
				}
				if dialed < n {
					p2p.reject.Clear()
				}
			}
		}
	}
}

func (p2p *PeerToPeer) syncSeeds() (connectAndQuery bool) {
	role := p2p.getRole()
	if role.Has(p2pRoleRoot) {
		if (p2p.children.Len() + p2p.nephews.Len()) < 1 {
			connectAndQuery = true
		}
		for _, p := range p2p.friends.Array() {
			if !p.incomming {
				p2p.sendQuery(p)
			}
		}
	} else {
		parent := p2p.getParent()
		if parent == nil || p2p.uncles.Len() < DefaultUncleLimit {
			connectAndQuery = true
		}
		if parent != nil {
			p2p.sendQuery(parent)
		}
		for _, p := range p2p.uncles.Array() {
			p2p.sendQuery(p)
		}

		if role == p2pRoleSeed {
			roots := p2p.orphanages.GetByRoleAndIncomming(p2pRoleRoot, false, false)
			for _, p := range roots {
				p2p.sendQuery(p)
			}
		}
	}
	return
}

func (p2p *PeerToPeer) discoverFriends() {
	nones := p2p.friends.GetByRole(p2pRoleNone, true)
	for _, p := range nones {
		p2p.log.Println("discoverFriends", "not allowed connection from p2pRoleNone", p.id)
		p.Close("discoverFriends not allowed connection from p2pRoleNone")
	}
	seeds := p2p.friends.GetByRole(p2pRoleSeed, true)
	for _, p := range seeds {
		p2p.updatePeerConnectionType(p, p2pConnTypeNone)
	}
	//in_roots := p2p.orphanages.GetByRoleAndIncomming(p2pRoleRoot, false, true)
	//out_roots := p2p.orphanages.GetByRoleAndIncomming(p2pRoleRoot, false, false)
	roots := p2p.orphanages.GetByRole(p2pRoleRoot, false)
	for _, p := range roots {
		p2p.log.Println("discoverFriends", "p2pConnTypeFriend", p.id)
		p2p.updatePeerConnectionType(p, p2pConnTypeFriend)
	}
	for _, na := range p2p.roots.Array() {
		if p2p.self.netAddress != na &&
			!p2p.orphanages.HasNetAddresse(na) &&
			!p2p.friends.HasNetAddresse(na) {
			p2p.log.Println("discoverFriends", "dial to p2pRoleRoot", na)
			if err := p2p.dial(na); err != nil {
				p2p.roots.Remove(na)
			}
		}
	}
}

func (p2p *PeerToPeer) discoverParent(pr PeerRoleFlag) {
	//TODO connection between p2pRoleNone
	if p2p.getParent() != nil {
		p2p.log.Println("discoverParent", "nothing to do")
		return
	}

	if p2p.pre.Len() > 0 {
		p2p.log.Println("discoverParent", "waiting P2PConnectionResponse")
		return
	}

	peers := p2p.orphanages.GetByRoleAndIncomming(pr, false, false)
	if len(peers) < 1 {
		peers = p2p.uncles.GetByRoleAndIncomming(pr, false, false)
	}
	if len(peers) < 1 {
		return
	}
	sort.Slice(peers, func(i, j int) bool {
		il := len(peers[i].children)
		jl := len(peers[j].children)
		if il == jl {
			return peers[i].rtt.avg < peers[j].rtt.avg
		} else {
			return il < jl
		}
	})
	for _, p := range peers {
		if !p2p.reject.Contains(p) && !p2p.pre.Contains(p) {
			p2p.pre.Add(p)
			p2p.sendP2PConnectionRequest(p2pConnTypeParent, p)
			p2p.log.Println("discoverParent", "try p2pConnTypeParent", p.ID(), p.connType)
			return
		}
	}
}

func (p2p *PeerToPeer) discoverUncle(ur PeerRoleFlag) {
	if p2p.uncles.Len() >= DefaultUncleLimit {
		p2p.log.Println("discoverUncle", "nothing to do")
		return
	}

	n := DefaultUncleLimit - p2p.uncles.Len() - p2p.pre.Len()
	if n < 1 {
		p2p.log.Println("discoverUncle", "waiting P2PConnectionResponse")
		return
	}

	peers := p2p.orphanages.GetByRoleAndIncomming(ur, false, false)
	if len(peers) < 1 {
		return
	}
	sort.Slice(peers, func(i, j int) bool {
		il := peers[i].nephews
		jl := peers[j].nephews
		if il == jl {
			return peers[i].rtt.avg < peers[j].rtt.avg
		} else {
			return il < jl
		}
	})
	//sort.Slice(peers, func(i, j int) bool { return peers[i].rtt.avg < peers[j].rtt.avg })
	for _, p := range peers {
		if n < 1 {
			return
		}
		if !p2p.reject.Contains(p) && !p2p.pre.Contains(p) {
			p2p.pre.Add(p)
			p2p.sendP2PConnectionRequest(p2pConnTypeUncle, p)
			p2p.log.Println("discoverUncle", "try p2pConnTypeUncle", p.ID(), p.connType)
			n--
		}
	}
}

func (p2p *PeerToPeer) updatePeerConnectionType(p *Peer, connType PeerConnectionType) (updated bool) {
	if p.connType == connType {
		return
	}

	pre := p.connType
	var preset *PeerSet
	var tset *PeerSet
	var rset *PeerSet
	var limit int = -1
	switch pre {
	case p2pConnTypeNone:
		preset = p2p.orphanages
	case p2pConnTypeUncle:
		preset = p2p.uncles
	case p2pConnTypeNephew:
		preset = p2p.nephews
	case p2pConnTypeFriend:
		preset = p2p.friends
	}
	if preset != nil {
		preset.Remove(p)
	}

	p.connType = connType
	switch connType {
	case p2pConnTypeParent:
		p2p.setParent(p)
		p2p.reject.Clear()
		p2p.log.Println("updatePeerConnectionType", "complete", strPeerConnectionType[connType])
	case p2pConnTypeUncle:
		tset = p2p.uncles
		rset = p2p.reject
		limit = DefaultUncleLimit
	case p2pConnTypeChildren:
		tset = p2p.children
		limit = DefaultChildrenLimit
	case p2pConnTypeNephew:
		tset = p2p.nephews
		limit = DefaultNephewLimit
	case p2pConnTypeFriend:
		tset = p2p.friends
	case p2pConnTypeNone:
		tset = p2p.orphanages
	}

	updated = true
	if tset != nil {
		tset.Add(p)
		tl := tset.Len()
		if limit > -1 {
			if tl > limit {
				p.connType = pre
				tset.Remove(p)
				preset.Add(p)
				updated = false
			} else {
				if tl == limit {
					p2p.log.Println("updatePeerConnectionType", "complete", strPeerConnectionType[connType])
					if rset != nil {
						rset.Clear()
					}
				}
			}
		}
	}

	if updated {
		if pre == p2pConnTypeNone {
			p2p.onEvent(p2pEventJoin, p)
		}
		if connType == p2pConnTypeNone {
			p2p.onEvent(p2pEventLeave, p)
		}
	}
	return
}

//TODO timestamp or sequencenumber for validation (request,response pair)
type P2PConnectionRequest struct {
	ConnType PeerConnectionType
}

type P2PConnectionResponse struct {
	ReqConnType PeerConnectionType
	ConnType    PeerConnectionType
}

func (p2p *PeerToPeer) sendP2PConnectionRequest(connType PeerConnectionType, p *Peer) {
	m := &P2PConnectionRequest{ConnType: connType}
	pkt := newPacket(PROTO_P2P_CONN_REQ, p2p.encodeMsgpack(m), p2p.self.id)
	pkt.destPeer = p.id
	err := p.sendPacket(pkt)
	if err != nil {
		p2p.log.Println("Warning", "sendP2PConnectionRequest", err, p)
	} else {
		p2p.log.Println("sendP2PConnectionRequest", m, p)
	}
}
func (p2p *PeerToPeer) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := p2p.decodeMsgpack(pkt.payload, req)
	if err != nil {
		p2p.log.Println("Warning", "handleP2PConnectionRequest", err, p)
		return
	}
	p2p.log.Println("handleP2PConnectionRequest", req, p)
	m := &P2PConnectionResponse{ConnType: p2pConnTypeNone}
	switch req.ConnType {
	case p2pConnTypeParent:
		//TODO p2p.children condition
		switch p.connType {
		case p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, p2pConnTypeChildren)
		case p2pConnTypeNephew:
			p2p.updatePeerConnectionType(p, p2pConnTypeChildren)
		default:
			p2p.log.Println("handleP2PConnectionRequest", "ignore", req.ConnType, "from", p.connType)
		}
	case p2pConnTypeUncle:
		//TODO p2p.nephews condition
		switch p.connType {
		case p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, p2pConnTypeNephew)
		default:
			p2p.log.Println("handleP2PConnectionRequest", "ignore", req.ConnType, "from", p.connType)
		}
	default:
		p2p.log.Println("handleP2PConnectionRequest", "invalid reqConnType", req.ConnType, "from", p.connType)
	}
	m.ReqConnType = req.ConnType
	m.ConnType = p.connType

	rpkt := newPacket(PROTO_P2P_CONN_RESP, p2p.encodeMsgpack(m), p2p.self.id)
	rpkt.destPeer = p.id
	err = p.sendPacket(rpkt)
	if err != nil {
		p2p.log.Println("Warning", "handleP2PConnectionRequest", "sendP2PConnectionResponse", err, p)
	} else {
		p2p.log.Println("handleP2PConnectionRequest", "sendP2PConnectionResponse", m, p)
	}
}

func (p2p *PeerToPeer) handleP2PConnectionResponse(pkt *Packet, p *Peer) {
	resp := &P2PConnectionResponse{}
	err := p2p.decodeMsgpack(pkt.payload, resp)
	if err != nil {
		p2p.log.Println("Warning", "handleP2PConnectionResponse", err, p)
		return
	}
	p2p.log.Println("handleP2PConnectionResponse", resp, p)

	p2p.pre.Remove(p)
	switch resp.ReqConnType {
	case p2pConnTypeParent:
		if p2p.getParent() != nil {
			p2p.log.Println("handleP2PConnectionResponse already p2pConnTypeParent", resp, p)
			return
		}
		if resp.ConnType != p2pConnTypeChildren {
			p2p.reject.Add(p)
			return
		}
		switch p.connType {
		case p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, p2pConnTypeParent)
		case p2pConnTypeUncle:
			p2p.updatePeerConnectionType(p, p2pConnTypeParent)
		default:
			p2p.log.Println("Warning", "handleP2PConnectionResponse", "p2pConnTypeParent wrong connType", resp, p)
			p.CloseByError(fmt.Errorf("handleP2PConnectionResponse p2pConnTypeParent wrong connType:%v", p.connType))
		}
	case p2pConnTypeUncle:
		if p2p.uncles.Len() >= DefaultUncleLimit {
			p2p.log.Println("handleP2PConnectionResponse already p2pConnTypeUncle", resp, p)
			return
		}
		if resp.ConnType != p2pConnTypeNephew {
			p2p.reject.Add(p)
			return
		}
		switch p.connType {
		case p2pConnTypeNone:
			p2p.updatePeerConnectionType(p, p2pConnTypeUncle)
		default:
			p2p.log.Println("Warning", "handleP2PConnectionResponse", "p2pConnTypeUncle wrong connType", resp, p)
			p.CloseByError(fmt.Errorf("handleP2PConnectionResponse p2pConnTypeUncle wrong connType:%v", p.connType))
		}
	default:
		p2p.log.Println("handleP2PConnectionResponse", "invalid ReqConnType", resp, p)
	}
}
