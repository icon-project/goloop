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

	//Addresses
	seeds *NetAddressSet
	//Only for seed
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
	if err := p2p.transport.Dial(string(na), p2p.channel); err != nil {
		p2p.log.Println("Dial fail", na, err)
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
	Addr NetAddress
}

type QueryResultMessage struct {
	Role     PeerRoleFlag
	Seeds    []NetAddress
	Roots    []NetAddress
	Children []NetAddress
	Message  string
}

func (p2p *PeerToPeer) setRole(r PeerRoleFlag) {
	if p2p.self.role != r {
		p2p.self.role.Set(r)
		switch r {
		case p2pRoleNone:
			p2p.roots.Remove(p2p.self.netAddress)
			p2p.seeds.Remove(p2p.self.netAddress)
		case p2pRoleSeed:
			p2p.seeds.Merge(p2p.self.netAddress)
			p2p.roots.Remove(p2p.self.netAddress)
		case p2pRoleRoot:
			p2p.roots.Merge(p2p.self.netAddress)
			p2p.seeds.Remove(p2p.self.netAddress)
		case p2pRoleRootSeed:
			p2p.roots.Merge(p2p.self.netAddress)
			p2p.seeds.Merge(p2p.self.netAddress)
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
	m := &QueryMessage{Role: p2p.getRole(), Addr: p2p.self.netAddress}
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
		switch qm.Role {
		case p2pRoleNone:
			switch m.Role {
			case p2pRoleNone:
				m.Children = p2p.children.NetAddresses()
			case p2pRoleRoot:
				m.Message = "not allowed to query"
				//p.conn.Close()
			default: //p2pRoleSeed, ROLE_P2P_ROOTSEED
				m.Seeds = p2p.seeds.Array()
				m.Children = p2p.children.NetAddresses()
			}
		default: //between p2pRoleSeed, p2pRoleRoot, p2pRoleRootSeed
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			if qm.Role.Has(p2pRoleSeed) {
				p2p.log.Println("handleQuery seeds.Merge", qm.Addr)
				p2p.seeds.Merge(qm.Addr)
			}
			if qm.Role.Has(p2pRoleRoot) {
				p2p.log.Println("handleQuery roots.Merge", qm.Addr)
				p2p.roots.Merge(qm.Addr)
			}
			if m.Role == p2pRoleSeed {
				//p.conn will be disconnected
			}
		}
		p.role.Set(qm.Role)
		if p.netAddress == "" {
			p.netAddress = qm.Addr
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
	switch p2p.getRole() {
	case p2pRoleNone:
		switch p.role {
		case p2pRoleNone:
			//TODO p2p.preParent.Merge(qrm.Children)
		case p2pRoleSeed:
			p2p.seeds.Merge(qrm.Seeds...)
		case p2pRoleRoot:
			p2p.log.Println("handleQueryResult p2pRoleNone cannot query to p2pRoleRoot")
		case p2pRoleRootSeed:
			p2p.seeds.Merge(qrm.Seeds...)
		default:
			//TODO p2p.preParent.Merge(qrm.Children)
		}
	default: //between p2pRoleSeed, p2pRoleRoot, p2pRoleRootSeed
		p2p.seeds.Merge(qrm.Seeds...)
		p2p.roots.Merge(qrm.Roots...)
		//disconn root->seed , seed->seed,
		if !p.incomming && p.role == p2pRoleSeed {
			p2p.log.Println("handleQueryResult p2pRoleRoot no need outgoing connection with p2pRoleSeed ")
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
			if r := p2p.getRole(); r.Has(p2pRoleRoot) {
				roots := p2p.orphanages.RemoveByRole(p2pRoleRoot)
				for _, p := range roots {
					p2p.log.Println("discoveryRoutine p2pConnTypeFriend", p.id)
					p.connType = p2pConnTypeFriend
					p2p.friends.Add(p)
				}
				for _, s := range p2p.roots.Array() {
					if s != p2p.self.netAddress && !p2p.friends.HasNetAddresse(s) {
						p2p.log.Println("discoveryRoutine p2pRoleRoot", p2p.self, "dial to p2pRoleRoot", s)
						p2p.dial(s)
					}
				}
			} else {
				if p2p.parent == nil {
					p2p.discoverParent()
				} else if p2p.uncles.Len() < 1 { //TODO p2pConnTypeUncle condition
					p2p.discoverUncle()
				}
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
				!p2p.orphanages.HasNetAddresse(s) {
				p2p.log.Println("discoveryRoutine syncSeeds", p2p.self, "dial to p2pRoleSeed", s)
				p2p.dial(s)
			}
		}
		for _, p := range p2p.friends.Array() {
			if !p.incomming {
				p2p.sendQuery(p)
			}
		}
	}
}

func (p2p *PeerToPeer) discoverParent() {
	//TODO connection between p2pRoleNone
	if p2p.preParent.Len() < 1 {
		var parentRole PeerRoleFlag
		parentRole = p2pRoleSeed
		if p2p.self.role == p2pRoleSeed {
			parentRole = p2pRoleRoot
		}
		p := p2p.orphanages.GetByRoleAndIncomming(parentRole, false)
		if p != nil {
			p2p.orphanages.Remove(p)
			p2p.preParent.Add(p)
			p2p.sendP2PConnectionRequest(p2pConnTypeParent, p)
			p2p.log.Println("discoverParent try p2pConnTypeParent", p)
		} else {
			//TODO upgrade p2pConnTypeUncle to p2pConnTypeParent
			if p2p.self.role == p2pRoleSeed {
				for _, s := range p2p.roots.Array() {
					if s != p2p.self.netAddress && !p2p.uncles.HasNetAddresse(s) {
						p2p.log.Println("discoverParent", p2p.self, "dial to p2pRoleRoot", s)
						p2p.dial(s)
					}
				}
			} else {
				for _, s := range p2p.seeds.Array() {
					if s != p2p.self.netAddress && !p2p.uncles.HasNetAddresse(s) {
						p2p.log.Println("discoverParent", p2p.self, "dial to p2pRoleSeed", s)
						p2p.dial(s)
					}
				}
			}

		}
	}
}

func (p2p *PeerToPeer) discoverUncle() {
	var uncleRole PeerRoleFlag
	uncleRole = p2pRoleSeed
	if p2p.self.role == p2pRoleSeed {
		uncleRole = p2pRoleRoot
	}
	p := p2p.orphanages.GetByRoleAndIncomming(uncleRole, false)
	if p != nil {
		p2p.sendP2PConnectionRequest(p2pConnTypeUncle, p)
		p2p.log.Println("discoverUncle try p2pConnTypeUncle", p)
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
