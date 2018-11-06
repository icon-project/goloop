package block

import (
	"bytes"
	"errors"
	"sync"

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
	mutex *sync.Mutex
	sm    module.ServiceManager
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

func (ti *transitionImpl) newTransition(cb transitionCallback) *transition {
	if ti._valErr != nil {
		cb.onValidate(*(ti._valErr))
	}
	if ti._exeErr != nil {
		cb.onValidate(*(ti._exeErr))
	}
	if ti.running() {
		ti._cbs = append(ti._cbs, cb)
	}
	ti._nRef++
	return &transition{ti, cb}
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
		for i, c := range parent._children {
			if c == ti {
				last := len(parent._children) - 1
				parent._children[i] = parent._children[last]
				parent._children[last] = nil
				parent._children = parent._children[:last]
				break
			}
		}
		ti._mtransition = nil
	}
}

func (ti *transitionImpl) OnValidate(tr module.Transition, err error) {
	ti._setting.mutex.Lock()
	defer ti._setting.mutex.Unlock()
	if !ti.running() {
		return
	}
	ti._valErr = &err
	if err != nil {
		ti._cbs = nil
	}
	for _, cb := range ti._cbs {
		cb.onValidate(err)
	}
}

func (ti *transitionImpl) OnExecute(tr module.Transition, err error) {
	ti._setting.mutex.Lock()
	defer ti._setting.mutex.Unlock()
	if !ti.running() {
		return
	}
	ti._exeErr = &err
	ti._cbs = nil
	for _, cb := range ti._cbs {
		cb.onExecute(err)
	}
}

func (ti *transitionImpl) _addChild(mtr module.Transition) *transitionImpl {
	cti := &transitionImpl{
		_setting:     ti._setting,
		_mtransition: mtr,
		_parent:      ti,
	}
	var err error
	cti._canceler, err = mtr.Execute(cti)
	if err != nil {
		// TODO log
		return nil
	}
	ti._children = append(ti._children, cti)
	return cti
}

func (ti *transitionImpl) patch(
	patches module.TransactionList,
) *transitionImpl {
	for _, c := range ti._parent._children {
		if c._mtransition.PatchTransactions().Equal(patches) {
			return c
		}
	}
	c := ti._children[len(ti._children)-1]
	pmtr := ti._setting.sm.PatchTransition(c._mtransition, patches)
	return ti._parent._addChild(pmtr)
}

func (ti *transitionImpl) transit(
	txs module.TransactionList,
) *transitionImpl {
	cmtr, err := ti._setting.sm.CreateTransition(ti._mtransition, txs)
	if err != nil {
		return nil
	}
	return ti._addChild(cmtr)
}

func (ti *transitionImpl) propose() *transitionImpl {
	cmtr, err := ti._setting.sm.ProposeTransition(ti._mtransition)
	if err != nil {
		return nil
	}
	return ti._addChild(cmtr)
}

func (ti *transitionImpl) verifyResult(block module.Block) error {
	mtr := ti._mtransition
	if !bytes.Equal(mtr.Result(), block.Result()) {
		return errors.New("bad result")
	}
	if !bytes.Equal(mtr.LogBloom(), block.LogBloom()) {
		return errors.New("bad log bloom")
	}
	if !bytes.Equal(mtr.NextValidators().Hash(), block.NextValidators().Hash()) {
		return errors.New("bad next validators")
	}
	if !bytes.Equal(mtr.PatchReceipts().Hash(), block.PatchReceipts().Hash()) {
		return errors.New("bad patch receipts")
	}
	if !bytes.Equal(mtr.NormalReceipts().Hash(), block.NormalReceipts().Hash()) {
		return errors.New("bad normal receipts")
	}
	return nil
}

type transition struct {
	_ti *transitionImpl
	_cb transitionCallback
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
	res := tr._ti.cancel(tr._cb)
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
	ti := tr._ti.patch(patches)
	return ti.newTransition(cb)
}

func (tr *transition) transit(
	txs module.TransactionList,
	cb transitionCallback,
) *transition {
	if tr._ti == nil {
		return nil
	}
	ti := tr._ti.transit(txs)
	return ti.newTransition(cb)
}

func (tr *transition) propose(
	cb transitionCallback,
) *transition {
	if tr._ti == nil {
		return nil
	}
	ti := tr._ti.propose()
	return ti.newTransition(cb)
}

func (tr *transition) verifyResult(block module.Block) error {
	return tr._ti.verifyResult(block)
}

func newInitialTransition(
	mtransition module.Transition,
	mutex *sync.Mutex,
	sm module.ServiceManager,
) *transition {
	var nilErr error
	ti := &transitionImpl{
		_setting: &setting{
			mutex: mutex,
			sm:    sm,
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
