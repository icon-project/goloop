package network

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/ugorji/go/codec"
)

type PeerToPeer struct {
	channel         string
	ch              chan *Packet
	onPacketCbFuncs map[module.ProtocolInfo]packetCbFunc
	//[TBD] detecting duplicate transmission
	packetPool    map[uint64]*Packet
	mtxPacketPool sync.Mutex
	packetRw      *PacketReadWriter
	transport     module.NetworkTransport

	//Topology with Connected Peers
	self      *Peer
	parent    *Peer
	preParent *PeerSet
	children  *PeerSet
	uncles    *PeerSet
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
	allowedRoots *PeerIdSet
	allowedSeeds *PeerIdSet

	//codec
	mph *codec.MsgpackHandle

	//log
	log *logger
}

//can be created each channel
func newPeerToPeer(channel string, t module.NetworkTransport) *PeerToPeer {
	id := t.PeerID()
	netAddress := NetAddress(t.Address())
	p2p := &PeerToPeer{
		channel:         channel,
		ch:              make(chan *Packet),
		onPacketCbFuncs: make(map[module.ProtocolInfo]packetCbFunc),
		packetPool:      make(map[uint64]*Packet),
		packetRw:        NewPacketReadWriter(),
		transport:       t,
		//
		self:            &Peer{id: id, netAddress: netAddress},
		preParent:       NewPeerSet(),
		children:        NewPeerSet(),
		uncles:          NewPeerSet(),
		nephews:         NewPeerSet(),
		friends:         NewPeerSet(),
		orphanages:      NewPeerSet(),
		discoveryTicker: time.NewTicker(time.Duration(DefaultDiscoveryPeriodSec) * time.Second),
		seedTicker:      time.NewTicker(time.Duration(DefaultSeedPeriodSec) * time.Second),
		duplicated:      NewSet(),
		dialing:         NewNetAddressSet(),
		//
		seeds:         NewNetAddressSet(),
		roots:         NewNetAddressSet(),
		grandChildren: NewNetAddressSet(),
		//
		allowedRoots: NewPeerIdSet(),
		allowedSeeds: NewPeerIdSet(),
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

	go p2p.sendRoutine()
	go p2p.discoveryRoutine()
	return p2p
}

func (p2p *PeerToPeer) dial(na NetAddress) error {
	//TODO dialing context
	if !p2p.dialing.Add(na) {
		p2p.log.Println("Already Dialing", na)
		return nil
	}
	if err := p2p.transport.Dial(string(na), p2p.channel); err != nil {
		p2p.log.Println("Dial fail", na, err)
		p2p.dialing.Remove(na)
		return err
	}
	return nil
}

func (p2p *PeerToPeer) setPacketCbFunc(pi module.ProtocolInfo, cbFunc packetCbFunc) {
	if _, ok := p2p.onPacketCbFuncs[pi]; ok {
		p2p.log.Println("Warnning overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[pi] = cbFunc
}

//callback from PeerDispatcher.onPeer
func (p2p *PeerToPeer) onPeer(p *Peer) {
	p2p.log.Println("onPeer", p)
	if !p.incomming {
		p2p.dialing.Remove(p.netAddress)
	}
	if dp := p2p.getPeer(p.id); dp != nil {
		p2p.log.Println("Already exists connected Peer, close duplicated peer", dp)
		p2p.removePeer(dp)
		p2p.duplicated.Add(dp)
		dp.conn.Close()
	}
	p2p.orphanages.Add(p)
	if !p.incomming {
		p2p.sendQuery(p)
	} else {
		//peer can be children or nephews
	}
	//TODO discoveryRoutine with time.Ticker
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (p2p *PeerToPeer) onError(err error, p *Peer) {
	p2p.log.Println("onError", err, p)
	err = p.conn.Close()
	if err != nil {
		p2p.log.Println("onError p.conn.Close", err)
	}
	p2p.removePeer(p)
}

func (p2p *PeerToPeer) onClose(p *Peer) {
	p2p.log.Println("onClose", p)
	p2p.removePeer(p)
}

func (p2p *PeerToPeer) removePeer(p *Peer) {
	if p2p.duplicated.Remove(p) {
		return
	}
	switch p.connType {
	case p2pConnTypeNone:
		p2p.orphanages.Remove(p)
		p2p.preParent.Remove(p)
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
			p2p.log.Println("onPacket Ignore, undetermined PeerConnectionType")
		} else if cbFunc := p2p.onPacketCbFuncs[pkt.protocol]; cbFunc != nil {
			if !p2p.hasPacket(pkt) && !p2p.self.id.Equal(pkt.src) {
				p2p.putPacketPool(pkt)
				cbFunc(pkt, p)
			} else {
				p2p.log.Println("onPacket Ignore, duplicated", pkt.hashOfPacket)
			}
		}
	}
}

func (p2p *PeerToPeer) hasPacket(pkt *Packet) bool {
	defer p2p.mtxPacketPool.Unlock()
	p2p.mtxPacketPool.Lock()
	_, ok := p2p.packetPool[pkt.hashOfPacket]
	return ok
}
func (p2p *PeerToPeer) putPacketPool(pkt *Packet) {
	defer p2p.mtxPacketPool.Unlock()
	p2p.mtxPacketPool.Lock()
	p2p.packetPool[pkt.hashOfPacket] = pkt
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
		p2p.log.Println("Seed updated NetAddress old:", o, ", now:", p.netAddress, ",peerID:", p.id)
	}
	if c != "" {
		p2p.log.Println("Warnning Seed conflict NetAddress", p.netAddress, "removed:", c, ",now:", p.id)
	}
}
func (p2p *PeerToPeer) removeSeed(p *Peer) {
	p2p.seeds.RemoveByPeer(p)
}
func (p2p *PeerToPeer) addRoot(p *Peer) {
	c, o := p2p.roots.PutByPeer(p)
	if o != "" {
		p2p.log.Println("Root updated NetAddress old:", o, ", now:", p.netAddress, ",peerID:", p.id)
	}
	if c != "" {
		p2p.log.Println("Warnning Root conflict NetAddress", p.netAddress, "removed:", c, ",now:", p.id)
	}
}
func (p2p *PeerToPeer) removeRoot(p *Peer) {
	p2p.roots.RemoveByPeer(p)
}

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) {
	if p2p.self.role != r {
		p2p.self.role.Set(r)
		switch r {
		case p2pRoleNone:
			p2p.removeRoot(p2p.self)
			p2p.removeSeed(p2p.self)
		case p2pRoleSeed:
			p2p.addSeed(p2p.self)
			p2p.removeRoot(p2p.self)
		case p2pRoleRoot:
			p2p.addRoot(p2p.self)
			p2p.removeSeed(p2p.self)
		case p2pRoleRootSeed:
			p2p.addRoot(p2p.self)
			p2p.addSeed(p2p.self)
		}
	}
}

func (p2p *PeerToPeer) setRoleByAllowedSet() PeerRoleFlag {
	role := p2pRoleNone
	if p2p.isAllowedRole(p2pRoleRoot, p2p.self) {
		role |= p2pRoleRoot
	}
	if p2p.isAllowedRole(p2pRoleSeed, p2p.self) {
		role |= p2pRoleSeed
	}
	p2p.setRole(PeerRoleFlag(role))
	return p2p.self.role
}

func (p2p *PeerToPeer) getRole() PeerRoleFlag {
	return p2p.self.role
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.getRole()}
	pkt := NewPacket(PROTO_P2P_QUERY, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.sendPacket(pkt)
	p2p.log.Println("sendQuery", m, p)
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qm)
	if err != nil {
		p2p.log.Println("handleQuery", err)
		return
	}
	p2p.log.Println("handleQuery", qm, p)
	m := &QueryResultMessage{}
	m.Role = p2p.getRole()
	if p2p.isAllowedRole(qm.Role, p) {
		p.role.Set(qm.Role)
		if qm.Role == p2pRoleNone {
			p2p.removeSeed(p)
			p2p.removeRoot(p)

			m.Children = p2p.children.NetAddresses()
			switch m.Role {
			case p2pRoleSeed:
				m.Seeds = p2p.seeds.Array()
			case p2pRoleRoot:
				p2p.log.Println("handleQuery p2pRoleNone cannot query to p2pRoleRoot")
				m.Message = "not allowed to query"
				m.Children = m.Children[:0]
				//p.conn.Close()
			case p2pRoleRootSeed:
				//TODO hiding RootSeed role
				//m.Role.Set(p2pRoleSeed)
				m.Seeds = p2p.seeds.Array()
			}
		} else {

			switch qm.Role {
			case p2pRoleSeed:
				p2p.addSeed(p)
				p2p.removeRoot(p)
			case p2pRoleRoot:
				p2p.removeSeed(p)
				p2p.addRoot(p)
			case p2pRoleRootSeed:
				p2p.addRoot(p)
				p2p.addSeed(p)
			}
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			if m.Role == p2pRoleSeed {
				//p.conn will be disconnected
			}
		}
	} else {
		m.Message = "not exists allowedlist"
		//p.conn.Close()
	}
	rpkt := NewPacket(PROTO_P2P_QUERY_RESULT, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
	p.sendPacket(rpkt)
}

func (p2p *PeerToPeer) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qrm)
	if err != nil {
		p2p.log.Println("handleQueryResult", err)
		return
	}
	p2p.log.Println("handleQueryResult", qrm)
	p.role.Set(qrm.Role)
	role := p2p.getRole()
	if role == p2pRoleNone {
		switch qrm.Role {
		case p2pRoleNone:
			//TODO p2p.preParent.Merge(qrm.Children)
		case p2pRoleSeed:
			p2p.seeds.Merge(qrm.Seeds...)
		case p2pRoleRoot:
			p2p.log.Println("handleQueryResult p2pRoleNone cannot query to p2pRoleRoot")
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
			p2p.log.Println("handleQueryResult no need outgoing p2pRoleSeed connection from", role)
			p.conn.Close()
		}
	}
}

