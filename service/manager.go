package service

import (
	"encoding/json"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/scoreresult"
	ssync "github.com/icon-project/goloop/service/sync"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

const ConfigTransitionResultCacheEntryCount = 10
const ConfigTransitionResultCacheEntrySize = 1024 * 1024

type manager struct {
	// tx pool should be connected to transition for more than one branches.
	// Currently, it doesn't allow another branch, so add tx pool here.
	tm           *TransactionManager
	patchTxPool  *TransactionPool
	normalTxPool *TransactionPool

	patchMetric  *metric.TxMetric
	normalMetric *metric.TxMetric

	plt       Platform
	db        db.Database
	chain     module.Chain
	txReactor *TransactionReactor
	cm        contract.ContractManager
	eem       eeproxy.Manager
	trc       *transitionResultCache
	tsc       *TxTimestampChecker
	syncer    *ssync.Manager

	log log.Logger

	skipTxPatch atomic.Value
}

func NewManager(chain module.Chain, nm module.NetworkManager,
	eem eeproxy.Manager, plt Platform, contractDir string,
) (module.ServiceManager, error) {
	logger := chain.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "SV",
	})
	bk, err := chain.Database().GetBucket(db.TransactionLocatorByHash)
	if err != nil {
		logger.Warnf("FAIL to get bucket(%s) %v\n", db.TransactionLocatorByHash, err)
		return nil, err
	}

	pMetric := metric.NewTransactionMetric(chain.MetricContext(), metric.TxTypePatch)
	nMetric := metric.NewTransactionMetric(chain.MetricContext(), metric.TxTypeNormal)
	cm, err := plt.NewContractManager(chain.Database(), contractDir, logger)
	if err != nil {
		logger.Warnf("FAIL to create contractManager : %v\n", err)
		return nil, err
	}
	pTxPool := NewTransactionPool(module.TransactionGroupPatch,
		chain.PatchTxPoolSize(), bk, pMetric, logger)
	nTxPool := NewTransactionPool(module.TransactionGroupNormal,
		chain.NormalTxPoolSize(), bk, nMetric, logger)
	tsc := NewTimestampChecker()
	tm := NewTransactionManager(chain.NID(), tsc, pTxPool, nTxPool, bk, logger)
	syncm := ssync.NewSyncManager(chain.Database(), chain.NetworkManager(), plt, logger)

	mgr := &manager{
		patchMetric:  pMetric,
		normalMetric: nMetric,
		tm:           tm,
		db:           chain.Database(),
		chain:        chain,
		cm:           cm,
		plt:          plt,
		eem:          eem,
		syncer:       syncm,
		trc: newTransitionResultCache(chain.Database(), plt,
			ConfigTransitionResultCacheEntryCount,
			ConfigTransitionResultCacheEntrySize,
			logger),
		log: logger,
		tsc: tsc,
	}
	if nm != nil {
		mgr.txReactor = NewTransactionReactor(nm, tm)
	}
	return mgr, nil
}

