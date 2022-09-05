package service

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/service/eeproxy"
	ssync "github.com/icon-project/goloop/service/sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	RetryCount = 2
)

type transitionStep int

const (
	stepInited    transitionStep = iota // parent, patch/normalTxes and state are ready.
	stepValidated                       // Upon inited state, Txes are validated.
	stepValidating
	stepExecuting
	stepComplete // all information is ready. REMARK: InitTransition only has some result parts - result and nextValidators
	stepError    // fails validation or execution
	stepCanceled // canceled. requested to cancel after complete execution, just remain stepFinished
)

func (s transitionStep) String() string {
	switch s {
	case stepInited:
		return "stepInited"
	case stepValidating:
		return "stepValidating"
	case stepValidated:
		return "stepValidated"
	case stepExecuting:
		return "stepExecuting"
	case stepComplete:
		return "stepComplete"
	case stepError:
		return "stepError"
	case stepCanceled:
		return "stepCanceled"
	default:
		return fmt.Sprintf("stepUnknown(%d)", int(s))
	}
}

type transitionCallbackForTrace struct {
	info *module.TraceInfo
}

func (cb *transitionCallbackForTrace) OnValidate(tr module.Transition, e error) {
	if e != nil {
		cb.info.Callback.OnEnd(e)
	}
}

func (cb *transitionCallbackForTrace) OnExecute(tr module.Transition, e error) {
	cb.info.Callback.OnEnd(e)
}

type transitionID struct {
	dummy int
}

type transitionContext struct {
	db    db.Database
	cm    contract.ContractManager
	eem   eeproxy.Manager
	chain module.Chain
	log   log.Logger
	plt   base.Platform
	tsc   *TxTimestampChecker
	sass  state.AccountSnapshot
	tim   TXIDManager
}

func (tc *transitionContext) onWorldFinalize(wss state.WorldSnapshot) {
	ass := wss.GetAccountSnapshot(state.SystemID)
	if ass != nil && ass.StorageChangedAfter(tc.sass) {
		regulator := tc.chain.Regulator()
		as := scoredb.NewStateStoreWith(ass)
		timeout := scoredb.NewVarDB(as, state.VarCommitTimeout).Int64()
		interval := scoredb.NewVarDB(as, state.VarBlockInterval).Int64()
		if timeout > 0 || interval > 0 {
			regulator.SetBlockInterval(
				time.Duration(interval)*time.Millisecond,
				time.Duration(timeout)*time.Millisecond)
		}

		tsThreshold := scoredb.NewVarDB(as, state.VarTimestampThreshold).Int64()
		if tsThreshold > 0 {
			tc.tsc.SetThreshold(time.Duration(tsThreshold) * time.Millisecond)
			tc.tim.OnThresholdChange()
		}
		tc.sass = ass
	}
	tc.plt.OnExtensionSnapshotFinalization(wss.GetExtensionSnapshot(), tc.log)
}

type transition struct {
	parent *transition
	pid    *transitionID
	id     *transitionID
	bi     module.BlockInfo
	pbi    module.BlockInfo
	csi    module.ConsensusInfo

	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList

	*transitionContext

	cb module.TransitionCallback

	// internal processing state
	step  transitionStep
	mutex sync.Mutex

	result         []byte
	worldSnapshot  state.WorldSnapshot
	patchReceipts  module.ReceiptList
	normalReceipts module.ReceiptList
	logsBloom      txresult.LogsBloom

	transactionCount int
	executeDuration  time.Duration
	flushDuration    time.Duration

	syncer ssync.Syncer

	ti *module.TraceInfo

	ptxIDs   TXIDLogger
	ntxIDs   TXIDLogger
	ptxCount int
	ntxCount int
}

