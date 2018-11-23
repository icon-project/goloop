package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	// maximum number of transactions in a block
	// TODO it should be configured or received from block manager
	txMaxNumInBlock = 10
)

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	patchTxPool  *transactionPool
	normalTxPool *transactionPool

	db db.Database
}

func NewManager(chain module.Chain) module.ServiceManager {
	bk, _ := chain.GetDatabase().GetBucket(db.MerkleTrie)
	return &manager{
		patchTxPool:  NewtransactionPool(bk),
		normalTxPool: NewtransactionPool(bk),
		db:           chain.GetDatabase(),
	}
}

// ProposeTransition proposes a Transition following the parent Transition.
// parent transition should have a valid result.
// Returned Transition always passes validation.
func (m *manager) ProposeTransition(parent module.Transition) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	var timestamp int64 = time.Now().UnixNano() / 1000

	ws, _ := WorldStateFromSnapshot(pt.worldSnapshot)
	wc := NewWorldContext(ws, timestamp, pt.height+1)

	patchTxs := m.patchTxPool.candidate(wc, -1) // try to add all patches in the block
	maxTxNum := txMaxNumInBlock - len(patchTxs)
	var normalTxs []module.Transaction
	if maxTxNum > 0 {
		normalTxs = m.normalTxPool.candidate(wc, txMaxNumInBlock-len(patchTxs))
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]module.Transaction, 0)
	}

	// create transition instance and return it
	return newTransition(pt,
			NewTransactionListFromSlice(m.db, patchTxs),
			NewTransactionListFromSlice(m.db, normalTxs),
			true),
		nil
}

//{"accounts": [{"name": "god", "address": "hx5a05b58a25a1e5ea0f1d5715e1f655dffc1fb30a", "balance": "0x2961fff8ca4a62327800000"}, {"name": "treasury", "address": "hx1000000000000000000000000000000000000000", "balance": "0x0"}], "message": "A rhizome has no beginning or end; it is always in the middle, between things, interbeing, intermezzo. The tree is filiation, but the rhizome is alliance, uniquely alliance. The tree imposes the verb \"to be\" but the fabric of the rhizome is the conjunction, \"and ... and ...and...\"This conjunction carries enough force to shake and uproot the verb \"to be.\" Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.\n\n - Mille Plateaux, Gilles Deleuze & Felix Guattari\n\n\"Hyperconnect the world\""}
var genesisTx = "{\"accounts\": " +
	"[{\"name\": \"god\", " +
	"\"address\": \"hx5a05b58a25a1e5ea0f1d5715e1f655dffc1fb30a\", " +
	"\"balance\": \"0x2961fff8ca4a62327800000\"}, " +
	"{\"name\": \"treasury\", " +
	"\"address\": \"hx1000000000000000000000000000000000000000\", " +
	"\"balance\": \"0x0\"}], " +
	"\"message\": \"A rhizome has no beginning or end; " +
	"it is always in the middle, between things, interbeing, intermezzo. " +
	"The tree is filiation, but the rhizome is alliance, uniquely alliance. " +
	"The tree imposes the verb to be but the fabric of the rhizome is the conjunction, and ... and ...and...This conjunction carries enough force to shake and uproot the verb to be. Where are you going? Where are you coming from? What are you heading for? These are totally useless questions.- Mille Plateaux, Gilles Deleuze & Felix GuattariHyperconnect the world\"," +
	"\"validatorlist\": [\"01234\", \"12335\"]}"

func (m *manager) ProposeGenesisTransition(parent module.Transition) (module.Transition, error) {
	if pt, ok := parent.(*transition); ok {
		// TODO: temp code below to create genesis transaction. remove later
		ntx, err := NewTransactionFromJSON([]byte(genesisTx))
		if err != nil {
			log.Panicf("Failed to load genesis transaction")
			return nil, err
		}
		t := newTransition(pt,
			NewTransactionListFromSlice(m.db, nil),
			NewTransactionListFromSlice(pt.db, []module.Transaction{ntx}),
			true)

		return t, nil
	}
	return nil, common.ErrIllegalArgument
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte, valList module.ValidatorList, height int64) (module.Transition, error) {
	return newInitTransition(m.db, result, valList, height)
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
// TODO It has to receive timestamp
func (m *manager) CreateTransition(parent module.Transition, txList module.TransactionList) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}
	return newTransition(pt, nil, txList, false), nil
}

