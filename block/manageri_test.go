package block

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func assertHasValidGenesisBlock(t *testing.T, bm module.BlockManager) {
	blk, err := bm.GetLastBlock()
	assert.Nil(t, err, "error")
	assert.Equal(t, gheight, blk.Height(), "height")
	id := blk.ID()

	blk, err = bm.GetBlockByHeight(gheight)
	assert.Nil(t, err, "error")
	assert.Equal(t, gheight, blk.Height(), "height")
	assert.Equal(t, id, blk.ID(), "ID")

	blk, err = bm.GetBlock(id)
	assert.Nil(t, err, "error")
	assert.Equal(t, gheight, blk.Height(), "height")
	assert.Equal(t, id, blk.ID(), "ID")
}

type blockGenerator struct {
	t  *testing.T
	sm *testServiceManager
	bm module.BlockManager
}

func newBlockGenerator(t *testing.T, gtx *testTransaction) *blockGenerator {
	bg := &blockGenerator{}
	bg.t = t
	c := newTestChain(newMapDB(), gtx)
	bg.sm = newTestServiceManager(c.Database())
	var err error
	bg.bm, err = NewManager(c, nil, nil)
	assert.Nil(t, err)
	return bg
}

func (bg *blockGenerator) getBlock(n int64) module.Block {
	bg.generateUntil(n)
	blk, err := bg.bm.GetBlockByHeight(n)
	assert.Nil(bg.t, err, "GetBlockByHeight")
	return blk
}

func (bg *blockGenerator) getReaderForBlock(n int64) io.Reader {
	return getReaderForBlock(bg.t, bg.getBlock(n))
}

func getReaderForBlock(t *testing.T, blk module.Block) io.Reader {
	buf := bytes.NewBuffer(nil)
	assert.NoError(t, blk.MarshalHeader(buf))
	assert.NoError(t, blk.MarshalBody(buf))
	return buf
}

func (bg *blockGenerator) generateUntil(n int64) {
	blk, err := bg.bm.GetLastBlock()
	assert.Nil(bg.t, err, "GetLastBlock")
	for i := blk.Height(); i < n; i++ {
		pid := blk.ID()
		br := proposeSync(bg.bm, pid, newCommitVoteSetWithTimestamp(true, i))
		blk = br.blk
		err := bg.bm.Finalize(br.blk)
		assert.Nil(bg.t, err, "Finalize")
	}
}

type blockManagerTestSetUp struct {
	gtx      *testTransaction
	database db.Database
	chain    *testChain
	sm       *testServiceManager
	bm       module.BlockManager
	bg       *blockGenerator
}

func newBlockManagerTestSetUp(t *testing.T) *blockManagerTestSetUp {
	s := &blockManagerTestSetUp{}
	s.database = newMapDB()
	s.gtx = newGenesisTX(defaultValidators)
	s.chain = newTestChain(s.database, s.gtx)
	s.sm = s.chain.sm
	var err error
	s.bm, err = NewManager(s.chain, nil, nil)
	assert.Nil(t, err)
	s.chain.bm = s.bm
	s.bg = newBlockGenerator(t, s.gtx)
	return s
}

func getLastBlockID(t *testing.T, bm module.BlockManager) []byte {
	blk, err := bm.GetLastBlock()
	assert.Nil(t, err, "last block error")
	return blk.ID()
}

func getBadBlockID(t *testing.T, bm module.BlockManager) []byte {
	id := getLastBlockID(t, bm)
	pid := make([]byte, len(id))
	copy(pid, id)
	pid[0] = ^pid[0]
	return pid
}

type blockResult struct {
	blk      module.BlockCandidate
	err      error
	cberr    error
	cbCalled bool
}

func (br *blockResult) assertOK(t *testing.T) {
	assert.NotNil(t, br.blk, "block")
	assert.Nil(t, br.err, "return error")
	assert.Nil(t, br.cberr, "cb error")
	assert.True(t, br.cbCalled, "cb called")
}

func (br *blockResult) assertError(t *testing.T) {
	assert.Nil(t, br.blk, "block")
	assert.NotNil(t, br.err, "return error")
	assert.Nil(t, br.cberr, "cb error")
	assert.False(t, br.cbCalled, "cb called")
}

type cbResult struct {
	blk module.BlockCandidate
	err error
}

func proposeSync(bm module.BlockManager, pid []byte, vs module.CommitVoteSet) *blockResult {
	ch := make(chan cbResult)
	_, err := bm.Propose(pid, vs, func(blk module.BlockCandidate, err error) {
		ch <- cbResult{blk, err}
	})
	if err != nil {
		return &blockResult{nil, err, nil, false}
	}
	res := <-ch
	return &blockResult{res.blk, nil, res.err, true}
}

