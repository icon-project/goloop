package service

import (
	"errors"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
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
	patchTxPool  *transactionPool
	normalTxPool *transactionPool

	db db.Database
}

// TODO It should be declared in module package.
func NewManager(database db.Database) module.ServiceManager {
	bk, _ := database.GetBucket(db.MerkleTrie)
	return &manager{
		patchTxPool:  NewtransactionPool(bk),
		normalTxPool: NewtransactionPool(bk),
		db:           database,
	}
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
	var normalTxs []*transaction
	if maxTxNum > 0 {
		normalTxs = m.normalTxPool.candidate(state, txMaxNumInBlock-len(patchTxs))
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]*transaction, 0)
	}

	// create transition instance and return it
	return newTransition(pt,
			newTransactionList(m.db, patchTxs),
			newTransactionList(m.db, normalTxs),
			state,
			true),
		nil
}

// ProposeGenesisTransition proposes a Transition for Genesis
// with transactions of Genesis.
func (m *manager) ProposeGenesisTransition(parent module.Transition) (module.Transition, error) {
	if pt, ok := parent.(*transition); ok {
		// create transition instance and return it
		return newTransition(pt,
				newTransactionList(m.db, nil),
				newTransactionList(m.db, nil),
				trie_manager.NewMutable(m.db, nil),
				true),
			nil
	}
	return nil, common.ErrIllegalArgument
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte, valList module.ValidatorList, height int64) (module.Transition, error) {
	var err error
	var resultBytes resultBytes
	if len(result) == 0 {
		resultBytes = newEmptyResultBytes()
	} else {
		if resultBytes, err = newResultBytes(result); err != nil {
			return nil, errors.New("invalid result")
		}
	}
	if valList == nil {
		valList, _ = ValidatorListFromSlice(m.db, nil)
	}
	// TODO check if result isn't valid. Who's responsible?
	return newInitTransition(m.db, resultBytes, valList), nil
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
func (m *manager) CreateTransition(parent module.Transition, txList module.TransactionList) (module.Transition, error) {
	// check validity of transition
	pt, state, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	// check transaction type
	txlist, ok := txList.(*transactionlist)
	if !ok {
		return nil, common.ErrIllegalArgument
	}

	return newTransition(pt,
			newTransactionList(m.db, make([]*transaction, 0)),
			txlist,
			state,
			false),
		nil
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
	_, state, err := m.checkTransitionResult(pt)
	if err != nil {
		return nil
	}

	return newTransactionList(m.db, m.patchTxPool.candidate(state, -1))
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patchTxList module.TransactionList) module.Transition {
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
	var txList *transactionlist
	if patchTxList == nil {
		txList = newTransactionList(m.db, make([]*transaction, 0))
	} else {
		txList, ok = patchTxList.(*transactionlist)
		if !ok {
			return nil
		}
	}

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return newTransition(tst.parent, txList, tst.normalTransactions, state, false)
}

// Finalize finalizes data related to the transition. It usually stores
// data to a persistent storage. opt indicates which data are finalized.
// It should be called for every transition.
func (m *manager) Finalize(t module.Transition, opt int) {
	if tst, ok := t.(*transition); ok {
		if opt&module.FinalizeNormalTransaction == module.FinalizeNormalTransaction {
			tst.finalizeNormalTransaction()
			// Because transactionlist for transition is made only through peer and SendTransaction() call
			// transactionlist has slice of transactions in case that finalize() is called
			m.normalTxPool.removeList(tst.normalTransactions.txs)
		}
		if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
			tst.finalizePatchTransaction()
			m.normalTxPool.removeList(tst.patchTransactions.txs)
		}
		if opt&module.FinalizeResult == module.FinalizeResult {
			tst.finalizeResult()
		}
	}
}

// TransactionFromBytes returns a Transaction instance from bytes.
func (m *manager) TransactionFromBytes(b []byte, blockVersion int) module.Transaction {
	tx, _ := newTransaction(b)
	return tx
}

// TransactionListFromHash returns a TransactionList instance from
// the hash of transactions or nil when no transactions exist.
func (m *manager) TransactionListFromHash(hash []byte) module.TransactionList {
	// TODO nil if hash is invalid?
	return newTransactionListFromHash(m.db, hash)
}

// TransactionListFromSlice returns list of transactions.
func (m *manager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	// TODO What if transaction objects are created outside?
	// TODO: db should be passed as parameter for flush()
	//panic("not implemented")
	return newTransactionListFromList(m.db, txs)
}

// ReceiptFromTransactionID returns receipt from legacy receipt bucket.
func (m *manager) ReceiptFromTransactionID(id []byte) module.Receipt {
	return nil
}

// ReceiptListFromResult returns list of receipts from result.
func (m *manager) ReceiptListFromResult(result []byte, g module.TransactionGroup) module.ReceiptList {
	return nil
}

func (m *manager) checkTransitionResult(t module.Transition) (*transition, trie.Mutable, error) {
	// check validity of transition
	tst, ok := t.(*transition)
	if !ok || tst.step != stepComplete {
		return nil, nil, common.ErrIllegalArgument
	}
	state := trie_manager.NewMutable(m.db, tst.result.stateHash())

	return tst, state, nil
}

func (m *manager) SendTransaction(tx interface{}) ([]byte, error) {
	// TODO: apply changed API
	//newTx, err := newTransactionFromObject(tx)
	//if err != nil {
	//	log.Printf("Failed to create new transaction from object!. tx : %x\n", newTx.Bytes())
	//	return nil, err
	//}
	//if err = newTx.Verify(); err != nil {
	//	log.Printf("Failed to verify transaction. tx : %x\n", newTx.Bytes())
	//	return nil, err
	//}
	//hash := newTx.Hash()
	//if hash == nil {
	//	log.Printf("Failed to get hash from tx : %x\n", newTx.Bytes())
	//	return nil, errors.New("Invalid Transaction. Failed to get hash")
	//}
	//
	//var txPool *transactionPool
	//switch newTx.Group() {
	//case module.TransactionGroupNormal:
	//	txPool = m.normalTxPool
	//case module.TransactionGroupPatch:
	//	txPool = m.patchTxPool
	//default:
	//	log.Panicf("Wrong TransactionGroup. %v", newTx.Group())
	//}
	//
	//go txPool.add(newTx)
	//return hash, nil
	return nil, nil
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	valList, _ := ValidatorListFromHash(m.db, hash)
	return valList
}

// For test
func T_NewAccountState(db db.Database) AccountState {
	return newAccountState(db, nil)
}
