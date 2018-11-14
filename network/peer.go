package network

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/icon-project/goloop/common/crypto"
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
}

//can be created each channel
func newPeerToPeer(channel string) *PeerToPeer {
	c := GetConfig()
	p2p := &PeerToPeer{
		channel:         channel,
		ch:              make(chan *Packet),
		onPacketCbFuncs: make(map[module.ProtocolInfo]packetCbFunc),
		packetPool:      make([]Packet, 0),
		packetRw:        NewPacketReadWriter(bytes.NewBuffer(make([]byte, PacketBufferSize))),
		dialer:          NewDialer(channel),
		//
		self:     &Peer{id: NewPeerIdFromPublicKey(c.PublicKey)},
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
	}
	GetPeerDispatcher().registPeerToPeer(p2p)
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

var (
	mph = &codec.MsgpackHandle{}
)

func init() {
	mph.MapType = reflect.TypeOf(map[string]interface{}(nil))
}

func encodeMsgpack(v interface{}) []byte {
	b := make([]byte, PacketBufferSize)
	enc := codec.NewEncoderBytes(&b, mph)
	enc.MustEncode(v)
	return b
}

func decodeMsgpack(b []byte, v interface{}) error {
	dec := codec.NewDecoderBytes(b, mph)
	return dec.Decode(v)
}

const (
	PROTO_P2P_QUERY        = 0x0201
	PROTO_P2P_QUERY_RESULT = 0x0301
)

type QueryMessage struct {
	Role PeerRoleFlag
}

type QueryResultMessage struct {
	Seeds    []NetAddress
	Roots    []NetAddress
	Children []NetAddress
	Message  string
}

func (p2p *PeerToPeer) sendQuery(p *Peer) {
	m := &QueryMessage{Role: p2p.self.role}
	pkt := NewPacket(PROTO_P2P_QUERY, encodeMsgpack(m))
	pkt.src = p2p.self.id
	p.sendPacket(pkt)
}

func (p2p *PeerToPeer) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := decodeMsgpack(pkt.payload, qm)
	if err != nil {
		log.Println(err)
		return
	}
	m := &QueryResultMessage{}
	if p2p.isAllowedRole(qm.Role, p) {
		switch qm.Role {
		case p2pRoleNone:
			switch p2p.self.role {
			case p2pRoleNone:
				m.Children = p2p.children.NetAddresses()
			case p2pRoleRoot:
				m.Message = "not allowed"
				//p.conn.Close()
			default: //p2pRoleSeed, ROLE_P2P_ROOTSEED
				m.Seeds = p2p.seeds.Array()
				m.Children = p2p.children.NetAddresses()
			}
		default: //p2pRoleSeed, p2pRoleRoot, ROLE_P2P_ROOTSEED
			m.Seeds = p2p.seeds.Array()
			m.Roots = p2p.roots.Array()
		}
		p.role = qm.Role
	} else {
		m.Message = "not allowed"
		//p.conn.Close()
	}
	rpkt := NewPacket(PROTO_P2P_QUERY_RESULT, encodeMsgpack(m))
	rpkt.src = p2p.self.id
	p.sendPacket(rpkt)
}

func (p2p *PeerToPeer) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := decodeMsgpack(pkt.payload, qrm)
	if err != nil {
		log.Println(err)
		return
	}
	switch p2p.self.role {
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

		if pkt.src.IsNil() {
			pkt.src = p2p.self.id
		}
		p2p.packetRw.WritePacket(pkt)
		p2p.packetRw.Flush()
		if pkt.dest == p2pDestAny {
			//case broadcast
			if p2p.self.role >= p2pRoleRoot {
				p2p.sendToFriends(pkt)
				p2p.sendToDownside(pkt)
			} else {
				p2p.sendToDownside(pkt)
			}
			//case multicast : //TODO Routing or Flooding
			// if p2p.self.role >= p2pRoleRoot {
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
		return p2p.allowedRoots.IsEmpty() || p2p.allowedRoots.Has(p.id)
	case p2pRoleSeed:
		return p2p.allowedSeeds.IsEmpty() || p2p.allowedSeeds.Has(p.id)
	case p2pRoleRootSeed:
		return p2p.isAllowedRole(p2pRoleRoot, p) && p2p.isAllowedRole(p2pRoleSeed, p)
	default:
		return true
	}
}

type Peer struct {
	id         module.PeerID
	netAddress NetAddress
	pubKey     *crypto.PublicKey
	//
	conn     net.Conn
	reader   *PacketReader
	writer   *PacketWriter
	onPacket packetCbFunc
	//
	incomming bool
	channel   string
	rtt       PeerRTT
	connType  PeerConnectionType
	role      PeerRoleFlag
}

type packetCbFunc func(packet *Packet, peer *Peer)

//TODO define netAddress as IP:Port
type NetAddress string

//TODO define PeerRTT
type PeerRTT uint32

const (
	p2pDestAny  = 0x00
	p2pDestPeer = 0xFF
)

//TODO define PeerRTT
const (
	p2pRoleNone     = 0x00
	p2pRoleSeed     = 0x01
	p2pRoleRoot     = 0x02
	p2pRoleRootSeed = 0x03
)

type PeerRoleFlag byte

const (
	p2pConnTypeNone = iota
	p2pConnTypeParent
	p2pConnTypeChildren
	p2pConnTypeUncle
	p2pConnTypeNephew
	p2pConnTypeFriend
)

type PeerConnectionType byte

func newPeer(conn net.Conn, cbFunc packetCbFunc, incomming bool) *Peer {
	p := &Peer{
		conn:      conn,
		reader:    NewPacketReader(conn),
		writer:    NewPacketWriter(conn),
		incomming: incomming,
	}
	p.setPacketCbFunc(cbFunc)
	go p.receiveRoutine()
	return p
}

func (p *Peer) String() string {
	return fmt.Sprintf("{id:%s, addr:%v, in:%v, chan:%v, rtt:%v}",
		p.id, p.netAddress, p.incomming, p.channel, p.rtt)
}

func (p *Peer) Id() module.PeerID {
	return p.id
}

func (p *Peer) NetAddress() NetAddress {
	return p.netAddress
}

func (p *Peer) setPacketCbFunc(cbFunc packetCbFunc) {
	p.onPacket = cbFunc
}

//receive from bufio.Reader, unmarshalling and peerToPeer.onPacket
func (p *Peer) receiveRoutine() {
	defer p.conn.Close()
	for {
		pkt, h, err := p.reader.ReadPacket()
		if err != nil {
			//TODO
			// p.reader.Reset()
			log.Println(pkt, h, err)
			return
		}
		if rh := binary.BigEndian.Uint64(pkt.hashOfPacket); rh != h.Sum64() {
			log.Println("Invalid hashOfPacket :", rh, ",expected:", h.Sum64())
		}
		if p.onPacket != nil {
			p.onPacket(pkt, p)
		}
	}
}

//Send marshalled packet to peer
// func (p *Peer) sendFrom(rd io.Reader) error {
// 	n, err := p.writer.ReadFrom(rd)
// 	if err != nil {
// 		//TODO
// 		log.Println(n, err)
// 		return err
// 	}
// 	return p.writer.Flush()
// }

func (p *Peer) sendPacket(pkt *Packet) error {
	err := p.writer.WritePacket(pkt)
	if err != nil {
		//TODO
		log.Println(err)
		return err
	}
	return p.writer.Flush()
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
