package block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/chain/gs"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

const (
	configTraceBnode = false
)

const (
	keyLastBlockHeight = "block.lastHeight"
	genesisHeight      = 0
	ConfigCacheCap     = 10
)

// can be disposed either automatically or by force.
type bnode struct {
	parent   *bnode
	children []*bnode
	block    module.Block
	in       *transition
	preexe   *transition

	// a block candidate has a ref.
	// a child bnode has a ref to parent.
	// manager has a ref to finalized.
	nRef int
}

func (bn *bnode) RefCount() int {
	return bn.nRef
}

func (bn *bnode) String() string {
	return fmt.Sprintf("%p{nRef:%d ID:%s}", bn, bn.nRef, common.HexPre(bn.block.ID()))
}

type ServiceManager interface {
	module.TransitionManager
	TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error)
	GetChainID(result []byte) (int64, error)
	GetNetworkID(result []byte) (int64, error)
	GetNextBlockVersion(result []byte) int
	ImportResult(result []byte, vh []byte, src db.Database) error
	GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error)
	TransactionListFromHash(hash []byte) module.TransactionList
	ReceiptListFromResult(result []byte, g module.TransactionGroup) (module.ReceiptList, error)
	SendTransaction(result []byte, height int64, tx interface{}) ([]byte, error)
	ValidatorListFromHash(hash []byte) module.ValidatorList
	TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList
	SendTransactionAndWait(result []byte, height int64, tx interface{}) ([]byte, <-chan interface{}, error)
	WaitTransactionResult(id []byte) (<-chan interface{}, error)
	ExportResult(result []byte, vh []byte, dst db.Database) error
	BTPSectionFromResult(result []byte) (module.BTPSection, error)
	NextProofContextMapFromResult(result []byte) (module.BTPProofContextMap, error)
}

type Chain interface {
	Database() db.Database
	Wallet() module.Wallet
	ServiceManager() module.ServiceManager
	NID() int
	CID() int
	GenesisStorage() module.GenesisStorage
	CommitVoteSetDecoder() module.CommitVoteSetDecoder
	Genesis() []byte
}

type LocatorManager interface {
	GetLocator(id []byte) (*module.TransactionLocator, error)
}

type chainContext struct {
	syncer  syncer
	chain   Chain
	sm      ServiceManager
	lm      LocatorManager
	log     log.Logger
	running bool
	trtr    RefTracer
	srcUID  []byte
}

type finalizationCB = func(module.Block) bool

type handlerContext struct {
	*manager
}

func (c handlerContext) GetBlockByHeight(height int64) (module.Block, error) {
	// without acquiring lock
	return c.manager.getBlockByHeight(height)
}

type manager struct {
	*chainContext
	bntr  RefTracer
	nmap  map[string]*bnode
	cache *cache

	finalized       *bnode
	finalizationCBs []finalizationCB
	timestamper     module.Timestamper

	// pcm for last finalized block verification
	pcmForLastBlock module.BTPProofContextMap
	// next pcm in the last finalized block's result
	nextPCM module.BTPProofContextMap

	handlers       handlerList
	activeHandlers handlerList
	handlerContext handlerContext
}

type handlerList []base.BlockHandler

func (hl handlerList) upTo(version int) handlerList {
	for i, h := range hl {
		if h.Version() == version {
			return hl[:i+1]
		}
	}
	return hl
}

func (hl handlerList) forVersion(version int) (base.BlockHandler, bool) {
	for i := len(hl) - 1; i >= 0; i-- {
		if hl[i].Version() == version {
			return hl[i], true
		}
	}
	return nil, false
}

