package block

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	configTraceTransition = false
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

type transitionImpl struct {
	_chainContext *chainContext
	_mtransition  module.Transition    // nil iff disposed
	_canceler     func() bool          // can be nil if not cancelable
	_cbs          []transitionCallback // empty iff not running
	_valErr       *error
	_exeErr       *error
	_nRef         int             // count only transactions
	_parent       *transitionImpl // nil if parent is not accessible
	_children     []*transitionImpl
	_sync         bool // true if sync transition
}

func (ti *transitionImpl) RefCount() int {
	return ti._nRef
}

func (ti *transitionImpl) String() string {
	bi := ti._mtransition.BlockInfo()
	if bi != nil {
		return fmt.Sprintf("%p{nRef:%d H:%d}", ti, ti._nRef, bi.Height())
	}
	return fmt.Sprintf("%p{nRef:%d}", ti, ti._nRef)
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
	if configTraceTransition {
		ti._chainContext.trtr.TraceRef(ti)
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
	if configTraceTransition {
		ti._chainContext.trtr.TraceUnref(ti)
	}
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
	ti._chainContext.syncer.begin()
	defer ti._chainContext.syncer.end()
	if !ti._chainContext.running || !ti.running() {
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
	ti._chainContext.syncer.begin()
	defer ti._chainContext.syncer.end()
	if !ti._chainContext.running || !ti.running() {
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
) (*transition, error) {
	cti := &transitionImpl{
		_chainContext: ti._chainContext,
		_mtransition:  mtr,
		_parent:       ti,
		_nRef:         1,
	}
	tr := &transition{cti, cb}
	cti._cbs = append(cti._cbs, tr)
	var err error
	cti._canceler, err = mtr.Execute(cti)
	if err != nil {
		return nil, err
	}
	ti._children = append(ti._children, cti)
	if configTraceTransition {
		ti._chainContext.trtr.TraceNew(cti)
	}
	return tr, nil
}

func (ti *transitionImpl) patch(
	patches module.TransactionList,
	bi module.BlockInfo,
	cb transitionCallback,
) (*transition, error) {
	nmtr := ti._chainContext.sm.PatchTransition(ti._mtransition, patches, bi)
	// a sync transition has higher priority
	for _, s := range ti._parent._children {
		if s._sync && nmtr.Equal(s._mtransition) {
			return s._newTransition(cb), nil
		}
	}
	for _, s := range ti._parent._children {
		if nmtr.Equal(s._mtransition) {
			return s._newTransition(cb), nil
		}
	}
	return ti._parent._addChild(nmtr, cb)
}

func (ti *transitionImpl) transit(
	txs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	cb transitionCallback,
	validated bool,
) (*transition, error) {
	cmtr, err := ti._chainContext.sm.CreateTransition(ti._mtransition, txs, bi, csi, validated)
	if err != nil {
		return nil, err
	}
	// a sync transition has higher priority
	for _, c := range ti._children {
		if c._sync && cmtr.Equal(c._mtransition) {
			return c._newTransition(cb), nil
		}
	}
	for _, c := range ti._children {
		if cmtr.Equal(c._mtransition) {
			return c._newTransition(cb), nil
		}
	}

	return ti._addChild(cmtr, cb)
}

func (ti *transitionImpl) propose(bi module.BlockInfo, csi module.ConsensusInfo, cb transitionCallback) (*transition, error) {
	cmtr, err := ti._chainContext.sm.ProposeTransition(ti._mtransition, bi, csi)
	if err != nil {
		return nil, err
	}
	// a sync transition has higher priority
	for _, c := range ti._children {
		if c._sync && cmtr.Equal(c._mtransition) {
			return c._newTransition(cb), nil
		}
	}
	for _, c := range ti._children {
		if cmtr.Equal(c._mtransition) {
			return c._newTransition(cb), nil
		}
	}
	return ti._addChild(cmtr, cb)
}

func (ti *transitionImpl) sync(result []byte, vlHash []byte, cb transitionCallback) (*transition, error) {
	cmtr := ti._chainContext.sm.CreateSyncTransition(ti._mtransition, result, vlHash, false)
	if cmtr == nil {
		return nil, errors.New("fail to createSyncTransition")
	}
	res, err := ti._parent._addChild(cmtr, cb)
	if err != nil {
		return nil, err
	}
	res._ti._sync = true
	return res, nil
}

func (ti *transitionImpl) verifyResult(block module.BlockData) error {
	mtr := ti._mtransition
	if !bytes.Equal(mtr.Result(), block.Result()) {
		return errors.Errorf("bad result calc:%x block:%x", mtr.Result(), block.Result())
	}
	if !mtr.LogsBloom().Equal(block.LogsBloom()) {
		return errors.Errorf("bad log bloom calc:%x block:%x", mtr.LogsBloom().Bytes(), block.LogsBloom().Bytes())
	}
	if !bytes.Equal(mtr.NextValidators().Hash(), block.NextValidatorsHash()) {
		return errors.Errorf("bad next validators calc:%x block:%x", mtr.NextValidators().Hash(), block.NextValidatorsHash())
	}
	bs := mtr.BTPSection()
	trNSF := bs.Digest().NetworkSectionFilter()
	blkNSF := block.NetworkSectionFilter()
	if !bytes.Equal(trNSF.Bytes(), blkNSF.Bytes()) {
		return errors.Errorf("bad nsFilter calc:%x block:%x", trNSF, blkNSF)
	}
	bd, err := block.BTPDigest()
	if err != nil {
		return err
	}
	if !bytes.Equal(bs.Digest().Hash(), bd.Hash()) {
		return errors.Errorf("bad digest hash calc:%x block:%x", bs.Digest().Hash(), bd.Hash())
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
	bi module.BlockInfo,
	cb transitionCallback,
) (*transition, error) {
	if tr._ti == nil {
		return nil, nil
	}
	return tr._ti.patch(patches, bi, cb)
}

func (tr *transition) transit(
	txs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	cb transitionCallback,
	validated bool,
) (*transition, error) {
	if tr._ti == nil {
		return nil, nil
	}
	return tr._ti.transit(txs, bi, csi, cb, validated)
}

func (tr *transition) propose(bi module.BlockInfo, csi module.ConsensusInfo, cb transitionCallback) (*transition, error) {
	if tr._ti == nil {
		return nil, nil
	}
	return tr._ti.propose(bi, csi, cb)
}

func (tr *transition) sync(result []byte, vlHash []byte, cb transitionCallback) (*transition, error) {
	if tr._ti == nil {
		return nil, nil
	}
	return tr._ti.sync(result, vlHash, cb)
}

func (tr *transition) newTransition(cb transitionCallback) *transition {
	if tr._ti == nil {
		return nil
	}
	return tr._ti._newTransition(cb)
}

func (tr *transition) verifyResult(block module.BlockData) error {
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
	chainContext *chainContext,
) *transition {
	var nilErr error
	ti := &transitionImpl{
		_chainContext: chainContext,
		_mtransition:  mtransition,
		_valErr:       &nilErr,
		_exeErr:       &nilErr,
		_nRef:         1,
	}
	tr := &transition{
		_ti: ti,
		_cb: nil,
	}
	if configTraceTransition {
		ti._chainContext.trtr.TraceNew(ti)
	}
	return tr
}
