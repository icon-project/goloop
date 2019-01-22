package network

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/go-errors/errors"
	"github.com/ugorji/go/codec"

	"github.com/icon-project/goloop/module"
)

type PeerToPeer struct {
	channel         string
	sendQueue       *WeightQueue
	alternateQueue  *Queue
	sendTicker      *time.Ticker
	onPacketCbFuncs map[uint16]packetCbFunc
	onErrorCbFuncs  map[uint16]errorCbFunc
	onEventCbFuncs  map[string]map[uint16]eventCbFunc
	packetPool      *PacketPool
	packetRw        *PacketReadWriter
	dialer          *Dialer

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

	//Discovery
	discoveryTicker *time.Ticker
	seedTicker      *time.Ticker
	duplicated      *Set
	dialing         *NetAddressSet

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
	//codec
	mph *codec.MsgpackHandle

	//log
	log *logger
}

type eventCbFunc func(evt string, p *Peer)

const (
	p2pEventJoin       = "join"
	p2pEventLeave      = "leave"
	p2pEventDuplicate  = "duplicate"
	p2pEventNotAllowed = "not allowed"
)

//can be crea`ted each channel
func newPeerToPeer(channel string, self *Peer, d *Dialer) *PeerToPeer {
	p2p := &PeerToPeer{
		channel:         channel,
		sendQueue:       NewWeightQueue(DefaultSendQueueSize, DefaultSendQueueMaxPriority+1),
		alternateQueue:  NewQueue(DefaultSendQueueSize),
		sendTicker:      time.NewTicker(DefaultAlternateSendPeriod),
		onPacketCbFuncs: make(map[uint16]packetCbFunc),
		onErrorCbFuncs:  make(map[uint16]errorCbFunc),
		onEventCbFuncs:  make(map[string]map[uint16]eventCbFunc),
		packetPool:      NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen),
		packetRw:        NewPacketReadWriter(),
		dialer:          d,
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
		dialing:         NewNetAddressSet(),
		//
		seeds:         NewNetAddressSet(),
		roots:         NewNetAddressSet(),
		grandChildren: NewNetAddressSet(),
		//
		allowedRoots: NewPeerIDSet(),
		allowedSeeds: NewPeerIDSet(),
		allowedPeers: NewPeerIDSet(),
		//
		mph: &codec.MsgpackHandle{},
		//
		log: newLogger("PeerToPeer", fmt.Sprintf("%s.%s", channel, self.id)),
	}
	p2p.mph.MapType = reflect.TypeOf(map[string]interface{}(nil))
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
				p.CloseByError(errors.New("onUpdate not allowed connection"))
			}
		}
	}

	p2p.log.excludes = []string{
		//"onPeer",
		//"onClose",
		//"onEvent",
		"onPacket",
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
	}

	go p2p.sendRoutine()
	go p2p.alternateSendRoutine()
	go p2p.discoverRoutine()
	return p2p
}

func (p2p *PeerToPeer) dial(na NetAddress) error {
	//TODO dialing context
	if !p2p.dialing.Add(na) {
		p2p.log.Println("Warning", "Already Dialing", na)
		return nil
	}
	if err := p2p.dialer.Dial(string(na)); err != nil {
		p2p.log.Println("Warning", "Dial fail", na, err)
		p2p.dialing.Remove(na)
		return err
	}
	return nil
}