func importSync(bm module.BlockManager, r io.Reader) *blockResult {
	ch := make(chan cbResult)
	_, err := bm.Import(r, 0, func(blk module.BlockCandidate, err error) {
		ch <- cbResult{blk, err}
	})
	if err != nil {
		return &blockResult{nil, err, nil, false}
	}
	res := <-ch
	return &blockResult{res.blk, nil, res.err, true}
}

func TestBlockManager_New_HasValidGenesisBlock(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	bm := s.bm
	assertHasValidGenesisBlock(t, bm)
	blk, _ := bm.GetLastBlock()
	id := blk.ID()
	assert.Equal(t, s.gtx.Data.Effect.NextValidators.Bytes(), blk.NextValidators().Bytes())

	c := newTestChain(s.chain.Database(), s.chain.gtx)
	var err error
	bm, err = NewManager(c, nil, nil)
	assert.Nil(t, err)
	assertHasValidGenesisBlock(t, bm)
	blk, _ = bm.GetLastBlock()
	assert.Equal(t, id, blk.ID(), "ID")
	assert.Equal(t, s.gtx.Data.Effect.NextValidators.Bytes(), blk.NextValidators().Bytes())
}

func TestBlockManager_Propose_ErrorOnBadParent(t *testing.T) {
	bm := newBlockManagerTestSetUp(t).bm
	pid := getBadBlockID(t, bm)
	br := proposeSync(bm, pid, newCommitVoteSet(false))
	br.assertError(t)
}

func TestBlockManager_Propose_ErrorOnInvalidCommitVoteSet(t *testing.T) {
	bm := newBlockManagerTestSetUp(t).bm
	pid := getLastBlockID(t, bm)

	cases := []struct {
		vs module.CommitVoteSet
		ok bool
	}{
		{newCommitVoteSet(false), false},
		{newCommitVoteSet(true), true},
	}
	// for height 1
	for _, c := range cases {
		br := proposeSync(bm, pid, c.vs)
		if c.ok {
			br.assertOK(t)
			err := bm.Finalize(br.blk)
			assert.Nil(t, err, "finalize error")
			pid = br.blk.ID()
		} else {
			br.assertError(t)
		}
	}
	// for height 2
	for _, c := range cases {
		br := proposeSync(bm, pid, c.vs)
		if c.ok {
			br.assertOK(t)
			err := bm.Finalize(br.blk)
			assert.Nil(t, err, "finalize error")
			pid = br.blk.ID()
		} else {
			br.assertError(t)
		}
	}
}

func TestBlockManager_Propose_ReturnsValidBlock(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	bm := s.bm
	sm := s.sm
	tx := newTestTransaction()
	tx.Data.Effect.NextValidators = newRandomTestValidatorList(2)
	_, err := sm.SendTransaction(nil, 0, tx)
	assert.NoError(t, err)
	pid := getLastBlockID(t, bm)
	br := proposeSync(bm, pid, newCommitVoteSet(true))
	br.assertOK(t)
	blk := br.blk
	assert.Equal(t, gheight+1, blk.Height(), "height")
	assert.Equal(t, pid, blk.PrevID(), "prevID")
	assert.Equal(t, s.chain.Wallet().Address().Bytes(), blk.Proposer().Bytes())
	assert.Equal(t, s.gtx.Data.Effect.NextValidators.Bytes(), blk.NextValidators().Bytes())
	ntxs := blk.NormalTransactions()
	assert.NotNil(t, ntxs, "normal transactions")
	tx2, err := ntxs.Get(0)
	assert.Nil(t, err, "0th transaction")
	ttx2 := tx2.(*testTransaction)
	assert.NotNil(t, ttx2, "casting to testTransaction")
	assert.Equal(t, tx, ttx2, "transaction")
	br = proposeSync(bm, br.blk.ID(), newCommitVoteSet(true))
	br.assertOK(t)
	assert.Equal(t, tx.Data.Effect.NextValidators.Bytes(), br.blk.NextValidators().Bytes(), "validator list")
}

func TestBlockManager_Propose_Cancel(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	ec := make(chan struct{})
	s.sm.setTransitionExeChan(ec)
	pid := getLastBlockID(t, s.bm)
	br := proposeSync(s.bm, pid, newCommitVoteSet(true))
	blk := br.blk
	pid = blk.ID()

	canceler, err := s.bm.Propose(pid, newCommitVoteSet(true), func(blk module.BlockCandidate, err error) {
		assert.Fail(t, "canceled proposal cb was called")
	})
	assert.Nil(t, err, "propose return error")
	res := canceler.Cancel()
	assert.Equal(t, true, res, "canceler result")
}

