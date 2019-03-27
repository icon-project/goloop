package fastsync

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tReactorItem struct {
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

type tPacket struct {
	pi module.ProtocolInfo
	b  []byte
	id module.PeerID
}

type tNetworkManagerStatic struct {
	common.Mutex
	procList []*tNetworkManager
}

var nms = tNetworkManagerStatic{}

type tNetworkManager struct {
	*tNetworkManagerStatic
	id       module.PeerID
	wakeUpCh chan struct{}

	reactorItems []*tReactorItem
	peers        []*tNetworkManager
	drop         bool
	recvBuf      []*tPacket
}

type tProtocolHandler struct {
	nm *tNetworkManager
	ri *tReactorItem
}

func newTNetworkManagerForPeerID(id module.PeerID) *tNetworkManager {
	nm := &tNetworkManager{
		tNetworkManagerStatic: &nms,
		id:                    id,
		wakeUpCh:              make(chan struct{}, 1),
	}
	go nm.process()
	runtime.SetFinalizer(nm, (*tNetworkManager).dispose)
	return nm
}

func (nm *tNetworkManager) dispose() {
	close(nm.wakeUpCh)
}

func newTNetworkManager() *tNetworkManager {
	return newTNetworkManagerForPeerID(createAPeerID())
}

func (nm *tNetworkManager) GetPeers() []module.PeerID {
	nm.Lock()
	defer nm.Unlock()

	res := make([]module.PeerID, len(nm.peers))
	for i := range nm.peers {
		res[i] = nm.peers[i].id
	}
	return res
}

func (nm *tNetworkManager) RegisterReactor(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	nm.Lock()
	defer nm.Unlock()

	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *tNetworkManager) RegisterReactorForStreams(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	return nm.RegisterReactor(name, reactor, piList, priority)
}

func (nm *tNetworkManager) join(nm2 *tNetworkManager) {
	nm.Lock()
	defer nm.Unlock()

	nm.peers = append(nm.peers, nm2)
	nm2.peers = append(nm2.peers, nm)
	ri := make([]*tReactorItem, len(nm.reactorItems))
	copy(ri, nm.reactorItems)
	id2 := nm2.id
	ri2 := make([]*tReactorItem, len(nm2.reactorItems))
	copy(ri2, nm.reactorItems)
	id := nm.id
	nm.CallAfterUnlock(func() {
		for _, r := range ri {
			r.reactor.OnJoin(id2)
		}
		for _, r := range ri2 {
			r.reactor.OnJoin(id)
		}
	})
}

func (nm *tNetworkManager) onReceiveUnicast(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	nm.Lock()
	nm.recvBuf = append(nm.recvBuf, &tPacket{pi, b, from})
	nm.Unlock()
	select {
	case nm.wakeUpCh <- struct{}{}:
	default:
	}
}

func (nm *tNetworkManager) process() {
	for {
		select {
		case _, more := <-nm.wakeUpCh:
			if !more {
				return
			}
		}
		nm.Lock()
		recvBuf := nm.recvBuf
		nm.recvBuf = nil
		reactorItems := make([]*tReactorItem, len(nm.reactorItems))
		copy(reactorItems, nm.reactorItems)
		nm.Unlock()
		for _, p := range recvBuf {
			for _, r := range reactorItems {
				r.reactor.OnReceive(p.pi, p.b, p.id)
			}
		}
	}
}

func (ph *tProtocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	ph.nm.Lock()
	defer ph.nm.Unlock()

	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.id.Equal(id) {
			peer := p
			id := ph.nm.id
			ph.nm.CallAfterUnlock(func() {
				peer.onReceiveUnicast(pi, b, id)
			})
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func createAPeerID() module.PeerID {
	return network.NewPeerIDFromAddress(wallet.New().Address())
}

type tBlockHeader struct {
	Height int64
	ID     []byte
	Prev   []byte
}

type tBlockBody struct {
	B []byte
}

type tBlock struct {
	tBlockHeader
	tBlockBody
}

func newTBlock(height int64, id []byte, prev []byte, b []byte) module.Block {
	blk := &tBlock{
		tBlockHeader: tBlockHeader{
			Height: height,
			ID:     id,
			Prev:   prev,
		},
		tBlockBody: tBlockBody{
			B: b,
		},
	}
	return blk
}

func (b *tBlock) Version() int {
	panic("not implemented")
}

func (b *tBlock) ID() []byte {
	return b.tBlockHeader.ID
}

func (b *tBlock) Height() int64 {
	return b.tBlockHeader.Height
}

func (b *tBlock) PrevID() []byte {
	return b.Prev
}

func (b *tBlock) NextValidators() module.ValidatorList {
	return nil
}

func (b *tBlock) Votes() module.CommitVoteSet {
	return &tCommitVoteSet{b.Prev}
}

func (b *tBlock) NormalTransactions() module.TransactionList {
	panic("not implemented")
}

func (b *tBlock) PatchTransactions() module.TransactionList {
	panic("not implemented")
}

func (b *tBlock) Timestamp() int64 {
	panic("not implemented")
}

func (b *tBlock) Proposer() module.Address {
	panic("not implemented")
}

func (b *tBlock) LogBloom() module.LogBloom {
	panic("not implemented")
}

func (b *tBlock) Result() []byte {
	panic("not implemented")
}

func (b *tBlock) MarshalHeader(w io.Writer) error {
	bs := codec.MustMarshalToBytes(&b.tBlockHeader)
	_, err := w.Write(bs)
	return err
}

func (b *tBlock) MarshalBody(w io.Writer) error {
	_, err := w.Write(codec.MustMarshalToBytes(&b.tBlockBody))
	return err
}

func (b *tBlock) ToJSON(rcpVersion int) (interface{}, error) {
	panic("not implemented")
}

type tCommitVoteSet struct {
	b []byte
}

func newTCommitVoteSet(b []byte) module.CommitVoteSet {
	return &tCommitVoteSet{b: b}
}

func (vs *tCommitVoteSet) Verify(block module.Block, validators module.ValidatorList) error {
	return nil
}

func (vs *tCommitVoteSet) Bytes() []byte {
	return vs.b
}

func (vs *tCommitVoteSet) Hash() []byte {
	panic("not implemented")
}

type tBlockManager struct {
	bmap map[int64]module.Block
}

func newTBlockManager() *tBlockManager {
	bm := &tBlockManager{
		bmap: make(map[int64]module.Block),
	}
	return bm
}

func (bm *tBlockManager) SetBlock(height int64, blk module.Block) {
	bm.bmap[height] = blk
}

func (bm *tBlockManager) GetBlockByHeight(height int64) (module.Block, error) {
	blk := bm.bmap[height]
	if blk == nil {
		// TODO
		return nil, errors.New("NoBlock")
	}
	return blk, nil
}

func (bm *tBlockManager) NewBlockFromReader(r io.Reader) (module.Block, error) {
	var bh tBlockHeader
	err := codec.Unmarshal(r, &bh)
	if err != nil {
		return nil, err
	}
	var bb tBlockBody
	err = codec.Unmarshal(r, &bb)
	if err != nil {
		return nil, err
	}
	return &tBlock{
		tBlockHeader: bh,
		tBlockBody:   bb,
	}, nil
}

type tReactor struct {
	ch chan interface{}
}

type tReceiveEvent struct {
	pi module.ProtocolInfo
	b  []byte
	//msg interface{}
	id module.PeerID
}

type tFailureEvent struct {
	err error
	pi  module.ProtocolInfo
	b   []byte
}

type tJoinEvent struct {
	id module.PeerID
}

type tLeaveEvent struct {
	id module.PeerID
}

func newTReactor() *tReactor {
	return &tReactor{ch: make(chan interface{}, 5)}
}

func (r *tReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	r.ch <- tReceiveEvent{pi, b, id}
	return false, nil
}

func (r *tReactor) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	r.ch <- tFailureEvent{err, pi, b}
}

func (r *tReactor) OnJoin(id module.PeerID) {
	r.ch <- tJoinEvent{id}
}

func (r *tReactor) OnLeave(id module.PeerID) {
	r.ch <- tLeaveEvent{id}
}

type fastSyncTestSetUp struct {
	t         *testing.T
	bm        *tBlockManager
	votes     [][]byte
	rawBlocks [][]byte
	blks      []module.Block
}

func newFastSyncTestSetUp(t *testing.T) *fastSyncTestSetUp {
	s := &fastSyncTestSetUp{}
	s.t = t
	s.bm = newTBlockManager()
	s.votes = make([][]byte, tNumBlocks)
	s.rawBlocks = make([][]byte, tNumBlocks)
	s.blks = make([]module.Block, tNumBlocks)
	for i := 0; i < tNumBlocks; i++ {
		var b []byte
		if i < tNumLongBlocks {
			b = createABytes(configChunkSize * 10)
		} else {
			b = createABytes(2)
		}
		if i > 0 {
			s.blks[i] = newTBlock(int64(i), b[:1], s.blks[i-1].ID(), b[1:])
			s.votes[i] = s.blks[i-1].ID()
		} else {
			s.blks[i] = newTBlock(int64(i), b[:1], nil, b[1:])
			s.votes[i] = nil
		}
		buf := bytes.NewBuffer(nil)
		err := s.blks[i].MarshalHeader(buf)
		assert.Nil(s.t, err)
		err = s.blks[i].MarshalBody(buf)
		assert.Nil(s.t, err)
		s.rawBlocks[i] = buf.Bytes()
		s.bm.SetBlock(int64(i), s.blks[i])
	}
	return s
}
