package sync

import (
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

const (
	configSyncPriority   = 3
	configExpiredTime    = 500  // in millisecond
	configMaxExpiredTime = 1200 // in millisecond
)

var c = codec.MP

type Syncer interface {
	ForceSync() (*Result, error)
	Stop()
	Finalize() error
}

type Platform interface {
	NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot
}

type SyncerImpl interface {
	onReceive(pi module.ProtocolInfo, b []byte, p *peer)
	onJoin(p *peer)
	onLeave(id module.PeerID)
}

type Manager struct {
	log    log.Logger
	pool   *peerPool
	server *server
	client *client
	db     db.Database
	syncer SyncerImpl
	ds     *dataSyncer
	mutex  sync.Mutex
	plt    Platform
}

type Result struct {
	Wss            state.WorldSnapshot
	PatchReceipts  module.ReceiptList
	NormalReceipts module.ReceiptList
}

func (m *Manager) OnReceive(pi module.ProtocolInfo, b []byte,
	id module.PeerID) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.log.Tracef("OnReceive pi(%s), id(%s), syncing(%t)\n", pi, id, m.syncer != nil)
	p := m.pool.getPeer(id)
	if p == nil {
		m.log.Tracef("peer(%s) is not valid\n", id)
		return false, nil
	}
	switch pi {
	case protoHasNode, protoRequestNodeData:
		m.server.onReceive(pi, b, p)
	case protoResult, protoNodeData:
		if m.syncer != nil {
			m.syncer.onReceive(pi, b, p)
		}
	}
	return false, nil
}

func (m *Manager) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	m.log.Tracef("Manager OnFailure err(%+v), pi(%s)\n", err, pi)
}

func (m *Manager) OnJoin(id module.PeerID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.log.Tracef("Manager OnJoin syncing(%t)\n", m.syncer != nil)
	np := &peer{
		id:      id,
		reqID:   0,
		expired: configExpiredTime,
		log:     m.log,
	}
	m.pool.push(np)
	if m.syncer != nil {
		m.syncer.onJoin(np)
	}
}

func (m *Manager) OnLeave(id module.PeerID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.log.Tracef("Manager OnLeave id(%s)\n", id)
	p := m.pool.getPeer(id)
	if p == nil {
		return
	}
	if m.syncer != nil {
		m.syncer.onLeave(id)
	}
	m.pool.remove(id)
}

func (m *Manager) SetSyncHandler(sh SyncerImpl, on bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if on {
		if m.syncer == m.ds {
			m.ds.Deactivate()
		}
		m.syncer = sh
	} else {
		m.syncer = m.ds
		m.ds.Activate(m.pool)
	}
}

func (m *Manager) AddRequest(id db.BucketID, key []byte) error {
	return m.ds.AddRequest(id, key)
}

func (m *Manager) NewSyncer(ah, prh, nrh, vh, ed []byte, noBuffer bool) Syncer {
	return newSyncer(
		m.db, m.client, m.pool, m.plt,
		ah, prh, nrh, vh, ed, m.log,
		noBuffer,
		m.SetSyncHandler)
}

func (m *Manager) Term() {
	m.ds.Term()
}

func (m *Manager) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.syncer = m.ds
	m.ds.Start()
}

func NewSyncManager(db db.Database, nm module.NetworkManager, plt Platform, logger log.Logger) *Manager {
	logger = logger.WithFields(log.Fields{log.FieldKeyModule: "statesync"})
	logger.Debugln("NewSyncManager")
	m := new(Manager)
	ph, err := nm.RegisterReactorForStreams("statesync", module.ProtoStateSync, m, protocol, configSyncPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		logger.Panicf("Failed to register reactor for stateSync\n")
		return nil
	}
	m.db = db
	m.plt = plt
	m.log = logger

	server := newServer(db, ph, logger)
	m.server = server

	client := newClient(ph, logger)
	m.client = client
	m.pool = newPeerPool()

	m.ds = newDataSyncer(db, client, logger)
	return m
}
