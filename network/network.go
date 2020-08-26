package network

import (
	"strconv"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
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
	priority         map[module.ProtocolInfo]uint8

	mtx sync.RWMutex

	pd *PeerDispatcher
	//log
	logger log.Logger

	//monitor
	mtr *metric.NetworkMetric
}

func NewManager(c module.Chain, nt module.NetworkTransport, trustSeeds string, roles ...module.Role) module.NetworkManager {
	t := nt.(*transport)
	self := &Peer{id: t.PeerID(), netAddress: NetAddress(t.Address())}
	channel := ChannelOfNetID(c.NetID())
	mtr := metric.NewNetworkMetric(c.MetricContext())
	networkLogger := c.Logger().WithFields(log.Fields{log.FieldKeyModule: "NM"})
	networkLogger.Infof("NetworkManager use channel=%s for cid=%#x nid=%#x", channel, c.CID(), c.NID())
	m := &manager{
		channel:          channel,
		p2p:              newPeerToPeer(channel, self, t.GetDialer(channel), mtr, networkLogger),
		roles:            make(map[module.Role]*PeerIDSet),
		destByRole:       make(map[module.Role]byte),
		roleByDest:       make(map[byte]module.Role),
		protocolHandlers: make(map[string]*protocolHandler),
		priority:         make(map[module.ProtocolInfo]uint8),
		pd:               t.pd,
		logger:           networkLogger,
		mtr:              mtr,
	}

	//Create default protocolHandler for P2P topology management
	m.roles[module.ROLE_SEED] = m.p2p.allowedSeeds
	m.roles[module.ROLE_VALIDATOR] = m.p2p.allowedRoots
	m.roles[module.ROLE_NORMAL] = m.p2p.allowedPeers
	m.destByRole[module.ROLE_SEED] = p2pRoleSeed
	m.destByRole[module.ROLE_VALIDATOR] = p2pRoleRoot
	m.destByRole[module.ROLE_NORMAL] = p2pRoleNone //same as broadcast

	m.SetInitialRoles(roles...)
	m.SetTrustSeeds(trustSeeds)

	m.logger.Debugln("NewManager", channel)
	return m
}

func (m *manager) Channel() string {
	return m.channel
}
func (m *manager) PeerID() module.PeerID {
	return m.p2p.getID()
}

func (m *manager) GetPeers() []module.PeerID {
	arr := m.p2p.getPeers(true)
	l := make([]module.PeerID, len(arr))
	for i, p := range arr {
		l[i] = p.id
	}
	return l
}

func (m *manager) Term() {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	_ = m._stop()
	m.logger.Debugln("Term protocolHandlers")
	for _, ph := range m.protocolHandlers {
		m.logger.Debugln("Term", ph.name)
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
		return errors.InvalidNetworkError.Errorf("P2PChannelConflict(channel=%s)", m.channel)
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

func (m *manager) RegisterReactor(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if priority < 1 || priority > DefaultSendQueueMaxPriority {
		log.Panicf("priority must be positive value and less than %d", DefaultSendQueueMaxPriority)
	}

	if _, ok := m.protocolHandlers[name]; ok {
		return nil, ErrAlreadyRegisteredReactor
	}

	if _, ok := m.priority[pi]; ok {
		return nil, ErrAlreadyRegisteredReactor
	}

	ph := newProtocolHandler(m, pi, piList, reactor, name, priority, m.logger)
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

func (m *manager) hasProtocolHandler(pi module.ProtocolInfo) bool {
	defer m.mtx.RUnlock()
	m.mtx.RLock()
	_, ok := m.priority[pi]
	return ok
}

func (m *manager) SetWeight(pi module.ProtocolInfo, weight int) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	return m.p2p.sendQueue.SetWeight(int(pi.ID()), weight)
}

func (m *manager) unicast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, id module.PeerID) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	if DefaultPacketPayloadMax < len(bytes) {
		return ErrIllegalArgument
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

func (m *manager) multicast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, role module.Role) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	if _, ok := m.roles[role]; !ok {
		return ErrNotRegisteredRole
	}
	if DefaultPacketPayloadMax < len(bytes) {
		return ErrIllegalArgument
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = m.destByRole[role]
	pkt.ttl = 0
	pkt.priority = m.priority[pi]
	return m.p2p.Send(pkt)
}

func (m *manager) broadcast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, bt module.BroadcastType) error {
	if !m.hasProtocolHandler(pi) {
		return ErrNotRegisteredReactor
	}
	if DefaultPacketPayloadMax < len(bytes) {
		return ErrIllegalArgument
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = p2pDestAny
	pkt.ttl = bt.TTL()
	pkt.priority = m.priority[pi]
	pkt.forceSend = bt.ForceSend()
	return m.p2p.Send(pkt)
}

func (m *manager) relay(pkt *Packet) error {
	if pkt.ttl != 0 || pkt.dest == p2pDestPeer {
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
		m.logger.Debugln("SetRole", "ignore", version, "must greater than", s.version)
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

func (m *manager) SetTrustSeeds(seeds string) {
	ss := strings.Split(seeds, ",")
	nas := make([]NetAddress, 0)
	for _, s := range ss {
		if s != "" {
			na := NetAddress(s)
			if na != m.p2p.getNetAddress() {
				nas = append(nas, na)
			}
		}
	}
	m.p2p.trustSeeds.ClearAndAdd(nas...)
}

func (m *manager) SetInitialRoles(roles ...module.Role) {
	role := PeerRoleFlag(p2pRoleNone)
	for _, r := range roles {
		switch r {
		case module.ROLE_SEED:
			role.SetFlag(p2pRoleSeed)
		case module.ROLE_VALIDATOR:
			role.SetFlag(p2pRoleRoot)
		default:
			m.logger.Infoln("SetRoles", "ignored role", r)
		}
	}
	m.p2p.setRole(role)
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

func ChannelOfNetID(id int) string {
	return strconv.FormatInt(int64(id), 16)
}
