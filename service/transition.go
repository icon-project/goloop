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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/service/eeproxy"
	ssync "github.com/icon-project/goloop/service/sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
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

type transition struct {
	parent *transition
	bi     module.BlockInfo

	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList

	db    db.Database
	cm    contract.ContractManager
	eem   eeproxy.Manager
	chain module.Chain
	log   log.Logger

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
}

type transitionResult struct {
	StateHash         []byte
	PatchReceiptHash  []byte
	NormalReceiptHash []byte
}

func newTransitionResultFromBytes(bs []byte) (*transitionResult, error) {
	tresult := new(transitionResult)
	if _, err := codec.UnmarshalFromBytes(bs, tresult); err != nil {
		return nil, err
	}
	return tresult, nil
}

func (tr *transitionResult) Bytes() []byte {
	if bs, err := codec.MarshalToBytes(tr); err != nil {
		log.Debug("Fail to marshal transitionResult")
		return nil
	} else {
		return bs
	}
}

func patchTransition(t *transition, patchTXs module.TransactionList) *transition {
	if patchTXs == nil {
		patchTXs = transaction.NewTransactionListFromSlice(t.db, nil)
	}
	return &transition{
		parent:             t.parent,
		bi:                 t.bi,
		patchTransactions:  patchTXs,
		normalTransactions: t.normalTransactions,
		db:                 t.db,
		cm:                 t.cm,
		eem:                t.eem,
		chain:              t.chain,
		log:                t.log,
		step:               stepInited,
		tsc:                t.tsc,
	}
}

func newTransition(parent *transition, patchtxs module.TransactionList,
	normaltxs module.TransactionList, bi module.BlockInfo, alreadyValidated bool,
	logger log.Logger,
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
	}
}

// all parameters should be valid.
func newInitTransition(db db.Database, result []byte,
	validatorList module.ValidatorList, cm contract.ContractManager,
	em eeproxy.Manager, chain module.Chain,
	logger log.Logger,
	tsc *TxTimestampChecker,
) (*transition, error) {
	var tresult transitionResult
	if len(result) > 0 {
		if _, err := codec.UnmarshalFromBytes(result, &tresult); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("InvalidResult(%x)", result)
		}
	}
	ws := state.NewWorldState(db, tresult.StateHash, validatorList)

	return &transition{
		patchTransactions:  transaction.NewTransactionListFromSlice(db, nil),
		normalTransactions: transaction.NewTransactionListFromSlice(db, nil),
		bi:                 &blockInfo{int64(0), int64(0)},
		db:                 db,
		cm:                 cm,
		eem:                em,
		step:               stepComplete,
		worldSnapshot:      ws.GetSnapshot(),
		chain:              chain,
		log:                logger,
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

// Execute executes this transition.
// The result is asynchronously notified by cb. canceler can be used
// to cancel it after calling Execute. After canceler returns true,
// all succeeding cb functions may not be called back.
// REMARK: It is assumed to be called once. Any additional call returns
// error.
func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
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
	t.cb = cb
	if t.syncer == nil {
		go t.executeSync(t.step == stepExecuting)
	} else {
		//	// TODO : stepSyncing
		go t.forceSync()
	}

	return t.cancelExecution, nil
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

func (t *transition) newWorldContext() (state.WorldContext, error) {
	var ws state.WorldState
	if t.parent != nil {
		var err error
		ws, err = state.WorldStateFromSnapshot(t.parent.worldSnapshot)
		if err != nil {
			return nil, err
		}
	} else {
		ws = state.NewWorldState(t.db, nil, nil)
	}
	return state.NewWorldContext(ws, t.bi), nil
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

func (t *transition) forceSync() {
	// TODO skip t.cb.OnValidate()
	t.log.Debugf("syncer = %v\n", t.syncer)

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
	}
	t.result = tresult.Bytes()
	t.reportExecution(nil)
}

func (t *transition) executeSync(alreadyValidated bool) {
	var normalCount, patchCount int
	if !alreadyValidated {
		wc, err := t.newWorldContext()
		if err != nil {
			t.reportValidation(err)
			return
		}
		patchCount, err = t.validateTxs(t.patchTransactions, wc)
		if err != nil {
			t.reportValidation(err)
			return
		}
		normalCount, err = t.validateTxs(t.normalTransactions, wc)
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

	wc, err := t.newWorldContext()
	if err != nil {
		t.reportExecution(err)
		return
	}
	ctx := contract.NewContext(wc, t.cm, t.eem, t.chain, t.log)

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
	trbal := tr.GetBalance()
	trbal.Add(trbal, gatheredFee)
	tr.SetBalance(trbal)

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
	}
	t.result = tresult.Bytes()

	t.reportExecution(nil)
}

func (t *transition) validateTxs(l module.TransactionList, wc state.WorldContext) (int, error) {
	if l == nil {
		return 0, nil
	}
	cnt := 0
	tsRange := NewTimestampRangeFor(wc)
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.canceled() {
			return 0, ErrTransitionInterrupted
		}

		txi, _, err := i.Get()
		if err != nil {
			return 0, errors.Wrap(err, "validateTxs: fail to get transaction")
		}
		tx := txi.(transaction.Transaction)

		if err := tx.Verify(); err != nil {
			return 0, err
		}
		if err := tsRange.CheckTx(tx); err != nil {
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
	if ctx.SkipTransactionEnabled() {
		fakeBuf := make([]txresult.Receipt, len(rctBuf))
		wss := ctx.GetSnapshot()
		err := t.executeTxsSequential(l, ctx, fakeBuf)
		if err != nil {
			t.log.Warnf("It fails to execute transactions err=%v", err)
			t.log.Debugf("Failed reason err=%+v", err)
			return err
		}
		// TODO dump result for survey
		ctx.Reset(wss)
		for idx := 0; idx < len(rctBuf); idx++ {
			rct := txresult.NewReceipt(fakeBuf[idx].To())
			zero := big.NewInt(0)
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