func patchTransition(t *transition, bi module.BlockInfo, patchTXs module.TransactionList) *transition {
	if patchTXs == nil {
		patchTXs = transaction.NewTransactionListFromSlice(t.db, nil)
	}
	if len(patchTXs.Hash()) == 0 {
		bi = nil
	}
	return &transition{
		parent:             t.parent,
		pid:                t.pid,
		id:                 new(transitionID),
		bi:                 t.bi,
		pbi:                bi,
		csi:                t.csi,
		patchTransactions:  patchTXs,
		normalTransactions: t.normalTransactions,
		transitionContext:  t.transitionContext,
		step:               stepInited,
	}
}

func newTransition(
	parent *transition,
	patchtxs module.TransactionList,
	normaltxs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	alreadyValidated bool,
) *transition {
	var step transitionStep
	if alreadyValidated {
		step = stepValidated
	} else {
		step = stepInited
	}

	if patchtxs == nil {
		patchtxs = transaction.NewTransactionListFromSlice(parent.db, nil)
	}
	if normaltxs == nil {
		normaltxs = transaction.NewTransactionListFromSlice(parent.db, nil)
	}
	return &transition{
		parent:             parent,
		pid:                parent.id,
		id:                 new(transitionID),
		bi:                 bi,
		csi:                csi,
		patchTransactions:  patchtxs,
		normalTransactions: normaltxs,
		transitionContext:  parent.transitionContext,
		step:               step,
	}
}

func newWorldSnapshot(database db.Database, plt base.Platform, result []byte, vl module.ValidatorList) (state.WorldSnapshot, error) {
	var stateHash, extensionData []byte
	if len(result) > 0 {
		tr, err := newTransitionResultFromBytes(result)
		if err != nil {
			return nil, err
		}
		stateHash = tr.StateHash
		extensionData = tr.ExtensionData
	}
	var ess state.ExtensionSnapshot
	if plt != nil {
		ess = plt.NewExtensionSnapshot(database, extensionData)
	}
	return state.NewWorldSnapshot(database, stateHash, vl, ess), nil
}

// all parameters should be valid.
func newInitTransition(dbase db.Database,
	result []byte,
	validatorList module.ValidatorList,
	cm contract.ContractManager,
	em eeproxy.Manager, chain module.Chain,
	logger log.Logger, plt base.Platform,
	tsc *TxTimestampChecker,
	tim TXIDManager,
) (*transition, error) {
	wss, err := newWorldSnapshot(dbase, plt, result, validatorList)
	if err != nil {
		return nil, err
	}
	tr := &transition{
		id:                 new(transitionID),
		patchTransactions:  transaction.NewTransactionListFromSlice(dbase, nil),
		normalTransactions: transaction.NewTransactionListFromSlice(dbase, nil),
		patchReceipts:      txresult.NewReceiptListFromHash(dbase, nil),
		normalReceipts:     txresult.NewReceiptListFromHash(dbase, nil),
		bi:                 common.NewBlockInfo(0, 0),
		transitionContext: &transitionContext{
			db:    dbase,
			cm:    cm,
			eem:   em,
			chain: chain,
			log:   logger,
			plt:   plt,
			tsc:   tsc,
			tim:   tim,
		},
		step:          stepComplete,
		result:        result,
		worldSnapshot: wss,
		ptxIDs:        tim.NewLogger(module.TransactionGroupPatch, 0, 0),
		ntxIDs:        tim.NewLogger(module.TransactionGroupNormal, 0, 0),
	}
	return tr, nil
}

func (t *transition) PatchTransactions() module.TransactionList {
	return t.patchTransactions
}
func (t *transition) NormalTransactions() module.TransactionList {
	return t.normalTransactions
}

func (t *transition) PatchReceipts() module.ReceiptList {
	return t.patchReceipts
}
func (t *transition) NormalReceipts() module.ReceiptList {
	return t.normalReceipts
}

