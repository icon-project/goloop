package block

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

// TODO overall error handling log? return error?
// TODO import, finalize V1
// TODO refactor code using bucketFor

var dbCodec = codec.MP

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
	return m.chain.GetDatabase()
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
	votes       module.VoteList
}

func (bn *bnode) addChild(c *bnode) {
	bn.children = append(bn.children, c)
	c.parent = bn
}

func (bn *bnode) dispose() {
	for _, c := range bn.children {
		c.dispose()
	}
	bn.in.dispose()
	bn.preexe.dispose()
}

func (t *task) cb(block module.Block, err error) {
	cb := t._cb
	t.manager.syncer.callLater(func() {
		cb(block, err)
	})
}

func (m *manager) _import(
	r io.Reader,
	cb func(module.Block, error),
) (*importTask, error) {
	block := m.newBlockFromReader(r)
	bn := m.nmap[string(block.PrevID())]
	if bn == nil {
		return nil, common.ErrIllegalArgument
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
		return nil, common.ErrUnknown
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
		return true
	case validatingOut:
		it.stop()
		return true
	default:
		return false
	}
}

func (it *importTask) onValidate(err error) {
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
		bn := &bnode{
			block:  it.block,
			in:     it.in.newTransition(nil),
			preexe: it.out.newTransition(nil),
		}
		it.manager.nmap[string(bn.block.ID())] = bn
		it.stop()
		it.state = validatedOut
		it.cb(it.block, err)
	}
}

func (it *importTask) onExecute(err error) {
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
		it.out = it.in.transit(it.block.NormalTransactions(), it)
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
	votes module.VoteList,
	cb func(module.Block, error),
) (*proposeTask, error) {
	bn := m.nmap[string(parentID)]
	if bn == nil {
		return nil, common.ErrIllegalArgument
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
		return true
	default:
		return false
	}
}

func (pt *proposeTask) onValidate(err error) {
	if err != nil {
		pt.stop()
		pt.cb(nil, err)
		return
	}
}

func (pt *proposeTask) onExecute(err error) {
	if err != nil {
		pt.stop()
		pt.cb(nil, err)
		return
	}
	tr := pt.in.propose(nil)
	if tr == nil {
		pt.stop()
		pt.cb(nil, err)
	}
	pmtr := pt.in.mtransition()
	mtr := tr.mtransition()
	block := &blockV2{
		height:             pt.parentBlock.Height() + 1,
		timestamp:          time.Now(),
		proposer:           pt.manager.chain.GetWallet().GetAddress(),
		prevID:             pt.parentBlock.ID(),
		logBloom:           pmtr.LogBloom(),
		result:             pmtr.Result(),
		patchTransactions:  mtr.PatchTransactions(),
		normalTransactions: mtr.NormalTransactions(),
		nextValidators:     pmtr.NextValidators(),
		votes:              pt.votes,
	}
	bn := &bnode{
		block:  block,
		in:     pt.in.newTransition(nil),
		preexe: tr,
	}
	pt.manager.nmap[string(bn.block.ID())] = bn
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
	// TODO if last block is v1 block
	m := &manager{
		chain: chain,
		sm:    sm,
	}
	chainPropBucket := m.bucketFor(db.ChainProperty)
	if chainPropBucket == nil {
		return nil
	}
	var height int64
	err := chainPropBucket.get(raw("block.height"), &height)
	if err == common.ErrNotFound {
		chainPropBucket.set(raw("block.height"), 0)
		height = 0
		//	TODO H handle genesis block
	} else if err != nil {
		return nil
	}
	hashByHeightBucket := m.bucketFor(db.BlockHeaderHashByHeight)
	hash, err := hashByHeightBucket.getBytes(&height)
	if err != nil {
		return nil
	}
	lastFinalized := m.GetBlock(hash)
	mtr, _ := m.sm.CreateInitialTransition(lastFinalized.Result(), lastFinalized.NextValidators(), lastFinalized.Height()-1)
	if mtr == nil {
		return nil
	}
	tr := newInitialTransition(mtr, &m.syncer, sm)
	bn := &bnode{
		block: lastFinalized,
		in:    tr,
	}
	bn.preexe = tr.transit(lastFinalized.NormalTransactions(), nil)
	m.finalized = bn
	m.nmap[string(lastFinalized.ID())] = bn
	return m
}

func (m *manager) GetBlock(id []byte) module.Block {
	// TODO handle v1
	hb := m.bucketFor(db.BytesByHash)
	if hb == nil {
		panic("cannot get bucket BytesByHash")
	}
	headerBytes, err := hb.getBytes(raw(id))
	if err != nil {
		return nil
	}
	if headerBytes != nil {
		return m.newBlockFromHeaderReader(bytes.NewReader(headerBytes))
	}
	return nil
}

func (m *manager) Import(
	r io.Reader,
	cb func(module.Block, error),
) (func() bool, error) {
	m.syncer.begin()
	defer m.syncer.end()

	it, err := m._import(r, cb)
	if err != nil {
		return nil, err
	}
	return func() bool {
		m.syncer.begin()
		defer m.syncer.end()
		return it.cancel()
	}, nil
}