func (m *manager) Start() {
	if m.txReactor != nil {
		m.txReactor.Start(m.chain.Wallet())
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
func (m *manager) ProposeTransition(parent module.Transition, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}

	ws, _ := state.WorldStateFromSnapshot(pt.worldSnapshot)
	wc := state.NewWorldContext(ws, bi, csi, m.plt)

	baseTx, err := m.plt.NewBaseTransaction(wc)
	if err != nil {
		return nil, err
	}
	maxTxCount := m.chain.Regulator().MaxTxCount()
	txSizeInBlock := m.chain.MaxBlockTxBytes()
	normalTxs, _ := m.tm.Candidate(module.TransactionGroupNormal, wc, txSizeInBlock, maxTxCount)
	if baseTx != nil {
		normalTxs = append([]module.Transaction{baseTx}, normalTxs...)
	}

	// create transition instance and return it
	return newTransition(
			pt,
			transaction.NewTransactionListFromSlice(m.db, nil),
			transaction.NewTransactionListFromSlice(m.db, normalTxs),
			bi,
			csi,
			m.plt,
			true,
		),
		nil
}

// CreateInitialTransition creates an initial Transition with result and
// vs validators.
func (m *manager) CreateInitialTransition(result []byte,
	valList module.ValidatorList,
) (module.Transition, error) {
	var stateHash []byte
	var ess state.ExtensionSnapshot
	if len(result) > 0 {
		if tr, err := newTransitionResultFromBytes(result); err != nil {
			return nil, errors.IllegalArgumentError.Wrap(err, "InvalidResultBytes")
		} else {
			stateHash = tr.StateHash
			ess = m.plt.NewExtensionSnapshot(m.db, tr.ExtensionData)
		}
	}
	return newInitTransition(m.db, stateHash, valList, ess, m.cm, m.eem, m.chain, m.log, m.plt, m.tsc)
}

// CreateTransition creates a Transition following parent Transition with txs
// transactions.
// parent transition should have a valid result.
func (m *manager) CreateTransition(
	parent module.Transition,
	txs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
) (module.Transition, error) {
	// check validity of transition
	pt, err := m.checkTransitionResult(parent)
	if err != nil {
		return nil, err
	}
	return newTransition(pt, nil, txs, bi, csi, m.plt, false), nil
}

func (m *manager) SendPatch(data module.Patch) error {
	if data.Type() == module.PatchTypeSkipTransaction {
		patch, ok := data.(module.SkipTransactionPatch)
		if !ok {
			return InvalidPatchDataError.New("Invalid Skip Transaction Patch Data")
		}
		if patch.Height() < 1 {
			return InvalidPatchDataError.Errorf(
				"InvalidHeightValue(height=%d)", patch.Height())
		}
		m.skipTxPatch.Store(patch)
		return nil
	} else {
		return InvalidPatchDataError.New("UnknownPatch")
	}
}

// GetPatches returns all patch transactions based on the parent transition.
// If it doesn't have any patches, it returns nil.
func (m *manager) GetPatches(parent module.Transition, bi module.BlockInfo) module.TransactionList {
	// In fact, state is not necessary for patch transaction candidate validation,
	// but add the following same as that of normal transaction.
	pt, ok := parent.(*transition)
	if !ok {
		m.log.Panicf("Illegal transition for GetPatches type=%T", parent)
		return nil
	}

	ws, err := state.WorldStateFromSnapshot(pt.worldSnapshot)
	if err != nil {
		m.log.Panicf("Fail to creating world state from snapshot")
		return nil
	}

	wc := state.NewWorldContext(ws, bi, nil, m.plt)

	txs, size := m.tm.Candidate(module.TransactionGroupPatch, wc, m.chain.MaxBlockTxBytes(), 0)

	p, _ := m.skipTxPatch.Load().(module.SkipTransactionPatch)
	if p != nil {
		m.log.Debugf("GetPatches() skipTxPatch=%+v wc.BlockHeight()=%d", p, wc.BlockHeight())
		if p.Height()+1 == wc.BlockHeight() {
			tx, err := transaction.NewPatchTransaction(
				p, m.chain.NID(), wc.BlockTimeStamp(), m.chain.Wallet())
			if err != nil {
				m.log.Panicf("Fail to make transaction from patch err=%+v", err)
			}
			size += len(tx.Bytes())
			txs = append(txs, tx)
		}
	}
	return transaction.NewTransactionListFromSlice(m.db, txs)
}

// PatchTransition creates a Transition by overwriting patches on the transition.
// It doesn't return same instance as transition, but new Transition instance.
func (m *manager) PatchTransition(t module.Transition, patchTxList module.TransactionList,
	bi module.BlockInfo,
) module.Transition {
	pt, ok := t.(*transition)
	if !ok {
		m.log.Panicf("Illegal transition for GetPatches type=%T", t)
		return nil
	}
	m.log.Debugf("PatchTransition(patchTxs=<%x>)", patchTxList.Hash())

	// If there is no way to validate patches, then set 'alreadyValidated' to
	// true. It'll skip unnecessary validation for already validated normal
	// transactions.
	return patchTransition(pt, bi, patchTxList)
}

func (m *manager) CreateSyncTransition(t module.Transition, result []byte, vlHash []byte) module.Transition {
	m.log.Debugf("CreateSyncTransition result(%#x), vlHash(%#x)\n", result, vlHash)
	tr, ok := t.(*transition)
	if !ok {
		m.log.Panicf("Illegal transition for CreateSyncTransition type=%T", t)
		return nil
	}
	ntr := newTransition(
		tr.parent, tr.patchTransactions, tr.normalTransactions, tr.bi, tr.csi, m.plt, true)
	r, _ := newTransitionResultFromBytes(result)
	ntr.syncer = m.syncer.NewSyncer(r.StateHash,
		r.PatchReceiptHash, r.NormalReceiptHash, vlHash, r.ExtensionData)
	return ntr
}

// Finalize finalizes data related to the transition. It usually stores
// data to a persistent storage. opt indicates which data are finalized.
// It should be called for every transition.
func (m *manager) Finalize(t module.Transition, opt int) error {
	if tst, ok := t.(*transition); ok {
		if opt&module.FinalizeNormalTransaction == module.FinalizeNormalTransaction {
			if err := tst.finalizeNormalTransaction(); err != nil {
				return err
			}
			// Because transactionlist for transition is made only through peer and SendTransaction() call
			// transactionlist has slice of transactions in case that finalize() is called
			m.tm.RemoveTxs(module.TransactionGroupNormal, tst.normalTransactions)
			m.tm.RemoveOldTxByBlockTS(module.TransactionGroupNormal, tst.bi.Timestamp())
		}
		if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction {
			if err := tst.finalizePatchTransaction(); err != nil {
				return err
			}
			m.tm.RemoveTxs(module.TransactionGroupPatch, tst.patchTransactions)
			m.tm.RemoveOldTxByBlockTS(module.TransactionGroupPatch, tst.bi.Timestamp())
		}
		if opt&module.FinalizeResult == module.FinalizeResult {
			if err := tst.finalizeResult(false); err != nil {
				return err
			}
			m.tm.NotifyFinalized(tst.patchTransactions, tst.patchReceipts, tst.normalTransactions, tst.normalReceipts)
			now := time.Now()
			m.patchMetric.OnFinalize(tst.patchTransactions.Hash(), now)
			m.normalMetric.OnFinalize(tst.normalTransactions.Hash(), now)
		}
	} else {
		panic("FAIL type assertion. Not transition pointer type")
	}
	return nil
}

// TransactionFromBytes returns a Transaction instance from bytes.
func (m *manager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := transaction.NewTransaction(b)
	if err != nil {
		m.log.Warnf("sm.TransactionFromBytes() fails with err=%+v", err)
		return nil, err
	}
	return tx, nil
}

func (m *manager) GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := transaction.NewGenesisTransaction(b)
	if err != nil {
		m.log.Warnf("sm.GenesisTransactionFromBytes() fails with err=%+v", err)
		return nil, err
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
	case module.BlockVersion0:
		return transaction.NewTransactionListV1FromSlice(txs)
	case module.BlockVersion1, module.BlockVersion2:
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
func (m *manager) ReceiptListFromResult(result []byte, g module.TransactionGroup) (module.ReceiptList, error) {
	if rl, err := m.trc.GetReceipts(result, g); err != nil {
		return nil, err
	} else {
		return rl, nil
	}
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

func newTransaction(txi interface{}) (transaction.Transaction, error) {
	switch txo := txi.(type) {
	case []byte:
		ntx, err := transaction.NewTransactionFromJSON(txo)
		if err != nil {
			return nil, errors.WithCode(err, InvalidTransactionError)
		}
		return ntx.(transaction.Transaction), nil
	case string:
		ntx, err := transaction.NewTransactionFromJSON([]byte(txo))
		if err != nil {
			return nil, errors.WithCode(err, InvalidTransactionError)
		}
		return ntx.(transaction.Transaction), nil
	case transaction.Transaction:
		return txo, nil
	default:
		return nil, InvalidTransactionError.Errorf("UnknownType(%T)", txi)
	}
}

func (m *manager) SendTransactionAndWait(txi interface{}) ([]byte, <-chan interface{}, error) {
	newTx, err := newTransaction(txi)
	if err != nil {
		return nil, nil, err
	}
	chn, err := m.tm.AddAndWait(newTx)
	if err == nil {
		if err := m.txReactor.PropagateTransaction(newTx); err != nil {
			if !network.NotAvailableError.Equals(err) {
				m.log.Tracef("FAIL to propagate tx err=%+v", err)
			}
		}
	}
	if err == nil || err == ErrCommittedTransaction {
		return newTx.ID(), chn, err
	}
	return nil, nil, err
}

func (m *manager) WaitTransactionResult(id []byte) (<-chan interface{}, error) {
	return m.tm.WaitResult(id)
}

func (m *manager) SendTransaction(txi interface{}) ([]byte, error) {
	newTx, err := newTransaction(txi)
	if err != nil {
		return nil, err
	}

	if err := m.tm.Add(newTx, true); err != nil {
		return nil, err
	}

	if err := m.txReactor.PropagateTransaction(newTx); err != nil {
		if !network.NotAvailableError.Equals(err) {
			m.log.Tracef("FAIL to propagate tx err=%+v", err)
		}
	}
	return newTx.ID(), nil
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
		return nil, InvalidQueryError.Errorf("FailToParse(%s)", string(js))
	}
	if jso.DataType == nil || *jso.DataType != contract.DataTypeCall {
		return nil, InvalidQueryError.New("InvalidDataType")
	}

	var wc state.WorldContext
	if wss, err := m.trc.GetWorldSnapshot(resultHash, vl.Hash()); err == nil {
		ws := state.NewReadOnlyWorldState(wss)
		wc = state.NewWorldContext(ws, bi, nil, m.plt)
	} else {
		return nil, err
	}

	qh, err := NewQueryHandler(m.cm, &jso.To, jso.Data)
	if err != nil {
		return nil, err
	}
	return qh.Query(contract.NewContext(wc, m.cm, m.eem, m.chain, m.log, nil))
}

func (m *manager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	valList, _ := m.trc.GetValidatorSnapshot(hash)
	return valList
}

func (m *manager) GetBalance(result []byte, addr module.Address) (*big.Int, error) {
	wss, err := m.trc.GetWorldSnapshot(result, nil)
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
	wss, err := m.trc.GetWorldSnapshot(result, nil)
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
	wss, err := m.trc.GetWorldSnapshot(result, nil)
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

func (m *manager) GetChainID(result []byte) (int64, error) {
	wss, err := m.trc.GetWorldSnapshot(result, nil)
	if err != nil {
		return 0, err
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	nidVar := scoredb.NewVarDB(as, state.VarChainID)
	if nidVar.Bytes() == nil {
		return 0, errors.ErrNotFound
	}
	return nidVar.Int64(), nil
}

func (m *manager) GetAPIInfo(result []byte, addr module.Address) (module.APIInfo, error) {
	if !addr.IsContract() {
		return nil, NotContractAddressError.Errorf("Given Address(%s) isn't contract", addr)
	}
	wss, err := m.trc.GetWorldSnapshot(result, nil)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(addr.ID())
	if ass == nil {
		return nil, NoActiveContractError.Errorf("No account for %s", addr)
	}
	info, err := ass.APIInfo()
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, NoActiveContractError.Errorf("Account(%s) doesn't have active contract", addr)
	}
	return info, nil
}

func (m *manager) GetMembers(result []byte) (module.MemberList, error) {
	wss, err := m.trc.GetWorldSnapshot(result, nil)
	if err != nil {
		return nil, err
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	return newMemberList(ass), nil
}

func (m *manager) GetRoundLimit(result []byte, vl int) int64 {
	wss, err := m.trc.GetWorldSnapshot(result, nil)
	if err != nil {
		return 0
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	factor := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Int64()
	if factor == 0 {
		return 0
	}
	limit := contract.RoundLimitFactorToRound(vl, factor)
	m.log.Debugf("Validators:%d RoundLimitFactor:%d --> RoundLimit:%d",
		vl, factor, limit)
	return limit
}

func (m *manager) GetMinimizeBlockGen(result []byte) bool {
	wss, err := m.trc.GetWorldSnapshot(result, nil)
	if err != nil {
		return false
	}
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	return scoredb.NewVarDB(as, state.VarMinimizeBlockGen).Bool()
}

func (m *manager) HasTransaction(id []byte) bool {
	return m.tm.HasTx(id)
}

func (m *manager) WaitForTransaction(
	parent module.Transition,
	bi module.BlockInfo,
	cb func(),
) bool {
	pt := parent.(*transition)
	ws, _ := state.WorldStateFromSnapshot(pt.worldSnapshot)
	wc := state.NewWorldContext(ws, bi, nil, m.plt)

	return m.tm.Wait(wc, cb)
}

func (m *manager) ExportResult(result []byte, vh []byte, d db.Database) error {
	r, err := newTransitionResultFromBytes(result)
	if err != nil {
		return err
	}
	e := merkle.NewCopyContext(m.db, d)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.NormalReceiptHash)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.PatchReceiptHash)
	ess := m.plt.NewExtensionWithBuilder(e.Builder(), r.ExtensionData)
	state.NewWorldSnapshotWithBuilder(e.Builder(), r.StateHash, vh, ess)
	return e.Run()
}

func (m *manager) ImportResult(result []byte, vh []byte, src db.Database) error {
	r, err := newTransitionResultFromBytes(result)
	if err != nil {
		return err
	}
	e := merkle.NewCopyContext(src, m.db)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.NormalReceiptHash)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.PatchReceiptHash)
	es := m.plt.NewExtensionWithBuilder(e.Builder(), r.ExtensionData)
	state.NewWorldSnapshotWithBuilder(e.Builder(), r.StateHash, vh, es)
	return e.Run()
}

func (m *manager) ExecuteTransaction(result []byte, vh []byte, js []byte, bi module.BlockInfo) (module.Receipt, error) {
	tx, err := transaction.NewTransactionFromJSON(js)
	if err != nil {
		return nil, err
	}
	if err := tx.Verify(); err != nil && !transaction.InvalidSignatureError.Equals(err) {
		return nil, scoreresult.InvalidParameterError.Wrap(err, "InvalidTransaction")
	}

	txh, err := tx.GetHandler(m.cm)
	if err != nil {
		return nil, err
	}
	defer txh.Dispose()

	var wc state.WorldContext
	if wss, err := m.trc.GetWorldSnapshot(result, vh); err == nil {
		ws, err := state.WorldStateFromSnapshot(wss)
		if err != nil {
			return nil, err
		}
		wc = state.NewWorldContext(ws, bi, nil, m.plt)
	} else {
		return nil, err
	}
	ctx := contract.NewContext(wc, m.cm, m.eem, m.chain, m.log, nil)
	ctx.SetTransactionInfo(&state.TransactionInfo{
		Group:     module.TransactionGroupNormal,
		Index:     0,
		Hash:      tx.ID(),
		From:      tx.From(),
		Timestamp: tx.Timestamp(),
		Nonce:     tx.Nonce(),
	})
	ctx.UpdateSystemInfo()

	return txh.Execute(ctx, true)
}
