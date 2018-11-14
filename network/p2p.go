package network

import (
	"container/list"
	"fmt"
	"log"
	"reflect"

	"github.com/icon-project/goloop/module"
	"github.com/ugorji/go/codec"
)

type PeerToPeer struct {
	channel         string
	ch              chan *Packet
	onPacketCbFuncs map[module.ProtocolInfo]packetCbFunc
	//[TBD] detecting duplicate transmission
	packetPool []Packet
	packetRw   *PacketReadWriter
	dialer     *Dialer

	//Topology with Connected Peers
	self     *Peer
	parent   *Peer
	children *PeerList
	uncles   *PeerList
	nephews  *PeerList
	//Only for root, parent is nil, uncles is empty
	friends *PeerList

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
func newPeerToPeer(channel string, id module.PeerID, addr NetAddress) *PeerToPeer {
	p2p := &PeerToPeer{
		channel:         channel,
		ch:              make(chan *Packet),
		onPacketCbFuncs: make(map[module.ProtocolInfo]packetCbFunc),
		packetPool:      make([]Packet, 0),
		packetRw:        NewPacketReadWriter(),
		dialer:          GetDialer(channel),
		//
		self:     &Peer{id: id, netAddress: addr},
		children: NewPeerList(),
		uncles:   NewPeerList(),
		nephews:  NewPeerList(),
		friends:  NewPeerList(),
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
	go p2p.sendGoRoutine()
	return p2p
}

func (p2p *PeerToPeer) setPacketCbFunc(pi module.ProtocolInfo, cbFunc packetCbFunc) {
	if _, ok := p2p.onPacketCbFuncs[pi]; ok {
		log.Println("Warnning overwrite packetCbFunc", pi)
	}
	p2p.onPacketCbFuncs[pi] = cbFunc
}

//callback from PeerDispatcher.onPacket
func (p2p *PeerToPeer) onPeer(p *Peer) {
	log.Println("PeerToPeer.onPeer", p)
	if !p.incomming {
		p2p.sendQuery(p)
	} else {
		//peer can be children or nephews
	}
	//TODO discoveryRoutine with time.Ticker
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (p2p *PeerToPeer) onError(err error, p *Peer) {

}

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
			}
		}
	} else {
		if p.connType == p2pConnTypeNone {
			log.Println("Ignoring packet, because undetermined PeerConnectionType is not allowed to handle")
		}
		if cbFunc := p2p.onPacketCbFuncs[pkt.protocol]; cbFunc != nil {
			cbFunc(pkt, p)
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
		case p2pRoleRoot:
			log.Println("Wrong situation")
		default:
			//TODO p2p.preParent.Merge(qrm.Children)
			p2p.seeds.Merge(qrm.Seeds)
		}
	default:
		p2p.seeds.Merge(qrm.Seeds)
		p2p.roots.Merge(qrm.Roots)
	}
}

func (p2p *PeerToPeer) sendToFriends(pkt *Packet) {
	p2p.sendToPeers(pkt, p2p.friends)
}

func (p2p *PeerToPeer) sendToUpside(pkt *Packet) {
	p2p.sendToPeer(pkt, p2p.parent)
	//TODO after next period
	//p2p.sendToPeers(pkt, p2p.uncles)
}

func (p2p *PeerToPeer) sendToDownside(pkt *Packet) {
	p2p.sendToPeers(pkt, p2p.children)
	//TODO after next period
	//p2p.sendToPeers(pkt, p2p.nephews)
}

func (p2p *PeerToPeer) sendToPeer(pkt *Packet, p *Peer) {
	if p != nil {
		p2p.packetRw.WriteTo(p.conn)
	}
}

func (p2p *PeerToPeer) sendToPeers(pkt *Packet, peers *PeerList) {
	for e := peers.Front(); e != nil; e = e.Next() {
		p := e.Value.(*Peer)
		p2p.packetRw.WriteTo(p.conn)
	}
}

func (p2p *PeerToPeer) sendGoRoutine() {
	select {
	case pkt := <-p2p.ch:
		log.Println("PeerToPeer.sendGoRoutine", pkt)

		if pkt.src == nil {
			pkt.src = p2p.self.id
		}
		p2p.packetRw.WritePacket(pkt)
		p2p.packetRw.Flush()
		if pkt.dest == p2pDestAny {
			//case broadcast
			if p2p.getSelfRole() >= p2pRoleRoot {
				p2p.sendToFriends(pkt)
				p2p.sendToDownside(pkt)
			} else {
				p2p.sendToDownside(pkt)
			}
			//case multicast : //TODO Routing or Flooding
			// if p2p.getSelfRole() >= p2pRoleRoot {
			// 	p2p.sendToFriends(pkt)
			// } else {
			// 	p2p.sendToUpside(pkt)
			// }
		} else {

		}
	}
}

func (p2p *PeerToPeer) isAllowedRole(role PeerRoleFlag, p *Peer) bool {
	switch role {
	case p2pRoleRoot:
		log.Println("p2p.isAllowedRoots", p2p.allowedRoots)
		return p2p.allowedRoots.IsEmpty() || p2p.allowedRoots.Has(p.id)
	case p2pRoleSeed:
		log.Println("p2p.allowedSeeds", p2p.allowedRoots)
		return p2p.allowedSeeds.IsEmpty() || p2p.allowedSeeds.Has(p.id)
	case p2pRoleRootSeed:
		return p2p.isAllowedRole(p2pRoleRoot, p) && p2p.isAllowedRole(p2pRoleSeed, p)
	default:
		return false
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