func (hl handlerList) last() base.BlockHandler {
	return hl[len(hl)-1]
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

const (
	exportBlock = 0x1 << iota
	exportValidator
	exportResult
	exportTransaction
	exportIndex
	exportReserved
	exportHashable = exportBlock | exportValidator | exportResult | exportTransaction
	exportAll      = exportReserved - 1
)

type task struct {
	manager *manager
	_cb     func(module.BlockCandidate, error)
	in      *transition
	state   taskState
}

type importTask struct {
	task
	out   *transition
	block module.BlockData
	csi   module.ConsensusInfo
	flags int
}

type proposeTask struct {
	task
	parentBlock module.Block
	votes       module.CommitVoteSet
	csi         module.ConsensusInfo
}

func (m *manager) addNode(par *bnode, bn *bnode) {
	par.children = append(par.children, bn)
	bn.parent = par
	par.nRef++
	if configTraceBnode {
		m.bntr.TraceRef(par)
	}
	m.nmap[string(bn.block.ID())] = bn
}

func (m *manager) newCandidate(bn *bnode) *blockCandidate {
	bn.nRef++
	if configTraceBnode {
		m.bntr.TraceRef(bn)
	}
	return &blockCandidate{
		Block: bn.block.(base.Block),
		m:     m,
	}
}

func (m *manager) unrefNode(bn *bnode) {
	bn.nRef--
	if configTraceBnode {
		m.bntr.TraceUnref(bn)
	}
	if bn.nRef == 0 {
		par := bn.parent
		m.removeNode(bn)
		if par != nil {
			for i, c := range par.children {
				if c == bn {
					last := len(par.children) - 1
					par.children[i] = par.children[last]
					par.children[last] = nil
					par.children = par.children[:last]
				}
			}
		}
	}
}

func (m *manager) _removeNode(bn *bnode) {
	for _, c := range bn.children {
		m._removeNode(c)
	}
	bn.in.dispose()
	bn.preexe.dispose()
	bn.parent = nil
	if configTraceBnode {
		m.bntr.TraceDispose(bn)
	}
	delete(m.nmap, string(bn.block.ID()))
}

func (m *manager) removeNode(bn *bnode) {
	for _, c := range bn.children {
		m._removeNode(c)
	}
	bn.in.dispose()
	bn.preexe.dispose()
	if bn.parent != nil {
		m.unrefNode(bn.parent)
		bn.parent = nil
	}
	if configTraceBnode {
		m.bntr.TraceDispose(bn)
	}
	delete(m.nmap, string(bn.block.ID()))
}

func (m *manager) removeNodeExcept(bn *bnode, except *bnode) {
	for _, c := range bn.children {
		if c == except {
			c.parent = nil
		} else {
			m._removeNode(c)
		}
	}
	bn.in.dispose()
	bn.preexe.dispose()
	if bn.parent != nil {
		m.unrefNode(bn.parent)
		bn.parent = nil
	}
	if configTraceBnode {
		m.bntr.TraceDispose(bn)
	}
	delete(m.nmap, string(bn.block.ID()))
}

func (t *task) cb(block module.BlockCandidate, err error) {
	cb := t._cb
	t.manager.syncer.callLater(func() {
		go cb(block, err)
	})
}

func (m *manager) _import(
	block module.BlockData,
	flags int,
	cb func(module.BlockCandidate, error),
) (*importTask, error) {
	bn := m.nmap[string(block.PrevID())]
	if bn == nil {
		return nil, errors.Errorf("InvalidPreviousID(%x)", block.PrevID())
	}
	csi, err := m.verifyNewBlock(block, bn.block)
	if err != nil {
		return nil, err
	}
	it := &importTask{
		block: block,
		csi:   csi,
		flags: flags,
		task: task{
			manager: m,
			_cb:     cb,
		},
	}
	it.state = executingIn
	it.in, err = bn.preexe.patch(it.block.PatchTransactions(), block, it)
	if err != nil {
		return nil, err
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

func (it *importTask) Cancel() bool {
	it.manager.syncer.begin()
	defer it.manager.syncer.end()

	switch it.state {
	case executingIn:
		it.stop()
	case validatingOut:
		it.stop()
	default:
		it.manager.log.Debugf("Cancel Import: Ignored\n")
		return false
	}
	it.manager.log.Debugf("Cancel Import: OK\n")
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
		var bn *bnode
		var ok bool
		if bn, ok = it.manager.nmap[string(it.block.ID())]; !ok {
			validatedBlock := it.block.NewBlock(it.in.mtransition())
			bn = &bnode{
				block:  validatedBlock,
				in:     it.in.newTransition(nil),
				preexe: it.out.newTransition(nil),
			}
			if configTraceBnode {
				it.manager.bntr.TraceNew(bn)
			}
			pbn := it.manager.nmap[string(it.block.PrevID())]
			it.manager.addNode(pbn, bn)
		}
		it.stop()
		it.state = validatedOut
		it.cb(it.manager.newCandidate(bn), err)
	}
}

func (it *importTask) onExecute(err error) {
	it.manager.syncer.callLaterInLock(func() {
		it._onExecute(err)
	})
}

func (it *importTask) _handleExecutionError(err error) {
	var tr *transition
	it.manager.log.Warnf("error during import : %+v", err)
	if it.flags&module.ImportByForce != 0 {
		tr, err = it.in.sync(it.block.Result(), it.block.NextValidatorsHash(), it)
		if err == nil {
			it.in.dispose()
			it.in = tr
			return
		}
	}
	it.stop()
	it.cb(nil, err)
}

func (it *importTask) _onExecute(err error) {
	if it.state == executingIn {
		if err != nil {
			it._handleExecutionError(err)
			return
		}
		err = it.in.verifyResult(it.block)
		if err != nil {
			it._handleExecutionError(err)
			return
		}
		validated := it.flags&module.ImportByForce != 0
		it.out, err = it.in.transit(it.block.NormalTransactions(), it.block, it.csi, it, validated)
		if err != nil {
			it.stop()
			it.cb(nil, err)
			return
		}
		it.state = validatingOut
		return
	}
}

func (m *manager) _propose(
	parentID []byte,
	votes module.CommitVoteSet,
	cb func(module.BlockCandidate, error),
) (*proposeTask, error) {
	bn := m.nmap[string(parentID)]
	if bn == nil {
		return nil, errors.Errorf("NoParentBlock(id=<%x>)", parentID)
	}
	csi, _, err := m.verifyProofForLastBlock(bn.block, votes)
	if err != nil {
		return nil, err
	}
	pt := &proposeTask{
		task: task{
			manager: m,
			_cb:     cb,
		},
		parentBlock: bn.block,
		votes:       votes,
		csi:         csi,
	}
	pt.state = executingIn
	bi := common.NewBlockInfo(bn.block.Height()+1, votes.Timestamp())
	patches := m.sm.GetPatches(
		bn.in.mtransition(),
		bi,
	)
	pt.in, err = bn.preexe.patch(patches, bi, pt)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func (pt *proposeTask) stop() {
	if pt.in != nil {
		pt.in.dispose()
	}
	pt.state = stopped
}

func (pt *proposeTask) Cancel() bool {
	pt.manager.syncer.begin()
	defer pt.manager.syncer.end()

	switch pt.state {
	case executingIn:
		pt.stop()
	default:
		pt.manager.log.Debugf("Cancel Propose: Ignored\n")
		return false
	}
	pt.manager.log.Debugf("Cancel Propose: OK\n")
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
	timestamp := pt.votes.Timestamp()
	if pt.manager.timestamper != nil {
		timestamp = pt.manager.timestamper.GetBlockTimestamp(height, timestamp)
	}
	tr, err := pt.in.propose(common.NewBlockInfo(height, timestamp), pt.csi, nil)
	if err != nil {
		pt.stop()
		pt.cb(nil, err)
		return
	}
	pmtr := pt.in.mtransition()
	mtr := tr.mtransition()
	block := pt.manager.activeHandlers.last().NewBlock(
		height,
		timestamp,
		pt.manager.chain.Wallet().Address(),
		pt.parentBlock,
		pmtr.LogsBloom(),
		pmtr.Result(),
		pmtr.PatchTransactions(),
		mtr.NormalTransactions(),
		pmtr.NextValidators(),
		pt.votes,
		pmtr.BTPSection(),
	)
	var bn *bnode
	var ok bool
	if bn, ok = pt.manager.nmap[string(block.ID())]; !ok {
		bn = &bnode{
			block:  block,
			in:     pt.in.newTransition(nil),
			preexe: tr,
		}
		if configTraceBnode {
			pt.manager.bntr.TraceNew(bn)
		}
		pbn := pt.manager.nmap[string(block.PrevID())]
		pt.manager.addNode(pbn, bn)
	} else {
		tr.dispose()
	}
	pt.stop()
	pt.state = validatedOut
	pt.cb(pt.manager.newCandidate(bn), nil)
}

// NewManager creates BlockManager.
func NewManager(
	chain module.Chain,
	timestamper module.Timestamper,
	handlers []base.BlockHandler,
) (module.BlockManager, error) {
	logger := chain.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "BM",
	})
	logger.Debugf("NewBlockManager\n")

	if handlers == nil {
		handlers = []base.BlockHandler{NewBlockV2Handler(chain)}
	}

	lm, err := chain.GetLocatorManager()
	if err != nil {
		return nil, err
	}
	m := &manager{
		chainContext: &chainContext{
			chain:   chain,
			sm:      chain.ServiceManager(),
			lm:      lm,
			log:     logger,
			running: true,
			srcUID:  module.GetSourceNetworkUID(chain),
		},
		nmap:        make(map[string]*bnode),
		cache:       newCache(ConfigCacheCap),
		timestamper: timestamper,
		handlers:    handlers,
	}
	m.activeHandlers = m.handlers.upTo(
		m.sm.GetNextBlockVersion(nil),
	)
	m.handlerContext.manager = m
	m.bntr.Logger = chain.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "BM|BNODE",
	})
	m.chainContext.trtr.Logger = chain.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "BM|TRANS",
	})
	chainPropBucket, err := m.bucketFor(db.ChainProperty)
	if err != nil {
		return nil, err
	}

	var height int64
	err = chainPropBucket.Get(db.Raw(keyLastBlockHeight), &height)
	if errors.NotFoundError.Equals(err) || (err == nil && height == 0) {
		if err := m.finalizeGenesis(); err != nil {
			return nil, err
		}
		m.activeHandlers = m.handlers.upTo(m.sm.GetNextBlockVersion(
			m.finalized.block.Result(),
		))
		if err := m.initializePCM(); err != nil {
			return nil, err
		}
		return m, nil
	} else if err != nil {
		return nil, err
	}
	lastFinalized, err := m.getBlockByHeightWithHandlerList(height, m.handlers)
	if err != nil {
		return nil, err
	}
	if nid, err := m.sm.GetNetworkID(lastFinalized.Result()); err != nil {
		return nil, err
	} else if int(nid) != m.chain.NID() {
		return nil, errors.InvalidNetworkError.Errorf(
			"InvalidNetworkID Database.NID=%#x Chain.NID=%#x", nid, m.chain.NID())
	}
	m.activeHandlers = m.handlers.upTo(m.sm.GetNextBlockVersion(
		lastFinalized.Result(),
	))

	var cid int
	if gBlock, err := m.getBlockByHeight(0); err == nil {
		if tx, err := gBlock.NormalTransactions().Get(0); err == nil {
			if gtx, ok := tx.(transaction.GenesisTransaction); ok {
				cid = gtx.CID()
			} else {
				return nil, errors.InvalidStateError.New("InvalidGenesisTransaction")
			}
		} else {
			return nil, errors.InvalidStateError.New("NoGenesisTransaction")
		}
	} else {
		if id, err := m.sm.GetChainID(lastFinalized.Result()); err != nil {
			return nil, err
		} else {
			cid = int(id)
		}
	}
	if cid != m.chain.CID() {
		return nil, errors.InvalidNetworkError.Errorf(
			"InvalidChainID Database.CID=%#x Chain.CID=%#x",
			cid, m.chain.CID())
	}

	mtr, _ := m.sm.CreateInitialTransition(lastFinalized.Result(), lastFinalized.NextValidators())
	if mtr == nil {
		return nil, err
	}
	tr := newInitialTransition(mtr, m.chainContext)
	bn := &bnode{
		block: lastFinalized,
		in:    tr,
	}
	if err := m.sm.Finalize(mtr, module.FinalizeResult); err != nil {
		return nil, err
	}
	csi, err := m.newConsensusInfo(lastFinalized)
	if err != nil {
		return nil, err
	}
	bn.preexe, err = tr.transit(lastFinalized.NormalTransactions(), lastFinalized, csi, nil, true)
	if err != nil {
		return nil, err
	}
	// This ensures that locators are flushed to the database.
	err = m.sm.Finalize(bn.preexe.mtransition(), module.FinalizeNormalTransaction)
	if err != nil {
		return nil, err
	}
	m.finalized = bn
	bn.nRef++
	if configTraceBnode {
		m.bntr.TraceNew(bn)
	}
	m.nmap[string(lastFinalized.ID())] = bn
	if err := m.initializePCM(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *manager) initializePCM() error {
	lastBlk := m.finalized.block
	nextPCM, err := lastBlk.NextProofContextMap()
	if err != nil {
		return err
	}
	m.nextPCM = nextPCM
	if lastBlk.Height() > 0 {
		blk, err := m.getBlockByHeight(lastBlk.Height() - 1)
		if err != nil {
			return err
		}
		pcm, err := blk.NextProofContextMap()
		if err != nil {
			return err
		}
		m.pcmForLastBlock = pcm
	} else {
		m.pcmForLastBlock = btp.ZeroProofContextMap
	}
	return nil
}