func (m *manager) Propose(
	parentID []byte,
	votes module.VoteList,
	cb func(module.Block, error),
) (canceler func() bool, err error) {
	m.syncer.begin()
	defer m.syncer.end()

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
	// TODO notify import/propose error due to finalization
	bn := m.nmap[string(block.ID())]
	if bn == nil || bn.parent != m.finalized {
		return common.ErrIllegalArgument
	}

	for _, c := range m.finalized.children {
		if c != bn {
			c.dispose()
		}
	}
	m.finalized.dispose()

	m.finalized = bn
	m.finalized.parent = nil

	m.sm.Finalize(
		bn.in.mtransition(),
		module.FinalizePatchTransaction|module.FinalizeResult,
	)
	m.sm.Finalize(bn.preexe.mtransition(), module.FinalizeNormalTransaction)

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
			lb.set(raw(tr.Hash()), trLoc)
		}
		for it := block.NormalTransactions().Iterator(); it.Has(); it.Next() {
			tr, i, _ := it.Get()
			trLoc := transactionLocator{
				BlockHeight:      block.Height(),
				TransactionGroup: module.TransactionGroupNormal,
				IndexInGroup:     i,
			}
			lb.set(raw(tr.Hash()), trLoc)
		}
		b := m.bucketFor(db.BlockHeaderHashByHeight)
		if b == nil {
			return common.ErrUnknown
		}
		b.set(block.Height(), raw(block.ID()))
	}
	// TODO update DB for v1 : blockV1, trLocatorByHash
	return nil
}

func (m *manager) voteListFromHash(hash []byte) module.VoteList {
	hb := m.bucketFor(db.BytesByHash)
	if hb == nil {
		return nil
	}
	bs, err := hb.getBytes(hash)
	if err != nil {
		return nil
	}
	dec := m.chain.VoteListDecoder()
	return dec(bs)
}

func (m *manager) newBlockFromHeaderReader(r io.Reader) module.Block {
	var header blockV2HeaderFormat
	v2codec.Unmarshal(r, &header)
	patches := m.sm.TransactionListFromHash(header.PatchTransactionsHash)
	normalTxs := m.sm.TransactionListFromHash(header.NormalTransactionsHash)
	nextValidators := m.sm.ValidatorListFromHash(header.NextValidatorsHash)
	votes := m.voteListFromHash(header.VotesHash)
	if patches == nil || normalTxs == nil || nextValidators == nil ||
		votes == nil {
		return nil
	}
	return &blockV2{
		height:             header.Height,
		timestamp:          timeFromUnixMicro(header.Timestamp),
		proposer:           common.NewAccountAddress(header.Proposer),
		prevID:             header.PrevID,
		logBloom:           header.LogBloom,
		result:             header.Result,
		patchTransactions:  patches,
		normalTransactions: normalTxs,
		nextValidators:     nextValidators,
		votes:              votes,
	}
}

func (m *manager) newTransactionListFromBSS(
	bss [][]byte,
	version int,
) module.TransactionList {
	ts := make([]module.Transaction, len(bss))
	for i, bs := range bss {
		ts[i] = m.sm.TransactionFromBytes(bs, version)
	}
	return m.sm.TransactionListFromSlice(ts, version)
}

func (m *manager) newBlockFromReader(r io.Reader) module.Block {
	// TODO return error? log error?
	// TODO handle v1
	var blockFormat blockV2Format
	v2codec.Unmarshal(r, &blockFormat)
	patches := m.newTransactionListFromBSS(
		blockFormat.PatchTransactions,
		common.BlockVersion2,
	)
	if bytes.Equal(patches.Hash(), blockFormat.PatchTransactionsHash) {
		return nil
	}
	normalTxs := m.newTransactionListFromBSS(
		blockFormat.NormalTransactions,
		common.BlockVersion2,
	)
	if bytes.Equal(normalTxs.Hash(), blockFormat.NormalTransactionsHash) {
		return nil
	}
	nextValidators := m.sm.ValidatorListFromHash(blockFormat.NextValidatorsHash)
	if bytes.Equal(nextValidators.Hash(), blockFormat.NextValidatorsHash) {
		return nil
	}
	votes := m.chain.VoteListDecoder()(blockFormat.Votes)
	if bytes.Equal(votes.Hash(), blockFormat.VotesHash) {
		return nil
	}
	return &blockV2{
		height:             blockFormat.Height,
		timestamp:          timeFromUnixMicro(blockFormat.Timestamp),
		proposer:           common.NewAccountAddress(blockFormat.Proposer),
		prevID:             blockFormat.PrevID,
		logBloom:           blockFormat.LogBloom,
		result:             blockFormat.Result,
		patchTransactions:  patches,
		normalTransactions: normalTxs,
		nextValidators:     nextValidators,
		votes:              votes,
	}
}

// TODO GetTransactionInfo
func (m *manager) GetTransactionInfo(id []byte) module.TransactionInfo {
	return nil
}
