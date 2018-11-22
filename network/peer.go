package network

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

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
	onClose  closeCbFunc
	mtx      sync.Mutex
	//
	incomming bool
	channel   string
	rtt       PeerRTT
	connType  PeerConnectionType
	role      PeerRoleFlag
}

type packetCbFunc func(pkt *Packet, p *Peer)
type errorCbFunc func(err error, p *Peer)
type closeCbFunc func(p *Peer)

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

func (pr *PeerRoleFlag) Set(o PeerRoleFlag) {
	*pr = o
}

func (pr *PeerRoleFlag) SetFlag(o PeerRoleFlag) {
	*pr |= o
}
func (pr *PeerRoleFlag) UnSetFlag(o PeerRoleFlag) {
	*pr &= (^o)
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
	p.setErrorCbFunc(func(err error, p *Peer) {
		p.conn.Close()
	})
	p.setCloseCbFunc(func(p *Peer) {
		//ignore
	})
	go p.receiveRoutine()
	return p
}

func (p *Peer) String() string {
	return fmt.Sprintf("{id:%v, addr:%v, in:%v, channel:%v, role:%v, rtt:%v}",
		p.id, p.netAddress, p.incomming, p.channel, p.role, p.rtt)
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

func (p *Peer) setCloseCbFunc(cbFunc closeCbFunc) {
	p.onClose = cbFunc
}

func (p *Peer) setRole(role PeerRoleFlag) {
	p.role.Set(role)
}

//receive from bufio.Reader, unmarshalling and peerToPeer.onPacket
func (p *Peer) receiveRoutine() {
	defer func() {
		if err := recover(); err != nil {
			//TODO recover()
			log.Fatal(err)
		}
		p.conn.Close()
	}()
	for {
		pkt, h, err := p.reader.ReadPacket()
		if err != nil {
			if oe, ok := err.(*net.OpError); ok { //after p.conn.Close()
				//referenced from golang.org/x/net/http2/server.go isClosedConnError
				if strings.Contains(oe.Err.Error(), "use of closed network connection") {
					p.onClose(p)
				}
			} else if err == io.EOF || err == io.ErrUnexpectedEOF { //half Close (recieved tcp close)
				p.onClose(p)
			} else {
				//TODO
				// p.reader.Reset()
				p.onError(err, p)
			}
			return
		}
		if pkt.hashOfPacket != h.Sum64() {
			log.Println("Peer.receiveRoutine Invalid hashOfPacket :", pkt.hashOfPacket, ",expected:", h.Sum64())
			continue
		} else {
			p.onPacket(pkt, p)
		}
	}
}

//Send marshalled packet to peer
// func (p *Peer) sendFrom(rd io.Reader) error {
// 	n, err := p.writer.ReadFrom(rd)
// 	if err != nil {
// 		//TODO
// 		return err
// 	}
// 	return p.writer.Flush()
// }

func (p *Peer) sendPacket(pkt *Packet) {
	defer p.mtx.Unlock()
	p.mtx.Lock()
	if err := p.writer.WritePacket(pkt); err != nil {
		log.Printf("Peer.sendPacket WritePacket onError %T %#v %s", err, err, p.String())
		//TODO
		p.onError(err, p)
	} else if err := p.writer.Flush(); err != nil {
		log.Printf("Peer.sendPacket Flush onError %T %#v %s", err, err, p.String())
		//TODO
		p.onError(err, p)
	}
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
