package network

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type manager struct {
	channel string
	p2p     *PeerToPeer
	//
	roles      map[module.Role]*PeerIDSet
	destByRole map[module.Role]byte
	roleByDest map[byte]module.Role
	//
	protocolHandlers map[string]*protocolHandler
	priority         map[protocolInfo]uint8

	mtx sync.RWMutex

	pd *PeerDispatcher
	//log
	log *logger

	//monitor
	mtr *metric.NetworkMetric
}

func NewManager(c module.Chain, nt module.NetworkTransport, initialSeed string, roles ...module.Role) module.NetworkManager {
	t := nt.(*transport)
	self := &Peer{id: t.PeerID(), netAddress: NetAddress(t.Address())}
	channel := strconv.FormatInt(int64(c.NID()), 16)
	mtr := metric.NewNetworkMetric(c.MetricContext())
	m := &manager{
		channel:          channel,
		p2p:              newPeerToPeer(channel, self, t.GetDialer(channel), mtr),
		roles:            make(map[module.Role]*PeerIDSet),
		destByRole:       make(map[module.Role]byte),
		roleByDest:       make(map[byte]module.Role),
		protocolHandlers: make(map[string]*protocolHandler),
		priority:         make(map[protocolInfo]uint8),
		pd:               t.pd,
		log:              newLogger("NetworkManager", channel),
		mtr:              mtr,
	}

	//Create default protocolHandler for P2P topology management
	m.roles[module.ROLE_SEED] = m.p2p.allowedSeeds
	m.roles[module.ROLE_VALIDATOR] = m.p2p.allowedRoots
	m.roles[module.ROLE_NORMAL] = m.p2p.allowedPeers
	m.destByRole[module.ROLE_SEED] = p2pRoleSeed
	m.destByRole[module.ROLE_VALIDATOR] = p2pRoleRoot
	m.destByRole[module.ROLE_NORMAL] = p2pRoleNone //same as broadcast

	role := PeerRoleFlag(p2pRoleNone)
	for _, r := range roles {
		switch r {
		case module.ROLE_SEED:
			role.SetFlag(p2pRoleSeed)
		case module.ROLE_VALIDATOR:
			role.SetFlag(p2pRoleRoot)
		default:
			m.log.Println("Warning", "NewManager", "ignored role", r)
		}
	}
	m.p2p.setRole(role)
	if initialSeed != "" {
		m.p2p.seeds.Add(NetAddress(initialSeed))
	}

	m.log.Println("NewManager", channel)
	m.log.excludes = []string{
		"SetRole",
	}
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

func (m *manager) Term() {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	_ = m._stop()
	m.log.Println("Term protocolHandlers")
	for _, ph := range m.protocolHandlers {
		m.log.Println("Term", ph.name)
		ph.Term()
	}
}

func (m *manager) Start() error {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if m.p2p.IsStarted() {
		return nil
	}
	if !m.pd.registerPeerToPeer(m.p2p) {
		log.Panicf("already registered p2p %s", m.channel)
	}
	m.p2p.Start()
	return nil
}

func (m *manager) _stop() error {
	if !m.p2p.IsStarted() {
		return nil
	}
	if !m.pd.unregisterPeerToPeer(m.p2p) {
		log.Panicf("already unregistered p2p %s", m.channel)
	}
	m.p2p.Stop()
	return nil
}

func (m *manager) Stop() error {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	return m._stop()
}

func (m *manager) RegisterReactor(name string, r module.Reactor, spiList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if priority < 1 || priority > DefaultSendQueueMaxPriority {
		log.Panicf("priority must be positive value and less than %d", DefaultSendQueueMaxPriority)
	}

	if _, ok := m.protocolHandlers[name]; ok {
		return nil, ErrAlreadyRegisteredReactor
	}

	//TODO protocolInfo management
	pi := newProtocolInfo(byte(len(m.protocolHandlers))+1, 0)
	ph := newProtocolHandler(m, pi, spiList, r, name, priority)
	m.p2p.setCbFunc(pi, ph.onPacket, ph.onFailure, ph.onEvent, p2pEventJoin, p2pEventLeave, p2pEventDuplicate)

	m.protocolHandlers[name] = ph
	m.priority[pi] = priority
	return ph, nil
}

func (m *manager) UnregisterReactor(reactor module.Reactor) error {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	for name, ph := range m.protocolHandlers {
		if ph.reactor == reactor {
			ph.Term()
			m.p2p.unsetCbFunc(ph.protocol)
			delete(m.protocolHandlers, name)
			delete(m.priority, ph.protocol)
			return nil
		}
	}
	return ErrNotRegisteredReactor
}

func (m *manager) hasProtocolHandler(pi protocolInfo) bool {
	defer m.mtx.RUnlock()
	m.mtx.RLock()
	_, ok := m.priority[pi]
	return ok
}

func (m *manager) SetWeight(pi protocolInfo, weight int) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	return m.p2p.sendQueue.SetWeight(int(pi.ID()), weight)
}

