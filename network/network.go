package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/icon-project/goloop/module"
)

type manager struct {
	channel string
	p2p     *PeerToPeer
	//
	roles       map[module.Role]*PeerIDSet
	authorities map[module.Authority]*RoleSet
	destByRole  map[module.Role]byte
	//
	protocolHandlers map[string]*protocolHandler
	priority         map[protocolInfo]uint8

	//log
	log *logger
}

func NewManager(channel string, t module.NetworkTransport, roles ...module.Role) module.NetworkManager {
	m := &manager{
		channel:          channel,
		p2p:              newPeerToPeer(channel, t),
		roles:            make(map[module.Role]*PeerIDSet),
		authorities:      make(map[module.Authority]*RoleSet),
		destByRole:       make(map[module.Role]byte),
		protocolHandlers: make(map[string]*protocolHandler),
		priority:         make(map[protocolInfo]uint8),
		log:              newLogger("NetworkManager", channel),
	}

	//Create default protocolHandler for P2P topology management
	m.roles[module.ROLE_SEED] = m.p2p.allowedSeeds
	m.roles[module.ROLE_VALIDATOR] = m.p2p.allowedRoots
	m.destByRole[module.ROLE_SEED] = p2pRoleSeed
	m.destByRole[module.ROLE_VALIDATOR] = p2pRoleRoot

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

func (m *manager) Channel() string {
	return m.channel
}
func (m *manager) PeerID() module.PeerID {
	return m.p2p.self.id
}

func (m *manager) GetPeers() []module.PeerID {
	arr := m.p2p.getPeers(true)
	l := make([]module.PeerID, len(arr))
	for i, p := range arr {
		l[i] = p.ID()
	}
	return l
}

func (m *manager) RegisterReactor(name string, r module.Reactor, spiList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	if priority < 1 || priority > DefaultSendQueueMaxPriority {
		log.Panicf("priority must be positive value and less than %d", DefaultSendQueueMaxPriority)
	}

	if _, ok := m.protocolHandlers[name]; ok {
		return nil, ErrAlreadyRegisteredReactor
	}

	//TODO protocolInfo management
	pi := newProtocolInfo(byte(len(m.protocolHandlers))+1, 0)
	ph := newProtocolHandler(m, pi, spiList, r, name, priority)
	m.p2p.setCbFunc(pi, ph.onPacket, ph.onError, ph.onEvent, p2pEventJoin, p2pEventLeave, p2pEventDuplicate)

	m.protocolHandlers[name] = ph
	m.priority[pi] = priority
	return ph, nil
}

func (m *manager) SetWeight(pi protocolInfo, weight int) error {
	return m.p2p.sendQueue.SetWeight(int(pi.ID()), weight)
}

func (m *manager) unicast(pi protocolInfo, spi protocolInfo, bytes []byte, id module.PeerID) error {
	pkt := NewPacket(pi, spi, bytes)
	pkt.protocol = pi
	pkt.dest = p2pDestPeer
	pkt.destPeer = id
	pkt.priority = m.priority[pi]
	err := m.p2p.send(pkt)
	return NewNetworkError(err)
}

//TxMessage,PrevoteMessage, Send to Validators
func (m *manager) multicast(pi protocolInfo, spi protocolInfo, bytes []byte, role module.Role) error {
	if _, ok := m.roles[role]; !ok {
		return ErrNotRegisteredRole
	}
	//TODO Check authority
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = m.destByRole[role]
	pkt.priority = m.priority[pi]
	return m.p2p.send(pkt)
}

//ProposeMessage,PrecommitMessage,BlockMessage, Send to Citizen
func (m *manager) broadcast(pi protocolInfo, spi protocolInfo, bytes []byte, broadcastType module.BroadcastType) error {
	//TODO Check authority
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = p2pDestAny
	pkt.ttl = byte(broadcastType)
	pkt.priority = m.priority[pi]
	err := m.p2p.send(pkt)
	if err == ErrNotAvailable && pkt.dest == p2pDestAny &&
		pkt.ttl != 1 && m.p2p.self.compareRole(p2pRoleNone, true) {
		err = nil
	}
	return err
}

func (m *manager) relay(pkt *Packet) error {
	//TODO Check authority
	if pkt.ttl == 1 {
		return errors.New("not allowed relay")
	}
	pkt.priority = m.priority[pkt.protocol]
	err := m.p2p.send(pkt)
	if err == ErrNotAvailable && pkt.dest == p2pDestAny && m.p2p.self.compareRole(p2pRoleNone, true) {
		err = nil
	}
	return err
}

func (m *manager) getRolePeerIDSet(role module.Role) *PeerIDSet {
	s, ok := m.roles[role]
	if !ok {
		s := NewPeerIDSet()
		m.roles[role] = s
		m.destByRole[role] = byte(len(m.roles) + p2pDestPeerGroup)
	}
	return s
}

func (m *manager) SetRole(role module.Role, peers ...module.PeerID) {
	s := m.getRolePeerIDSet(role)
	s.ClearAndAdd(peers...)
}

func (m *manager) GetPeersByRole(role module.Role) []module.PeerID {
	s := m.getRolePeerIDSet(role)
	return s.Array()
}

func (m *manager) AddRole(role module.Role, peers ...module.PeerID) {
	s := m.getRolePeerIDSet(role)
	for _, p := range peers {
		if !s.Contains(p) {
			s.Add(p)
		}
	}
}

func (m *manager) RemoveRole(role module.Role, peers ...module.PeerID) {
	s := m.getRolePeerIDSet(role)
	for _, p := range peers {
		s.Remove(p)
	}
}

func (m *manager) HasRole(role module.Role, id module.PeerID) bool {
	s := m.getRolePeerIDSet(role)
	return s.Contains(id)
}

func (m *manager) Roles(id module.PeerID) []module.Role {
	var i int
	s := make([]module.Role, 0, len(m.roles))
	for k, v := range m.roles {
		if v.Contains(id) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

func (m *manager) getAuthorityRoleSet(authority module.Authority) *RoleSet {
	s, ok := m.authorities[authority]
	if !ok {
		s := NewRoleSet()
		m.authorities[authority] = s
	}
	return s
}

func (m *manager) GrantAuthority(authority module.Authority, roles ...module.Role) {
	s := m.getAuthorityRoleSet(authority)
	for _, r := range roles {
		if !s.Contains(r) {
			s.Add(r)
		}
	}
}

func (m *manager) DenyAuthority(authority module.Authority, roles ...module.Role) {
	l := m.getAuthorityRoleSet(authority)
	for _, r := range roles {
		l.Remove(r)
	}
}

func (m *manager) HasAuthority(authority module.Authority, role module.Role) bool {
	l := m.getAuthorityRoleSet(authority)
	return l.Contains(role)
}

func (m *manager) Authorities(role module.Role) []module.Authority {
	var i int
	s := make([]module.Authority, len(m.authorities))
	for k, v := range m.authorities {
		if v.Contains(role) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
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
	return newProtocolInfo(id, version)
}
func NewProtocolInfoFrom(pi module.ProtocolInfo) module.ProtocolInfo {
	return NewProtocolInfo(pi.ID(), pi.Version())
}
func newProtocolInfo(id byte, version byte) protocolInfo {
	return protocolInfo(int(id)<<8 | int(version))
}
func newProtocolInfoFrom(b []byte) protocolInfo {
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

type NetworkError struct {
	error
	IsTemporary bool
}
func(e *NetworkError) Temporary() bool {return e.IsTemporary}
func NewNetworkError(err error) module.NetworkError{
	if err != nil {
		isTemporary := false
		switch err {
		case ErrQueueOverflow:
			isTemporary = true
		}
		return &NetworkError{err, isTemporary}
	}
	return nil
}