func (m *manager) Term() {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return
	}

	m.log.Debugf("Term block manager\n")

	m.removeNode(m.finalized)
	m.finalized = nil
	m.running = false
	for i := 0; i < len(m.finalizationCBs); i++ {
		cb := m.finalizationCBs[i]
		m.syncer.callLater(func() {
			cb(nil)
		})
	}
}

func (m *manager) GetBlock(id []byte) (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	return m.getBlock(id)
}

func (m *manager) getBlock(id []byte) (module.Block, error) {
	blk := m.cache.Get(id)
	if blk != nil {
		return blk.Copy(), nil
	}
	return m.doGetBlock(id)
}

func (m *manager) doGetBlock(id []byte) (module.Block, error) {
	for i := len(m.activeHandlers) - 1; i >= 0; i-- {
		h := m.activeHandlers[i]
		blk, err := h.GetBlock(id)
		if errors.Is(err, errors.ErrUnsupported) {
			continue
		} else if err != nil {
			return nil, err
		}
		m.cache.Put(blk)
		return blk.Copy(), nil
	}
	return nil, errors.NotFoundError.Errorf("block not found %x", id)
}

func (m *manager) Import(
	r io.Reader,
	flags int,
	cb func(module.BlockCandidate, error),
) (module.Canceler, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	m.log.Debugf("Import(%x)\n", r)

	v, r, err := PeekVersion(r)
	if err != nil {
		return nil, err
	}
	h, ok := m.activeHandlers.forVersion(v)
	if !ok {
		return nil, errors.UnsupportedError.Errorf("unsupported block version %d", v)
	}
	block, err := h.NewBlockDataFromReader(r)
	if err != nil {
		return nil, err
	}
	it, err := m._import(block, flags, cb)
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (m *manager) ImportBlock(
	block module.BlockData,
	flags int,
	cb func(module.BlockCandidate, error),
) (module.Canceler, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	m.log.Debugf("ImportBlock(%x)\n", block.ID())

	it, err := m._import(block, flags, cb)
	if err != nil {
		return nil, err
	}
	return it, nil
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

func (m *manager) finalizeGenesis() error {
	gns := m.chain.GenesisStorage()
	gt, err := gns.Type()
	if err != nil {
		return transaction.InvalidGenesisError.Wrap(err, "UnknownGenesisType")
	}
	switch gt {
	case module.GenesisNormal:
		_, err := m.finalizeGenesisBlock(nil, 0,
			m.chain.CommitVoteSetDecoder()(nil))
		return err
	case module.GenesisPruned:
		return errors.InvalidStateError.Errorf("start with PrunedGenesis without reset")
	}
	return errors.InvalidStateError.Errorf("InvalidGenesisType(type=%d)", gt)
}

func (m *manager) finalizeGenesisBlock(
	proposer module.Address,
	timestamp int64,
	votes module.CommitVoteSet,
) (block module.Block, err error) {
	m.log.Debugf("FinalizeGenesisBlock()\n")
	if m.finalized != nil {
		return nil, errors.InvalidStateError.New("InvalidState")
	}
	mtr, err := m.sm.CreateInitialTransition(nil, nil)
	if err != nil {
		return nil, err
	}
	in := newInitialTransition(mtr, m.chainContext)
	ch := make(chan error)
	gtxbs := m.chain.Genesis()
	gtx, err := m.sm.GenesisTransactionFromBytes(
		gtxbs, m.activeHandlers.last().Version(),
	)
	if err != nil {
		return nil, err
	}
	if !gtx.ValidateNetwork(m.chain.NID()) {
		return nil, errors.InvalidNetworkError.Errorf(
			"Invalid Network ID config=%#x genesis=%s", m.chain.NID(), gtxbs)
	}
	gtxl := m.sm.TransactionListFromSlice(
		[]module.Transaction{gtx}, m.activeHandlers.last().Version(),
	)
	m.syncer.begin()
	csi := common.NewConsensusInfo(nil, nil, nil)
	gtr, err := in.transit(gtxl, common.NewBlockInfo(0, timestamp), csi, &channelingCB{ch: ch}, true)
	if err != nil {
		m.syncer.end()
		return nil, err
	}
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
	bn.block = m.activeHandlers.last().NewBlock(
		genesisHeight,
		timestamp,
		proposer,
		nil,
		mtr.LogsBloom(),
		mtr.Result(),
		gtr.mtransition().PatchTransactions(),
		gtr.mtransition().NormalTransactions(),
		gtr.mtransition().NextValidators(),
		votes,
		mtr.BTPSection(),
	)
	if configTraceBnode {
		m.bntr.TraceNew(bn)
	}
	m.nmap[string(bn.block.ID())] = bn
	err = m.finalize(bn, false)
	if err != nil {
		return nil, err
	}
	err = m.sm.Finalize(gtr.mtransition(), module.FinalizeNormalTransaction|module.FinalizePatchTransaction|module.FinalizeResult|module.KeepingParent)
	if err != nil {
		return nil, err
	}
	return bn.block, nil
}

func (m *manager) Propose(
	parentID []byte,
	votes module.CommitVoteSet,
	cb func(module.BlockCandidate, error),
) (canceler module.Canceler, err error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	m.log.Debugf("Propose(<%x>, %v)\n", parentID, votes)

	pt, err := m._propose(parentID, votes, cb)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func (m *manager) bucketFor(id db.BucketID) (*db.CodedBucket, error) {
	return db.NewCodedBucket(m.db(), id, nil)
}

func (m *manager) Finalize(block module.BlockCandidate) error {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return errors.New("not running")
	}

	bn := m.nmap[string(block.ID())]
	if bn == nil || bn.parent != m.finalized {
		return errors.Errorf("InvalidStatusForBlock(id=<%x>", block.ID())
	}
	return m.finalize(bn, true)
}

func (m *manager) finalize(bn *bnode, updatePCM bool) error {
	// TODO notify import/propose error due to finalization
	// TODO update nmap
	block := bn.block

	if m.finalized != nil {
		m.removeNodeExcept(m.finalized, bn)
		err := m.sm.Finalize(
			bn.in.mtransition(),
			module.FinalizePatchTransaction|module.FinalizeResult,
		)
		if err != nil {
			return err
		}
	}
	err := m.sm.Finalize(bn.preexe.mtransition(), module.FinalizeNormalTransaction)
	if err != nil {
		return err
	}

	m.finalized = bn
	bn.nRef++
	if configTraceBnode {
		m.bntr.TraceRef(bn)
	}

	err = block.(base.BlockVersionSpec).FinalizeHeader(m.chain.Database())
	if err != nil {
		return err
	}
	nextVer := m.sm.GetNextBlockVersion(m.finalized.in.mtransition().Result())
	if m.activeHandlers.last().Version() != nextVer {
		m.activeHandlers = m.handlers.upTo(nextVer)
	}

	chainProp, err := m.bucketFor(db.ChainProperty)
	if err != nil {
		return err
	}
	if err = chainProp.Set(db.Raw(keyLastBlockHeight), block.Height()); err != nil {
		return err
	}

	if updatePCM {
		nextPCM, err := m.nextPCM.Update(m.finalized.block)
		if err != nil {
			return err
		}
		m.pcmForLastBlock = m.nextPCM
		m.nextPCM = nextPCM
	}

	m.cache.Put(m.finalized.block)

	m.log.Debugf("Finalize(%x)\n", block.ID())
	for i := 0; i < len(m.finalizationCBs); {
		cb := m.finalizationCBs[i]
		if cb(block) {
			last := len(m.finalizationCBs) - 1
			m.finalizationCBs[i] = m.finalizationCBs[last]
			m.finalizationCBs[last] = nil
			m.finalizationCBs = m.finalizationCBs[:last]
			continue
		}
		i++
	}
	return nil
}

func newProposer(bs []byte) (module.Address, error) {
	if bs != nil {
		addr, err := common.NewAddress(bs)
		if err != nil {
			return nil, errors.CriticalFormatError.Wrapf(err,
				"InvalidProposer(bs=%#x)", bs)
		} else {
			return addr, nil
		}
	}
	return nil, nil
}

func (m *manager) NewBlockDataFromReader(r io.Reader) (module.BlockData, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	v, r, err := PeekVersion(r)
	if err != nil {
		return nil, err
	}
	h, ok := m.activeHandlers.forVersion(v)
	if !ok {
		return nil, errors.UnsupportedError.Errorf("unsupported block version %d", v)
	}
	return h.NewBlockDataFromReader(r)
}

type transactionInfo struct {
	_txBlock module.Block
	_index   int
	_group   module.TransactionGroup
	_sm      ServiceManager
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

func (txInfo *transactionInfo) Transaction() (module.Transaction, error) {
	var txs module.TransactionList
	if txInfo._group == module.TransactionGroupNormal {
		txs = txInfo._txBlock.NormalTransactions()
	} else {
		txs = txInfo._txBlock.PatchTransactions()
	}
	return txs.Get(txInfo._index)
}

func (txInfo *transactionInfo) GetReceipt() (module.Receipt, error) {
	if txInfo._rBlock != nil {
		rl, err := txInfo._sm.ReceiptListFromResult(
			txInfo._rBlock.Result(), txInfo._group)
		if err != nil {
			return nil, err
		}
		rct, err := rl.Get(txInfo._index)
		if err != nil {
			return nil, err
		}
		return rct, nil
	} else {
		return nil, ErrResultNotFinalized
	}
}

func (m *manager) GetTransactionInfo(id []byte) (module.TransactionInfo, error) {
	loc, err := m.getTransactionLocator(id)
	if err != nil {
		return nil, err
	}

	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	return m.makeTransactionInfo(loc)
}

func (m *manager) getTransactionInfo(id []byte) (module.TransactionInfo, error) {
	loc, err := m.getTransactionLocator(id)
	if err != nil {
		return nil, err
	}
	return m.makeTransactionInfo(loc)
}

func (m *manager) makeTransactionInfo(loc *module.TransactionLocator) (module.TransactionInfo, error) {
	block, err := m.getBlockByHeight(loc.BlockHeight)
	if err != nil {
		return nil, errors.InvalidStateError.Wrapf(err, "block h=%d not found", loc.BlockHeight)
	}

	var rblock module.Block
	if loc.TransactionGroup == module.TransactionGroupNormal {
		if m.finalized.block.Height() < loc.BlockHeight+1 {
			rblock = nil
		} else {
			rblock, err = m.getBlockByHeight(loc.BlockHeight + 1)
			if err != nil {
				return nil, err
			}
		}
	} else {
		rblock = block
	}
	return &transactionInfo{
		_txBlock: block,
		_index:   loc.IndexInGroup,
		_group:   loc.TransactionGroup,
		_sm:      m.sm,
		_rBlock:  rblock,
	}, nil
}

func (m *manager) getTransactionLocator(id []byte) (*module.TransactionLocator, error) {
	if loc, err := m.chainContext.lm.GetLocator(id); err != nil {
		return nil, errors.NotFoundError.Wrapf(err, "not found tx=%#x", id)
	} else if loc == nil {
		return nil, errors.NotFoundError.Errorf("not found tx=%#x", id)
	} else {
		return loc, nil
	}
}

func (m *manager) SendTransactionAndWait(result []byte, height int64, txi interface{}) ([]byte, <-chan interface{}, error) {
	id, rc, err := m.sm.SendTransactionAndWait(result, height, txi)
	if err == nil {
		return id, rc, nil
	}

	if err != service.ErrCommittedTransaction {
		return nil, nil, err
	}

	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, nil, errors.New("not running")
	}
	c, err := m.waitTransactionResult(id)
	return id, c, err
}

func (m *manager) WaitTransactionResult(id []byte) (<-chan interface{}, error) {
	ch, err := m.sm.WaitTransactionResult(id)
	if err == nil {
		return ch, nil
	}
	if err != service.ErrCommittedTransaction {
		return nil, err
	}

	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}
	return m.waitTransactionResult(id)
}

func (m *manager) waitTransactionResult(id []byte) (<-chan interface{}, error) {
	loc, err := m.chainContext.lm.GetLocator(id)
	if err != nil {
		return nil, errors.NotFoundError.Wrap(err, "Not found")
	} else if loc == nil {
		return nil, errors.NotFoundError.New("Not found")
	}
	var rBlockHeight int64
	if loc.TransactionGroup == module.TransactionGroupNormal {
		rBlockHeight = loc.BlockHeight + 1
	} else {
		rBlockHeight = loc.BlockHeight
	}

	fc := make(chan interface{}, 1)
	if rBlockHeight > m.finalized.block.Height() {
		m.finalizationCBs = append(m.finalizationCBs, func(blk module.Block) bool {
			if blk == nil {
				close(fc)
				return true
			}
			if blk.Height() == rBlockHeight {
				if info, err := m.getTransactionInfo(id); err != nil {
					fc <- err
				} else {
					fc <- info
				}
				close(fc)
				return true
			}
			return false
		})
		return fc, nil
	}
	ti, err := m.getTransactionInfo(id)
	if err != nil {
		fc <- err
	} else {
		fc <- ti
	}
	close(fc)
	return fc, nil
}

func (m *manager) GetBlockByHeight(height int64) (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	return m.getBlockByHeight(height)
}

func (m *manager) getBlockByHeight(height int64) (module.Block, error) {
	return m.getBlockByHeightWithHandlerList(height, m.activeHandlers)
}

func (m *manager) getBlockByHeightWithHandlerList(
	height int64,
	hl handlerList,
) (module.Block, error) {
	blk := m.cache.GetByHeight(height)
	if blk != nil {
		return blk.Copy(), nil
	}
	return m.doGetBlockByHeight(height, hl)
}

func (m *manager) doGetBlockByHeight(
	height int64,
	hl handlerList,
) (module.Block, error) {
	if m.finalized != nil && height > m.finalized.block.Height() {
		return nil, errors.NotFoundError.Errorf("no block for %d", height)
	}
	// For now, assume all versions have same height to hash database structure
	dbase := m.chain.Database()
	headerHashByHeight, err := db.NewCodedBucket(
		dbase,
		db.BlockHeaderHashByHeight,
		nil,
	)
	if err != nil {
		return nil, err
	}
	hash, err := headerHashByHeight.GetBytes(height)
	if err != nil {
		return nil, err
	}
	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return nil, err
	}
	br := bytes.NewReader(headerBytes)
	v, r, err := PeekVersion(br)
	if err != nil {
		return nil, err
	}
	h, ok := hl.forVersion(v)
	if !ok {
		return nil, errors.UnsupportedError.Errorf("unsupported block version %d", v)
	}
	blk, err := h.NewBlockFromHeaderReader(r)
	if err != nil {
		return nil, err
	}
	m.cache.Put(blk)
	return blk.Copy(), err
}

