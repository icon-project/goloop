package network

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/ugorji/go/codec"
)

type PeerToPeer struct {
	channel         string
	sendQueue       *Queue
	alternateQueue  *Queue
	sendTicker      *time.Ticker
	onPacketCbFuncs map[uint16]packetCbFunc
	onErrorCbFuncs  map[uint16]errorCbFunc
	onEventCbFuncs  map[string]map[uint16]eventCbFunc
	packetPool      *PacketPool
	packetRw        *PacketReadWriter
	transport       module.NetworkTransport

	//Topology with Connected Peers
	self      *Peer
	parent    *Peer
	preParent *PeerSet
	children  *PeerSet
	uncles    *PeerSet
	preUncles *PeerSet
	nephews   *PeerSet
	//Only for root, parent is nil, uncles is empty
	friends *PeerSet
	//Discovery
	orphanages      *PeerSet
	discoveryTicker *time.Ticker
	seedTicker      *time.Ticker
	duplicated      *Set
	dialing         *NetAddressSet

	//Addresses
	seeds *NetAddressSet
	//For seed, root
	roots *NetAddressSet
	//[TBD] 2hop peers of current tree for status change
	grandParent   NetAddress
	grandChildren *NetAddressSet

	//managed PeerId
	allowedRoots *PeerIDSet
	allowedSeeds *PeerIDSet

	//codec
	mph *codec.MsgpackHandle

	//log
	log *logger
}

type eventCbFunc func(evt string, p *Peer)
type eventHandler struct {
	evt     string
	onEvent eventCbFunc
}

const (
	p2pEventJoin  = "join"
	p2pEventLeave = "leave"
)

//can be created each channel
func newPeerToPeer(channel string, t module.NetworkTransport) *PeerToPeer {
	id := t.PeerID()
	netAddress := NetAddress(t.Address())
	p2p := &PeerToPeer{
		channel:         channel,
		sendQueue:       NewQueue(DefaultSendQueueSize),
		alternateQueue:  NewQueue(DefaultSendQueueSize),
		sendTicker:      time.NewTicker(DefaultAlternateSendPeriod),
		onPacketCbFuncs: make(map[uint16]packetCbFunc),
		onErrorCbFuncs:  make(map[uint16]errorCbFunc),
		onEventCbFuncs:  make(map[string]map[uint16]eventCbFunc),
		packetPool:      NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen),
		packetRw:        NewPacketReadWriter(),
		transport:       t,
		//
		self:            &Peer{id: id, netAddress: netAddress},
		preParent:       NewPeerSet(),
		children:        NewPeerSet(),
		uncles:          NewPeerSet(),
		preUncles:       NewPeerSet(),
		nephews:         NewPeerSet(),
		friends:         NewPeerSet(),
		orphanages:      NewPeerSet(),
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
		//
		mph: &codec.MsgpackHandle{},
		//
		log: newLogger("PeerToPeer", fmt.Sprintf("%s.%s", channel, id)),
	}
	p2p.mph.MapType = reflect.TypeOf(map[string]interface{}(nil))
	p2p.allowedRoots.onUpdate = func() {
		p2p.setRoleByAllowedSet()
	}
	p2p.allowedSeeds.onUpdate = func() {
		p2p.setRoleByAllowedSet()
	}
	t.(*transport).pd.registPeerToPeer(p2p)
	p2p.log.excludes = []string{
		"sendQuery",
		"handleQuery",
		"sendRoutine",
		"alternateSendRoutine",
		"sendToPeer",
		"onPacket",
		"onPeer",
	}

	go p2p.sendRoutine()
	go p2p.alternateSendRoutine()
	go p2p.discoveryRoutine()
	return p2p
}

func (p2p *PeerToPeer) dial(na NetAddress) error {
	//TODO dialing context
	if !p2p.dialing.Add(na) {
		p2p.log.Println("Warning", "Already Dialing", na)
		return nil
	}
	if err := p2p.transport.Dial(string(na), p2p.channel); err != nil {
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
		m := p2p.onEventCbFuncs[evt]
		if m == nil {
			m = make(map[uint16]eventCbFunc)
			p2p.onEventCbFuncs[evt] = m
		}
		m[k] = evtFunc
	}
}