// startExecution executes this transition.
// The result is asynchronously notified by cb. canceler can be used
// to cancel it after calling Execute. After canceler returns true,
// all succeeding cb functions may not be called back.
// REMARK: It is assumed to be called once. Any additional call returns
// error.
func (t *transition) startExecution(setup func() error) (canceler func() bool, err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	switch t.step {
	case stepInited:
		t.step = stepValidating
	case stepValidated:
		t.step = stepExecuting
	default:
		return nil, errors.InvalidStateError.Errorf("Invalid transition state: %s", t.step)
	}

	if err := setup(); err != nil {
		return nil, err
	}

	if t.syncer == nil {
		go t.doExecute(t.step == stepExecuting)
	} else {
		go t.doForceSync()
	}

	return t.cancelExecution, nil
}

func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	if cb == nil {
		return nil, errors.IllegalArgumentError.New("TraceCallbackIsNil")
	}

	return t.startExecution(func() error {
		t.cb = cb
		return nil
	})
}

func (t *transition) ExecuteForTrace(ti module.TraceInfo) (canceler func() bool, err error) {
	if ti.Callback == nil {
		return nil, errors.IllegalArgumentError.New("TraceCallbackIsNil")
	}
	switch ti.Group {
	case module.TransactionGroupNormal:
		if _, err := t.normalTransactions.Get(ti.Index); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("InvalidTransactionIndex(n=%d)", ti.Index)
		}
	case module.TransactionGroupPatch:
		if _, err := t.patchTransactions.Get(ti.Index); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("InvalidTransactionIndex(n=%d)", ti.Index)
		}
	default:
		return nil, errors.IllegalArgumentError.Errorf("UnknownTransactionGroup(%d)", ti.Group)
	}

	// no need to validate the tx again for trace
	t.step = stepValidated

	return t.startExecution(func() error {
		if t.syncer != nil {
			return errors.InvalidStateError.New("TraceWithSyncTransition")
		}
		t.ti = &ti
		t.cb = &transitionCallbackForTrace{info: t.ti}
		return nil
	})
}

// Result returns service manager defined result bytes.
func (t *transition) Result() []byte {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.step != stepComplete {
		return nil
	}
	return t.result
}

// NextValidators returns the addresses of validators as a result of
// transaction processing.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) NextValidators() module.ValidatorList {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.step != stepComplete {
		return nil
	}
	return t.worldSnapshot.GetValidatorSnapshot()
}

// LogsBloom returns log bloom filter for this transition.
// It may return nil before cb.OnExecute is called back by Execute.
func (t *transition) LogsBloom() module.LogsBloom {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.step != stepComplete {
		return nil
	}
	return &t.logsBloom
}

func (t *transition) BlockInfo() module.BlockInfo {
	return t.bi
}

func (t *transition) newWorldContext(execution bool) (state.WorldContext, error) {
	var ws state.WorldState
	if t.parent != nil {
		var err error
		ws, err = state.WorldStateFromSnapshot(t.parent.worldSnapshot)
		if err != nil {
			return nil, err
		}
	} else {
		ws = state.NewWorldState(t.db, nil, nil, nil)
	}
	if execution {
		ws.EnableNodeCache()
	}
	return state.NewWorldContext(ws, t.bi, t.csi, t.plt), nil
}

func (t *transition) newContractContext(wc state.WorldContext) contract.Context {
	priority := eeproxy.ForTransaction
	if t.ti != nil {
		priority = eeproxy.ForQuery
	}
	return contract.NewContext(wc, t.cm, t.eem, t.chain, t.log, t.ti, priority)
}

func (t *transition) reportValidation(e error) bool {
	locker := common.LockForAutoCall(&t.mutex)
	defer locker.Unlock()

	t.log.Debugf("reportValidation(err=%+v)", e)

	switch t.step {
	case stepValidating, stepExecuting:
		if e != nil {
			t.step = stepError
		} else {
			t.step = stepExecuting
		}
		locker.CallAfterUnlock(func() {
			t.cb.OnValidate(t, e)
		})
		return true
	case stepCanceled:
		t.log.Tracef("Ignore error err=%+v", e)
	default:
		t.log.Tracef("Invalid state %s for err=%+v", t.step, e)
	}
	return false
}

