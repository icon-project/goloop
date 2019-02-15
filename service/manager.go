package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/icon-project/goloop/service/tx"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	patchTxPool  *tx.TransactionPool
	normalTxPool *tx.TransactionPool

	db        db.Database
	chain     module.Chain
	txReactor *tx.TransactionReactor
	cm        contract.ContractManager
	eem       eeproxy.Manager
}

func NewManager(chain module.Chain, nm module.NetworkManager,
	eem eeproxy.Manager, contractDir string,
) module.ServiceManager {
	bk, _ := chain.Database().GetBucket(db.TransactionLocatorByHash)

	mgr := &manager{
		patchTxPool:  tx.NewTransactionPool(bk),
		normalTxPool: tx.NewTransactionPool(bk),
		db:           chain.Database(),
		chain:        chain,
		cm:           contract.NewContractManager(chain.Database(), contractDir),
		eem:          eem,
	}
	if nm != nil {
		mgr.txReactor = tx.NewTransactionReactor(nm, mgr.patchTxPool, mgr.normalTxPool)
	}
	return mgr
}

// ProposeTransition proposes a Transition following the parent Transition.
// parent transition should have a valid result.
// Returned Transition always passes validation.
func (m *manager) ProposeTransition(parent module.Transition, bi module.BlockInfo,
) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	ws, _ := state.WorldStateFromSnapshot(pt.worldSnapshot)
	wc := state.NewWorldContext(ws, bi)

	patchTxs := m.patchTxPool.Candidate(wc, -1) // try to add all patches in the block
	maxTxNum := tx.TxMaxNumInBlock - len(patchTxs)
	var normalTxs []module.Transaction
	if maxTxNum > 0 {
		normalTxs = m.normalTxPool.Candidate(wc, tx.TxMaxNumInBlock-len(patchTxs))
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]module.Transaction, 0)
	}

	// create transition instance and return it
	return newTransition(pt,
			tx.NewTransactionListFromSlice(m.db, patchTxs),
			tx.NewTransactionListFromSlice(m.db, normalTxs),
			bi, true),
		nil
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte,
	valList module.ValidatorList,
) (module.Transition, error) {
	return newInitTransition(m.db, result, valList, m.cm, m.eem)
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
func (m *manager) CreateTransition(parent module.Transition,
	txList module.TransactionList, bi module.BlockInfo,
) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}
	return newTransition(pt, nil, txList, bi, false), nil
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

	ws, err := state.WorldStateFromSnapshot(pt.worldSnapshot)
	if err != nil {
		log.Panicf("Fail to creating world state from snapshot")
	}

	wc := state.NewWorldContext(ws, pt.bi)
	return tx.NewTransactionListFromSlice(m.db, m.patchTxPool.Candidate(wc, -1))
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patchTxList module.TransactionList,
) module.Transition {
	pt, ok := t.(*transition)
	if !ok {
		log.Panicf("Illegal transition for GetPatches type=%T", t)
		return nil
	}

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return newTransition(pt.parent, patchTxList, pt.normalTransactions, pt.bi, false)
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
			m.normalTxPool.RemoveList(tst.normalTransactions)
			m.normalTxPool.RemoveOldTXs(tst.bi.Timestamp() - tx.ConfigTXTimestampBackwardMargin)
		}
		if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
			tst.finalizePatchTransaction()
			m.patchTxPool.RemoveList(tst.patchTransactions)
			m.patchTxPool.RemoveOldTXs(tst.bi.Timestamp() - tx.ConfigTXTimestampBackwardMargin)
		}
		if opt&module.FinalizeResult == module.FinalizeResult {
			tst.finalizeResult()
		}
	}
}

// TransactionFromBytes returns a Transaction instance from bytes.
func (m *manager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := tx.NewTransaction(b)
	if err != nil {
		log.Printf("sm.TransactionFromBytes() fails with err=%+v", err)
	}
	return tx, nil
}

// TransactionListFromHash returns a TransactionList instance from
// the hash of transactions or nil when no transactions exist.
func (m *manager) TransactionListFromHash(hash []byte) module.TransactionList {
	return tx.NewTransactionListFromHash(m.db, hash)
}