//callback from PeerDispatcher.onPeer
func (p2p *PeerToPeer) onPeer(p *Peer) {
	p2p.log.Println("onPeer", p)
	if !p.incomming {
		p2p.dialing.Remove(p.netAddress)
	}
	if dp := p2p.getPeer(p.id, false); dp != nil {
		p2p.log.Println("Warning", "Already exists connected Peer, close duplicated peer", dp)
		p2p.removePeer(dp)
		p2p.duplicated.Add(dp)
		dp.Close()
	}
	p2p.orphanages.Add(p)
	if !p.incomming {
		p2p.sendQuery(p)
	} else {
		//peer can be children or nephews
	}
	//TODO discoveryRoutine with time.Ticker
}

//callback from Peer.sendRoutine or Peer.receiveRoutine
func (p2p *PeerToPeer) onError(err error, p *Peer, pkt *Packet) {
	p2p.log.Println("Warning", "onError", err, p, pkt)

	p.Close()

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
	p2p.log.Println("onClose", p)
	if p2p.removePeer(p) {
		p2p.onEvent(p2pEventLeave, p)
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

	if p.hasRole(p2pRoleSeed) {
		p2p.removeSeed(p)
		p2p.seeds.Add(p.netAddress)
	}
	if p.hasRole(p2pRoleRoot) {
		p2p.removeRoot(p)
		p2p.roots.Add(p.netAddress)
	}
	isLeave = true
	switch p.connType {
	case p2pConnTypeNone:
		p2p.orphanages.Remove(p)
		isLeave = false
	case p2pConnTypePre:
		p2p.preParent.Remove(p)
		p2p.preUncles.Remove(p)
		isLeave = false
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
			case PROTO_P2P_CONN_REQ:
				p2p.handleP2PConnectionRequest(pkt, p)
			case PROTO_P2P_CONN_RESP:
				p2p.handleP2PConnectionResponse(pkt, p)
			}
		}
	} else {
		if p.connType == p2pConnTypeNone {
			p2p.log.Println("Warning", "onPacket", "Ignore, undetermined PeerConnectionType")
		} else if pkt.ttl == 1 && !p.id.Equal(pkt.src) {
			p2p.log.Println("Warning", "onPacket", "Ignore, Invalid Packet.src:", pkt.src, ",expected:", p.id)
		} else if cbFunc := p2p.onPacketCbFuncs[pkt.protocol.Uint16()]; cbFunc != nil {
			if !p2p.packetPool.Contains(pkt) && !p2p.self.id.Equal(pkt.src) {
				p2p.packetPool.Put(pkt)
				cbFunc(pkt, p)
			} else {
				p2p.log.Println("onPacket", "Ignore, duplicated", pkt.hashOfPacket)
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

type QueryMessage struct {
	Role PeerRoleFlag
}

type QueryResultMessage struct {
	Role     PeerRoleFlag
	Seeds    []NetAddress
	Roots    []NetAddress
	Children []NetAddress
	Message  string
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

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) {
	if !p2p.self.eqaulRole(r) {
		p2p.self.setRole(r)
		p2p.applyPeerRole(p2p.self)
	}
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
	p2p.log.Println("setRoleByAllowedSet", p2p.getRole())
	return role
}

func (p2p *PeerToPeer) getRole() PeerRoleFlag {
	return p2p.self.getRole()
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.getRole()}
	pkt := NewPacket(PROTO_P2P_QUERY, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.rtt.Start()
	p.send(pkt)
	p2p.log.Println("sendQuery", m, p)
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qm)
	if err != nil {
		p2p.log.Println("Warning", "handleQuery", err)
		return
	}
	p2p.log.Println("handleQuery", qm, p)
	m := &QueryResultMessage{}
	m.Role = p2p.getRole()
	if p2p.isAllowedRole(qm.Role, p) {
		p.setRole(qm.Role)
		p2p.applyPeerRole(p)
		if qm.Role == p2pRoleNone {
			m.Children = p2p.children.NetAddresses()
			switch m.Role {
			case p2pRoleSeed:
				m.Seeds = p2p.seeds.Array()
			case p2pRoleRoot:
				p2p.log.Println("Warning", "handleQuery", "p2pRoleNone cannot query to p2pRoleRoot")
				m.Message = "not allowed to query"
				m.Children = nil
				//p.Close()
			case p2pRoleRootSeed:
				//TODO hiding RootSeed role
				m.Role = p2pRoleSeed
				m.Seeds = p2p.seeds.Array()
			}
		} else {
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			if m.Role == p2pRoleSeed {
				//p.conn will be disconnected
			}
		}
	} else {
		m.Message = "not exists allowedlist"
		//p.Close()
	}
	rpkt := NewPacket(PROTO_P2P_QUERY_RESULT, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
	p.send(rpkt)
}

func (p2p *PeerToPeer) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qrm)
	if err != nil {
		p2p.log.Println("Warning", "handleQueryResult", err)
		return
	}
	p2p.log.Println("handleQueryResult", qrm)
	p.rtt.Stop()
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
				p2p.log.Println("Warning", "handleQueryResult", "p2pRoleNone cannot query to p2pRoleRoot")
			case p2pRoleRootSeed:
				//TODO hiding RootSeed role
				p2p.seeds.Merge(qrm.Seeds...)
			default:
				//TODO p2p.preParent.Merge(qrm.Children)
			}
		} else {
			p2p.seeds.Merge(qrm.Seeds...)
			p2p.roots.Merge(qrm.Roots...)
			//disconn root->seed , seed->seed,
			if !p.incomming && qrm.Role == p2pRoleSeed {
				p2p.log.Println("handleQueryResult", "no need outgoing p2pRoleSeed connection from", role)
				p.Close()
			}
		}
	} else {
		p2p.log.Println("handleQueryResult", "not exists allowedlist", p)
		//p.Close()
	}
}

