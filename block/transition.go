package block

import (
	"bytes"
	"errors"
	"log"

	"github.com/icon-project/goloop/module"
)

type exeState int

const (
	notValidated exeState = iota
	validated
	executed
)

type transitionCallback interface {
	onValidate(error)
	onExecute(error)
}

type setting struct {
	syncer *syncer
	sm     module.ServiceManager
}

type transitionImpl struct {
	_setting     *setting
	_mtransition module.Transition    // nil iff disposed
	_canceler    func() bool          // can be nil if not cancelable
	_cbs         []transitionCallback // empty iff not running
	_valErr      *error
	_exeErr      *error
	_nRef        int             // count only transactions
	_parent      *transitionImpl // nil if parent is not accessible
	_children    []*transitionImpl
}

type transition struct {
	_ti *transitionImpl // nil iff disposed

	// nil iff cb was called, tr was canceled or original cb was nil
	_cb transitionCallback
}

func (ti *transitionImpl) running() bool {
	return len(ti._cbs) != 0
}

func (ti *transitionImpl) err() error {
	if ti._valErr != nil && *ti._valErr != nil {
		return *ti._valErr
	}
	if ti._exeErr != nil && *ti._exeErr != nil {
		return *ti._exeErr
	}
	return nil
}

func (ti *transitionImpl) exeState() exeState {
	if ti._valErr == nil || *ti._valErr != nil {
		return notValidated
	}
	if ti._exeErr == nil || *ti._exeErr != nil {
		return validated
	}
	return executed
}

func (ti *transitionImpl) _newTransition(cb transitionCallback) *transition {
	tr := &transition{ti, cb}
	ti._nRef++
	if ti._valErr != nil {
		tr.onValidate(*(ti._valErr))
	}
	if ti._exeErr != nil {
		tr.onExecute(*(ti._exeErr))
	}
	if ti.running() {
		ti._cbs = append(ti._cbs, tr)
	}
	return tr
}

func (ti *transitionImpl) cancel(tncb transitionCallback) bool {
	if !ti.running() {
		return false
	}
	for i, cb := range ti._cbs {
		if cb == tncb {
			last := len(ti._cbs) - 1
			ti._cbs[i] = ti._cbs[last]
			ti._cbs[last] = nil
			ti._cbs = ti._cbs[:last]
			if len(ti._cbs) == 0 {
				ti._canceler()
			}
			return true
		}
	}
	return false
}

func (ti *transitionImpl) unref() {
	ti._nRef--
	if ti._nRef == 0 {
		for _, c := range ti._children {
			c._parent = nil
		}
		ti._children = nil
		parent := ti._parent
		if parent != nil {
			for i, c := range parent._children {
				if c == ti {
					last := len(parent._children) - 1
					parent._children[i] = parent._children[last]
					parent._children[last] = nil
					parent._children = parent._children[:last]
					break
				}
			}
		}
		ti._mtransition = nil
	}
}

func (ti *transitionImpl) OnValidate(tr module.Transition, err error) {
	ti._setting.syncer.begin()
	defer ti._setting.syncer.end()
	if !ti.running() {
		return
	}
	ti._valErr = &err
	cbs := make([]transitionCallback, len(ti._cbs))
	copy(cbs, ti._cbs)
	if err != nil {
		ti._cbs = nil
	}
	for _, cb := range cbs {
		cb.onValidate(err)
	}
}

func (ti *transitionImpl) OnExecute(tr module.Transition, err error) {
	ti._setting.syncer.begin()
	defer ti._setting.syncer.end()
	if !ti.running() {
		return
	}
	ti._exeErr = &err
	cbs := ti._cbs
	ti._cbs = nil
	for _, cb := range cbs {
		cb.onExecute(err)
	}
}

