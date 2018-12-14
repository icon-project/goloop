package network

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"

	"github.com/icon-project/goloop/module"
)

type manager struct {
	channel     string
	memberships map[string]*membership
	p2p         *PeerToPeer
	log         *logger
}

func NewManager(channel string, t module.NetworkTransport, roles ...module.Role) module.NetworkManager {
	m := &manager{
		channel:     channel,
		memberships: make(map[string]*membership),
		p2p:         newPeerToPeer(channel, t),
		log:         newLogger("NetworkManager", channel),
	}

	//Create default membership for P2P topology management
	dms := m.GetMembership(DefaultMembershipName).(*membership)
	dms.roles[module.ROLE_SEED] = m.p2p.allowedSeeds
	dms.roles[module.ROLE_VALIDATOR] = m.p2p.allowedRoots
	dms.destByRole[module.ROLE_SEED] = p2pRoleSeed
	dms.destByRole[module.ROLE_VALIDATOR] = p2pRoleRoot

	role := PeerRoleFlag(p2pRoleNone)
	for _, r := range roles {
		switch r {
		case module.ROLE_SEED:
			role.SetFlag(p2pRoleSeed)
		case module.ROLE_VALIDATOR:
			role.SetFlag(p2pRoleRoot)
		default:
			m.log.Println("Ignore role", r)
		}
	}
	m.p2p.setRole(role)

	m.log.Println("NewManager", channel)
	return m
}

//TODO Multiple membership version
func (m *manager) GetMembership(name string) module.Membership {
	ms, ok := m.memberships[name]
	if !ok {
		pi := m.getProtocolInfo(name)
		ms = newMembership(name, pi, m.p2p)
		m.memberships[name] = ms
	}
	return ms
}

func (m *manager) GetPeers() []module.PeerID {
	arr := m.p2p.getPeers(true)
	l := make([]module.PeerID, len(arr))
	for i, p := range arr {
		l[i] = p.ID()
	}
	return l
}

//TODO protocolInfo management
func (m *manager) getProtocolInfo(name string) module.ProtocolInfo {
	if name == DefaultMembershipName {
		return PROTO_DEF_MEMBER
	}
	id := PROTO_DEF_MEMBER.ID() + byte(len(m.memberships))
	return NewProtocolInfo(id, 0)
}

type logger struct {
	name     string
	prefix   string
	excludes []string
}

func newLogger(name string, prefix string) *logger {
	//l := log.New(os.Stdout, fmt.Sprintf("[%s] %s", prefix, name), log.LstdFlags)
	return &logger{name, prefix, make([]string, 0)}
}

func (l *logger) printable(v interface{}) bool {
	for _, e := range singletonLoggerExcludes {
		if e == l.name {
			return false
		}
	}

	if len(l.excludes) < 1 {
		return true
	}
	s, ok := v.(string)
	if !ok {
		return true
	}
	for _, e := range l.excludes {
		if strings.HasPrefix(s, e) {
			return false
		}
	}
	return true
}

func (l *logger) Println(v ...interface{}) {
	if v[0] == "Warning" || l.printable(v[0]) {
		//%T : type //%#v
		s := fmt.Sprintf("[%s] %s", l.prefix, l.name)
		w := make([]interface{}, len(v)+1)
		copy(w[1:], v)
		w[0] = s
		log.Println(w...)
	}
}

func (l *logger) Printf(format string, v ...interface{}) {
	if l.printable(format) {
		s := fmt.Sprintf(format, v...)
		l.Println(s)
	}
}

type protocolInfo uint16

func NewProtocolInfo(id byte, version byte) module.ProtocolInfo {
	return protocolInfo(int(id)<<8 | int(version))
}
func NewProtocolInfoFrom(pi module.ProtocolInfo) module.ProtocolInfo {
	return NewProtocolInfo(pi.ID(), pi.Version())
}
func newProtocolInfo(b []byte) protocolInfo {
	return protocolInfo(binary.BigEndian.Uint16(b[:2]))
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
	//return fmt.Sprintf("{ID:%#02x,Ver:%#02x}", pi.ID(), pi.Version())
	return fmt.Sprintf("{%#04x}", pi.Uint16())
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
