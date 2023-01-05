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

type transportForManager interface {
	GetDialer(channel string) *Dialer
	addProtocol(channel string, pi module.ProtocolInfo)
	removeProtocol(channel string, pi module.ProtocolInfo)
	registerPeerHandler(channel string, ph PeerHandler, mtr *metric.NetworkMetric) bool
	unregisterPeerHandler(channel string)
}

type manager struct {
	channel string
	p2p     *PeerToPeer
	//
	protocolHandlers map[uint16]*protocolHandler

	mtx sync.RWMutex

	t transportForManager
	//log
	logger log.Logger

	//monitor
	mtr *metric.NetworkMetric

	streamReactors []*streamReactor
}

func NewManager(c module.Chain, nt module.NetworkTransport, trustSeeds string, roles ...module.Role) module.NetworkManager {
	m := &manager{
		channel:          ChannelOfNetID(c.NetID()),
		protocolHandlers: make(map[uint16]*protocolHandler),
		t:                nt.(transportForManager),
		logger:           c.Logger().WithFields(log.Fields{log.FieldKeyModule: "NM"}),
		mtr:              metric.NewNetworkMetric(c.MetricContext()),
	}
	m.p2p = newPeerToPeer(
		m.channel,
		&Peer{id: nt.PeerID(), netAddress: NetAddress(nt.Address())},
		m.t.GetDialer(m.channel),
		m.mtr,
		m.logger)

	m.SetInitialRoles(roles...)
	m.SetTrustSeeds(trustSeeds)

	m.p2p.setConnectionLimit(p2pConnTypeChildren, c.ChildrenLimit())
	m.p2p.setConnectionLimit(p2pConnTypeNephew, c.NephewsLimit())

	m.logger.Infof("NetworkManager use channel=%s for cid=%#x nid=%#x",
		m.channel, c.CID(), c.NID())
	return m
}

func toPeerIDs(ps []*Peer) []module.PeerID {
	l := make([]module.PeerID, len(ps))
	for i, p := range ps {
		l[i] = p.ID()
	}
	return l
}

func (m *manager) GetPeers() []module.PeerID {
	return toPeerIDs(m.p2p.getPeers())
}

func (m *manager) getPeersByProtocol(pi module.ProtocolInfo) []module.PeerID {
	return toPeerIDs(m.p2p.getPeersByProtocol(pi))
}

func (m *manager) Term() {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if m.p2p.IsStarted() {
		m.t.unregisterPeerHandler(m.channel)
		for _, pi := range m.p2p.supportedProtocols() {
			m.t.addProtocol(m.channel, pi)
		}
		m.p2p.Stop()
	}
	m.logger.Debugln("Term protocolHandlers")
	for k, ph := range m.protocolHandlers {
		m.logger.Debugln("Term", ph.getName())
		ph.Term()
		m.t.removeProtocol(m.channel, module.ProtocolInfo(k))
	}
}

func (m *manager) Start() error {
	defer m.mtx.Unlock()
	m.mtx.Lock()

	if m.p2p.IsStarted() {
		return nil
	}
	if !m.t.registerPeerHandler(m.channel, m.p2p, m.mtr) {
		return errors.InvalidNetworkError.Errorf("P2PChannelConflict(channel=%s)", m.channel)
	}
	for _, pi := range m.p2p.supportedProtocols() {
		m.t.addProtocol(m.channel, pi)
	}
	m.p2p.Start()
	return nil
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
		for _, spi := range spis {
			has := false
			for _, subProtocol := range piList {
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
		m.t.addProtocol(m.channel, pi)
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
		if ph.getReactor() == reactor {
			ph.Term()
			m.p2p.unsetCbFunc(module.ProtocolInfo(k))
			delete(m.protocolHandlers, k)
			m.t.removeProtocol(m.channel, module.ProtocolInfo(k))
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

func (m *manager) send(pkt *Packet) error {
	ph, ok := m.getProtocolHandler(pkt.protocol)
	if !ok {
		return ErrNotRegisteredReactor
	}
	pkt.priority = ph.getPriority()
	return m.p2p.Send(pkt)
}

func (m *manager) SetRole(version int64, role module.Role, peers ...module.PeerID) {
	s := m.p2p.getAllowed(role)
	if s.version < version {
		s.version = version
		s.ClearAndAdd(peers...)
	} else {
		m.logger.Debugln("SetRole", "ignore", version, "must greater than", s.version)
	}
}

func (m *manager) GetPeersByRole(role module.Role) []module.PeerID {
	return m.p2p.getAllowed(role).Array()
}

func (m *manager) AddRole(role module.Role, peers ...module.PeerID) {
	m.p2p.getAllowed(role).Merge(peers...)
}

func (m *manager) RemoveRole(role module.Role, peers ...module.PeerID) {
	m.p2p.getAllowed(role).Removes(peers...)
}

func (m *manager) HasRole(role module.Role, id module.PeerID) bool {
	return m.p2p.getAllowed(role).Contains(id)
}

func (m *manager) Roles(id module.PeerID) []module.Role {
	var roles []module.Role
	for r := module.ROLE_NORMAL; r < module.ROLE_RESERVED; r++ {
		if m.p2p.getAllowed(r).Contains(id) {
			roles = append(roles, r)
		}
	}
	return roles
}

func (m *manager) SetTrustSeeds(seeds string) {
	var ss []NetAddress
	for _, s := range strings.Split(seeds, ",") {
		ss = append(ss, NetAddress(s))
	}
	m.p2p.setTrustSeeds(ss)
}

func (m *manager) SetInitialRoles(roles ...module.Role) {
	m.p2p.setRole(NewPeerRoleFlag(roles...))
}

func ChannelOfNetID(id int) string {
	return strconv.FormatInt(int64(id), 16)
}