func (t *transition) reportExecution(e error) bool {
	locker := common.LockForAutoCall(&t.mutex)
	defer locker.Unlock()

	t.log.Debugf("reportExecution(err=%+v)", e)

	switch t.step {
	case stepExecuting:
		if e != nil {
			t.log.Tracef("Execution failed with err=%+v", e)
			t.step = stepError
		} else {
			t.step = stepComplete
		}
		locker.CallAfterUnlock(func() {
			t.cb.OnExecute(t, e)
		})
		return true
	case stepCanceled:
		t.log.Tracef("Ignore error err=%+v", e)
	default:
		t.log.Tracef("Invalid state %s for err=%+v", t.step, e)
	}
	return false
}

func (t *transition) canceled() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.step == stepCanceled
}

func (t *transition) ensureRecordTXIDs(force bool) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.ensureRecordTXIDsInLock(force)
}

func (t *transition) ensureRecordTXIDsInLock(force bool) error {
	if t.ntxIDs != nil && t.ptxIDs != nil {
		return nil
	}
	if t.pbi != nil {
		t.ptxIDs = t.parent.ptxIDs.NewLogger(t.pbi.Height(), t.pbi.Timestamp())
	} else {
		t.ptxIDs = t.parent.ptxIDs.NewLogger(0, 0)
	}
	t.ntxIDs = t.parent.ntxIDs.NewLogger(t.bi.Height(), t.bi.Timestamp())
	if count, err := t.validateTXIDs(t.patchTransactions, t.ptxIDs, force); err != nil {
		return err
	} else {
		t.ptxCount = count
	}
	if count, err := t.validateTXIDs(t.normalTransactions, t.ntxIDs, force); err != nil {
		return err
	} else {
		t.ntxCount = count
	}
	return nil
}

func (t *transition) commitTXIDs(group module.TransactionGroup) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if err := t.ensureRecordTXIDsInLock(true); err != nil {
		return err
	}
	switch group {
	case module.TransactionGroupPatch:
		return t.ptxIDs.Commit()
	case module.TransactionGroupNormal:
		return t.ntxIDs.Commit()
	default:
		return errors.IllegalArgumentError.Errorf("InvalidTXGroup(group=%d)", group)
	}
}

func (t *transition) doForceSync() {
	if err := t.ensureRecordTXIDs(true); err != nil {
		t.reportValidation(err)
		return
	}
	t.reportValidation(nil)

	sr, err := t.syncer.ForceSync()
	if err != nil {
		t.reportExecution(err)
		return
	}
	t.logsBloom.SetInt64(0)
	for _, receipts := range []module.ReceiptList{sr.PatchReceipts, sr.NormalReceipts} {
		for itr := receipts.Iterator(); itr.Has(); itr.Next() {
			r, _ := itr.Get()
			t.logsBloom.Merge(r.LogsBloom())
		}
	}

	t.patchReceipts = sr.PatchReceipts
	t.normalReceipts = sr.NormalReceipts
	t.worldSnapshot = sr.Wss
	tresult := transitionResult{
		t.worldSnapshot.StateHash(),
		t.patchReceipts.Hash(),
		t.normalReceipts.Hash(),
		t.worldSnapshot.ExtensionData(),
	}
	t.result = tresult.Bytes()
	t.log.Debugf("ForceSyncDone(result=%#x)", t.result)
	t.reportExecution(nil)
}

func (t *transition) validateTXIDs(list module.TransactionList, logger TXIDLogger, force bool) (int, error) {
	count := 0
	for i := list.Iterator(); i.Has(); i.Next() {
		tx, _, err := i.Get()
		if err != nil {
			return 0, err
		}
		if err := logger.Add(tx.ID(), force); err != nil {
			return count, err
		}
		count += 1
	}
	return count, nil
}