func (p2p *PeerToPeer) sendToPeer(pkt *Packet, p *Peer) error {
	if p == nil {
		return ErrNotAvailable
	}
	if pkt.src == nil {
		pkt.src = p2p.self.id
	}
	p2p.log.Println("sendToPeer", pkt, p)
	return p.send(pkt)
}

func (p2p *PeerToPeer) sendToPeers(pkt *Packet, peers *PeerSet) {
	for _, p := range peers.Array() {
		//p2p.packetRw.WriteTo(p.writer)
		p2p.sendToPeer(pkt, p)
	}
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
			p2p.log.Println("sendRoutine", pkt)
			if pkt.src == nil {
				pkt.src = p2p.self.id
			}
			// p2p.packetRw.WritePacket(pkt)

			switch pkt.dest {
			case p2pDestPeer:
			case p2pDestAny:
				switch pkt.protocol {
				case PROTO_DEF_MEMBER:
					p2p.sendToPeers(pkt, p2p.friends)
					p2p.sendToPeers(pkt, p2p.children)
					if !p2p.alternateQueue.Push(ctx) {
						p2p.log.Println("Warning", "sendRoutine", "alternateQueue Push failure", pkt.protocol, pkt.subProtocol)
					}
				default:
					//TODO gossip
					//send to all connected peer
					p2p.sendToPeers(pkt, p2p.friends)
					p2p.sendToPeer(pkt, p2p.parent)
					p2p.sendToPeers(pkt, p2p.uncles)
					p2p.sendToPeers(pkt, p2p.children)
					p2p.sendToPeers(pkt, p2p.nephews)
				}
			case p2pRoleRoot: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
				p2p.sendToPeers(pkt, p2p.friends)
				p2p.sendToPeer(pkt, p2p.parent)
				if !p2p.alternateQueue.Push(ctx) {
					p2p.log.Println("Warning", "sendRoutine", "alternateQueue Push failure", pkt.protocol, pkt.subProtocol)
				}
			//case p2pRoleSeed:
			default: //p2pDestPeerGroup < dest < p2pDestPeer
				//TODO multicast Routing or Flooding
			}
		}
	}
}

