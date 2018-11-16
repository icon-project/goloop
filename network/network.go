package network

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/icon-project/goloop/module"
)

type manager struct {
	channel     string
	memberships map[string]module.Membership
	peerToPeer  *PeerToPeer
}

func newManager(channel string, id module.PeerID, addr NetAddress, d *Dialer) *manager {
	m := &manager{
		channel:     channel,
		memberships: make(map[string]module.Membership),
		peerToPeer:  newPeerToPeer(channel, id, addr, d),
	}

	//Create default membership for P2P topology management
	dms := m.GetMembership(DefaultMembershipName).(*membership)
	dms.roles[module.ROLE_SEED] = m.peerToPeer.allowedSeeds
	dms.roles[module.ROLE_VALIDATOR] = m.peerToPeer.allowedRoots
	dms.destByRole[module.ROLE_SEED] = p2pRoleSeed
	dms.destByRole[module.ROLE_VALIDATOR] = p2pRoleRoot

	log.Println("network.newManager", channel, id, addr)
	return m
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

//TODO protocolInfo management
func (m *manager) getProtocolInfo(name string) module.ProtocolInfo {
	pi := module.ProtocolInfo(PROTO_DEF_MEMBER)
	if name == DefaultMembershipName {
		return pi
	} else {
		return NewProtocolInfoWithIdVersion(pi.ID()+byte(len(m.memberships)), 0)
	}
}

type protocolInfo uint16

func NewProtocolInfo(b []byte) module.ProtocolInfo {
	return protocolInfo(binary.BigEndian.Uint16(b[:2]))
}
func NewProtocolInfoWithIdVersion(id byte, version byte) module.ProtocolInfo {
	return protocolInfo(int(id)<<8 | int(version))
}
func (pi protocolInfo) ID() byte {
	return byte(pi >> 8)
}
func (pi protocolInfo) Version() byte {
	return byte(pi)
}
func (pi protocolInfo) Copy(b []byte) {
	binary.BigEndian.PutUint16(b[:2], uint16(pi))
}
func (pi protocolInfo) String() string {
	return fmt.Sprintf("{ID:%#02x,Ver:%#02x}", pi.ID(), pi.Version())
}
func (pi protocolInfo) Uint16() uint16 {
	return uint16(pi)
}

//////////////////if using marshall/unmarshall of membership
type MessageMembership interface {
	//set marshaller each message type << extends
	UnicastMessage(message struct{}, id module.PeerID) error
	MulticastMessage(message struct{}, authority module.Authority) error
	BroadcastMessage(message struct{}, broadcastType module.BroadcastType) error

	//callback from PeerToPeer.onPacket()
	//using worker pattern {pool or each packet or none} for reactor
	onPacket(packet Packet, peer Peer)
	//from Peer.sendRoutine()
	onError()
}

type PacketReactor interface {
	OnPacket(packet Packet, id module.PeerID)
}

type MessageReactor interface {
	module.Reactor

	//Empty list일경우 모든 값에 대해 Callback이 호출된다.
	SubProtocols() map[module.ProtocolInfo]interface{}

	OnMarshall(subProtocol module.ProtocolInfo, message interface{}) ([]byte, error)
	//nil을 리턴할경우
	OnUnmarshall(subProtocol module.ProtocolInfo, bytes []byte) (interface{}, error)

	//goRoutine by Membership.onPacket() like worker pattern
	OnMessage(message interface{}, id module.PeerID)
}

////////////util classes
type StringList struct {
	*list.List
}

func NewStringList() *StringList {
	return &StringList{list.New()}
}

func (l *StringList) get(v string) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(string); s == v {
			return e
		}
	}
	return nil
}

func (l *StringList) Remove(v string) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *StringList) Has(v string) bool {
	return l.get(v) != nil
}