func TestBlockManager_Import_ErrorOnBadParent(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	r := s.bg.getReaderForBlock(3)
	br := importSync(s.bm, r)
	br.assertError(t)
}

func TestBlockManager_Import_OK(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	for i := int64(1); i < 10; i++ {
		r := s.bg.getReaderForBlock(i)
		br := importSync(s.bm, r)
		br.assertOK(t)
		assert.NoError(t, s.bm.Finalize(br.blk))
	}
}

func TestBlockManager_Import_Cancel(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	ec := make(chan struct{})
	r := s.bg.getReaderForBlock(1)
	s.sm.setTransitionExeChan(ec)
	canceler, err := s.bm.Import(r, 0, func(blk module.BlockCandidate, err error) {
		assert.Fail(t, "canceled import cb was called")
	})
	assert.Nil(t, err, "import return error")
	res := canceler.Cancel()
	assert.Equal(t, true, res, "canceler result")
}

func TestBlockManager_Import_BadTimestamp(t *testing.T) {
	s := newBlockManagerTestSetUp(t)

	// height 1 - OK
	r := s.bg.getReaderForBlock(1)
	br := importSync(s.bm, r)
	br.assertOK(t)
	assert.NoError(t, s.bm.Finalize(br.blk))

	// height 2 - alter timestamp
	blk := s.bg.getBlock(2)
	assert.NotNil(t, blk)
	blk.(*blockV2).timestamp = blk.(*blockV2).timestamp + 10
	blk.(*blockV2)._id = nil
	r = getReaderForBlock(t, blk)
	br = importSync(s.bm, r)
	// TODO: check if the observed error is the expected error
	br.assertError(t)
}

func TestBlockManager_Import_NonAscendingTimestamp(t *testing.T) {
	s := newBlockManagerTestSetUp(t)

	// height 1 - OK
	r := s.bg.getReaderForBlock(1)
	br := importSync(s.bm, r)
	br.assertOK(t)
	assert.NoError(t, s.bm.Finalize(br.blk))

	// height 2 - change timestamp (2 -> 10)
	blk := s.bg.getBlock(2)
	assert.NotNil(t, blk)
	votes := newCommitVoteSetWithTimestamp(true, 10)
	oriVotes := blk.(*blockV2).votes
	blk.(*blockV2).votes = votes
	oriTimestamp := blk.(*blockV2).timestamp
	blk.(*blockV2).timestamp = votes.Timestamp()
	oriID := blk.(*blockV2)._id
	blk.(*blockV2)._id = nil
	r = getReaderForBlock(t, blk)
	prevHash := blk.ID()
	blk.(*blockV2).votes = oriVotes
	blk.(*blockV2).timestamp = oriTimestamp
	blk.(*blockV2)._id = oriID
	br = importSync(s.bm, r)
	br.assertOK(t)
	assert.NoError(t, s.bm.Finalize(br.blk))

	// height 3 - do not change timestamp (3 -> 3)
	blk = s.bg.getBlock(3)
	assert.NotNil(t, blk)
	blk.(*blockV2).prevID = prevHash
	blk.(*blockV2)._id = nil
	r = getReaderForBlock(t, blk)
	br = importSync(s.bm, r)
	// TODO: check if the observed error is the expected error
	br.assertError(t)
}

func TestBlockManager_WaitForBlock_Nonblock(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	const height = int64(1)
	r := s.bg.getReaderForBlock(height)
	br := importSync(s.bm, r)
	br.assertOK(t)
	assert.NoError(t, s.bm.Finalize(br.blk))
	bch, err := s.bm.WaitForBlock(height)
	assert.NoError(t, err)
	blk := <-bch
	assert.Equal(t, blk.Height(), height)
	assert.Equal(t, blk.ID(), br.blk.ID())
}

func TestBlockManager_WaitForBlock_Block(t *testing.T) {
	s := newBlockManagerTestSetUp(t)
	const height = int64(3)
	bch, err := s.bm.WaitForBlock(height)
	assert.Nil(t, err)
	var br *blockResult
	for i := int64(1); i <= height; i++ {
		r := s.bg.getReaderForBlock(i)
		br = importSync(s.bm, r)
		br.assertOK(t)
		select {
		case blk := <-bch:
			assert.Failf(t, "unexpected return from WaitForBlock", "blk=%v", blk)
		default:
		}
		assert.NoError(t, s.bm.Finalize(br.blk))
	}
	blk := <-bch
	assert.Equal(t, blk.Height(), height)
	if assert.NotNil(t, br) && br != nil {
		assert.Equal(t, blk.ID(), br.blk.ID())
	}
}
