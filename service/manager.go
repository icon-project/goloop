package service

import (
	"errors"
	"io"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
	"github.com/icon-project/goloop/module"
)

const (
	// maximum number of transactions in a block
	// TODO it should be configured or received from block manager
	txMaxNumInBlock = 10
)

////////////////////
// Service Manager
////////////////////

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	patchTxPool  *txPool
	normalTxPool *txPool

	trieManager trie.Manager
}

// TODO It should be declared in module package.
func NewManager(db db.Database) module.ServiceManager {
	// TODO change not to use mpt package directly
	return &manager{
		patchTxPool:  new(txPool),
		normalTxPool: new(txPool),
		trieManager:  mpt.NewManager(db)}
}

// ProposeTransition proposes a Transition following the parent Transition.
// parent transition should have a valid result.
// Returned Transition always passes validation.
func (m *manager) ProposeTransition(parent module.Transition) (module.Transition, error) {
	// check validity of transition
	pt, state, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	// find validated transactions
	patchTxs := m.patchTxPool.candidate(state, -1) // try to add all patches in the block
	maxTxNum := txMaxNumInBlock - len(patchTxs)
	var normalTxs []tx
	if maxTxNum > 0 {
		normalTxs = m.normalTxPool.candidate(state, txMaxNumInBlock-len(patchTxs))
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]tx, 0)
	}

	// create transition instance and return it
	return newTransition(pt, &txList{txs: patchTxs}, &txList{txs: normalTxs}, state, true), nil
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte, vs []module.Validator) (module.Transition, error) {
	resultBytes, err := newResultBytes(result)
	if err != nil {
		return nil, errors.New("Invalid result")
	}
	// TODO check if result is valid. Who's responsible?
	// TODO set validatorList correctly
	return newInitTransition(m.trieManager, resultBytes, nil), nil
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
func (m *manager) CreateTransition(parent module.Transition, txs module.TransactionList) (module.Transition, error) {
	// check validity of transition
	pt, state, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	// check transaction type
	txlist, ok := txs.(*txList)
	if !ok {
		return nil, common.ErrIllegalArgument
	}

	return newTransition(pt, &txList{txs: make([]tx, 0)}, txlist, state, false), nil
}

// GetPatches returns all patch transactions based on the parent transition.
// If it doesn't have any patches, it returns nil.
func (m *manager) GetPatches(parent module.Transition) module.TransactionList {
	// In fact, state is not necessary for patch transaction candidate validation,
	// but add the following same as that of normal transaction.
	pt, ok := parent.(*transition)
	if !ok {
		return nil
	}
	_, state, err := m.checkTransitionResult(pt.parent)
	if err != nil {
		return nil
	}

	return &txList{txs: m.patchTxPool.candidate(state, -1)}
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patches module.TransactionList) module.Transition {
	// type checking
	pt, ok := t.(*transition)
	if !ok {
		return nil
	}
	tst, state, err := m.checkTransitionResult(pt.parent)
	if err != nil {
		return nil
	}

	// prepare patch transaction list
	var txs *txList
	if patches == nil {
		txs = &txList{txs: make([]tx, 0)}
	} else {
		txs, ok = patches.(*txList)
		if !ok {
			return nil
		}
	}

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return newTransition(tst.parent, txs, tst.normalTransactions, state, false)
}

// Finalize finalizes data related to the transition. It usually stores
// data to a persistent storage. opt indicates which data are finalized.
// It should be called for every transition.
func (m *manager) Finalize(t module.Transition, opt int) {
	if tst, ok := t.(*transition); ok {
		tst.finalize(opt)
	}
}

// TransactionFromReader returns a Transaction instance from bytes
// read by Reader.
func (m *manager) TransactionFromReader(r io.Reader) module.Transaction {
	tx, _ := newTx(r)
	return tx
}

func (m *manager) checkTransitionResult(t module.Transition) (*transition, trie.Mutable, error) {
	// check validity of transition
	tst, ok := t.(*transition)
	if !ok || tst.step != stepComplete {
		return nil, nil, common.ErrIllegalArgument
	}
	state := m.trieManager.NewMutable(tst.result.stateHash())

	return tst, state, nil
}
