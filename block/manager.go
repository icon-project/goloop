package block

import (
	"bytes"
	"io"
	"time"

	"github.com/icon-project/goloop/common/db"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type bnode struct {
	parent   *bnode
	children []*bnode
	block    module.Block
	in       *transition
	preexe   *transition
}

type manager struct {
	chain     module.Chain
	sm        module.ServiceManager
	syncer    *syncer
	finalized *bnode
	nmap      map[string]*bnode
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
	var blockV2ForCodec blockV2ForCodec
	v2codec.Unmarshal(r, &blockV2ForCodec)
	block := newBlockV2(&blockV2ForCodec)
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
	bparam := &blockV2Param{
		parent:             pt.parentBlock,
		timestamp:          time.Now(),
		proposer:           pt.manager.chain.GetWallet().GetAddress(),
		logBloom:           pmtr.LogBloom(),
		result:             pmtr.Result(),
		patchTransactions:  mtr.PatchTransactions(),
		normalTransactions: mtr.NormalTransactions(),
		nextValidators:     pmtr.NextValidators(),
		votes:              pt.votes,
	}
	block := newBlockV2FromParam(bparam)
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
	return &manager{
		chain: chain,
		sm:    sm,
	}
}

func (m *manager) GetBlock(id []byte) module.Block {
	bucket, err := m.db().GetBucket(db.BlockHeaderByHash)
	if err != nil {
		panic("cannot get bucket BlockHeaderByHash")
	}
	headerBytes, err := bucket.Get(id)
	if err != nil {
		return nil
	}
	if headerBytes != nil {
		var blockV2HeaderForCodec blockV2HeaderForCodec
		err = v2codec.Unmarshal(
			bytes.NewReader(headerBytes),
			blockV2HeaderForCodec,
		)
		if err != nil {
			return nil
		}
		return newBlockV2FromHeaderForCodec(&blockV2HeaderForCodec)
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
	return nil
}

// TODO dummy
func (m *manager) GetTransactionInfo(id []byte) module.TransactionInfo {
	return nil
}