func (t *transition) doExecute(alreadyValidated bool) {
	if !alreadyValidated {
		if err := t.ensureRecordTXIDs(false); err != nil {
			t.reportValidation(err)
			return
		}
		wc, err := t.newWorldContext(false)
		if err != nil {
			t.reportValidation(err)
			return
		}

		if err := t.plt.OnValidateTransactions(wc, t.patchTransactions, t.normalTransactions); err != nil {
			t.reportValidation(err)
			return
		}

		var tsr TimestampRange
		if t.pbi != nil {
			tsr = NewTimestampRange(t.pbi.Timestamp(),
				TransactionTimestampThreshold(wc, module.TransactionGroupPatch))
		} else {
			tsr = NewDummyTimeStampRange()
		}
		err = t.validateTxs(t.patchTransactions, wc, tsr)
		if err != nil {
			t.reportValidation(err)
			return
		}
		tsr = NewTxTimestampRangeFor(wc, module.TransactionGroupNormal)
		err = t.validateTxs(t.normalTransactions, wc, tsr)
		if err != nil {
			t.reportValidation(err)
			return
		}
	} else {
		if err := t.ensureRecordTXIDs(true); err != nil {
			t.reportValidation(err)
			return
		}

		wc, err := t.newWorldContext(false)
		if err != nil {
			t.reportValidation(err)
			return
		}
		if err := t.plt.OnValidateTransactions(wc, t.patchTransactions, t.normalTransactions); err != nil {
			t.reportValidation(err)
			return
		}
	}

	if !t.reportValidation(nil) {
		return
	}

	wc, err := t.newWorldContext(true)
	if err != nil {
		t.reportExecution(err)
		return
	}
	ctx := t.newContractContext(wc)
	ctx.ClearCache()
	ctx.SetProperty(contract.PropInitialSnapshot, ctx.GetSnapshot())

	startTime := time.Now()

	t.log.Debugf("Transition.doExecute: height=%d csi=%v", ctx.BlockHeight(), ctx.ConsensusInfo())

	if err := t.plt.OnExecutionBegin(ctx, t.log); err != nil {
		t.reportExecution(err)
		return
	}
	patchReceipts := make([]txresult.Receipt, t.ptxCount)
	if err := t.executeTxsSequential(t.patchTransactions, ctx, patchReceipts); err != nil {
		t.reportExecution(err)
		return
	}
	normalReceipts := make([]txresult.Receipt, t.ntxCount)
	if err := t.executeTxs(t.normalTransactions, ctx, normalReceipts); err != nil {
		t.reportExecution(err)
		return
	}
	cumulativeSteps := big.NewInt(0)
	gatheredFee := big.NewInt(0)
	virtualFee := new(big.Int)

	t.logsBloom.SetInt64(0)
	fixLostFeeByDeposit := ctx.Revision().Has(module.FixLostFeeByDeposit)
	for _, receipts := range [][]txresult.Receipt{patchReceipts, normalReceipts} {
		for _, r := range receipts {
			used := r.StepUsed()
			cumulativeSteps.Add(cumulativeSteps, used)
			r.SetCumulativeStepUsed(cumulativeSteps)

			virtualFee.Add(virtualFee, new(big.Int).Mul(used, r.StepPrice()))
			if fixLostFeeByDeposit {
				gatheredFee.Add(gatheredFee, r.Fee())
			} else {
				gatheredFee.Add(gatheredFee, r.FeeByEOA())
			}

			t.logsBloom.Merge(r.LogsBloom())
		}
	}
	t.patchReceipts = txresult.NewReceiptListFromSlice(t.db, patchReceipts)
	t.normalReceipts = txresult.NewReceiptListFromSlice(t.db, normalReceipts)

	// save gathered fee to treasury
	tr := ctx.GetAccountState(ctx.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Add(tb, gatheredFee))

	er := NewExecutionResult(t.patchReceipts, t.normalReceipts, virtualFee, gatheredFee)
	if err := t.plt.OnExecutionEnd(ctx, er, t.log); err != nil {
		t.reportExecution(err)
		return
	}

	t.worldSnapshot = ctx.GetSnapshot()

	txDuration := time.Now().Sub(startTime)
	txCount := t.ntxCount + t.ptxCount
	t.transactionCount = txCount
	t.executeDuration = txDuration

	elapsedMS := float64(txDuration/time.Microsecond) / 1000
	t.log.Infof("Transactions: %6d  Elapsed: %9.3f ms  PerTx: %7.1f µs  TPS: %9.2f",
		txCount, elapsedMS,
		elapsedMS*1000/float64(txCount),
		float64(txCount)/elapsedMS*1000)

	tresult := transitionResult{
		t.worldSnapshot.StateHash(),
		t.patchReceipts.Hash(),
		t.normalReceipts.Hash(),
		t.worldSnapshot.ExtensionData(),
	}
	t.result = tresult.Bytes()

	t.reportExecution(nil)
}

