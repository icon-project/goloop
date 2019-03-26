package fastsync

import (
	"io"
	"runtime"
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/pkg/errors"
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
	sync.Mutex
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
		wakeUpCh:              make(chan struct{}),
	}
	go nm.process()
	runtime.SetFinalizer(nm, (*tNetworkManager).dispose)
	return nm
}

func (nm *tNetworkManager) dispose() {
	nm.Lock()
	defer nm.Unlock()
	close(nm.wakeUpCh)
}

func newTNetworkManager() *tNetworkManager {
	return newTNetworkManagerForPeerID(createAPeerID())
}

func (nm *tNetworkManager) GetPeers() []module.PeerID {
	res := make([]module.PeerID, len(nm.peers))
	for i := range nm.peers {
		res[i] = nm.peers[i].id
	}
	return res
}

func (nm *tNetworkManager) RegisterReactor(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
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
	nm.peers = append(nm.peers, nm2)
	nm2.peers = append(nm2.peers, nm)
	for _, r := range nm.reactorItems {
		r.reactor.OnJoin(nm2.id)
	}
	for _, r := range nm2.reactorItems {
		r.reactor.OnJoin(nm.id)
	}
}

func (nm *tNetworkManager) onReceiveUnicast(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	nm.recvBuf = append(nm.recvBuf, &tPacket{pi, b, from})
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
	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.id.Equal(id) {
			p.onReceiveUnicast(pi, b, ph.nm.id)
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
