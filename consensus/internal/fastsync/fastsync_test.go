package fastsync

import (
	"bytes"
	"io"
	"testing"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
)

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

func (b *tBlock) Marshal(w io.Writer) error {
	if err := b.MarshalHeader(w); err!=nil {
		return err
	}
	return b.MarshalBody(w)
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
