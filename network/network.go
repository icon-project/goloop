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
	protocolHandlers map[uint16]*protocolHandler

	mtx sync.RWMutex

	pd *PeerDispatcher
	cn *ChannelNegotiator
	//log
	logger log.Logger

	//monitor
	mtr *metric.NetworkMetric

	streamReactors []*streamReactor
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
		protocolHandlers: make(map[uint16]*protocolHandler),
		pd:               t.pd,
		cn:               t.cn,
		logger:           networkLogger,
		mtr:              mtr,
	}
	for _, pi := range m.p2p.supportedProtocols() {
		m.cn.addProtocol(m.channel, pi)
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

	m.p2p.setConnectionLimit(p2pConnTypeChildren, c.ChildrenLimit())
	m.p2p.setConnectionLimit(p2pConnTypeNephew, c.NephewsLimit())

	m.logger.Debugln("NewManager", channel)
	return m
}

func (m *manager) Channel() string {
	return m.channel
}
func (m *manager) PeerID() module.PeerID {
	return m.p2p.ID()
}

func toPeerIDs(ps []*Peer) []module.PeerID {
	l := make([]module.PeerID, len(ps))
	for i, p := range ps {
		l[i] = p.ID()
	}
	return l
}

func (m *manager) GetPeers() []module.PeerID {
	return toPeerIDs(m.p2p.getPeers(true))
}

func (m *manager) getPeersByProtocol(pi module.ProtocolInfo) []module.PeerID {
	return toPeerIDs(m.p2p.getPeersByProtocol(pi, true))
}

func (m *manager) Term() {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	_ = m._stop()
	m.logger.Debugln("Term protocolHandlers")
	for _, ph := range m.protocolHandlers {
		m.logger.Debugln("Term", ph.name)
		ph.Term()
		m.cn.removeProtocol(m.channel, ph.protocol)
	}

	for _, pi := range m.p2p.supportedProtocols() {
		m.cn.removeProtocol(m.channel, pi)
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

func (m *manager) RegisterReactor(
	name string,
	pi module.ProtocolInfo,
	reactor module.Reactor,
	piList []module.ProtocolInfo,
	priority uint8,
	policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if priority < 1 || priority > DefaultSendQueueMaxPriority {
		log.Panicf("priority must be positive value and less than %d", DefaultSendQueueMaxPriority)
	}

	k := pi.Uint16()
	ph, ok := m.protocolHandlers[k]
	if ok {
		if ph.getName() != name || ph.getPriority() != priority || ph.getPolicy() != policy {
			return nil, errors.WithStack(ErrIllegalArgument)
		}
		spis := ph.getSubProtocols()
		if len(spis) != len(piList) {
			return nil, errors.WithStack(ErrIllegalArgument)
		}
		for _, subProtocol := range piList {
			has := false
			for _, spi := range spis {
				if subProtocol.Uint16() == spi.Uint16() {
					has = true
					break
				}
			}
			if !has {
				return nil, errors.WithStack(ErrIllegalArgument)
			}
		}
		ph.setReactor(reactor)
	} else {
		if m.p2p.IsStarted() {
			m.logger.Debugln("RegisterReactor, p2p started")
		}

		ph = newProtocolHandler(m, pi, piList, reactor, name, priority, policy, m.logger)
		m.p2p.setCbFunc(pi, ph.onPacket, ph.onEvent, p2pEventJoin, p2pEventLeave, p2pEventDuplicate)
		m.protocolHandlers[k] = ph
		m.cn.addProtocol(m.channel, pi)
	}
	return ph, nil
}

func (m *manager) UnregisterReactor(reactor module.Reactor) error {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if sr := m.tryUnregisterStreamReactor(reactor); sr != nil {
		reactor = sr
	}

	for k, ph := range m.protocolHandlers {
		if ph.reactor == reactor {
			ph.Term()
			m.p2p.unsetCbFunc(ph.protocol)
			delete(m.protocolHandlers, k)
			m.cn.removeProtocol(m.channel, module.ProtocolInfo(k))
			return nil
		}
	}
	return ErrNotRegisteredReactor
}

func (m *manager) getProtocolHandler(pi module.ProtocolInfo) (*protocolHandler, bool) {
	defer m.mtx.RUnlock()
	m.mtx.RLock()

	ph, ok := m.protocolHandlers[pi.Uint16()]
	return ph, ok
}

func (m *manager) SetWeight(pi module.ProtocolInfo, weight int) error {
	if _, ok := m.getProtocolHandler(pi); !ok {
		return ErrNotRegisteredReactor
	}
	return m.p2p.sendQueue.SetWeight(int(pi.ID()), weight)
}

func (m *manager) unicast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, id module.PeerID) error {
	ph, ok := m.getProtocolHandler(pi)
	if !ok {
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
	pkt.priority = ph.getPriority()
	pkt.src = m.PeerID()
	pkt.forceSend = true
	p := m.p2p.getPeerByProtocol(id, pkt.protocol, true)
	return p.sendPacket(pkt)
}

func (m *manager) multicast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, role module.Role) error {
	ph, ok := m.getProtocolHandler(pi)
	if !ok {
		return ErrNotRegisteredReactor
	}
	if _, ok = m.roles[role]; !ok {
		return ErrNotRegisteredRole
	}
	if DefaultPacketPayloadMax < len(bytes) {
		return ErrIllegalArgument
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = m.destByRole[role]
	pkt.ttl = 0
	pkt.priority = ph.getPriority()
	return m.p2p.Send(pkt)
}

func (m *manager) broadcast(pi module.ProtocolInfo, spi module.ProtocolInfo, bytes []byte, bt module.BroadcastType) error {
	ph, ok := m.getProtocolHandler(pi)
	if !ok {
		return ErrNotRegisteredReactor
	}
	if DefaultPacketPayloadMax < len(bytes) {
		return ErrIllegalArgument
	}
	pkt := NewPacket(pi, spi, bytes)
	pkt.dest = p2pDestAny
	pkt.ttl = bt.TTL()
	pkt.priority = ph.getPriority()
	pkt.forceSend = bt.ForceSend()
	return m.p2p.Send(pkt)
}

func (m *manager) relay(pkt *Packet) error {
	if pkt.ttl != 0 || pkt.dest == p2pDestPeer {
		return errors.New("not allowed relay")
	}
	ph, ok := m.getProtocolHandler(pkt.protocol)
	if !ok {
		return ErrNotRegisteredReactor
	}
	pkt.priority = ph.getPriority()
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
	m.p2p.trustSeeds.Clear()
	ss := strings.Split(seeds, ",")
	for _, s := range ss {
		if na := NetAddress(s); len(na) != 0 && na != m.p2p.NetAddress() {
			m.p2p.trustSeeds.Add(NetAddress(s))
		}
	}
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

func ChannelOfNetID(id int) string {
	return strconv.FormatInt(int64(id), 16)
}