func (p2p *PeerToPeer) alternateSendRoutine() {
	var m = make(map[uint64]*Packet)
	for {
		select {
		case <-p2p.alternateQueue.Wait():
			for {
				ctx := p2p.alternateQueue.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				m[pkt.hashOfPacket] = pkt
			}
		case <-p2p.sendTicker.C:
			for _, pkt := range m {
				switch pkt.dest {
				case p2pDestPeer:
				case p2pDestAny:
					switch pkt.protocol {
					case PROTO_DEF_MEMBER:
						p2p.sendToPeers(pkt, p2p.nephews)
						p2p.log.Println("alternateSendRoutine", "nephews", p2p.nephews.Len(), pkt)
					}
				case p2pRoleRoot: //multicast to reserved role : p2pDestAny < dest <= p2pDestPeerGroup
					p2p.sendToPeers(pkt, p2p.uncles)
					p2p.log.Println("alternateSendRoutine", "uncles", p2p.uncles.Len(), pkt)
				//case p2pRoleSeed:
				default: //p2pDestPeerGroup < dest < p2pDestPeer
				}
				delete(m, pkt.hashOfPacket)
			}
		}
	}
}

func (p2p *PeerToPeer) send(pkt *Packet) error {
	if p2p.numberOfPeer(true) < 1 {
		p2p.log.Println("Warning", "send", "Not Available", pkt.protocol, pkt.subProtocol)
		return ErrNotAvailable
	}
	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	if ok := p2p.sendQueue.Push(ctx); !ok {
		p2p.log.Println("Warning", "send", "Queue Push failure", pkt.protocol, pkt.subProtocol)
		return ErrQueueOverflow
	}
	return nil
}

type p2pContextKey string

var (
	p2pContextKeyPacket = p2pContextKey("packet")
	p2pContextKeyPeer   = p2pContextKey("peer")
	p2pContextKeyEvent  = p2pContextKey("event")
	p2pContextKeyError  = p2pContextKey("error")
	p2pContextKeyDone   = p2pContextKey("done")
)

func (p2p *PeerToPeer) getPeer(id module.PeerID, onlyJoin bool) *Peer {
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
		if p := p2p.preParent.GetByID(id); p != nil {
			return p
		} else if p := p2p.preUncles.GetByID(id); p != nil {
			return p
		} else if p := p2p.orphanages.GetByID(id); p != nil {
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
		arr = append(arr, p2p.preParent.Array()...)
		arr = append(arr, p2p.preUncles.Array()...)
		arr = append(arr, p2p.orphanages.Array()...)
	}
	return arr
}