func (p2p *PeerToPeer) sendToFriends(pkt *Packet) {
	p2p.sendToPeers(pkt, p2p.friends)
}

func (p2p *PeerToPeer) sendToUpside(pkt *Packet) {
	if p2p.parent != nil {
		p2p.parent.sendPacket(pkt)
	}
	//TODO after next period
	//p2p.sendToPeers(pkt, p2p.uncles)
}

func (p2p *PeerToPeer) sendToDownside(pkt *Packet) {
	p2p.sendToPeers(pkt, p2p.children)
	//TODO after next period
	//p2p.sendToPeers(pkt, p2p.nephews)
}

func (p2p *PeerToPeer) sendToPeers(pkt *Packet, peers *PeerSet) {
	for _, p := range peers.Array() {
		//p2p.packetRw.WriteTo(p.writer)
		p.sendPacket(pkt)
		p2p.log.Println("sendToPeers", pkt, p)
		//p2p.sendToPeer(pkt, p)
	}
}

func (p2p *PeerToPeer) sendRoutine() {
	//TODO goroutine exit
	for {
		select {
		case pkt := <-p2p.ch:
			p2p.log.Println("sendRoutine", pkt)

			if pkt.src == nil {
				pkt.src = p2p.self.id
			}
			// p2p.packetRw.WritePacket(pkt)
			// p2p.packetRw.Flush()

			switch pkt.dest {
			case p2pDestAny: //broadcast
				if p2p.getRole() >= p2pRoleRoot {
					p2p.sendToFriends(pkt)
					p2p.sendToDownside(pkt)
				} else {
					p2p.sendToDownside(pkt)
				}
			case p2pRoleRoot:
				if p2p.getRole() >= p2pRoleRoot {
					p2p.sendToFriends(pkt)
				} else {
					p2p.sendToUpside(pkt)
				}
			//case p2pRoleSeed:
			default: //TODO p2pRoleSeed, multicast Routing or Flooding
			}
		}
	}
}