// TransactionListFromSlice returns list of transactions.
func (m *manager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	switch version {
	case module.BlockVersion1:
		return tx.NewTransactionListV1FromSlice(txs)
	case module.BlockVersion2:
		return tx.NewTransactionListFromSlice(m.db, txs)
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
	if tresult, err := newTransitionResultFromBytes(result); err == nil {
		if g == module.TransactionGroupPatch {
			return txresult.NewReceiptListFromHash(m.db, tresult.PatchReceiptHash)
		} else {
			return txresult.NewReceiptListFromHash(m.db, tresult.NormalReceiptHash)
		}
	} else {
		log.Printf("Fail to unmarshal result bytes err=%+v", err)
	}
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

func (m *manager) SendTransaction(txi interface{}) ([]byte, error) {
	var newTx tx.Transaction
	switch txo := txi.(type) {
	case []byte:
		ntx, err := tx.NewTransactionFromJSON(txo)
		if err != nil {
			return nil, err
		}
		newTx = ntx.(tx.Transaction)
	case string:
		ntx, err := tx.NewTransactionFromJSON([]byte(txo))
		if err != nil {
			return nil, err
		}
		newTx = ntx.(tx.Transaction)
	case tx.Transaction:
		newTx = txo
	default:
		return nil, fmt.Errorf("IllegalTransactionType:%T", txi)
	}

	if err := newTx.Verify(); err != nil {
		log.Printf("Failed to verify transaction. tx : %x\n", newTx.Bytes())
		return nil, err
	}
	hash := newTx.ID()
	if hash == nil {
		log.Printf("Failed to get hash from tx : %x\n", newTx.Bytes())
		return nil, errors.New("Invalid Transaction. Failed to get hash")
	}

	var txPool *tx.TransactionPool
	switch newTx.Group() {
	case module.TransactionGroupNormal:
		txPool = m.normalTxPool
	case module.TransactionGroupPatch:
		txPool = m.patchTxPool
	default:
		log.Panicf("Wrong TransactionGroup. %v", newTx.Group())
	}

	if err := txPool.Add(newTx); err == nil {
		m.txReactor.PropagateTransaction(tx.ProtocolPropagateTransaction, newTx)
	} else {
		return hash, err
	}
	return hash, nil
}

func (m *manager) Call(resultHash []byte, js []byte, bi module.BlockInfo,
) (module.Status, interface{}, error) {
	type callJSON struct {
		From     common.Address  `json:"from"`
		To       common.Address  `json:"to"`
		DataType *string         `json:"dataType"`
		Data     json.RawMessage `json:"data"`
	}

	var jso callJSON
	if json.Unmarshal(js, &jso) != nil {
		return module.StatusSystemError, nil, errors.New("Fail to parse JSON RPC")
	}

	var wc state.WorldContext
	if tresult, err := newTransitionResultFromBytes(resultHash); err == nil {
		ws := state.NewWorldState(m.db, tresult.StateHash, nil)
		wc = state.NewWorldContext(ws, bi)
	} else {
		return module.StatusSystemError, err.Error(), nil
	}

	qh, err := contract.NewQueryHandler(m.cm, &jso.From, &jso.To,
		jso.DataType, jso.Data)
	if err != nil {
		return module.StatusSystemError, err.Error(), nil
	}
	status, result := qh.Query(contract.NewContext(wc, m.cm, m.eem))
	return status, result, nil
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	valList, _ := state.ValidatorListFromHash(m.db, hash)
	return valList
}

func (m *manager) GetBalance(result []byte, addr module.Address) *big.Int {
	if tresult, err := newTransitionResultFromBytes(result); err == nil {
		ws := state.NewWorldSnapshot(m.db, tresult.StateHash, nil)
		ass := ws.GetAccountSnapshot(addr.ID())
		if ass == nil {
			return big.NewInt(0)
		}
		return ass.GetBalance()
	}
	return big.NewInt(0)
}

type blockInfo struct {
	height    int64
	timestamp int64
}

func newBlockInfo(h, ts int64) *blockInfo {
	return &blockInfo{height: h, timestamp: ts}
}
func (bi *blockInfo) Height() int64 {
	return bi.height
}

func (bi *blockInfo) Timestamp() int64 {
	return bi.timestamp
}