func (m *manager) GetLastBlock() (module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	return m.finalized.block, nil
}

func (m *manager) WaitForBlock(height int64) (<-chan module.Block, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	bch := make(chan module.Block, 1)

	blk, err := m.getBlockByHeight(height)
	if err == nil {
		bch <- blk
		return bch, nil
	} else if !errors.NotFoundError.Equals(err) {
		return nil, err
	}

	m.finalizationCBs = append(m.finalizationCBs, func(blk module.Block) bool {
		if blk == nil {
			close(bch)
			return true
		}
		if blk.Height() == height {
			bch <- blk
			close(bch)
			return true
		}
		return false
	})
	return bch, nil
}

func (m *manager) WaitForTransaction(parentID []byte, cb func()) (bool, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return false, errors.New("not running")
	}

	bn := m.nmap[string(parentID)]
	if bn == nil {
		return false, nil
	}
	return m.sm.WaitForTransaction(bn.in.mtransition(), bn.block, cb), nil
}

func (m *manager) DupBlockCandidate(bc *blockCandidate) *blockCandidate {
	m.syncer.begin()
	defer m.syncer.end()

	bn := m.nmap[string(bc.ID())]
	if bn != nil {
		bn.nRef++
		if configTraceBnode {
			m.bntr.TraceRef(bn)
		}
	}
	res := *bc
	return &res
}