// GetPatches returns all patch transactions based on the parent transition.
// If it doesn't have any patches, it returns nil.
func (m *manager) GetPatches(parent module.Transition) module.TransactionList {
	// In fact, state is not necessary for patch transaction candidate validation,
	// but add the following same as that of normal transaction.
	pt, ok := parent.(*transition)
	if !ok {
		log.Panicf("Illegal transition for GetPatches type=%T", parent)
		return nil
	}

	ws, err := WorldStateFromSnapshot(pt.worldSnapshot)
	if err != nil {
		log.Panicf("Fail to creating world state from snapshot")
	}

	// TODO we need to get proper time stamp value and height.
	wc := NewWorldContext(ws, 0, 0)
	return NewTransactionListFromSlice(m.db, m.patchTxPool.candidate(wc, -1))
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patchTxList module.TransactionList) module.Transition {
	pt, ok := t.(*transition)
	if !ok {
		log.Panicf("Illegal transition for GetPatches type=%T", t)
		return nil
	}

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return newTransition(pt.parent, patchTxList, pt.normalTransactions, false)
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
			m.normalTxPool.removeList(tst.normalTransactions)
		}
		if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
			tst.finalizePatchTransaction()
			m.normalTxPool.removeList(tst.patchTransactions)
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
	switch version {
	case module.BlockVersion1:
		return NewTransactionListV1FromSlice(txs)
	case module.BlockVersion2:
		return NewTransactionListFromSlice(m.db, txs)
	default:
		return nil
	}
}

// ReceiptFromTransactionID returns receipt from legacy receipt bucket.
func (m *manager) ReceiptFromTransactionID(id []byte) module.Receipt {
	return nil
}

// ReceiptListFromResult returns list of receipts from result.
func (m *manager) ReceiptListFromResult(result []byte, g module.TransactionGroup) module.ReceiptList {
	return nil
}

func (m *manager) checkTransitionResult(t module.Transition) (*transition, error) {
	if t == nil {
		return nil, nil
	}
	tst, ok := t.(*transition)
	if !ok || tst.step != stepComplete {
		return nil, common.ErrIllegalArgument
	}
	return tst, nil
}

func (m *manager) SendTransaction(tx interface{}) ([]byte, error) {

	var newTx *transaction
	switch txo := tx.(type) {
	case []byte:
		ntx, err := NewTransactionFromJSON(txo)
		if err != nil {
			return nil, err
		}
		newTx = ntx.(*transaction)
	case string:
		ntx, err := NewTransactionFromJSON([]byte(txo))
		if err != nil {
			return nil, err
		}
		newTx = ntx.(*transaction)
	case *transaction:
		newTx = txo
	default:
		return nil, fmt.Errorf("IllegalTransactoinType:%T", tx)
	}

	if err := newTx.Verify(); err != nil {
		log.Printf("Failed to verify transaction. tx : %x\n", newTx.Bytes())
		return nil, err
	}
	hash := newTx.Hash()
	if hash == nil {
		log.Printf("Failed to get hash from tx : %x\n", newTx.Bytes())
		return nil, errors.New("Invalid Transaction. Failed to get hash")
	}

	var txPool *transactionPool
	switch newTx.Group() {
	case module.TransactionGroupNormal:
		txPool = m.normalTxPool
	case module.TransactionGroupPatch:
		txPool = m.patchTxPool
	default:
		log.Panicf("Wrong TransactionGroup. %v", newTx.Group())
	}

	go txPool.add(newTx)
	return hash, nil
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	valList, _ := ValidatorListFromHash(m.db, hash)
	return valList
}
