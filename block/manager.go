package block

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

// TODO overall error handling log? return error?
// TODO import, finalize V1
// TODO refactor code using bucketFor
// TODO wait for transaction available

var dbCodec = codec.MP
var logger *log.Logger

const (
	keyLastBlockHeight = "block.lastHeight"
	genesisHeight      = 0
)

type transactionLocator struct {
	BlockHeight      int64
	TransactionGroup module.TransactionGroup
	IndexInGroup     int
}

type bnode struct {
	parent   *bnode
	children []*bnode
	block    module.Block
	in       *transition
	preexe   *transition
}

type manager struct {
	syncer syncer

	chain     module.Chain
	sm        module.ServiceManager
	nmap      map[string]*bnode
	finalized *bnode
}

func (m *manager) db() db.Database {
	return m.chain.Database()
}

type taskState int

const (
	executingIn taskState = iota
	validatingOut
	validatedOut
	stopped
)

type task struct {
	manager *manager
	_cb     func(module.Block, error)
	in      *transition
	state   taskState
}

type importTask struct {
	task
	out   *transition
	block module.Block
}

type proposeTask struct {
	task
	parentBlock module.Block
	votes       module.CommitVoteSet
}

func (m *manager) addNode(par *bnode, bn *bnode) {
	par.children = append(par.children, bn)
	bn.parent = par
	m.nmap[string(bn.block.ID())] = bn
}

func (m *manager) removeNode(bn *bnode) {
	for _, c := range bn.children {
		m.removeNode(c)
	}
	bn.in.dispose()
	bn.preexe.dispose()
	delete(m.nmap, string(bn.block.ID()))
}

func (m *manager) removeNodeExcept(bn *bnode, except *bnode) {
	for _, c := range bn.children {
		if c != except {
			m.removeNode(c)
			c.parent = nil
		}
	}
	bn.in.dispose()
	bn.preexe.dispose()
	delete(m.nmap, string(bn.block.ID()))
}

func (t *task) cb(block module.Block, err error) {
	cb := t._cb
	t.manager.syncer.callLater(func() {
		cb(block, err)
	})
}

func (m *manager) _import(
	block module.Block,
	cb func(module.Block, error),
) (*importTask, error) {
	bn := m.nmap[string(block.PrevID())]
	if bn == nil {
		return nil, errors.Errorf("InvalidPreviousID(%x)", block.PrevID())
	}
	var validators module.ValidatorList
	if block.Height() == 1 {
		validators = nil
	} else {
		pprev, err := m.getBlock(bn.block.PrevID())
		if err != nil {
			logger.Panicf("cannot get prev prev block %x\n", bn.block.PrevID())
		}
		validators = pprev.NextValidators()
	}
	if err := verifyBlock(block, bn.block, validators); err != nil {
		return nil, err
	}
	it := &importTask{
		block: block,
		task: task{
			manager: m,
			_cb:     cb,
		},
	}
	it.state = executingIn
	it.in = bn.preexe.patch(it.block.PatchTransactions(), it)
	if it.in == nil {
		return nil, errors.New("FailToPatch")
	}
	return it, nil
}

func (it *importTask) stop() {
	if it.in != nil {
		it.in.dispose()
	}
	if it.out != nil {
		it.out.dispose()
	}
	it.state = stopped
}

func (it *importTask) cancel() bool {
	switch it.state {
	case executingIn:
		it.stop()
	case validatingOut:
		it.stop()
	default:
		logger.Printf("Cancel Import: Ignored\n")
		return false
	}
	logger.Printf("Cancel Import: OK\n")
	return true
}

func (it *importTask) onValidate(err error) {
	it.manager.syncer.callLaterInLock(func() {
		it._onValidate(err)
	})
}

