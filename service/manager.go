package service

import (
	"encoding/json"
	"log"
	"math/big"
	"time"

	"github.com/icon-project/goloop/service/scoreresult"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

// Maximum size in bytes for transaction in a block.
// TODO it should be configured or received from block manager
const ConfigMaxTxBytesInABlock = 1024 * 1024
const ConfigWorldSnapshotCacheSize = 10

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	patchTxPool  *TransactionPool
	normalTxPool *TransactionPool

	patchMetric  *metric.TxMetric
	normalMetric *metric.TxMetric

	db        db.Database
	chain     module.Chain
	txReactor *TransactionReactor
	cm        contract.ContractManager
	eem       eeproxy.Manager
	wsc       *worldSnapshotCache
}

func NewManager(chain module.Chain, nm module.NetworkManager,
	eem eeproxy.Manager, chainRoot string,
) module.ServiceManager {
	bk, err := chain.Database().GetBucket(db.TransactionLocatorByHash)
	if err != nil {
		log.Printf("FAIL to get bucket(%s) %v\n", db.TransactionLocatorByHash, err)
		return nil //, err
	}

	pMetric := metric.NewTransactionMetric(chain.MetricContext(), metric.TxTypePatch)
	nMetric := metric.NewTransactionMetric(chain.MetricContext(), metric.TxTypeNormal)
	cm, err := contract.NewContractManager(chain.Database(), chainRoot)
	if err != nil {
		log.Printf("FAIL to create contractManager : %v\n", err)
		return nil //, err
	}

	mgr := &manager{
		patchMetric:  pMetric,
		normalMetric: nMetric,
		patchTxPool:  NewTransactionPool(chain.NID(), bk, pMetric),
		normalTxPool: NewTransactionPool(chain.NID(), bk, nMetric),
		db:           chain.Database(),
		chain:        chain,
		cm:           cm,
		eem:          eem,
		wsc:          newWorldSnapshotCache(chain.Database(), ConfigWorldSnapshotCacheSize),
	}
	if nm != nil {
		mgr.txReactor = NewTransactionReactor(nm, mgr.patchTxPool, mgr.normalTxPool)
	}
	return mgr
}

func (m *manager) Start() {
	if m.txReactor != nil {
		m.txReactor.Start()
	}
}

func (m *manager) Term() {
	if m.txReactor != nil {
		m.txReactor.Stop()
	}
	m.chain = nil
	m.cm = nil
	m.eem = nil
	m.db = nil
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

	maxTxCount := m.chain.Regulator().MaxTxCount()
	txSizeInBlock := ConfigMaxTxBytesInABlock

	patchTxs, size := m.patchTxPool.Candidate(wc, txSizeInBlock, maxTxCount) // try to add all patches in the block
	txSizeInBlock -= size
	maxTxCount -= len(patchTxs)

	var normalTxs []module.Transaction
	if txSizeInBlock > 0 && maxTxCount > 0 {
		normalTxs, _ = m.normalTxPool.Candidate(wc, txSizeInBlock, maxTxCount)
	} else {
		// what if patches already exceed the limit of transactions? It usually
		// doesn't happen but...
		normalTxs = make([]module.Transaction, 0)
	}

	// create transition instance and return it
	return newTransition(pt,
			transaction.NewTransactionListFromSlice(m.db, patchTxs),
			transaction.NewTransactionListFromSlice(m.db, normalTxs),
			bi, true),
		nil
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte,
	valList module.ValidatorList,
) (module.Transition, error) {
	return newInitTransition(m.db, result, valList, m.cm, m.eem, m.chain)
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
	txs, _ := m.patchTxPool.Candidate(wc, ConfigMaxTxBytesInABlock, 0)
	return transaction.NewTransactionListFromSlice(m.db, txs)
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
			m.normalTxPool.RemoveOldTXs(tst.bi.Timestamp() - transaction.ConfigTXTimestampBackwardMargin)
		}
		if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
			tst.finalizePatchTransaction()
			m.patchTxPool.RemoveList(tst.patchTransactions)
			m.patchTxPool.RemoveOldTXs(tst.bi.Timestamp() - transaction.ConfigTXTimestampBackwardMargin)
		}
		if opt&module.FinalizeResult == module.FinalizeResult {
			tst.finalizeResult()
			now := time.Now()
			m.patchMetric.OnFinalize(tst.patchTransactions.Hash(), now)
			m.normalMetric.OnFinalize(tst.normalTransactions.Hash(), now)
		}
	}
}

// TransactionFromBytes returns a Transaction instance from bytes.
func (m *manager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := transaction.NewTransaction(b)
	if err != nil {
		log.Printf("sm.TransactionFromBytes() fails with err=%+v", err)
	}
	return tx, nil
}

// TransactionListFromHash returns a TransactionList instance from
// the hash of transactions or nil when no transactions exist.
func (m *manager) TransactionListFromHash(hash []byte) module.TransactionList {
	return transaction.NewTransactionListFromHash(m.db, hash)
}

