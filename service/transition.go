package service

import (
	"fmt"
	"math/big"
	"sync"
	"time"

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

type transitionID struct{}

type transition struct {
	parent *transition
	pid    *transitionID
	id     *transitionID
	bi     module.BlockInfo
	pbi    module.BlockInfo

	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList

	db    db.Database
	cm    contract.ContractManager
	eem   eeproxy.Manager
	chain module.Chain
	log   log.Logger
	plt   Platform

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
	tsc              *TxTimestampChecker

	syncer ssync.Syncer

	ti *module.TraceInfo
}

func patchTransition(t *transition, patchTXs module.TransactionList) *transition {
	if patchTXs == nil {
		patchTXs = transaction.NewTransactionListFromSlice(t.db, nil)
	}
	return &transition{
		parent:             t.parent,
		pid:                t.pid,
		id:                 new(transitionID),
		bi:                 t.bi,
		pbi:                nil,
		patchTransactions:  patchTXs,
		normalTransactions: t.normalTransactions,
		db:                 t.db,
		cm:                 t.cm,
		eem:                t.eem,
		chain:              t.chain,
		log:                t.log,
		plt:                t.plt,
		step:               stepInited,
		tsc:                t.tsc,
	}
}

func newTransition(parent *transition, patchtxs module.TransactionList,
	normaltxs module.TransactionList, bi module.BlockInfo, alreadyValidated bool,
	logger log.Logger, plt Platform,
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
		patchTransactions:  patchtxs,
		normalTransactions: normaltxs,
		db:                 parent.db,
		cm:                 parent.cm,
		tsc:                parent.tsc,
		eem:                parent.eem,
		step:               step,
		chain:              parent.chain,
		log:                logger,
		plt:                plt,
	}
}