func (it *importTask) _onValidate(err error) {
	if it.state == executingIn {
		if err != nil {
			it.stop()
			it.cb(nil, err)
			return
		}
	} else if it.state == validatingOut {
		if err != nil {
			it.stop()
			it.cb(nil, err)
			return
		}
		if _, ok := it.manager.nmap[string(it.block.ID())]; !ok {
			bn := &bnode{
				block:  it.block,
				in:     it.in.newTransition(nil),
				preexe: it.out.newTransition(nil),
			}
			pbn := it.manager.nmap[string(it.block.PrevID())]
			it.manager.addNode(pbn, bn)
		}
		it.stop()
		it.state = validatedOut
		it.cb(it.block, err)
	}
}

func (it *importTask) onExecute(err error) {
	it.manager.syncer.callLaterInLock(func() {
		it._onExecute(err)
	})
}

func (it *importTask) _onExecute(err error) {
	if it.state == executingIn {
		if err != nil {
			it.stop()
			it.cb(nil, err)
			return
		}
		err = it.in.verifyResult(it.block)
		if err != nil {
			it.stop()
			it.cb(nil, err)
			return
		}
		it.out = it.in.transit(it.block.NormalTransactions(), it.block, it)
		if it.out == nil {
			it.stop()
			it.cb(nil, common.ErrUnknown)
			return
		}
		it.state = validatingOut
		return
	}
}

func (m *manager) _propose(
	parentID []byte,
	votes module.CommitVoteSet,
	cb func(module.Block, error),
) (*proposeTask, error) {
	bn := m.nmap[string(parentID)]
	if bn == nil {
		return nil, errors.Errorf("NoParentBlock(id=<%x>)", parentID)
	}
	var validators module.ValidatorList
	if bn.block.Height() == 0 {
		validators = nil
	} else {
		pprev, err := m.getBlock(bn.block.PrevID())
		if err != nil {
			logger.Panicf("cannot get prev prev block %x\n", bn.block.PrevID())
		}
		validators = pprev.NextValidators()
	}
	if err := votes.Verify(bn.block, validators); err != nil {
		return nil, err
	}
	pt := &proposeTask{
		task: task{
			manager: m,
			_cb:     cb,
		},
		parentBlock: bn.block,
		votes:       votes,
	}
	pt.state = executingIn
	patches := m.sm.GetPatches(bn.in.mtransition())
	pt.in = bn.preexe.patch(patches, pt)
	if pt.in == nil {
		return nil, common.ErrUnknown
	}
	return pt, nil
}

func (pt *proposeTask) stop() {
	if pt.in != nil {
		pt.in.dispose()
	}
	pt.state = stopped
}

func (pt *proposeTask) cancel() bool {
	switch pt.state {
	case executingIn:
		pt.stop()
	default:
		logger.Printf("Cancel Propose: Ignored\n")
		return false
	}
	logger.Printf("Cancel Propose: OK\n")
	return true
}

func (pt *proposeTask) onValidate(err error) {
	pt.manager.syncer.callLaterInLock(func() {
		pt._onValidate(err)
	})
}

func (pt *proposeTask) _onValidate(err error) {
	if err != nil {
		pt.stop()
		pt.cb(nil, err)
		return
	}
}

func (pt *proposeTask) onExecute(err error) {
	pt.manager.syncer.callLaterInLock(func() {
		pt._onExecute(err)
	})
}

func (pt *proposeTask) _onExecute(err error) {
	if err != nil {
		pt.stop()
		pt.cb(nil, err)
		return
	}
	height := pt.parentBlock.Height() + 1
	timestamp := unixMicroFromTime(time.Now())
	tr := pt.in.propose(newBlockInfo(height, timestamp), nil)
	if tr == nil {
		pt.stop()
		pt.cb(nil, err)
	}
	pmtr := pt.in.mtransition()
	mtr := tr.mtransition()
	block := &blockV2{
		height:             height,
		timestamp:          timestamp,
		proposer:           pt.manager.chain.Wallet().Address(),
		prevID:             pt.parentBlock.ID(),
		logBloom:           pmtr.LogBloom(),
		result:             pmtr.Result(),
		patchTransactions:  mtr.PatchTransactions(),
		normalTransactions: mtr.NormalTransactions(),
		nextValidators:     pmtr.NextValidators(),
		votes:              pt.votes,
	}
	if _, ok := pt.manager.nmap[string(block.ID())]; !ok {
		bn := &bnode{
			block:  block,
			in:     pt.in.newTransition(nil),
			preexe: tr,
		}
		pbn := pt.manager.nmap[string(block.PrevID())]
		pt.manager.addNode(pbn, bn)
	}
	pt.stop()
	pt.state = validatedOut
	pt.cb(block, nil)
	return
}

