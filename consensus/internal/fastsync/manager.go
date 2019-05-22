package fastsync

import (
	"math"

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
	Block() module.Block
	Votes() module.CommitVoteSet
	Consume()
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
		prevBlock module.Block,
		f module.CommitVoteSetDecoder,
		cb FetchCallback,
	) (canceler func() bool, err error)
}

type manager struct {
	server *server
	client *client
}

func (m *manager) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	switch pi {
	case protoBlockRequest, protoCancelAllBlockRequests:
		m.server.onReceive(pi, b, id)
	case protoBlockMetadata, protoBlockData:
		m.client.onReceive(pi, b, id)
	}
	return false, nil
}

func (m *manager) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	panic("not implemented")
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
	prev module.Block,
	f module.CommitVoteSetDecoder,
	cb FetchCallback,
) (canceler func() bool, err error) {
	if end < 0 {
		end = math.MaxInt64
	}
	fr, err := m.client.fetchBlocks(begin, end, prev, f, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		return fr.cancel()
	}, nil
}

func NewManager(nm module.NetworkManager, bm module.BlockManager) (Manager, error) {
	return newManager(nm, bm)
}

func newManager(nm module.NetworkManager, bm module.BlockManager) (Manager, error) {
	m := &manager{}
	ph, err := nm.RegisterReactorForStreams("fastsync", m, protocols, configFastSyncPriority)
	if err != nil {
		return nil, err
	}
	m.server = newServer(nm, ph, bm)
	m.client = newClient(nm, ph, bm)
	return m, nil
}