func (p2p *PeerToPeer) setCbFunc(pi module.ProtocolInfo, pktFunc packetCbFunc,
	errFunc errorCbFunc, evtFunc eventCbFunc, evts ...string) {
	k := pi.Uint16()
	if _, ok := p2p.onPacketCbFuncs[k]; ok {
		p2p.log.Println("Warning", "overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[k] = pktFunc
	p2p.onErrorCbFuncs[k] = errFunc
	for _, evt := range evts {
		p2p.setEventCbFunc(evt, k, evtFunc)
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

//callback from PeerDispatcher.onPeer
func (p2p *PeerToPeer) onPeer(p *Peer) {
	p2p.log.Println("onPeer", p)
	if !p.incomming {
		p2p.dialing.Remove(p.netAddress)
	}
	if !p2p.allowedPeers.IsEmpty() && !p2p.allowedPeers.Contains(p.id) {
		p2p.onEvent(p2pEventNotAllowed, p)
		p.CloseByError(errors.New("onPeer not allowed connection"))
		return
	}
	if dp := p2p.getPeer(p.id, false); dp != nil {
		if p2p.removePeer(dp) {
			p2p.onEvent(p2pEventDuplicate, p)
		}
		p2p.duplicated.Add(dp)
		if dp.incomming == p.incomming {
			dp.CloseByError(errors.New("onPeer duplicated peer"))
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

	p.CloseByError(err)

	//Peer.receiveRoutine
	//// bufio.Reader.Read error except {net.OpError, io.EOF, io.ErrUnexpectedEOF}
	//Peer.sendRoutine
	//// net.Conn.SetWriteDeadline error
	//// bufio.Writer.Write error
	//// bufio.Writer.Flush error

	if pkt != nil {
		if cbFunc, ok := p2p.onErrorCbFuncs[pkt.protocol.Uint16()]; ok {
			cbFunc(err, p, pkt)
		}
	}
}

func (p2p *PeerToPeer) onClose(p *Peer) {
	p2p.log.Println("onClose", p.CloseInfo(), p)
	if p2p.removePeer(p) {
		p2p.onEvent(p2pEventLeave, p)

		for ctx := p.q.Pop(); ctx != nil; ctx = p.q.Pop() {
			c := ctx.Value(p2pContextKeyCounter).(*Counter)
			c.close++
			if c.close == c.enqueue {
				//TODO onFailure ErrNotAvailable
			}
		}
	}
}

func (p2p *PeerToPeer) onEvent(evt string, p *Peer) {
	p2p.log.Println("onEvent", evt, p)
	if m, ok := p2p.onEventCbFuncs[evt]; ok {
		for _, cbFunc := range m {
			cbFunc(evt, p)
		}
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
		p2p.parent = nil
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
		if pkt.ttl == 1 && !p.id.Equal(pkt.src) {
			p2p.log.Println("Warning", "onPacket", "Drop, Invalid 1hop-src:", pkt.src, ",expected:", p.id, pkt.protocol, pkt.subProtocol)
		} else if p2p.self.id.Equal(pkt.src) {
			p2p.log.Println("Warning", "onPacket", "Drop, Invalid self-src", pkt.src, pkt.protocol, pkt.subProtocol)
		} else if cbFunc := p2p.onPacketCbFuncs[pkt.protocol.Uint16()]; cbFunc != nil {
			if p.connType == p2pConnTypeNone {
				//TODO drop from p.connType == p2pConnTypeNone
				p2p.log.Println("Warning", "onPacket", "undetermined PeerConnectionType", pkt.protocol, pkt.subProtocol)
			}
			if pkt.ttl == 1 || p2p.packetPool.Put(pkt) {
				cbFunc(pkt, p)
			} else {
				//TODO drop counting each (protocol,subProtocol)
				p2p.log.Println("onPacket", "Drop, Duplicated by hash", pkt.protocol, pkt.subProtocol, pkt.hashOfPacket, p.id)
			}
		}
	}
}

func (p2p *PeerToPeer) encodeMsgpack(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.NewEncoderBytes(&b, p2p.mph)
	enc.MustEncode(v)
	return b
}

func (p2p *PeerToPeer) decodeMsgpack(b []byte, v interface{}) error {
	dec := codec.NewDecoderBytes(b, p2p.mph)
	return dec.Decode(v)
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
	p2p.log.Println("setRoleByAllowedSet", p2p.getRole())
	return role
}

func (p2p *PeerToPeer) getRole() PeerRoleFlag {
	return p2p.self.getRole()
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.getRole()}
	pkt := newPacket(PROTO_P2P_QUERY, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
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
		if qm.Role == p2pRoleNone {
			switch m.Role {
			case p2pRoleSeed:
			case p2pRoleRoot:
				//TODO hiding Root role
				p2p.log.Println("Warning", "handleQuery", "p2pRoleNone cannot query to p2pRoleRoot", p)
				m.Message = "not allowed to query"
				m.Seeds = nil
				m.Children = nil
				m.Nephews = nil
				//p.Close()
			case p2pRoleRootSeed:
				//TODO hiding RootSeed role
				m.Role = p2pRoleSeed
			}
		} else {
			m.Roots = p2p.roots.Array()
		}
	} else {
		m.Message = "not exists allowedlist"
	}
	rpkt := newPacket(PROTO_P2P_QUERY_RESULT, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
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
		if role == p2pRoleNone {
			switch qrm.Role {
			case p2pRoleNone:
				//TODO p2p.preParent.Merge(qrm.Children)
			case p2pRoleSeed:
				p2p.seeds.Merge(qrm.Seeds...)
			case p2pRoleRoot:
				p2p.log.Println("Warning", "handleQueryResult", "p2pRoleNone cannot query to p2pRoleRoot", p)
				p.CloseByError(errors.New("handleQueryResult p2pRoleNone cannot query to p2pRoleRoot"))
				return
			case p2pRoleRootSeed:
				//TODO hiding RootSeed role
				p2p.seeds.Merge(qrm.Seeds...)
			default:
			}
		} else {
			p2p.seeds.Merge(qrm.Seeds...)
			p2p.roots.Merge(qrm.Roots...)
		}
	} else {
		p2p.log.Println("handleQueryResult", "not exists allowedlist", p)
		p.CloseByError(errors.New("handleQueryResult not exists allowedlist"))
		return
	}

	m := &RttMessage{Last: p.rtt.last, Average: p.rtt.avg}
	rpkt := newPacket(PROTO_P2P_RTT_REQ, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
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
	rpkt := newPacket(PROTO_P2P_RTT_RESP, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
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

func (p2p *PeerToPeer) sendToFriends(ctx context.Context) {
	//TODO clustered, using gateway
	p2p.sendToPeers(ctx, p2p.friends)
}

func (p2p *PeerToPeer) sendRoutine() {
	// TODO goroutine exit
	for {
		<-p2p.sendQueue.Wait()
		for {
			ctx := p2p.sendQueue.Pop()
			if ctx == nil {
				break
			}
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			c := ctx.Value(p2pContextKeyCounter).(*Counter)
			p2p.log.Println("sendRoutine", pkt)
			if pkt.src == nil {
				pkt.src = p2p.self.id
			}
			// p2p.packetRw.WritePacket(pkt)
			r := p2p.getRole()
			switch pkt.dest {
			case p2pDestPeer:
				p := p2p.getPeer(pkt.destPeer, true)
				_ = p.send(ctx)
			case p2pDestAny:
				if pkt.ttl == 1 {
					if r.Has(p2pRoleRoot) {
						p2p.sendToFriends(ctx)
					}
					_ = p2p.parent.send(ctx)
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
					_ = p2p.parent.send(ctx)
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
					_ = p2p.parent.send(ctx)
					c.alternate = p2p.uncles.Len()
				}
			default: //p2pDestPeerGroup < dest < p2pDestPeer
				//TODO multicast Routing or Flooding
			}

			if c.alternate < 1 {
				if c.peer < 1 {
					//TODO onFailure ErrNotAvailable
				} else if c.enqueue < 1 {
					//TODO onFailure ErrQueueOverflow
				}
			} else if !p2p.alternateQueue.Push(ctx) && c.enqueue < 1 {
				//TODO onFailure ErrQueueOverflow
			}
		}
	}
}

func (p2p *PeerToPeer) alternateSendRoutine() {
	var m = make(map[uint64]context.Context)
	for {
		select {
		case <-p2p.alternateQueue.Wait():
			for {
				ctx := p2p.alternateQueue.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
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
				if c.peer < 1 {
					//TODO onFailure ErrNotAvailable
				} else if c.enqueue < 1 {
					//TODO onFailure ErrQueueOverflow
				}
				delete(m, pkt.hashOfPacket)
			}
		}
	}
}

func (p2p *PeerToPeer) send(pkt *Packet) error {
	if !p2p.available(pkt) {
		//p2p.log.Println("Warning", "send", "Not Available", pkt.dest, pkt.protocol, pkt.subProtocol)
		return ErrNotAvailable
	}

	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, &Counter{})
	if ok := p2p.sendQueue.Push(ctx, int(pkt.protocol.ID())); !ok {
		p2p.log.Println("Warning", "send", "Queue Push failure", pkt.protocol, pkt.subProtocol)
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

type Counter struct {
	peer      int
	alternate int
	enqueue   int
	//
	duplicate int
	overflow  int
	close     int
}

func (p2p *PeerToPeer) getPeer(id module.PeerID, onlyJoin bool) *Peer {
	if id == nil {
		return nil
	}
	if p2p.parent != nil && p2p.parent.id.Equal(id) {
		return p2p.parent
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
	if p2p.parent != nil {
		arr = append(arr, p2p.parent)
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
	return p2p.self.netAddress == na ||
		(p2p.parent != nil && p2p.parent.netAddress == na) ||
		p2p.uncles.HasNetAddresse(na) ||
		p2p.children.HasNetAddresse(na) ||
		p2p.nephews.HasNetAddresse(na) ||
		p2p.friends.HasNetAddresse(na) ||
		p2p.orphanages.HasNetAddresse(na)
}

func (p2p *PeerToPeer) hasNetAddresseAndIncomming(na NetAddress, incomming bool) bool {
	return p2p.self.netAddress == na ||
		(p2p.parent != nil && p2p.parent.netAddress == na) ||
		p2p.uncles.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.children.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.nephews.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.friends.HasNetAddresseAndIncomming(na, incomming) ||
		p2p.orphanages.HasNetAddresseAndIncomming(na, incomming)
}

func (p2p *PeerToPeer) connections() map[PeerConnectionType]int {
	m := make(map[PeerConnectionType]int)
	m[p2pConnTypeParent] = 0
	if p2p.parent != nil {
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
		//TODO multicast Routing or Flooding
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
	//TODO goroutine exit
	for {
		select {
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
				if p2p.parent != nil {
					p2p.discoverUncle(pr)
				}

				n := DefaultUncleLimit + 1 - p2p.uncles.Len() - p2p.pre.Len()
				if p2p.parent != nil {
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
		if p2p.parent == nil || p2p.uncles.Len() < DefaultUncleLimit {
			connectAndQuery = true
		}
		if p2p.parent != nil {
			p2p.sendQuery(p2p.parent)
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
	if p2p.parent != nil {
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
		p2p.parent = p
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
	pkt := newPacket(PROTO_P2P_CONN_REQ, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
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

	rpkt := newPacket(PROTO_P2P_CONN_RESP, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
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
		if p2p.parent != nil {
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