func (m *manager) DisposeBlockCandidate(bc *blockCandidate) {
	m.syncer.begin()
	defer m.syncer.end()

	bn := m.nmap[string(bc.ID())]
	if bn == nil {
		return
	}
	m.unrefNode(bn)
}

func hasBits(v int, bits int) bool {
	return (v & bits) == bits
}

func (m *manager) ExportGenesis(blk module.BlockData, votes module.CommitVoteSet, gsw module.GenesisStorageWriter) error {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return errors.New("not running")
	}

	height := blk.Height()

	if votes == nil {
		if nblk, err := m.getBlockByHeight(height + 1); err != nil {
			return errors.Wrapf(err, "fail to get next block(height=%d) for votes", height+1)
		} else {
			votes = nblk.Votes()
		}
	}

	cid, err := m.sm.GetChainID(blk.Result())
	if err != nil {
		return errors.Wrap(err, "fail to get CID")
	}

	nid, err := m.sm.GetNetworkID(blk.Result())
	if err != nil {
		return errors.Wrap(err, "fail to get NID")
	}

	pg := &gs.PrunedGenesis{
		CID:    common.HexInt32{Value: int32(cid)},
		NID:    common.HexInt32{Value: int32(nid)},
		Height: common.HexInt64{Value: height},
		Block:  blk.ID(),
		Votes:  votes.Hash(),
	}
	g, err := json.Marshal(pg)
	if err != nil {
		return errors.Wrapf(err, "fail to marshal genesis=%+v", pg)
	}

	if err := gsw.WriteGenesis(g); err != nil {
		return errors.Wrap(err, "fail to write genesis")
	}

	if _, err := gsw.WriteData(votes.Bytes()); err != nil {
		return errors.Wrap(err, "fail to write votes")
	}
	return nil
}

