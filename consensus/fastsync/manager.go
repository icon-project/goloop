package fastsync

import (
	"math"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	configFastSyncPriority = 4
)

type ServerCallback interface {
	OnBeginServing(id module.PeerID)
	OnEndServing(id module.PeerID)
}

type BlockResult interface {
	Block() module.BlockData
	Votes() []byte
	Consume()
	Reject()
}

type FetchCallback interface {
	OnBlock(br BlockResult)
	OnEnd(err error)
}

type Manager interface {
	StartServer()
	StopServer()
	FetchBlocks(
		begin int64,
		end int64,
		cb FetchCallback,
	) (canceler func() bool, err error)
	Term()
}

type manager struct {
	nm     module.NetworkManager
	server *server
	client *client
}

func (m *manager) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	switch pi {
	case ProtoBlockRequest, ProtoCancelAllBlockRequests:
		m.server.onReceive(pi, b, id)
	case ProtoBlockMetadata, ProtoBlockData:
		m.client.onReceive(pi, b, id)
	}
	return false, nil
}

func (m *manager) OnJoin(id module.PeerID) {
	m.server.onJoin(id)
	m.client.onJoin(id)
}

func (m *manager) OnLeave(id module.PeerID) {
	m.server.onLeave(id)
	m.client.onLeave(id)
}

func (m *manager) StartServer() {
	m.server.start()
}

func (m *manager) StopServer() {
	m.server.stop()
}

func (m *manager) FetchBlocks(
	begin int64,
	end int64,
	cb FetchCallback,
) (canceler func() bool, err error) {
	if end < 0 {
		end = math.MaxInt64
	}
	fr, err := m.client.fetchBlocks(begin, end, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		return fr.cancel()
	}, nil
}

func (m *manager) Term() {
	if m.nm != nil {
		err := m.nm.UnregisterReactor(m)
		if err != nil {
			log.Warnf("fastsync.manager.Term: error=%+v", err)
		}
		m.nm = nil
	}
}

func NewManager(
	nm module.NetworkManager,
	bm module.BlockManager,
	bpp BlockProofProvider,
	logger log.Logger,
	maxBlockBytes int,
) (Manager, error) {
	m := &manager{
		nm: nm,
	}
	m.server = newServer(nm, nil, bm, bpp, logger)
	m.client = newClient(nm, nil, bm, logger, maxBlockBytes)

	// lock to prevent enter server.onJoin / client.onJoin
	m.server.Lock()
	defer m.server.Unlock()
	m.client.Lock()
	defer m.client.Unlock()
	ph, err := nm.RegisterReactorForStreams("fastsync", module.ProtoFastSync, m, protocols, configFastSyncPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		return nil, err
	}
	m.server.ph = ph
	m.client.ph = ph
	return m, nil
}

func NewManagerOnlyForClient(
	nm module.NetworkManager,
	bdf module.BlockDataFactory,
	logger log.Logger,
	maxBlockBytes int,
) (Manager, error) {
	m := &manager{
		nm: nm,
	}
	m.server = newServer(nm, nil, nil, nil, logger)
	m.client = newClient(nm, nil, bdf, logger, maxBlockBytes)

	// lock to prevent enter server.onJoin / client.onJoin
	m.server.Lock()
	defer m.server.Unlock()
	m.client.Lock()
	defer m.client.Unlock()
	ph, err := nm.RegisterReactorForStreams("fastsync", module.ProtoFastSync, m, protocols, configFastSyncPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		return nil, err
	}
	m.server.ph = ph
	m.client.ph = ph
	return m, nil
}

type BlockProofProvider interface {
	GetBlockProof(h int64, opt int32) (proof []byte, err error)
}
