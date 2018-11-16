package network

import (
	"container/list"
	"fmt"
	"log"
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
	dialer        *Dialer

	//Topology with Connected Peers
	self      *Peer
	parent    *Peer
	preParent *PeerList
	children  *PeerList
	uncles    *PeerList
	nephews   *PeerList
	//Only for root, parent is nil, uncles is empty
	friends *PeerList
	//Discovery
	orphanages      *PeerList
	discoveryTicker *time.Ticker
	seedTicker      *time.Ticker

	//Addresses
	seeds *NetAddressList
	//Only for seed
	roots *NetAddressList
	//[TBD] 2hop peers of current tree for status change
	grandParent   NetAddress
	grandChildren *NetAddressList

	//managed PeerId
	allowedRoots *PeerIdList
	allowedSeeds *PeerIdList

	//codec
	mph *codec.MsgpackHandle
}

//can be created each channel
func newPeerToPeer(channel string, id module.PeerID, addr NetAddress, d *Dialer) *PeerToPeer {
	p2p := &PeerToPeer{
		channel:         channel,
		ch:              make(chan *Packet),
		onPacketCbFuncs: make(map[module.ProtocolInfo]packetCbFunc),
		packetPool:      make(map[uint64]*Packet),
		packetRw:        NewPacketReadWriter(),
		dialer:          d,
		//
		self:            &Peer{id: id, netAddress: addr},
		preParent:       NewPeerList(),
		children:        NewPeerList(),
		uncles:          NewPeerList(),
		nephews:         NewPeerList(),
		friends:         NewPeerList(),
		orphanages:      NewPeerList(),
		discoveryTicker: time.NewTicker(time.Duration(DefaultDiscoveryPeriodSec) * time.Second),
		seedTicker:      time.NewTicker(time.Duration(DefaultSeedPeriodSec) * time.Second),
		//
		seeds:         NewNetAddressList(),
		roots:         NewNetAddressList(),
		grandChildren: NewNetAddressList(),
		//
		allowedRoots: NewPeerIdList(),
		allowedSeeds: NewPeerIdList(),
		//
		mph: &codec.MsgpackHandle{},
	}
	p2p.mph.MapType = reflect.TypeOf(map[string]interface{}(nil))
	go p2p.sendRoutine()
	go p2p.discoveryRoutine()
	return p2p
}

