package network

import (
	"github.com/icon-project/goloop/module"
)

type manager struct {
	channel     string
	memberships []module.Membership
	peerToPeer  *PeerToPeer
}

//can be created each channel
func NewNetworkManager(channel string) module.NetworkManager {
	return &manager{
		channel:    channel,
		peerToPeer: NewPeerToPeer(channel),
	}
}

func (m *manager) Start() {
	m.peerToPeer.Start()
}

func (m *manager) Stop() {
	m.peerToPeer.Stop()
}

func (m *manager) GetMembership(name string) module.Membership {
	return nil
}

//////////////////if using marshall/unmarshall of membership
type MessageMembership interface {
	//set marshaller each message type << extends
	UnicastMessage(message struct{}, peerId module.PeerId) error
	MulticastMessage(message struct{}, authority module.Authority) error
	BroadcastMessage(message struct{}, broadcastType module.BroadcastType) error

	//callback from PeerToPeer.onPacket()
	//using worker pattern {pool or each packet or none} for reactor
	onPacket(packet Packet, peer Peer)
	//from Peer.sendGoRoutine()
	onError()
}

type PacketReactor interface {
	OnPacket(packet Packet, peerId module.PeerId)
}

type MessageReactor interface {
	module.Reactor

	//Empty list일경우 모든 값에 대해 Callback이 호출된다.
	SubProtocols() map[module.ProtocolInfo]interface{}

	OnMarshall(subProtocol module.ProtocolInfo, message interface{}) ([]byte, error)
	//nil을 리턴할경우
	OnUnmarshall(subProtocol module.ProtocolInfo, bytes []byte) (interface{}, error)

	//goRoutine by Membership.onPacket() like worker pattern
	OnMessage(message interface{}, peerId module.PeerId)
}