func (ti *transitionImpl) _addChild(
	mtr module.Transition,
	cb transitionCallback,
) *transition {
	cti := &transitionImpl{
		_setting:     ti._setting,
		_mtransition: mtr,
		_parent:      ti,
		_nRef:        1,
	}
	tr := &transition{cti, cb}
	cti._cbs = append(cti._cbs, tr)
	var err error
	cti._canceler, err = mtr.Execute(cti)
	if err != nil {
		log.Println("Transition.Execute failed : ", err)
		return nil
	}
	ti._children = append(ti._children, cti)
	return tr
}

func (ti *transitionImpl) patch(
	patches module.TransactionList,
	cb transitionCallback,
) *transition {
	for _, c := range ti._parent._children {
		if c._mtransition.PatchTransactions().Equal(patches) {
			return c._newTransition(cb)
		}
	}
	c := ti._children[len(ti._children)-1]
	pmtr := ti._setting.sm.PatchTransition(c._mtransition, patches)
	return ti._parent._addChild(pmtr, cb)
}

func (ti *transitionImpl) transit(
	txs module.TransactionList,
	bi module.BlockInfo,
	cb transitionCallback,
) *transition {
	cmtr, err := ti._setting.sm.CreateTransition(ti._mtransition, txs, bi)
	if err != nil {
		log.Println("ServiceManager.CreateTransition failed : ", err)
		return nil
	}
	return ti._addChild(cmtr, cb)
}

func (ti *transitionImpl) propose(bi module.BlockInfo, cb transitionCallback) *transition {
	cmtr, err := ti._setting.sm.ProposeTransition(ti._mtransition, bi)
	if err != nil {
		log.Println("ServiceManager.ProposeTransition failed : ", err)
		return nil
	}
	return ti._addChild(cmtr, cb)
}

func (ti *transitionImpl) verifyResult(block module.Block) error {
	mtr := ti._mtransition
	if !bytes.Equal(mtr.Result(), block.Result()) {
		return errors.New("bad result")
	}
	if !mtr.LogBloom().Equal(block.LogBloom()) {
		return errors.New("bad log bloom")
	}
	if !bytes.Equal(mtr.NextValidators().Hash(), block.NextValidators().Hash()) {
		return errors.New("bad next validators")
	}
	return nil
}

func (tr *transition) onValidate(err error) {
	cb := tr._cb
	if err != nil {
		tr._cb = nil
	}
	if cb != nil {
		cb.onValidate(err)
	}
}

func (tr *transition) onExecute(err error) {
	cb := tr._cb
	tr._cb = nil
	if cb != nil {
		cb.onExecute(err)
	}
}

func (tr *transition) dispose() {
	if tr._ti == nil {
		return
	}
	tr.cancel()
	ti := tr._ti
	tr._ti = nil
	ti.unref()
}

func (tr *transition) cancel() bool {
	if tr._ti == nil {
		return false
	}
	res := tr._ti.cancel(tr)
	tr._cb = nil
	return res
}

func (tr *transition) patch(
	patches module.TransactionList,
	cb transitionCallback,
) *transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti.patch(patches, cb)
}

func (tr *transition) transit(
	txs module.TransactionList,
	bi module.BlockInfo,
	cb transitionCallback,
) *transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti.transit(txs, bi, cb)
}

func (tr *transition) propose(bi module.BlockInfo, cb transitionCallback) *transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti.propose(bi, cb)
}

func (tr *transition) newTransition(cb transitionCallback) *transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti._newTransition(cb)
}

func (tr *transition) verifyResult(block module.Block) error {
	if tr._ti == nil {
		return nil
	}
	return tr._ti.verifyResult(block)
}

func (tr *transition) mtransition() module.Transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti._mtransition
}

func newInitialTransition(
	mtransition module.Transition,
	syncer *syncer,
	sm module.ServiceManager,
) *transition {
	var nilErr error
	ti := &transitionImpl{
		_setting: &setting{
			syncer: syncer,
			sm:     sm,
		},
		_mtransition: mtransition,
		_valErr:      &nilErr,
		_exeErr:      &nilErr,
		_nRef:        1,
	}
	tr := &transition{
		_ti: ti,
		_cb: nil,
	}
	return tr
}