func (t *transition) validateTxs(l module.TransactionList, wc state.WorldContext, tsr TimestampRange) error {
	if l == nil {
		return nil
	}
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.canceled() {
			return ErrTransitionInterrupted
		}

		txi, _, err := i.Get()
		if err != nil {
			return errors.Wrap(err, "validateTxs: fail to get transaction")
		}
		tx := txi.(transaction.Transaction)

		if !tx.ValidateNetwork(t.chain.NID()) {
			return errors.InvalidNetworkError.New("InvalidNetworkID")
		}
		if err := tx.Verify(); err != nil {
			return err
		}
		if err := tsr.CheckTx(tx); err != nil {
			return err
		}
		if err := tx.PreValidate(wc, true); err != nil {
			return err
		}
	}
	return nil
}

func (t *transition) executeTxs(l module.TransactionList, ctx contract.Context, rctBuf []txresult.Receipt) error {
	if l == nil {
		return nil
	}
	if ctx.SkipTransactionEnabled() {
		// it will skip skippable transactions
		return t.executeTxsSequential(l, ctx, rctBuf)
	}
	if cc := t.chain.ConcurrencyLevel(); cc > 1 {
		return t.executeTxsConcurrent(cc, l, ctx, rctBuf)
	}
	return t.executeTxsSequential(l, ctx, rctBuf)
}

func (t *transition) finalizeNormalTransaction() error {
	if err := t.commitTXIDs(module.TransactionGroupNormal); err != nil {
		return err
	}
	return t.normalTransactions.Flush()
}

func (t *transition) finalizePatchTransaction() error {
	if err := t.commitTXIDs(module.TransactionGroupPatch); err != nil {
		return err
	}
	return t.patchTransactions.Flush()
}

func (t *transition) finalizeResult(noFlush bool, keepParent bool) error {
	var worldTS time.Time
	startTS := time.Now()
	if !noFlush {
		if t.syncer != nil {
			worldTS = time.Now()
			if err := t.syncer.Finalize(); err != nil {
				return errors.Wrap(err, "Fail to finalize with syncer")
			}
		} else {
			if err := t.worldSnapshot.Flush(); err != nil {
				return err
			}
			worldTS = time.Now()
			if err := t.patchReceipts.Flush(); err != nil {
				return err
			}
			if err := t.normalReceipts.Flush(); err != nil {
				return err
			}
		}
	}
	if !keepParent {
		t.parent = nil
	}
	finalTS := time.Now()

	t.onWorldFinalize(t.worldSnapshot)
	t.chain.Regulator().OnTxExecution(t.transactionCount, t.executeDuration, finalTS.Sub(startTS))
	t.log.Infof("finalizeResult() total=%s world=%s receipts=%s",
		finalTS.Sub(startTS), worldTS.Sub(startTS), finalTS.Sub(worldTS))
	return nil
}