func (p2p *PeerToPeer) numberOfPeer(onlyJoin bool) int {
	n := p2p.uncles.Len()
	n += p2p.children.Len()
	n += p2p.nephews.Len()
	n += p2p.friends.Len()
	if p2p.parent != nil {
		n++
	}
	if !onlyJoin {
		n += p2p.preParent.Len()
		n += p2p.preUncles.Len()
		n += p2p.orphanages.Len()
	}
	return n
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
func (p2p *PeerToPeer) discoveryRoutine() {
	//TODO goroutine exit
	for {
		select {
		// case t := <-p2p.seedTicker.C:
		// p2p.log.Println("discoveryRoutine seedTicker", t)
		case <-p2p.seedTicker.C:
			p2p.syncSeeds()
		// case t := <-p2p.discoveryTicker.C:
		// p2p.log.Println("discoveryRoutine discoveryTicker", t)
		case <-p2p.discoveryTicker.C:
			r := p2p.getRole()
			switch r {
			case p2pRoleNone:
				p2p.discoverParent(p2pRoleSeed, p2p.seeds)
				p2p.discoverUncle(p2pRoleSeed, p2p.seeds)
			case p2pRoleSeed:
				p2p.discoverParent(p2pRoleRoot, p2p.roots)
				p2p.discoverUncle(p2pRoleRoot, p2p.roots)
			default:
				p2p.discoverFriends()
			}
		}
	}
}

func (p2p *PeerToPeer) syncSeeds() {
	switch p2p.getRole() {
	case p2pRoleNone:
		if p2p.parent != nil {
			p2p.sendQuery(p2p.parent)
		}
	case p2pRoleSeed:
		if p2p.parent != nil {
			p2p.sendQuery(p2p.parent)
		}
		for _, p := range p2p.uncles.Array() {
			if !p.incomming {
				p2p.sendQuery(p)
			}
		}
	default: //p2pRoleRoot, p2pRoleRootSeed
		for _, s := range p2p.seeds.Array() {
			if s != p2p.self.netAddress &&
				!p2p.friends.HasNetAddresse(s) &&
				!p2p.children.HasNetAddresse(s) &&
				!p2p.nephews.HasNetAddresse(s) &&
				!p2p.orphanages.HasNetAddresse(s) {
				p2p.log.Println("syncSeeds", "dial to p2pRoleSeed", s)
				if err := p2p.dial(s); err != nil {
					p2p.seeds.Remove(s)
				}
			}
		}
		for _, p := range p2p.friends.Array() {
			if !p.incomming {
				p2p.sendQuery(p)
			}
		}
	}
}

func (p2p *PeerToPeer) discoverFriends() {
	roots := p2p.orphanages.RemoveByRole(p2pRoleRoot)
	for _, p := range roots {
		p2p.log.Println("discoverFriends", "p2pConnTypeFriend", p.id)
		p.connType = p2pConnTypeFriend
		p2p.friends.Add(p)
		p2p.onEvent(p2pEventJoin, p)
	}
	for _, s := range p2p.roots.Array() {
		if s != p2p.self.netAddress && !p2p.friends.HasNetAddresse(s) {
			p2p.log.Println("discoverFriends", "dial to p2pRoleRoot", s)
			if err := p2p.dial(s); err != nil {
				p2p.roots.Remove(s)
			}
		}
	}
}

func (p2p *PeerToPeer) discoverParent(pr PeerRoleFlag, s *NetAddressSet) {
	//TODO connection between p2pRoleNone
	if p2p.parent != nil {
		p2p.log.Println("discoverParent", "nothing to do")
		return
	}

	if p2p.preParent.Len() > 0 {
		p2p.log.Println("discoverParent", "waiting P2PConnectionResponse")
		return
	}

	//TODO sort by rtt, sizeof(children)
	p := p2p.orphanages.GetByRoleAndIncomming(pr, false)
	if p != nil {
		p2p.orphanages.Remove(p)
		p.connType = p2pConnTypePre
		p2p.preParent.Add(p)
		p2p.sendP2PConnectionRequest(p2pConnTypeParent, p)
		p2p.log.Println("discoverParent", "try p2pConnTypeParent", p)
	} else {
		if p2p.uncles.Len() > 0 {
			//TODO sort by rtt, sizeof(children)
			p = p2p.uncles.GetByRoleAndIncomming(pr, false)
			if p != nil {
				p2p.preParent.Add(p)
				p2p.sendP2PConnectionRequest(p2pConnTypeParent, p)
				p2p.log.Println("discoverParent", "try p2pConnTypeParent from p2pConnTypeUncle", p)
			}
		} else {
			for _, na := range s.Array() {
				if na != p2p.self.netAddress &&
					!p2p.uncles.HasNetAddresse(na) &&
					!p2p.preUncles.HasNetAddresse(na) {
					p2p.log.Println("discoverParent", "dial to", na)
					if err := p2p.dial(na); err == nil {
						break
					}
				}
			}
		}
	}
}

func (p2p *PeerToPeer) discoverUncle(ur PeerRoleFlag, s *NetAddressSet) {
	if p2p.parent == nil {
		p2p.log.Println("discoverUncle", "parent is nil")
		return
	}

	if p2p.uncles.Len() >= DefaultUncleLimit {
		p2p.log.Println("discoverUncle", "nothing to do")
		return
	}

	needPreUncle := DefaultUncleLimit - p2p.uncles.Len() - p2p.preUncles.Len()
	if needPreUncle < 1 {
		p2p.log.Println("discoverUncle", "waiting P2PConnectionResponse")
		return
	}
	for i := 0; i < needPreUncle; i++ {
		//TODO sort by rtt, sizeof(children)
		p := p2p.orphanages.GetByRoleAndIncomming(ur, false)
		if p == nil {
			break
		}
		p2p.orphanages.Remove(p)
		p.connType = p2pConnTypePre
		p2p.preUncles.Add(p)
		p2p.sendP2PConnectionRequest(p2pConnTypeUncle, p)
		p2p.log.Println("discoverUncle", "try p2pConnTypeUncle", p)
	}

	needDial := DefaultUncleLimit - p2p.uncles.Len() - p2p.preUncles.Len()
	for _, na := range s.Array() {
		if na != p2p.self.netAddress &&
			na != p2p.parent.netAddress &&
			!p2p.uncles.HasNetAddresse(na) &&
			!p2p.preUncles.HasNetAddresse(na) {
			if needDial < 1 {
				break
			}
			p2p.log.Println("discoverUncle", "dial to", na)
			if err := p2p.dial(na); err == nil {
				needDial--
			}
		}
	}
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
	pkt := NewPacket(PROTO_P2P_CONN_REQ, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.send(pkt)
	p2p.log.Println("sendP2PConnectionRequest", m)
}
func (p2p *PeerToPeer) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := p2p.decodeMsgpack(pkt.payload, req)
	if err != nil {
		p2p.log.Println("Warning", "handleP2PConnectionRequest", err)
		return
	}
	p2p.log.Println("handleP2PConnectionRequest", req)
	m := &P2PConnectionResponse{ConnType: p2pConnTypeNone}
	switch req.ConnType {
	case p2pConnTypeParent:
		//TODO p2p.children condition
		switch p.connType {
		case p2pConnTypeNone:
			p2p.orphanages.Remove(p)
			p.connType = p2pConnTypeChildren
			p2p.children.Add(p)
			p2p.onEvent(p2pEventJoin, p)
		case p2pConnTypeNephew:
			p2p.nephews.Remove(p)
			p.connType = p2pConnTypeChildren
			p2p.children.Add(p)
		default:
			p2p.log.Println("handleP2PConnectionRequest", "ignore", req.ConnType, "from", p.connType)
		}
	case p2pConnTypeUncle:
		//TODO p2p.nephews condition
		switch p.connType {
		case p2pConnTypeNone:
			p2p.orphanages.Remove(p)
			p.connType = p2pConnTypeNephew
			p2p.nephews.Add(p)
			p2p.onEvent(p2pEventJoin, p)
		default:
			p2p.log.Println("handleP2PConnectionRequest", "ignore", req.ConnType, "from", p.connType)
		}
	default:
		p2p.log.Println("handleP2PConnectionRequest", "ignore", req.ConnType, "from", p.connType)
	}
	m.ReqConnType = req.ConnType
	m.ConnType = p.connType

	rpkt := NewPacket(PROTO_P2P_CONN_RESP, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
	p.send(rpkt)
}

func (p2p *PeerToPeer) handleP2PConnectionResponse(pkt *Packet, p *Peer) {
	resp := &P2PConnectionResponse{}
	err := p2p.decodeMsgpack(pkt.payload, resp)
	if err != nil {
		p2p.log.Println("Warning", "handleP2PConnectionResponse", err)
		return
	}
	p2p.log.Println("handleP2PConnectionResponse", resp)

	switch resp.ReqConnType {
	case p2pConnTypeParent:
		p2p.preParent.Remove(p)
		if p2p.parent == nil && resp.ConnType == p2pConnTypeChildren {
			switch p.connType {
			case p2pConnTypePre:
				p2p.parent = p
				p.connType = resp.ReqConnType
				p2p.onEvent(p2pEventJoin, p)
			case p2pConnTypeUncle:
				p2p.uncles.Remove(p)
				p2p.parent = p
				p.connType = resp.ReqConnType
			default:
				p2p.log.Println("Warning", "handleP2PConnectionResponse", "wrong", resp, p)
			}
		} else {
			p2p.log.Println("handleP2PConnectionResponse invalid", resp, p)
		}
	case p2pConnTypeUncle:
		p2p.preUncles.Remove(p)
		if p2p.uncles.Len() < DefaultUncleLimit && resp.ConnType == p2pConnTypeNephew {
			switch p.connType {
			case p2pConnTypePre:
				p2p.uncles.Add(p)
				p.connType = resp.ReqConnType
				p2p.onEvent(p2pEventJoin, p)
			default:
				p2p.log.Println("Warning", "handleP2PConnectionResponse", "wrong", resp, p)
			}
		} else {
			p2p.log.Println("handleP2PConnectionResponse", "invalid", resp, p)
		}
	default:
		p2p.log.Println("handleP2PConnectionRespons", "invalid not supported", resp, p)
	}

	if p.connType == p2pConnTypePre {
		p.connType = p2pConnTypeNone
		p2p.orphanages.Add(p)
	}
}