// NewManager creates BlockManager.
func NewManager(
	chain module.Chain,
	sm module.ServiceManager,
) module.BlockManager {
	prefix := fmt.Sprintf("%x|BM|", chain.Wallet().Address().Bytes()[1:3])
	logger = log.New(os.Stderr, prefix, log.Lshortfile|log.Lmicroseconds)
	// TODO if last block is v1 block
	m := &manager{
		chain: chain,
		sm:    sm,
		nmap:  make(map[string]*bnode),
	}
	chainPropBucket := m.bucketFor(db.ChainProperty)
	if chainPropBucket == nil {
		return nil
	}

	var height int64
	err := chainPropBucket.get(raw(keyLastBlockHeight), &height)
	if err == common.ErrNotFound {
		if _, err := m.finalizeGenesisBlock(nil, 0, chain.CommitVoteSetDecoder()(nil)); err != nil {
			return nil
		}
		return m
	} else if err != nil {
		return nil
	}
	hashByHeightBucket := m.bucketFor(db.BlockHeaderHashByHeight)
	hash, err := hashByHeightBucket.getBytes(&height)
	if err != nil {
		return nil
	}
	lastFinalized, err := m.getBlock(hash)
	mtr, _ := m.sm.CreateInitialTransition(lastFinalized.Result(), lastFinalized.NextValidators())
	if mtr == nil {
		return nil
	}
	tr := newInitialTransition(mtr, &m.syncer, sm)
	bn := &bnode{
		block: lastFinalized,
		in:    tr,
	}
	bn.preexe = tr.transit(lastFinalized.NormalTransactions(), lastFinalized, nil)
	m.finalized = bn
	if bn != nil {
		m.nmap[string(lastFinalized.ID())] = bn
	}
	return m
}

func (m *manager) GetBlock(id []byte) (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	return m.getBlock(id)
}

func (m *manager) getBlock(id []byte) (module.Block, error) {
	// TODO handle v1
	hb := m.bucketFor(db.BytesByHash)
	if hb == nil {
		panic("cannot get bucket BytesByHash")
	}
	headerBytes, err := hb.getBytes(raw(id))
	if err != nil {
		return nil, err
	}
	if headerBytes != nil {
		return m.newBlockFromHeaderReader(bytes.NewReader(headerBytes))
	}
	return nil, common.ErrUnknown
}

func (m *manager) Import(
	r io.Reader,
	cb func(module.Block, error),
) (func() bool, error) {
	m.syncer.begin()
	defer m.syncer.end()

	logger.Printf("Import(%x)\n", r)

	block, err := m.newBlockFromReader(r)
	if err != nil {
		return nil, err
	}
	it, err := m._import(block, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		m.syncer.begin()
		defer m.syncer.end()
		return it.cancel()
	}, nil
}

func (m *manager) ImportBlock(
	block module.Block,
	cb func(module.Block, error),
) (func() bool, error) {
	m.syncer.begin()
	defer m.syncer.end()

	logger.Printf("ImportBlock(%x)\n", block.ID())

	it, err := m._import(block, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		m.syncer.begin()
		defer m.syncer.end()
		return it.cancel()
	}, nil
}

type channelingCB struct {
	ch chan<- error
}

func (cb *channelingCB) onValidate(err error) {
	cb.ch <- err
}

func (cb *channelingCB) onExecute(err error) {
	cb.ch <- err
}