func (m *manager) ExportBlocks(from, to int64, dst db.Database, cb module.ProgressCallback) error {
	return m.ExportBlocksWithFlag(from, to, dst, exportAll, cb)
}

func (m *manager) ExportBlocksWithFlag(from, to int64, dst db.Database, flag int, cb module.ProgressCallback) error {
	ctx := merkle.NewCopyContext(m.db(), dst)
	ctx.SetProgressCallback(cb)
	if hasBits(flag, exportValidator) && from > 0 {
		// export the block for validators
		blk, err := m.GetBlockByHeight(from - 1)
		if err != nil {
			return errors.Wrapf(err, "fail to get previous block height=%d", from-1)
		}
		if err := m._export(blk, ctx, flag); err != nil {
			return errors.Wrapf(err, "fail to export block height=%d", blk.Height())
		}
		// export the block for voters
		if pid := blk.PrevID(); len(pid) > 0 {
			pblk, err := m.GetBlockByHeight(from - 2)
			if err != nil {
				return errors.Wrapf(err, "fail to get p-previous block height=%d", from-2)
			}
			if err := m._export(pblk, ctx, flag); err != nil {
				return errors.Wrapf(err, "fail to export block height=%d", pblk.Height())
			}
		}
	}

	for h := from; h <= to; h++ {
		blk, err := m.GetBlockByHeight(h)
		if err != nil {
			return errors.Wrapf(err, "fail to get a block height=%d", h)
		}
		if err := m._export(blk, ctx, flag); err != nil {
			return errors.Wrapf(err, "fail to export block height=%d", blk.Height())
		}
	}
	return nil
}

