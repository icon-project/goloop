package sync2

import (
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

const (
	configSyncPriority      = 3
	configExpiredTime       = 500         // in millisecond
	configMaxExpiredTime    = 1200        // in millisecond
	configMigrationInterval = time.Second // second
)

var (
	c         = codec.MP
	afterFunc = time.AfterFunc
)

type RequestCallback func(ver byte, dataLen int, id module.PeerID)

type Syncer interface {
	ForceSync() (*Result, error)
	Stop()
	Finalize() error
}

type PeerWatcher interface {
	OnPeerJoin(p *peer)
	OnPeerLeave(p *peer)
}

type SyncReactor interface {
	GetVersion() byte
	WatchPeers(watcher PeerWatcher) []*peer
}

type Platform interface {
	NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot
}

type Manager struct {
	logger log.Logger

	db       db.Database
	plt      Platform
	ds       *dataSyncer
	reactors []SyncReactor
}

type Result struct {
	Wss            state.WorldSnapshot
	PatchReceipts  module.ReceiptList
	NormalReceipts module.ReceiptList
	BTPDigest      module.BTPDigest
}

func (m *Manager) NewSyncer(ah, prh, nrh, vh, ed, bh []byte, noBuffer bool) Syncer {
	return newSyncerWithHashes(
		m.db, m.reactors, m.plt, ah, prh, nrh, vh, ed, bh, m.logger, noBuffer)
}

func (m *Manager) AddRequest(id db.BucketID, key []byte) error {
	return m.ds.AddRequest(id, key)
}

func (m *Manager) UnresolvedRequestCount() int {
	return m.ds.UnresolvedCount()
}

func (m *Manager) Start() {
	m.ds.Start()
}

func (m *Manager) Term() {
	m.ds.Term()
}

func NewSyncManager(database db.Database, nm module.NetworkManager, plt Platform, logger log.Logger) *Manager {
	logger = logger.WithFields(log.Fields{log.FieldKeyModule: "statesync2"})
	m := new(Manager)

	reactorV1 := newReactorV1(database, logger)
	ph, err := nm.RegisterReactorForStreams("statesync", module.ProtoStateSync, reactorV1, protocol, configSyncPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		logger.Panicf("Failed to register reactorV1 for stateSync")
		return nil
	}
	reactorV1.ph = ph
	m.reactors = append(m.reactors, reactorV1)

	reactorV2 := newReactorV2(database, logger)
	pi2 := module.NewProtocolInfo(module.ProtoStateSync.ID(), 1)
	ph2, err := nm.RegisterReactorForStreams("statesync2", pi2, reactorV2, protocolv2, configSyncPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		logger.Panicf("Failed to register reactorV2 for stateSync2")
		return nil
	}
	reactorV2.ph = ph2
	m.reactors = append(m.reactors, reactorV2)

	m.db = database
	m.plt = plt
	m.logger = logger

	m.ds = newDataSyncer(m.db, m.reactors, logger)
	return m
}