func (m *manager) finalizeGenesisBlock(
	proposer module.Address,
	timestamp int64,
	votes module.CommitVoteSet,
) (block module.Block, err error) {
	logger.Printf("FinalizeGenesisBlock()\n")
	if m.finalized != nil {
		return nil, common.ErrInvalidState
	}
	mtr, err := m.sm.CreateInitialTransition(nil, nil)
	if err != nil {
		return nil, err
	}
	in := newInitialTransition(mtr, &m.syncer, m.sm)
	ch := make(chan error)
	gtxbs := m.chain.Genesis()
	gtx, err := m.sm.TransactionFromBytes(gtxbs, module.BlockVersion2)
	if err != nil {
		return nil, err
	}
	gtxl := m.sm.TransactionListFromSlice([]module.Transaction{gtx}, module.BlockVersion2)
	m.syncer.begin()
	gtr := in.transit(gtxl, newBlockInfo(0, timestamp), &channelingCB{ch: ch})
	m.syncer.end()

	// wait for genesis transition execution
	// TODO rollback
	if err = <-ch; err != nil {
		return nil, err
	}
	if err = <-ch; err != nil {
		return nil, err
	}

	bn := &bnode{}
	bn.in = in
	bn.preexe = gtr
	bn.block = &blockV2{
		height:             genesisHeight,
		timestamp:          timestamp,
		proposer:           proposer,
		prevID:             nil,
		logBloom:           mtr.LogBloom(),
		result:             mtr.Result(),
		patchTransactions:  gtr.mtransition().PatchTransactions(),
		normalTransactions: gtr.mtransition().NormalTransactions(),
		nextValidators:     gtr.mtransition().NextValidators(),
		votes:              votes,
	}
	m.nmap[string(bn.block.ID())] = bn
	err = m.finalize(bn)
	m.sm.Finalize(gtr.mtransition(), module.FinalizeNormalTransaction|module.FinalizePatchTransaction|module.FinalizeResult)
	if err != nil {
		return nil, err
	}
	return bn.block, nil
}

func (m *manager) Propose(
	parentID []byte,
	votes module.CommitVoteSet,
	cb func(module.Block, error),
) (canceler func() bool, err error) {
	m.syncer.begin()
	defer m.syncer.end()

	logger.Printf("Propose(<%x>, %v)\n", parentID, votes)

	pt, err := m._propose(parentID, votes, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		m.syncer.begin()
		defer m.syncer.end()

		return pt.cancel()
	}, nil
}

func (m *manager) Commit(block module.Block) error {
	return nil
}

func (m *manager) bucketFor(id db.BucketID) *bucket {
	b, err := m.db().GetBucket(id)
	if err != nil {
		return nil
	}
	return &bucket{
		dbBucket: b,
		codec:    dbCodec,
	}
}

func (m *manager) Finalize(block module.Block) error {
	m.syncer.begin()
	defer m.syncer.end()

	bn := m.nmap[string(block.ID())]
	if bn == nil || bn.parent != m.finalized {
		return errors.Errorf("InvalidStatusForBlock(id=<%x>", block.ID())
	}
	return m.finalize(bn)
}

func (m *manager) finalize(bn *bnode) error {
	// TODO notify import/propose error due to finalization
	// TODO update nmap
	block := bn.block

	if m.finalized != nil {
		m.removeNodeExcept(m.finalized, bn)
		m.sm.Finalize(
			bn.in.mtransition(),
			module.FinalizePatchTransaction|module.FinalizeResult,
		)
	}
	m.sm.Finalize(bn.preexe.mtransition(), module.FinalizeNormalTransaction)

	m.finalized = bn

	if blockV2, ok := block.(*blockV2); ok {
		hb := m.bucketFor(db.BytesByHash)
		if hb == nil {
			return common.ErrUnknown
		}
		hb.put(blockV2._headerFormat())
		hb.set(raw(block.Votes().Hash()), raw(block.Votes().Bytes()))
		lb := m.bucketFor(db.TransactionLocatorByHash)
		for it := block.PatchTransactions().Iterator(); it.Has(); it.Next() {
			tr, i, _ := it.Get()
			trLoc := transactionLocator{
				BlockHeight:      block.Height(),
				TransactionGroup: module.TransactionGroupPatch,
				IndexInGroup:     i,
			}
			lb.set(raw(tr.ID()), trLoc)
		}
		for it := block.NormalTransactions().Iterator(); it.Has(); it.Next() {
			tr, i, _ := it.Get()
			trLoc := transactionLocator{
				BlockHeight:      block.Height(),
				TransactionGroup: module.TransactionGroupNormal,
				IndexInGroup:     i,
			}
			lb.set(raw(tr.ID()), trLoc)
		}
		b := m.bucketFor(db.BlockHeaderHashByHeight)
		if b == nil {
			return common.ErrUnknown
		}
		b.set(block.Height(), raw(block.ID()))
		chainProp := m.bucketFor(db.ChainProperty)
		chainProp.set(raw(keyLastBlockHeight), block.Height())
	}
	// TODO update DB for v1 : blockV1, trLocatorByHash
	return nil
}