// all parameters should be valid.
func newInitTransition(db db.Database,
	stateHash []byte,
	validatorList module.ValidatorList,
	es state.ExtensionSnapshot,
	cm contract.ContractManager,
	em eeproxy.Manager, chain module.Chain,
	logger log.Logger, plt Platform,
	tsc *TxTimestampChecker,
) (*transition, error) {
	ws := state.NewWorldState(db, stateHash, validatorList, es)

	return &transition{
		id:                 new(transitionID),
		patchTransactions:  transaction.NewTransactionListFromSlice(db, nil),
		normalTransactions: transaction.NewTransactionListFromSlice(db, nil),
		bi:                 common.NewBlockInfo(0, 0),
		db:                 db,
		cm:                 cm,
		eem:                em,
		step:               stepComplete,
		worldSnapshot:      ws.GetSnapshot(),
		chain:              chain,
		log:                logger,
		plt:                plt,
		tsc:                tsc,
	}, nil
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
	return state.NewWorldContext(ws, t.bi, t.plt), nil
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

func (t *transition) doForceSync() {
	t.reportValidation(nil)

	sr := t.syncer.ForceSync()
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
	t.reportExecution(nil)
}

func (t *transition) doExecute(alreadyValidated bool) {
	var normalCount, patchCount int
	if !alreadyValidated {
		wc, err := t.newWorldContext(false)
		if err != nil {
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
		patchCount, err = t.validateTxs(t.patchTransactions, wc, tsr)
		if err != nil {
			t.reportValidation(err)
			return
		}
		tsr = NewTxTimestampRangeFor(wc, module.TransactionGroupNormal)
		normalCount, err = t.validateTxs(t.normalTransactions, wc, tsr)
		if err != nil {
			t.reportValidation(err)
			return
		}
	} else {
		for i := t.patchTransactions.Iterator(); i.Has(); i.Next() {
			patchCount++
		}
		for i := t.normalTransactions.Iterator(); i.Has(); i.Next() {
			normalCount++
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
	ctx := contract.NewContext(wc, t.cm, t.eem, t.chain, t.log, t.ti)
	ctx.ClearCache()

	startTime := time.Now()

	patchReceipts := make([]txresult.Receipt, patchCount)
	if err := t.executeTxsSequential(t.patchTransactions, ctx, patchReceipts); err != nil {
		t.reportExecution(err)
		return
	}
	normalReceipts := make([]txresult.Receipt, normalCount)
	if err := t.executeTxs(t.normalTransactions, ctx, normalReceipts); err != nil {
		t.reportExecution(err)
		return
	}
	if err := t.plt.OnExecutionEnd(ctx); err != nil {
		t.reportExecution(err)
		return
	}

	cumulativeSteps := big.NewInt(0)
	gatheredFee := big.NewInt(0)
	fee := big.NewInt(0)

	t.logsBloom.SetInt64(0)
	for _, receipts := range [][]txresult.Receipt{patchReceipts, normalReceipts} {
		for _, r := range receipts {
			used := r.StepUsed()
			cumulativeSteps.Add(cumulativeSteps, used)
			r.SetCumulativeStepUsed(cumulativeSteps)

			fee.Mul(r.StepPrice(), used)
			gatheredFee.Add(gatheredFee, fee)

			t.logsBloom.Merge(r.LogsBloom())
		}
	}
	t.patchReceipts = txresult.NewReceiptListFromSlice(t.db, patchReceipts)
	t.normalReceipts = txresult.NewReceiptListFromSlice(t.db, normalReceipts)

	// save gathered fee to treasury
	tr := ctx.GetAccountState(ctx.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Add(tb, gatheredFee))

	t.worldSnapshot = ctx.GetSnapshot()

	txDuration := time.Now().Sub(startTime)
	txCount := patchCount + normalCount
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

func (t *transition) validateTxs(l module.TransactionList, wc state.WorldContext, tsr TimestampRange) (int, error) {
	if l == nil {
		return 0, nil
	}
	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.canceled() {
			return 0, ErrTransitionInterrupted
		}

		txi, _, err := i.Get()
		if err != nil {
			return 0, errors.Wrap(err, "validateTxs: fail to get transaction")
		}
		tx := txi.(transaction.Transaction)

		if !tx.ValidateNetwork(t.chain.NID()) {
			return 0, errors.InvalidNetworkError.New("InvalidNetworkID")
		}
		if err := tx.Verify(); err != nil {
			return 0, err
		}
		if err := tsr.CheckTx(tx); err != nil {
			return 0, err
		}
		if err := tx.PreValidate(wc, true); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (t *transition) executeTxs(l module.TransactionList, ctx contract.Context, rctBuf []txresult.Receipt) error {
	if l == nil {
		return nil
	}
	rev := ctx.Revision()
	if ctx.SkipTransactionEnabled() {
		zero := big.NewInt(0)
		for itr, idx := l.Iterator(), 0; itr.Has(); idx, _ = idx+1, itr.Next() {
			txi, _, _ := itr.Get()
			txo := txi.(transaction.Transaction)
			rct := txresult.NewReceipt(t.db, rev, txo.To())
			rct.SetResult(module.StatusSkipTransaction, zero, zero, nil)
			rct.SetCumulativeStepUsed(zero)
			rctBuf[idx] = rct
		}
		return nil
	}
	if cc := t.chain.ConcurrencyLevel(); cc > 1 {
		return t.executeTxsConcurrent(cc, l, ctx, rctBuf)
	}
	return t.executeTxsSequential(l, ctx, rctBuf)
}

func (t *transition) finalizeNormalTransaction() error {
	return t.normalTransactions.Flush()
}

func (t *transition) finalizePatchTransaction() error {
	return t.patchTransactions.Flush()
}

func (t *transition) finalizeResult() error {
	// TODO check worldTS
	var worldTS time.Time
	startTS := time.Now()
	if t.syncer != nil {
		log.Debugf("finalizeResult with syncer\n")
		if err := t.syncer.Finalize(); err != nil {
			log.Errorf("Failed to finalize with state syncer err(%+v)\n", err)
			return err
		}
		return nil
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
		t.parent = nil
	}
	finalTS := time.Now()

	regulator := t.chain.Regulator()
	ass := t.worldSnapshot.GetAccountSnapshot(state.SystemID)
	if ass != nil {
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
			t.tsc.SetThreshold(time.Duration(tsThreshold) * time.Millisecond)
		}
	}
	t.plt.OnExtensionSnapshotFinalization(t.worldSnapshot.GetExtensionSnapshot())
	regulator.OnTxExecution(t.transactionCount, t.executeDuration, finalTS.Sub(startTS))

	t.log.Infof("finalizeResult() total=%s world=%s receipts=%s",
		finalTS.Sub(startTS), worldTS.Sub(startTS), finalTS.Sub(worldTS))
	return nil
}

func (t *transition) cancelExecution() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.step != stepComplete && t.step != stepError {
		t.step = stepCanceled
	} else if t.step == stepComplete {
		return false
	}
	return true
}

func equalBlockInfo(bi1 module.BlockInfo, bi2 module.BlockInfo) bool {
	if bi1 == nil && bi2 == nil {
		return true
	}
	if bi1 == nil || bi2 == nil {
		return false
	}
	return bi1.Timestamp() == bi2.Timestamp() && bi1.Height() == bi2.Height()
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
		equalBlockInfo(t.bi, t2.bi) &&
		equalBlockInfo(t.pbi, t2.pbi) &&
		t.pid == t2.pid
}
