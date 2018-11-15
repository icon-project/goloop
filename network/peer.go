package network

import (
	"fmt"
	"log"
	"net"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type Peer struct {
	id         module.PeerID
	netAddress NetAddress
	pubKey     *crypto.PublicKey
	//
	conn     net.Conn
	reader   *PacketReader
	writer   *PacketWriter
	onPacket packetCbFunc
	onError  errorCbFunc
	//
	incomming bool
	channel   string
	rtt       PeerRTT
	connType  PeerConnectionType
	role      PeerRoleFlag
}

type packetCbFunc func(pkt *Packet, p *Peer)
type errorCbFunc func(err error, p *Peer)

//TODO define netAddress as IP:Port
type NetAddress string

//TODO define PeerRTT
type PeerRTT uint32

const (
	p2pRoleNone     = 0x00
	p2pRoleSeed     = 0x01
	p2pRoleRoot     = 0x02
	p2pRoleRootSeed = 0x03
)

//PeerRoleFlag as BitFlag MSB[_,_,_,_,_,_,Root,Seed]LSB
//TODO remove p2pRoleRootSeed
type PeerRoleFlag byte

func (pr *PeerRoleFlag) Has(o PeerRoleFlag) bool {
	return (*pr)&o == o
}

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
	return fmt.Sprintf("{id:%v, addr:%v, in:%v, chan:%v, rtt:%v}",
		p.id, p.netAddress, p.incomming, p.channel, p.rtt)
}

func (p *Peer) ID() module.PeerID {
	return p.id
}

func (p *Peer) NetAddress() NetAddress {
	return p.netAddress
}

func (p *Peer) setPacketCbFunc(cbFunc packetCbFunc) {
	p.onPacket = cbFunc
}

func (p *Peer) setErrorCbFunc(cbFunc errorCbFunc) {
	p.onError = cbFunc
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
			p.onError(err, p)
			return
		}
		if pkt.hashOfPacket != h.Sum64() {
			log.Println("Invalid hashOfPacket :", pkt.hashOfPacket, ",expected:", h.Sum64())
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
		p.onError(err, p)
		return err
	}
	return p.writer.Flush()
}

const (
	peerIDSize = 20 //common.AddressBytes
)

type peerID struct {
	*common.Address
}

func NewPeerID(b []byte) module.PeerID {
	return &peerID{common.NewAccountAddress(b)}
}

func NewPeerIDFromPublicKey(k *crypto.PublicKey) module.PeerID {
	return &peerID{common.NewAccountAddressFromPublicKey(k)}
}

func (pi *peerID) Copy(b []byte) {
	copy(b[:peerIDSize], pi.ID())
}

func (pi *peerID) Equal(a module.Address) bool {
	return a.Equal(pi.Address)
}

func (pi *peerID) String() string {
	return pi.Address.String()
}