func (m *manager) commitVoteSetFromHash(hash []byte) module.CommitVoteSet {
	hb := m.bucketFor(db.BytesByHash)
	if hb == nil {
		return nil
	}
	bs, err := hb.getBytes(raw(hash))
	if err != nil {
		return nil
	}
	dec := m.chain.CommitVoteSetDecoder()
	return dec(bs)
}

func newAddress(bs []byte) module.Address {
	if bs != nil {
		return common.NewAddress(bs)
	}
	return nil
}

func (m *manager) newBlockFromHeaderReader(r io.Reader) (module.Block, error) {
	var header blockV2HeaderFormat
	err := v2Codec.Unmarshal(r, &header)
	if err != nil {
		return nil, err
	}
	patches := m.sm.TransactionListFromHash(header.PatchTransactionsHash)
	if patches == nil {
		return nil, errors.Errorf("TranscationListFromHash(%x) failed", header.PatchTransactionsHash)
	}
	normalTxs := m.sm.TransactionListFromHash(header.NormalTransactionsHash)
	if normalTxs == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", header.NormalTransactionsHash)
	}
	nextValidators := m.sm.ValidatorListFromHash(header.NextValidatorsHash)
	if nextValidators == nil {
		return nil, errors.Errorf("ValidatorListFromHas(%x)", header.NextValidatorsHash)
	}
	votes := m.commitVoteSetFromHash(header.VotesHash)
	if votes == nil {
		return nil, errors.Errorf("commitVoteSetFromHash(%x) failed", header.VotesHash)
	}
	return &blockV2{
		height:             header.Height,
		timestamp:          header.Timestamp,
		proposer:           newAddress(header.Proposer),
		prevID:             header.PrevID,
		logBloom:           common.NewLogBloom(header.LogBloom),
		result:             header.Result,
		patchTransactions:  patches,
		normalTransactions: normalTxs,
		nextValidators:     nextValidators,
		votes:              votes,
	}, nil
}

func (m *manager) newTransactionListFromBSS(
	bss [][]byte,
	version int,
) (module.TransactionList, error) {
	ts := make([]module.Transaction, len(bss))
	for i, bs := range bss {
		if tx, err := m.sm.TransactionFromBytes(bs, version); err != nil {
			return nil, err
		} else {
			ts[i] = tx
		}
	}
	return m.sm.TransactionListFromSlice(ts, version), nil
}

func (m *manager) NewBlockFromReader(r io.Reader) (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	return m.newBlockFromReader(r)
}

