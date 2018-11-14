package network

import (
	"github.com/icon-project/goloop/module"
)

var (
	managers = make(map[string]module.NetworkManager)
)

type manager struct {
	channel     string
	memberships map[string]module.Membership
	peerToPeer  *PeerToPeer
}

const (
	DEF_MEMBERSHIP_NAME = ""
)

const (
	PROTO_CONTOL     = 0x0000
	PROTO_DEF_MEMBER = 0x0100
)

//can be created each channel
func GetNetworkManager(channel string) module.NetworkManager {
	mgr, ok := managers[channel]
	if !ok {
		m := &manager{
			channel:     channel,
			memberships: make(map[string]module.Membership),
			peerToPeer:  newPeerToPeer(channel),
		}
		//Create default membership for P2P topology management
		dms := m.GetMembership(DEF_MEMBERSHIP_NAME).(*membership)
		dms.roles[module.ROLE_VALIDATOR] = m.peerToPeer.allowedRoots
		dms.roles[module.ROLE_SEED] = m.peerToPeer.allowedSeeds

		mgr = m
		managers[channel] = m

	}
	return mgr
}

//TODO Multiple membership version
func (m *manager) GetMembership(name string) module.Membership {
	ms, ok := m.memberships[name]
	if !ok {
		pi := m.getProtocolInfo(name)
		ms = newMembership(name, pi, m.peerToPeer)
		m.memberships[name] = ms
	}
	return ms
}

func (m *manager) getProtocolInfo(name string) module.ProtocolInfo {
	pi := module.ProtocolInfo(PROTO_DEF_MEMBER)
	if name == DEF_MEMBERSHIP_NAME {
		return pi
	} else {
		return module.NewProtocolInfo(pi.Id()+byte(len(m.memberships)), 0)
	}
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