func (p2p *PeerToPeer) setPacketCbFunc(pi module.ProtocolInfo, cbFunc packetCbFunc) {
	if _, ok := p2p.onPacketCbFuncs[pi]; ok {
		log.Println("Warnning overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[pi] = cbFunc
}

//callback from PeerDispatcher.onPeer
func (p2p *PeerToPeer) onPeer(p *Peer) {
	log.Println("PeerToPeer.onPeer", p)
	p2p.orphanages.PushBack(p)
	if !p.incomming {
		p2p.sendQuery(p)
	} else {
		//peer can be children or nephews
	}
	//TODO discoveryRoutine with time.Ticker
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (p2p *PeerToPeer) onError(err error, p *Peer) {
	log.Println("PeerToPeer.onError", err)
	err = p.conn.Close()
	if err != nil {
		log.Println("PeerToPeer.onError close", err)
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
		log.Println("PeerToPeer.onPacket", pkt)
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
			log.Println("Ignoring packet, because undetermined PeerConnectionType is not allowed to handle")
		} else if cbFunc := p2p.onPacketCbFuncs[pkt.protocol]; cbFunc != nil {
			if !p2p.hasPacket(pkt) {
				p2p.putPacketPool(pkt)
				cbFunc(pkt, p)
			} else {
				log.Println("Duplicated packet, ignore", pkt.hashOfPacket)
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

func (p2p *PeerToPeer) getSelfRole() PeerRoleFlag {
	role := p2pRoleNone
	if p2p.allowedRoots.Has(p2p.self.id) {
		role |= p2pRoleRoot
	}
	if p2p.allowedSeeds.Has(p2p.self.id) {
		role |= p2pRoleSeed
	}
	prf := PeerRoleFlag(role)
	if p2p.self.role != prf {
		switch prf {
		case p2pRoleNone:
			p2p.roots.Remove(p2p.self.netAddress)
			p2p.seeds.Remove(p2p.self.netAddress)
		case p2pRoleSeed:
			p2p.seeds.Merge([]NetAddress{p2p.self.netAddress})
			p2p.roots.Remove(p2p.self.netAddress)
		case p2pRoleRoot:
			p2p.roots.Merge([]NetAddress{p2p.self.netAddress})
			p2p.seeds.Remove(p2p.self.netAddress)
		case p2pRoleRootSeed:
			p2p.roots.Merge([]NetAddress{p2p.self.netAddress})
			p2p.seeds.Merge([]NetAddress{p2p.self.netAddress})
		}
	}
	return prf
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.getSelfRole(), Addr: p2p.self.netAddress}
	pkt := NewPacket(PROTO_P2P_QUERY, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.sendPacket(pkt)
	log.Println("PeerToPeer.sendQuery", m)
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := p2p.decodeMsgpack(pkt.payload, qm)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("PeerToPeer.handleQuery", qm)
	m := &QueryResultMessage{}
	m.Role = p2p.getSelfRole()
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
		case p2pRoleSeed:
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			p2p.seeds.Merge([]NetAddress{qm.Addr})
		case p2pRoleRoot:
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			p2p.roots.Merge([]NetAddress{qm.Addr})
		case p2pRoleRootSeed:
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
			p2p.seeds.Merge([]NetAddress{qm.Addr})
			p2p.roots.Merge([]NetAddress{qm.Addr})
		default:
		}
		p.role = qm.Role
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
		log.Println(err)
		return
	}
	log.Println("PeerToPeer.handleQueryResult", qrm)
	p.role = qrm.Role
	switch p2p.getSelfRole() {
	case p2pRoleNone:
		switch p.role {
		case p2pRoleNone:
			//TODO p2p.preParent.Merge(qrm.Children)
		case p2pRoleSeed:
			p2p.seeds.Merge(qrm.Seeds)
		case p2pRoleRoot:
			log.Println("Wrong situation")
		case p2pRoleRootSeed:
			p2p.seeds.Merge(qrm.Seeds)
		default:
			//TODO p2p.preParent.Merge(qrm.Children)
		}
	default: //between p2pRoleSeed, p2pRoleRoot, p2pRoleRootSeed
		p2p.seeds.Merge(qrm.Seeds)
		p2p.roots.Merge(qrm.Roots)
		//disconn root->seed , seed->seed,
		if !p.incomming && p.role == p2pRoleSeed {
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

func (p2p *PeerToPeer) sendToPeers(pkt *Packet, peers *PeerList) {
	for e := peers.Front(); e != nil; e = e.Next() {
		p := e.Value.(*Peer)
		//p2p.packetRw.WriteTo(p.writer)
		p.sendPacket(pkt)
		log.Println("PeerToPeer.sendToPeers", p.id, pkt)
		//p2p.sendToPeer(pkt, p)
	}
}

func (p2p *PeerToPeer) sendRoutine() {
	//TODO goroutine exit
	for {
		select {
		case pkt := <-p2p.ch:
			log.Println("PeerToPeer.sendRoutine", pkt)

			if pkt.src == nil {
				pkt.src = p2p.self.id
			}
			p2p.packetRw.WritePacket(pkt)
			p2p.packetRw.Flush()

			switch pkt.dest {
			case p2pDestAny: //broadcast
				if p2p.getSelfRole() >= p2pRoleRoot {
					p2p.sendToFriends(pkt)
					p2p.sendToDownside(pkt)
				} else {
					p2p.sendToDownside(pkt)
				}
			case p2pRoleRoot:
				if p2p.getSelfRole() >= p2pRoleRoot {
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
		log.Println("PeerToPeer.sendToPeer", p.id, pkt)
	} else {
		log.Println("PeerToPeer.sendToPeer not found", id)
	}

}

func (p2p *PeerToPeer) isAllowedRole(role PeerRoleFlag, p *Peer) bool {
	switch role {
	case p2pRoleSeed:
		log.Println("PeerToPeer.allowedSeeds", p2p.allowedRoots)
		return p2p.allowedSeeds.IsEmpty() || p2p.allowedSeeds.Has(p.id)
	case p2pRoleRoot:
		log.Println("PeerToPeer.isAllowedRoots", p2p.allowedRoots)
		return p2p.allowedRoots.IsEmpty() || p2p.allowedRoots.Has(p.id)
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
		case t := <-p2p.seedTicker.C:
			log.Println("PeerToPeer.discoveryRoutine seedTicker", t)
			p2p.syncSeeds()
		case t := <-p2p.discoveryTicker.C:
			log.Println("PeerToPeer.discoveryRoutine discoveryTicker", t)
			if p2p.self.role.Has(p2pRoleRoot) {
				roots := p2p.orphanages.PopByRole(p2pRoleRoot)
				for _, p := range roots {
					log.Println("PeerToPeer.discoveryRoutine p2pConnTypeFriend", p.id)
					p.connType = p2pConnTypeFriend
					p2p.friends.PushBack(p)
				}
				for _, s := range p2p.roots.Array() {
					if s != p2p.self.netAddress && !p2p.friends.HasNetAddresse(s) {
						log.Println("PeerToPeer.discoveryRoutine p2pRoleRoot", p2p.self, "dial to p2pRoleRoot", s)
						p2p.dialer.Dial(string(s))
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
	switch p2p.getSelfRole() {
	case p2pRoleNone:
		if p2p.parent != nil {
			p2p.sendQuery(p2p.parent)
		}
	case p2pRoleSeed:
		if p2p.parent != nil {
			p2p.sendQuery(p2p.parent)
		}
		for e := p2p.uncles.Front(); e != nil; e = e.Next() {
			p := e.Value.(*Peer)
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
				log.Println("PeerToPeer.discoveryRoutine seedTicker", p2p.self, "dial to p2pRoleSeed", s)
				p2p.dialer.Dial(string(s))
			}
		}
		for e := p2p.friends.Front(); e != nil; e = e.Next() {
			p := e.Value.(*Peer)
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
			p2p.preParent.PushBack(p)
			p2p.sendConnectionRequest(p2pConnTypeParent, p)
			log.Println("PeerToPeer.discoverParent try p2pConnTypeParent", p)
		} else {
			//TODO upgrade p2pConnTypeUncle to p2pConnTypeParent
			for _, s := range p2p.seeds.Array() {
				if s != p2p.self.netAddress && !p2p.uncles.HasNetAddresse(s) {
					log.Println("PeerToPeer.discoverParent", p2p.self, "dial to", s)
					p2p.dialer.Dial(string(s))
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
		p2p.sendConnectionRequest(p2pConnTypeUncle, p)
		log.Println("PeerToPeer.discoverUncle try p2pConnTypeUncle", p)
	}
}

type P2PConnectionRequest struct {
	ConnType PeerConnectionType
}

type P2PConnectionResponse struct {
	ReqConnType PeerConnectionType
	ConnType    PeerConnectionType
}

func (p2p *PeerToPeer) sendConnectionRequest(connType PeerConnectionType, p *Peer) {
	m := &P2PConnectionRequest{ConnType: connType}
	pkt := NewPacket(PROTO_P2P_CONN_REQ, p2p.encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.sendPacket(pkt)
	log.Println("PeerToPeer.sendConnectionRequest", m)
}
func (p2p *PeerToPeer) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := p2p.decodeMsgpack(pkt.payload, req)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("PeerToPeer.handleP2PConnectionRequest", req)
	m := &P2PConnectionResponse{ConnType: p2pConnTypeNone}
	if p.connType == p2pConnTypeNone {
		switch req.ConnType {
		case p2pConnTypeParent:
			p2p.orphanages.Remove(p)
			p2p.children.PushBack(p)
			p.connType = p2pConnTypeChildren
			m.ReqConnType = req.ConnType
			m.ConnType = p.connType
			//TODO p2p.children condition
		case p2pConnTypeUncle:
			p2p.orphanages.Remove(p)
			p2p.nephews.PushBack(p)
			p.connType = p2pConnTypeNephew
			m.ReqConnType = req.ConnType
			m.ConnType = p.connType
			//TODO p2p.nephews condition
		default:
			log.Println("PeerToPeer.handleP2PConnectionRequest ignore", req.ConnType)
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
		log.Println(err)
		return
	}
	log.Println("PeerToPeer.handleP2PConnectionResponse", resp)
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
					log.Println("PeerToPeer.handleP2PConnectionResponse reject", resp.ReqConnType)
				}
			} else {
				log.Println("PeerToPeer.handleP2PConnectionResponse wrong", resp.ReqConnType)
			}
		case p2pConnTypeUncle:
			if resp.ConnType == p2pConnTypeNephew {
				p2p.orphanages.Remove(p)
				p2p.uncles.PushBack(p)
				p.connType = resp.ReqConnType
			} else {
				//TODO handle to reject p2pConnTypeUncle
				log.Println("PeerToPeer.handleP2PConnectionResponse reject", resp.ReqConnType)
			}
		default:
			log.Println("PeerToPeer.handleP2PConnectionResponse ignore", resp.ReqConnType)
		}
	}
}

type PeerList struct {
	*list.List
	addrs []NetAddress
}

func NewPeerList() *PeerList {
	return &PeerList{List: list.New(), addrs: make([]NetAddress, 0, 64)}
}

func (l *PeerList) get(v *Peer) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(*Peer); s == v {
			return e
		}
	}
	return nil
}

func (l *PeerList) Remove(v *Peer) bool {
	defer l.cache()
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *PeerList) Has(v *Peer) bool {
	return l.get(v) != nil
}

func (l *PeerList) cache() {
	l.addrs = l.addrs[:0]
	for e := l.Front(); e != nil; e = e.Next() {
		s := e.Value.(*Peer)
		l.addrs = append(l.addrs, s.netAddress)
	}
}

func (l *PeerList) NetAddresses() []NetAddress {
	if len(l.addrs) != l.Len() {
		l.cache()
	}
	return l.addrs[:]
}

func (l *PeerList) HasNetAddresse(o NetAddress) bool {
	if len(l.addrs) != l.Len() {
		l.cache()
	}
	for _, na := range l.addrs {
		if na == o {
			return true
		}
	}
	return false
}

func (l *PeerList) GetByID(id module.PeerID) *Peer {
	for e := l.Front(); e != nil; e = e.Next() {
		if p := e.Value.(*Peer); p.id.Equal(id) {
			return p
		}
	}
	return nil
}
func (l *PeerList) GetByRoleAndIncomming(role PeerRoleFlag, in bool) *Peer {
	for e := l.Front(); e != nil; e = e.Next() {
		if p := e.Value.(*Peer); p.incomming == in && p.role.Has(role) {
			return p
		}
	}
	return nil
}
func (l *PeerList) PopByRole(role PeerRoleFlag) []*Peer {
	tl := make([]*list.Element, 0, l.Len())
	for e := l.Front(); e != nil; e = e.Next() {
		if p := e.Value.(*Peer); p.role.Has(role) {
			tl = append(tl, e)
		}
	}
	ps := make([]*Peer, len(tl))
	for i, t := range tl {
		ps[i] = t.Value.(*Peer)
		l.List.Remove(t)
	}
	return ps
}

type NetAddressList struct {
	*list.List
	arr []NetAddress
}

func NewNetAddressList() *NetAddressList {
	return &NetAddressList{List: list.New()}
}

func (l *NetAddressList) get(v NetAddress) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(NetAddress); s == v {
			return e
		}
	}
	return nil
}

func (l *NetAddressList) Remove(v NetAddress) bool {
	defer l.cache()
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *NetAddressList) Has(v NetAddress) bool {
	return l.get(v) != nil
}

func (l *NetAddressList) cache() {
	l.arr = l.arr[:0]
	for e := l.Front(); e != nil; e = e.Next() {
		l.arr = append(l.arr, e.Value.(NetAddress))
	}
}

func (l *NetAddressList) Array() []NetAddress {
	if len(l.arr) != l.Len() {
		l.cache()
	}
	return l.arr[:]
}

func (l *NetAddressList) Merge(arr []NetAddress) {
	for _, na := range arr {
		if !l.Has(na) {
			l.PushBack(na)
			log.Println("NetAddressList.Merge", na)
		}
	}
	if len(l.arr) != l.Len() {
		l.cache()
	}
}

func (l *NetAddressList) String() string {
	s := make([]string, 0, l.Len())
	for e := l.Front(); e != nil; e = e.Next() {
		s = append(s, string(e.Value.(NetAddress)))
	}
	return fmt.Sprintf("%v", s)
}