func (p2p *PeerToPeer) getPeer(id module.PeerID) *Peer {
	if p2p.parent != nil && p2p.parent.id.Equal(id) {
		return p2p.parent
	} else if p := p2p.preParent.GetByID(id); p != nil {
		return p
	} else if p := p2p.uncles.GetByID(id); p != nil {
		return p
	} else if p := p2p.children.GetByID(id); p != nil {
		return p
	} else if p := p2p.nephews.GetByID(id); p != nil {
		return p
	} else if p := p2p.friends.GetByID(id); p != nil {
		return p
	} else if p := p2p.orphanages.GetByID(id); p != nil {
		return p
	} else {
		return nil
	}
}

func (p2p *PeerToPeer) sendToPeer(pkt *Packet, id module.PeerID) {
	p := p2p.getPeer(id)
	if p != nil {
		//p2p.packetRw.WriteTo(p.conn)
		if pkt.src == nil {
			pkt.src = p2p.self.id
		}
		p.sendPacket(pkt)
		p2p.log.Println("sendToPeer", pkt, id)
	} else {
		p2p.log.Println("sendToPeer not exists", pkt, id)
	}

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
				p2p.discoverParent(p2pRoleSeed, p2p.seeds.Array())
				p2p.discoverUncle(p2pRoleSeed, p2p.seeds.Array())
			case p2pRoleSeed:
				p2p.discoverParent(p2pRoleRoot, p2p.roots.Array())
				p2p.discoverUncle(p2pRoleRoot, p2p.roots.Array())
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
		if p2p.friends.Len() < 1 {
			for _, s := range p2p.seeds.Array() {
				if s != p2p.self.netAddress &&
					!p2p.friends.HasNetAddresse(s) &&
					!p2p.children.HasNetAddresse(s) &&
					!p2p.nephews.HasNetAddresse(s) &&
					!p2p.orphanages.HasNetAddresse(s) {
					p2p.log.Println("syncSeeds dial to p2pRoleSeed", s)
					p2p.dial(s)
				}
			}
		} else {

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
		p2p.log.Println("discoverFriends p2pConnTypeFriend", p.id)
		p.connType = p2pConnTypeFriend
		p2p.friends.Add(p)
	}
	for _, s := range p2p.roots.Array() {
		if s != p2p.self.netAddress && !p2p.friends.HasNetAddresse(s) {
			p2p.log.Println("discoverFriends dial to p2pRoleRoot", s)
			p2p.dial(s)
		}
	}
}