func (t *transition) cancelExecution() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.step != stepComplete && t.step != stepError {
		if t.step != stepCanceled && t.syncer != nil {
			t.syncer.Stop()
		}
		t.step = stepCanceled
	} else if t.step == stepComplete {
		return false
	}
	return true
}

func (t *transition) Equal(tr module.Transition) bool {
	t2 := tr.(*transition)

	if t == t2 {
		return true
	}
	if t == nil || t2 == nil {
		return false
	}

	return t.patchTransactions.Equal(t2.patchTransactions) &&
		t.normalTransactions.Equal(t2.normalTransactions) &&
		common.BlockInfoEqual(t.bi, t2.bi) &&
		common.BlockInfoEqual(t.pbi, t2.pbi) &&
		common.ConsensusInfoEqual(t.csi, t2.csi) &&
		t.pid == t2.pid
}

// NewInitTransition creates initial transition based on the last result.
// It's only for development purpose. So, normally it should not be used.
func NewInitTransition(
	db db.Database,
	result []byte,
	vl module.ValidatorList,
	cm contract.ContractManager,
	em eeproxy.Manager, chain module.Chain,
	logger log.Logger, plt base.Platform,
	tsc *TxTimestampChecker,
) (module.Transition, error) {
	tim, err := NewTXIDManager(db, tsc)
	if err != nil {
		return nil, err
	}
	if tr, err := newInitTransition(db, result, vl, cm, em, chain, logger, plt, tsc, tim); err != nil {
		return nil, err
	} else {
		return tr, nil
	}
}

// NewTransition creates new transition based on the parent to execute
// given transactions under given environments.
// It's only for development purpose. So, normally it should not be used.
func NewTransition(
	parent module.Transition,
	patchtxs module.TransactionList,
	normaltxs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	alreadyValidated bool,
) module.Transition {
	return newTransition(
		parent.(*transition),
		patchtxs,
		normaltxs,
		bi,
		csi,
		alreadyValidated,
	)
}

// FinalizeTransition finalize parts of transition result without
// updating other information of service manager.
// It's only for development purpose. So, normally it should not be used.
func FinalizeTransition(tr module.Transition, opt int, noFlush bool) error {
	tst := tr.(*transition)
	if opt&module.FinalizeNormalTransaction == module.FinalizeNormalTransaction && !noFlush {
		if err := tst.finalizeNormalTransaction(); err != nil {
			return err
		}
	}
	if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction && !noFlush {
		if err := tst.finalizePatchTransaction(); err != nil {
			return err
		}
	}
	if opt&module.FinalizeResult == module.FinalizeResult {
		keepParent := (opt & module.KeepingParent) != 0
		if err := tst.finalizeResult(noFlush, keepParent); err != nil {
			return err
		}
	}
	return nil
}

type SyncManager interface {
	NewSyncer(ah, prh, nrh, vh, ed []byte, noBuffer bool) ssync.Syncer
}

func NewSyncTransition(
	tr module.Transition,
	sm SyncManager,
	result []byte, vl []byte,
	noBuffer bool,
) module.Transition {
	tst := tr.(*transition)
	ntr := newTransition(tst.parent, tst.patchTransactions, tst.normalTransactions, tst.bi, tst.csi, true)
	r, _ := newTransitionResultFromBytes(result)
	ntr.syncer = sm.NewSyncer(r.StateHash, r.PatchReceiptHash, r.NormalReceiptHash, vl, r.ExtensionData, noBuffer)
	return ntr
}

func PatchTransition(
	tr module.Transition,
	bi module.BlockInfo,
	ptxs module.TransactionList,
) module.Transition {
	return patchTransition(tr.(*transition), bi, ptxs)
}