// TransactionListFromSlice returns list of transactions.
func (m *manager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	switch version {
	case module.BlockVersion1:
		return transaction.NewTransactionListV1FromSlice(txs)
	case module.BlockVersion2:
		return transaction.NewTransactionListFromSlice(m.db, txs)
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
		return nil, errors.ErrIllegalArgument
	}
	return tst, nil
}

func (m *manager) SendTransaction(txi interface{}) ([]byte, error) {
	var newTx transaction.Transaction
	switch txo := txi.(type) {
	case []byte:
		ntx, err := transaction.NewTransactionFromJSON(txo)
		if err != nil {
			return nil, errors.WithCode(err, InvalidTransactionError)
		}
		newTx = ntx.(transaction.Transaction)
	case string:
		ntx, err := transaction.NewTransactionFromJSON([]byte(txo))
		if err != nil {
			return nil, errors.WithCode(err, InvalidTransactionError)
		}
		newTx = ntx.(transaction.Transaction)
	case transaction.Transaction:
		newTx = txo
	default:
		return nil, ErrIllegalTransactionType
	}

	if err := newTx.Verify(); err != nil {
		return nil, InvalidTransactionError.Wrapf(err, "Failed to verify transaction. tx=<%x>\n", newTx.Bytes())
	}
	hash := newTx.ID()
	if hash == nil {
		return nil, InvalidTransactionError.Errorf("Failed to get hash from tx : %x\n", newTx.Bytes())
	}

	var txPool *TransactionPool
	switch newTx.Group() {
	case module.TransactionGroupNormal:
		txPool = m.normalTxPool
	case module.TransactionGroupPatch:
		txPool = m.patchTxPool
	default:
		log.Panicf("Wrong TransactionGroup. %v", newTx.Group())
	}

	if err := txPool.Add(newTx, true); err == nil {
		if err = m.txReactor.PropagateTransaction(ProtocolPropagateTransaction, newTx); err != nil {
			log.Printf("FAIL to propagate tx(%s)", err)
		}
	} else {
		return hash, err
	}
	return hash, nil
}

func (m *manager) Call(resultHash []byte,
	vl module.ValidatorList, js []byte, bi module.BlockInfo,
) (interface{}, error) {
	type callJSON struct {
		To       common.Address  `json:"to"`
		DataType *string         `json:"dataType"`
		Data     json.RawMessage `json:"data"`
	}

	var jso callJSON
	if json.Unmarshal(js, &jso) != nil {
		return nil, InvalidTransactionError.Errorf("FailToParse(%s)", string(js))
	}

	var wc state.WorldContext
	if tresult, err := newTransitionResultFromBytes(resultHash); err == nil {
		ws := state.NewReadOnlyWorldState(m.db, tresult.StateHash, vl)
		wc = state.NewWorldContext(ws, bi)
	} else {
		return nil, err
	}

	qh, err := NewQueryHandler(m.cm, &jso.To, jso.DataType, jso.Data)
	if err != nil {
		return nil, err
	}
	status, result := qh.Query(contract.NewContext(wc, m.cm, m.eem, m.chain))
	if status != module.StatusSuccess {
		return nil, scoreresult.NewBase(status, status.String())
	}
	return result, nil
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	valList, _ := state.ValidatorSnapshotFromHash(m.db, hash)
	return valList
}

func (m *manager) GetBalance(result []byte, addr module.Address) (*big.Int, error) {
	wss, err := m.wsc.GetWorldSnapshot(result)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(addr.ID())
	if ass == nil {
		return big.NewInt(0), nil
	}
	return ass.GetBalance(), nil
}

func (m *manager) GetTotalSupply(result []byte) (*big.Int, error) {
	wss, err := m.wsc.GetWorldSnapshot(result)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)

	if ts := tsVar.BigInt(); ts != nil {
		return ts, nil
	}
	return big.NewInt(0), nil
}

func (m *manager) GetNetworkID(result []byte) (int64, error) {
	wss, err := m.wsc.GetWorldSnapshot(result)
	if err != nil {
		return 0, err
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	nidVar := scoredb.NewVarDB(as, state.VarNetwork)
	if nidVar.Bytes() == nil {
		return 0, errors.ErrNotFound
	}
	return nidVar.Int64(), nil
}

func (m *manager) GetAPIInfo(result []byte, addr module.Address) (module.APIInfo, error) {
	if !addr.IsContract() {
		return nil, state.ErrNotContractAccount
	}
	wss, err := m.wsc.GetWorldSnapshot(result)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(addr.ID())
	info := ass.APIInfo()
	if info == nil {
		return nil, state.ErrNoActiveContract
	}
	return info, nil
}

func (m *manager) GetMembers(result []byte) (module.MemberList, error) {
	wss, err := m.wsc.GetWorldSnapshot(result)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	varConfig := scoredb.NewVarDB(as, state.VarServiceConfig)

	if (varConfig.Int64() & state.SysConfigMembership) == 0 {
		return newMemberList(nil), nil
	}
	return newMemberList(ass), nil
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
