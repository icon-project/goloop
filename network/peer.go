package network

import (
	"bufio"
	"net"

	"github.com/icon-project/goloop/module"
)

type PeerToPeer struct {
	channel         string
	ch              chan Packet
	onPacketCbFuncs map[module.ProtocolInfo]packetCbFunc
	listener        net.Listener
	self            Peer
	parent          Peer
	children        []Peer
	uncles          []Peer
	nephew          []Peer
	//[TBD] maintain address list within 2hop of current for status change
	grandParent   NetAddress
	grandChildren []NetAddress
	//[TBD] detecting duplicate transmission
	packetPool []Packet
}

//can be created each channel
func NewPeerToPeer(channel string) *PeerToPeer {
	return &PeerToPeer{channel: channel}
}

func (p2p *PeerToPeer) Start() {

}
func (p2p *PeerToPeer) Stop() {

}
func (p2p *PeerToPeer) onPacket(packet Packet, peer Peer) {

}
func (p2p *PeerToPeer) sendGoRoutine() {

}

type packetCbFunc func(packet Packet, peer Peer)

//srcPeerId, castType, destInfo, TTL(0:unlimited)
type Packet struct {
	protocol        module.ProtocolInfo //{Control, Membership, ...}
	subProtocol     module.ProtocolInfo
	lengthOfpayload int
	payload         []byte
	hashOfPacket    []byte
}

func NewPacket(subProtocol module.ProtocolInfo, payload []byte) *Packet {
	return &Packet{
		subProtocol:     subProtocol,
		lengthOfpayload: len(payload),
		payload:         payload[:],
	}
}

type Peer struct {
	id         module.PeerId
	netAddress NetAddress
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	onPacket   packetCbFunc
}

//TODO define netAddress as IP:Port
type NetAddress string

//Send marshalled packet to peer
func (p *Peer) send(bytes []byte) {

}

//receive from bufio.Reader, unmarshalling and peerToPeer.onPacket
func (p *Peer) receiveGoRoutine() {

}