func (m *manager) _export(blk module.Block, ctx *merkle.CopyContext, flag int) error {
	ctx.SetHeight(blk.Height())
	if hasBits(flag, exportResult) {
		if err := m.sm.ExportResult(blk.Result(), blk.NextValidatorsHash(), ctx.TargetDB()); err != nil {
			return err
		}
	}
	if hasBits(flag, exportTransaction) {
		transaction.NewTransactionListWithBuilder(ctx.Builder(), blk.PatchTransactions().Hash())
		transaction.NewTransactionListWithBuilder(ctx.Builder(), blk.NormalTransactions().Hash())
		if err := ctx.Run(); err != nil {
			return err
		}
	}
	if hasBits(flag, exportBlock) {
		buf := bytes.NewBuffer(nil)
		if err := blk.MarshalHeader(buf); err != nil {
			return err
		}
		if err := ctx.Set(db.BytesByHash, blk.ID(), buf.Bytes()); err != nil {
			return err
		}
		if err := ctx.Copy(db.BytesByHash, blk.Votes().Hash()); err != nil {
			return err
		}
		if err := ctx.Copy(db.BytesByHash, blk.NextValidatorsHash()); err != nil {
			return err
		}
	}
	if hasBits(flag, exportIndex|exportBlock) {
		hb := codec.BC.MustMarshalToBytes(blk.Height())
		if err := ctx.Copy(db.BlockHeaderHashByHeight, hb); err != nil {
			return err
		}
		if err := ctx.Set(db.ChainProperty, []byte(keyLastBlockHeight), hb); err != nil {
			return err
		}
	}
	if hasBits(flag, exportIndex|exportTransaction) {
		txs := blk.NormalTransactions()
		for itr := txs.Iterator(); itr.Has(); m.log.Must(itr.Next()) {
			tx, _, err := itr.Get()
			if err != nil {
				return err
			}
			if err := ctx.Copy(db.TransactionLocatorByHash, tx.ID()); err != nil {
				return err
			}
		}
		txs = blk.PatchTransactions()
		for itr := txs.Iterator(); itr.Has(); m.log.Must(itr.Next()) {
			tx, _, err := itr.Get()
			if err != nil {
				return err
			}
			if err := ctx.Copy(db.TransactionLocatorByHash, tx.ID()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *manager) GetGenesisData() (module.Block, module.CommitVoteSet, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, nil, errors.New("not running")
	}

	storage := m.chain.GenesisStorage()
	if genesisType, err := storage.Type(); err != nil {
		return nil, nil, err
	} else {
		if genesisType != module.GenesisPruned {
			return nil, nil, nil
		}
	}
	genesis := new(gs.PrunedGenesis)
	if err := json.Unmarshal(storage.Genesis(), genesis); err != nil {
		return nil, nil, transaction.InvalidGenesisError.Wrap(err, "invalid genesis")
	}
	bs, err := storage.Get(genesis.Votes)
	if err != nil {
		return nil, nil, transaction.InvalidGenesisError.Wrapf(err, "fail to get votes for hash=%x", genesis.Votes)
	}
	voteSetDecoder := m.chain.CommitVoteSetDecoder()
	block, err := m.getBlock(genesis.Block)
	if err != nil {
		return nil, nil, transaction.InvalidGenesisError.Wrapf(err, "fail to get block for id=%x", genesis.Block)
	}
	return block, voteSetDecoder(bs), nil
}

func (m *manager) NewConsensusInfo(blk module.Block) (module.ConsensusInfo, error) {
	m.syncer.begin()
	defer m.syncer.end()

	if !m.running {
		return nil, errors.New("not running")
	}

	return m.newConsensusInfo(blk)
}

func (m *manager) newConsensusInfo(blk module.Block) (module.ConsensusInfo, error) {
	pblk, err := m.getBlockByHeight(blk.Height() - 1)
	if err != nil {
		return nil, err
	}
	vl, err := pblk.(base.BlockVersionSpec).GetVoters(m.handlerContext)
	if err != nil {
		return nil, err
	}
	voted, err := blk.Votes().VerifyBlock(pblk, vl)
	if err != nil {
		return nil, err
	}
	return common.NewConsensusInfo(pblk.Proposer(), vl, voted), nil
}

func GetBlockHeaderHashByHeight(
	dbase db.Database,
	c codec.Codec,
	height int64,
) ([]byte, error) {
	headerHashByHeight, err := db.NewCodedBucket(
		dbase, db.BlockHeaderHashByHeight, c,
	)
	if err != nil {
		return nil, err
	}
	return headerHashByHeight.GetBytes(height)
}

func GetBlockVersion(
	dbase db.Database,
	c codec.Codec,
	height int64,
) (int, error) {
	if c == nil {
		c = codec.BC
	}
	headerHashByHeight, err := db.NewCodedBucket(
		dbase, db.BlockHeaderHashByHeight, c,
	)
	if err != nil {
		return -1, err
	}
	hash, err := headerHashByHeight.GetBytes(height)
	if err != nil {
		return -1, err
	}
	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return -1, err
	}

	br := bytes.NewReader(headerBytes)
	dec := c.NewDecoder(br)
	defer func() {
		_ = dec.Close()
	}()
	d2, err := dec.DecodeList()
	if err != nil {
		return -1, err
	}
	var version int
	if err = d2.Decode(&version); err != nil {
		return -1, err
	}
	return version, nil
}

func getHeaderField(dbase db.Database, c codec.Codec, height int64, index int) ([]byte, error) {
	if c == nil {
		c = codec.BC
	}
	headerHashByHeight, err := db.NewCodedBucket(
		dbase, db.BlockHeaderHashByHeight, c,
	)
	if err != nil {
		return nil, err
	}
	hash, err := headerHashByHeight.GetBytes(height)
	if err != nil {
		return nil, err
	}
	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(headerBytes)
	dec := c.NewDecoder(br)
	defer func() {
		_ = dec.Close()
	}()

	d2, err := dec.DecodeList()
	if err != nil {
		return nil, err
	}
	if index > 0 {
		if err = d2.Skip(index); err != nil {
			return nil, err
		}
	}
	var str []byte
	if err = d2.Decode(&str); err != nil {
		return nil, err
	}
	return str, nil
}

func GetCommitVoteListBytesForHeight(
	dbase db.Database,
	c codec.Codec,
	height int64,
) ([]byte, error) {
	votesHash, err := getHeaderField(dbase, c, height+1, 5)
	if err != nil {
		return nil, err
	}
	return db.DoGetWithBucketID(dbase, db.BytesByHash, votesHash)
}

func GetBlockResultByHeight(
	dbase db.Database,
	c codec.Codec,
	height int64,
) ([]byte, error) {
	return getHeaderField(dbase, c, height, 10)
}

func GetBTPDigestFromResult(
	dbase db.Database,
	c codec.Codec,
	result []byte,
) (module.BTPDigest, error) {
	if c == nil {
		c = codec.BC
	}
	dh, err := service.BTPDigestHashFromResult(result)
	if err != nil {
		return nil, err
	}
	if dh == nil {
		return btp.ZeroDigest, nil
	}
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	bs, err := bk.Get(dh)
	if err != nil {
		return nil, err
	}
	return btp.NewDigestFromBytes(bs)
}

func GetNextValidatorsByHeight(
	dbase db.Database,
	c codec.Codec,
	height int64,
) (module.ValidatorList, error) {
	if c == nil {
		c = codec.BC
	}
	validatorsHash, err := getHeaderField(dbase, c, height, 6)
	if err != nil {
		return nil, err
	}
	return state.ValidatorSnapshotFromHash(dbase, validatorsHash)
}

func GetLastHeightWithCodec(dbase db.Database, c codec.Codec) (int64, error) {
	if c == nil {
		c = codec.BC
	}
	bk, err := dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return 0, err
	}
	bs, err := bk.Get([]byte(keyLastBlockHeight))
	if err != nil || bs == nil {
		return 0, err
	}
	var height int64
	if _, err := c.UnmarshalFromBytes(bs, &height); err != nil {
		return 0, err
	}
	return height, nil
}

func GetLastHeight(dbase db.Database) (int64, error) {
	return GetLastHeightWithCodec(dbase, nil)
}

func GetLastHeightOf(dbase db.Database) int64 {
	height, _ := GetLastHeight(dbase)
	return height
}

func ResetDB(d db.Database, c codec.Codec, height int64) error {
	return SetLastHeight(d, c, height)
}

func SetLastHeight(dbase db.Database, c codec.Codec, height int64) error {
	bk, err := dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return err
	}
	if c == nil {
		c = codec.BC
	}
	err = bk.Set([]byte(keyLastBlockHeight), c.MustMarshalToBytes(height))
	if err != nil {
		return err
	}
	return nil
}