func (m *manager) unicast(pi protocolInfo, spi protocolInfo, bytes []byte, id module.PeerID) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.protocol = pi
	pkt.dest = p2pDestPeer
	pkt.ttl = 1
	pkt.destPeer = id
	pkt.priority = m.priority[pi]
	pkt.src = m.PeerID()
	pkt.forceSend = true
	p := m.p2p.getPeer(id, true)
	return p.sendPacket(pkt)
}

func (m *manager) multicast(pi protocolInfo, spi protocolInfo, bytes []byte, role module.Role) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	if _, ok := m.roles[role]; !ok {
		return ErrNotRegisteredRole
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = m.destByRole[role]
	pkt.ttl = 0
	pkt.priority = m.priority[pi]
	return m.p2p.Send(pkt)
}

func (m *manager) broadcast(pi protocolInfo, spi protocolInfo, bytes []byte, bt module.BroadcastType) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = p2pDestAny
	pkt.ttl = byte(bt)
	pkt.priority = m.priority[pi]
	if bt == module.BROADCAST_NEIGHBOR {
		pkt.forceSend = true
	}
	return m.p2p.Send(pkt)
}

func (m *manager) relay(pkt *Packet) error {
	if pkt.ttl == 1 || pkt.dest == p2pDestPeer {
		return errors.New("not allowed relay")
	}
	pkt.priority = m.priority[pkt.protocol]
	return m.p2p.Send(pkt)
}

func (m *manager) _getPeerIDSetByRole(role module.Role) *PeerIDSet {
	s, ok := m.roles[role]
	if !ok {
		s := NewPeerIDSet()
		m.roles[role] = s
		m.destByRole[role] = byte(len(m.roles) + p2pDestPeerGroup)
	}
	return s
}

func (m *manager) SetRole(version int64, role module.Role, peers ...module.PeerID) {
	s := m._getPeerIDSetByRole(role)
	if s.version < version {
		s.version = version
		s.ClearAndAdd(peers...)
	} else {
		m.log.Println("SetRole","ignore",version,"must greater than",s.version)
	}
}

func (m *manager) GetPeersByRole(role module.Role) []module.PeerID {
	s := m._getPeerIDSetByRole(role)
	return s.Array()
}

func (m *manager) AddRole(role module.Role, peers ...module.PeerID) {
	s := m._getPeerIDSetByRole(role)
	s.Merge(peers...)
}

func (m *manager) RemoveRole(role module.Role, peers ...module.PeerID) {
	s := m._getPeerIDSetByRole(role)
	s.Removes(peers...)
}

func (m *manager) HasRole(role module.Role, id module.PeerID) bool {
	s := m._getPeerIDSetByRole(role)
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

func (m *manager) getRoleByDest(dest byte) module.Role {
	return m.roleByDest[dest]
}

type logger struct {
	name     string
	prefix   string
	excludes []string
}

func newLogger(name string, prefix string) *logger {
	//l := log.New(os.Stdout, fmt.Sprintf("[%s] %s", prefix, name), log.LstdFlags)
	l := &logger{name: name, excludes: make([]string, 0)}
	l.SetPrefix(prefix)
	return l
}

func (l *logger) SetPrefix(prefix string) {
	if prefix == "" {
		l.prefix = fmt.Sprintf("%s ", l.name)
	} else {
		l.prefix = fmt.Sprintf("[%s] %s ", prefix, l.name)
	}
}

func (l *logger) printable(v interface{}) bool {
	for _, e := range ExcludeLoggers {
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
		//w := make([]interface{}, len(v)+1)
		//copy(w[1:], v)
		//w[0] = l.prefix
		_ = log.Output(2, l.prefix+fmt.Sprintln(v...))
	}
}

func (l *logger) Printf(format string, v ...interface{}) {
	if l.printable(format) {
		_ = log.Output(2, fmt.Sprintf(l.prefix+format, v...))
	}
}

type protocolInfo uint16

func NewProtocolInfo(v uint16) module.ProtocolInfo {
	return protocolInfo(v)
}
func newProtocolInfo(id byte, version byte) protocolInfo {
	return protocolInfo(int(id)<<8 | int(version))
}
func (pi protocolInfo) ID() byte {
	return byte(pi >> 8)
}
func (pi protocolInfo) Version() byte {
	return byte(pi)
}
func (pi protocolInfo) String() string {
	return fmt.Sprintf("{%#04x}", pi.Uint16())
}
func (pi protocolInfo) Uint16() uint16 {
	return uint16(pi)
}

type Error struct {
	error
	IsTemporary       bool
	Operation         string
	OperationArgument interface{}
}

func (e *Error) Temporary() bool { return e.IsTemporary }

func (e *Error) Unwrap() error { return e.error }

func NewBroadcastError(err error, bt module.BroadcastType) module.NetworkError {
	return newNetworkError(err, "broadcast", bt)
}
func NewMulticastError(err error, role module.Role) module.NetworkError {
	return newNetworkError(err, "multicast", role)
}
func NewUnicastError(err error, id module.PeerID) module.NetworkError {
	return newNetworkError(err, "unicast", id)
}
func newNetworkError(err error, op string, opArg interface{}) module.NetworkError {
	if err != nil {
		isTemporary := false
		if QueueOverflowError.Equals(err) {
			isTemporary = true
		}
		return &Error{err, isTemporary, op, opArg}
	}
	return nil
}