func (m *manager) newBlockFromReader(r io.Reader) (module.Block, error) {
	// TODO handle v1
	var blockFormat blockV2Format
	err := v2Codec.Unmarshal(r, &blockFormat.blockV2HeaderFormat)
	if err != nil {
		return nil, err
	}
	err = v2Codec.Unmarshal(r, &blockFormat.blockV2BodyFormat)
	if err != nil {
		return nil, err
	}
	patches, err := m.newTransactionListFromBSS(
		blockFormat.PatchTransactions,
		module.BlockVersion2,
	)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(patches.Hash(), blockFormat.PatchTransactionsHash) {
		return nil, errors.New("bad patch transactions hash")
	}
	normalTxs, err := m.newTransactionListFromBSS(
		blockFormat.NormalTransactions,
		module.BlockVersion2,
	)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(normalTxs.Hash(), blockFormat.NormalTransactionsHash) {
		return nil, errors.New("bad normal transactions hash")
	}
	nextValidators := m.sm.ValidatorListFromHash(blockFormat.NextValidatorsHash)
	if !bytes.Equal(nextValidators.Hash(), blockFormat.NextValidatorsHash) {
		return nil, errors.New("bad validator list hash")
	}
	votes := m.chain.CommitVoteSetDecoder()(blockFormat.Votes)
	if !bytes.Equal(votes.Hash(), blockFormat.VotesHash) {
		return nil, errors.New("bad vote list hash")
	}
	return &blockV2{
		height:             blockFormat.Height,
		timestamp:          blockFormat.Timestamp,
		proposer:           newAddress(blockFormat.Proposer),
		prevID:             blockFormat.PrevID,
		logBloom:           common.NewLogBloom(blockFormat.LogBloom),
		result:             blockFormat.Result,
		patchTransactions:  patches,
		normalTransactions: normalTxs,
		nextValidators:     nextValidators,
		votes:              votes,
	}, nil
}

type transactionInfo struct {
	_sm      module.ServiceManager
	_txID    []byte
	_txBlock module.Block
	_index   int
	_group   module.TransactionGroup
	_mtr     module.Transaction
	_rBlock  module.Block
}

func (txInfo *transactionInfo) Block() module.Block {
	return txInfo._txBlock
}

func (txInfo *transactionInfo) Index() int {
	return txInfo._index
}

func (txInfo *transactionInfo) Group() module.TransactionGroup {
	return txInfo._group
}

func (txInfo *transactionInfo) Transaction() module.Transaction {
	return txInfo._mtr
}

func (txInfo *transactionInfo) GetReceipt() module.Receipt {
	rblock := txInfo._rBlock
	if rblock != nil {
		rl := txInfo._sm.ReceiptListFromResult(rblock.Result(), txInfo._group)
		if rct, err := rl.Get(int(txInfo._index)); err == nil {
			return rct
		}
	}
	return nil
}

func (m *manager) GetTransactionInfo(id []byte) (module.TransactionInfo, error) {
	m.syncer.begin()
	defer m.syncer.end()

	return m.getTransactionInfo(id)
}

func (m *manager) getTransactionInfo(id []byte) (module.TransactionInfo, error) {
	// TODO handle V1 in GetTransactionInfo
	tlb := m.bucketFor(db.TransactionLocatorByHash)
	var loc transactionLocator
	err := tlb.get(raw(id), &loc)
	if err != nil {
		return nil, common.ErrNotFound
	}
	block, err := m.getBlockByHeight(loc.BlockHeight)
	if err != nil {
		return nil, common.ErrInvalidState
	}
	mtr, err := block.NormalTransactions().Get(loc.IndexInGroup)
	if err != nil {
		return nil, common.ErrInvalidState
	}
	rblock, err := m.getBlockByHeight(loc.BlockHeight + 1)
	if err != nil {
		rblock = nil
	}
	return &transactionInfo{
		_sm:      m.sm,
		_txID:    id,
		_txBlock: block,
		_index:   loc.IndexInGroup,
		_group:   loc.TransactionGroup,
		_mtr:     mtr,
		_rBlock:  rblock,
	}, nil
}

func (m *manager) GetBlockByHeight(height int64) (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	return m.getBlockByHeight(height)
}

func (m *manager) getBlockByHeight(height int64) (module.Block, error) {
	headerHashByHeight := m.bucketFor(db.BlockHeaderHashByHeight)
	hash, err := headerHashByHeight.getBytes(height)
	if err != nil {
		return nil, err
	}
	return m.getBlock(hash)
}

func (m *manager) GetLastBlock() (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	return m.getLastBlock()
}

func (m *manager) getLastBlock() (module.Block, error) {
	chainProp := m.bucketFor(db.ChainProperty)
	var height int64
	err := chainProp.get(raw(keyLastBlockHeight), &height)
	if err != nil {
		return nil, err
	}
	return m.getBlockByHeight(height)
}