func (p2p *PeerToPeer) discoverParent(pr PeerRoleFlag, nas []NetAddress) {
	//TODO connection between p2pRoleNone
	if p2p.parent == nil && p2p.preParent.Len() < 1 {
		p := p2p.orphanages.GetByRoleAndIncomming(pr, false)
		if p != nil {
			p2p.orphanages.Remove(p)
			p2p.preParent.Add(p)
			p2p.sendP2PConnectionRequest(p2pConnTypeParent, p)
			p2p.log.Println("discoverParent try p2pConnTypeParent", p)
		} else {
			//TODO upgrade p2pConnTypeUncle to p2pConnTypeParent
			for _, s := range nas {
				if s != p2p.self.netAddress && !p2p.uncles.HasNetAddresse(s) {
					p2p.log.Println("discoverParent dial to", s)
					p2p.dial(s)
				}
			}
		}
	}
}

func (p2p *PeerToPeer) discoverUncle(ur PeerRoleFlag, nas []NetAddress) {
	if p2p.parent != nil && p2p.uncles.Len() < 1 { //TODO p2pConnTypeUncle condition
		p := p2p.orphanages.GetByRoleAndIncomming(ur, false)
		if p != nil {
			p2p.sendP2PConnectionRequest(p2pConnTypeUncle, p)
			p2p.log.Println("discoverUncle try p2pConnTypeUncle", p)
		} else {
			for _, s := range nas {
				if s != p2p.self.netAddress && s != p2p.parent.netAddress && !p2p.uncles.HasNetAddresse(s) {
					p2p.log.Println("discoverUncle dial to", s)
					p2p.dial(s)
				}
			}
		}
	}
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
	pkt := NewPacket(PROTO_P2P_CONN_REQ, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.sendPacket(pkt)
	p2p.log.Println("sendP2PConnectionRequest", m)
}
func (p2p *PeerToPeer) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := p2p.decodeMsgpack(pkt.payload, req)
	if err != nil {
		p2p.log.Println(err)
		return
	}
	p2p.log.Println("handleP2PConnectionRequest", req)
	m := &P2PConnectionResponse{ConnType: p2pConnTypeNone}
	if p.connType == p2pConnTypeNone {
		switch req.ConnType {
		case p2pConnTypeParent:
			p2p.orphanages.Remove(p)
			p2p.children.Add(p)
			p.connType = p2pConnTypeChildren
			m.ReqConnType = req.ConnType
			m.ConnType = p.connType
			//TODO p2p.children condition
		case p2pConnTypeUncle:
			p2p.orphanages.Remove(p)
			p2p.nephews.Add(p)
			p.connType = p2pConnTypeNephew
			m.ReqConnType = req.ConnType
			m.ConnType = p.connType
			//TODO p2p.nephews condition
		default:
			p2p.log.Println("handleP2PConnectionRequest ignore", req.ConnType)
		}
	}

	rpkt := NewPacket(PROTO_P2P_CONN_RESP, p2p.encodeMsgpack(m))
	rpkt.src = p2p.self.id
	p.sendPacket(rpkt)
}
func (p2p *PeerToPeer) handleP2PConnectionResponse(pkt *Packet, p *Peer) {
	resp := &P2PConnectionResponse{}
	err := p2p.decodeMsgpack(pkt.payload, resp)
	if err != nil {
		p2p.log.Println(err)
		return
	}
	p2p.log.Println("handleP2PConnectionResponse", resp)
	if p.connType == p2pConnTypeNone {
		switch resp.ReqConnType {
		case p2pConnTypeParent:
			if p2p.parent == nil {
				if resp.ConnType == p2pConnTypeChildren {
					p2p.parent = p
					p.connType = resp.ReqConnType
					p2p.preParent.Remove(p)
				} else {
					//TODO handle to reject p2pConnTypeParent
					p2p.log.Println("handleP2PConnectionResponse reject", resp.ReqConnType)
				}
			} else {
				p2p.log.Println("handleP2PConnectionResponse wrong", resp.ReqConnType)
			}
		case p2pConnTypeUncle:
			if resp.ConnType == p2pConnTypeNephew {
				p2p.orphanages.Remove(p)
				p2p.uncles.Add(p)
				p.connType = resp.ReqConnType
			} else {
				//TODO handle to reject p2pConnTypeUncle
				p2p.log.Println("handleP2PConnectionResponse reject", resp.ReqConnType)
			}
		default:
			p2p.log.Println("handleP2PConnectionResponse ignore", resp.ReqConnType)
		}
	}
}
