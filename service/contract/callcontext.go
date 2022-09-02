package contract

import (
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	InterCallLimit = 1024
)

type ResultFlag int

const (
	ResultForceRerun ResultFlag = 1 << iota
)

type (
	CallContext interface {
		Context
		QueryMode() bool
		Call(handler ContractHandler, limit *big.Int) (error, *big.Int, *codec.TypedObj, module.Address)
		OnResult(status error, flags ResultFlag, stepUsed *big.Int, result *codec.TypedObj, addr module.Address)
		OnCall(handler ContractHandler, limit *big.Int)
		OnEvent(addr module.Address, indexed, data [][]byte)
		GetBalance(module.Address) *big.Int
		ReserveExecutor() error
		GetProxy(eeType state.EEType) eeproxy.Proxy
		Dispose()
		StepUsed() *big.Int
		SumOfStepUsed() *big.Int
		StepAvailable() *big.Int
		ApplySteps(t state.StepType, n int) bool
		ApplyCallSteps() error
		DeductSteps(s *big.Int) bool
		ResetStepLimit(s *big.Int)
		GetEventLogs(r txresult.Receipt)
		EnterQueryMode()
		SetFrameCodeID(id []byte)
		GetLastEIDOf(id []byte) int
		NewExecution() int
		GetReturnEID() int
		FrameID() int
		FrameLogger() *trace.Logger
		GetCustomLogs(name string, ot reflect.Type) CustomLogs
		SetFeeProportion(addr module.Address, portion int)
		RedeemSteps(s *big.Int) (*big.Int, error)
		GetRedeemLogs(r txresult.Receipt) bool
		ClearRedeemLogs()
		DoIOTask(func())
		ResultFlags() ResultFlag
	}
	callResultMessage struct {
		status   error
		stepUsed *big.Int
		result   *codec.TypedObj
		addr     module.Address
	}

	callRequestMessage struct {
		handler   ContractHandler
		stepLimit *big.Int
	}
)

const (
	unknownEID = 0
	initialEID = 1
)

const (
	unknownFID = 0
	baseFID    = 1 // ID for base frame  (Default + Input + Call)
	firstFID   = 2 // ID for first frame (Executor + Child)
)

type callContext struct {
	Context
	executor *eeproxy.Executor
	nextEID  int
	nextFID  int

	resultFlags ResultFlag

	lock   sync.Mutex
	frame  *callFrame
	waiter chan interface{}
	calls  int64

	timer   <-chan time.Time
	ioStart *time.Time
	ioTime  time.Duration

	log *trace.Logger
}

func prefixForFrame(id int) string {
	return fmt.Sprintf("FRAME[%d] ", id)
}

func NewCallContext(ctx Context, limit *big.Int, isQuery bool) CallContext {
	traceLogger := ctx.GetTraceLogger(module.EPhaseTransaction)
	frameLogger := traceLogger.WithTPrefix(prefixForFrame(baseFID))
	return &callContext{
		Context: ctx,
		nextEID: initialEID,
		nextFID: firstFID,
		frame:   NewFrame(nil, nil, limit, isQuery, frameLogger),

		waiter: make(chan interface{}, 8),
		log:    traceLogger,
	}
}

func (cc *callContext) QueryMode() bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	return cc.frame.isQuery
}

func (cc *callContext) Logger() log.Logger {
	return cc.log
}

func (cc *callContext) pushFrame(handler ContractHandler, limit *big.Int) *callFrame {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	logger := cc.log.WithTPrefix(prefixForFrame(cc.nextFID))
	handler.SetTraceLogger(logger)
	frame := NewFrame(cc.frame, handler, limit, false, logger)
	if !frame.isQuery {
		frame.snapshot = cc.GetSnapshot()
	}
	logger.OnFrameEnter(cc.frame.fid)
	frame.fid = cc.nextFID
	cc.nextFID += 1
	cc.frame = frame
	return frame
}

func (cc *callContext) popFrame(success bool) *callFrame {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	frame := cc.frame
	cc.frame.log.OnFrameExit(success, &frame.stepUsed)
	if !frame.isQuery {
		if success {
			frame.parent.applyFrameLogsOf(frame)
			frame.parent.applyFeePayerInfoOf(frame)
		} else {
			cc.Reset(frame.snapshot)
		}
	}
	if success {
		frame.parent.mergeLastEIDMap(frame)
	}
	cc.frame = frame.parent
	return frame
}

func (cc *callContext) FrameID() int {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	if cc.frame != nil {
		return cc.frame.fid
	} else {
		return unknownFID
	}
}

func (cc *callContext) FrameLogger() *trace.Logger {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	if cc.frame != nil {
		return cc.frame.log
	} else {
		return cc.log
	}
}

func (cc *callContext) isInAsyncFrame() bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	_, ok := cc.frame.handler.(AsyncContractHandler)
	return ok
}

func (cc *callContext) addLogToFrame(addr module.Address, indexed [][]byte, data [][]byte) error {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.frame.log.TSystemf("EVENT score=%s sig=%s indexed=%v data=%v",
		addr, indexed[0],
		common.SliceOfHexBytes(indexed[1:]),
		common.SliceOfHexBytes(data))
	cc.frame.addLog(addr, indexed, data)
	return nil
}

func (cc *callContext) validateStatus(status error) error {
	if status != nil && !cc.Context.Revision().ExpandErrorCode() {
		code, _ := scoreresult.StatusOf(status)
		if code > module.StatusLimitRev5 && code <= module.StatusLimit {
			status = scoreresult.WithStatus(status, module.StatusLimitRev5)
		}
	}
	return status
}

func (cc *callContext) Call(handler ContractHandler, limit *big.Int) (error, *big.Int, *codec.TypedObj, module.Address) {
	frame := cc.pushFrame(handler, limit)
	done, status, result, addr := cc.runFrame(frame)
	if done {
		cc.handleResult(frame, status, result, addr)
	} else {
		status, result, addr = cc.waitResult(frame)
	}
	return status, frame.getStepUsed(), result, addr
}

func (cc *callContext) runFrame(frame *callFrame) (bool, error, *codec.TypedObj, module.Address) {
	switch handler := frame.handler.(type) {
	case SyncContractHandler:
		status, result, addr := handler.ExecuteSync(cc)
		return true, cc.validateStatus(status), result, addr
	case AsyncContractHandler:
		if status := handler.ExecuteAsync(cc); status != nil {
			return true, cc.validateStatus(status), nil, nil
		}
		return false, nil, nil, nil
	default:
		cc.log.Panicf("UnsupportedHandler(handler=%T)", frame.handler)
		return true,
			scoreresult.UnknownFailureError.Errorf("UnsupportedHandler(handler=%T)", frame.handler),
			nil, nil
	}
}

func (cc *callContext) DoIOTask(f func()) {
	cc.lock.Lock()
	start := time.Now()
	cc.ioStart = &start
	cc.lock.Unlock()

	f()

	cc.lock.Lock()
	cc.ioTime += time.Now().Sub(start)
	cc.ioStart = nil
	cc.lock.Unlock()
}

func (cc *callContext) getTimer(update bool) <-chan time.Time {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	if cc.Revision().Has(module.LegacyNoTimeout) {
		if cc.timer == nil {
			cc.timer = make(chan time.Time)
		}
		return cc.timer
	}

	if cc.timer == nil {
		cc.timer = time.After(cc.TransactionTimeout())
		return cc.timer
	}
	if update {
		if cc.ioStart != nil {
			now := time.Now()
			cc.ioTime += now.Sub(*cc.ioStart)
			*cc.ioStart = now
		}
		if cc.ioTime > 0 {
			cc.timer = time.After(cc.ioTime)
			cc.ioTime = 0
			return cc.timer
		} else {
			return nil
		}
	} else {
		return cc.timer
	}
}

func (cc *callContext) waitResult(target *callFrame) (error, *codec.TypedObj, module.Address) {
	timer := cc.getTimer(false)
	for {
		select {
		case <-timer:
			timer = cc.getTimer(true)
			if timer != nil {
				continue
			}
			cc.cleanUpFrames(target, scoreresult.ErrTimeout)
			return scoreresult.ErrTimeout, nil, nil
		case msg := <-cc.waiter:
			switch msg := msg.(type) {
			case *callResultMessage:
				status := cc.validateStatus(msg.status)
				cc.DeductSteps(msg.stepUsed)
				if cc.handleResult(target, status, msg.result, msg.addr) {
					continue
				}
				return status, msg.result, msg.addr
			case *callRequestMessage:
				frame := cc.pushFrame(msg.handler, msg.stepLimit)
				if done, status, result, addr := cc.runFrame(frame); done {
					if cc.handleResult(target, status, result, addr) {
						continue
					}
					return status, result, addr
				}
			default:
				cc.log.Panicf("Invalid message=%[1]T %+[1]v", msg)
			}
		}
	}
}

func (cc *callContext) cleanUpFrames(target *callFrame, err error) {
	cc.log.Warnf("cleanUpFrames() TX=<%#x> err=%+v", cc.TransactionID(), err)
	l := common.Lock(&cc.lock)
	defer l.Unlock()
	achs := make([]AsyncContractHandler, 0, 16)
	for cc.frame != nil && cc.frame.handler != nil {
		frame := cc.frame
		cc.frame = frame.parent
		if ach, ok := frame.handler.(AsyncContractHandler); ok {
			achs = append(achs, ach)
		}
		if frame == target {
			break
		}
	}
	l.Unlock()

	if !target.isQuery {
		cc.Reset(target.snapshot)
	}
	for _, h := range achs {
		h.Dispose()
	}

	if cc.executor != nil {
		cc.executor.Kill()
		cc.executor = nil
	}
}

func (cc *callContext) handleResult(target *callFrame, status error, result *codec.TypedObj, addr module.Address) bool {
	if code := errors.CodeOf(status); code == scoreresult.TimeoutError ||
		code == errors.ExecutionFailError || errors.IsCriticalCode(code) {
		cc.cleanUpFrames(target, status)
		return false
	}

	current := cc.popFrame(status == nil)
	if current == nil {
		return false
	}

	if ach, ok := current.handler.(AsyncContractHandler); ok {
		ach.Dispose()
	}

	if current == target {
		return false
	}

	parent := current.parent
	if parent == nil || parent.handler == nil {
		cc.log.Panicf("ROOT frame shouldn't be reached or popped parent=%v", parent)
		return false
	}
	if ach, ok := parent.handler.(AsyncContractHandler); ok {
		err := ach.SendResult(status, current.getStepUsed(), result)
		if err != nil {
			cc.OnResult(err, 0, parent.getStepAvailable(), nil, nil)
		}
		return true
	} else {
		return false
	}
}

func (cc *callContext) OnResult(status error, flags ResultFlag, stepUsed *big.Int, result *codec.TypedObj, addr module.Address) {
	cc.resultFlags |= flags
	cc.sendMessage(&callResultMessage{
		status:   status,
		stepUsed: stepUsed,
		result:   result,
		addr:     addr,
	})
}

func (cc *callContext) OnCall(handler ContractHandler, limit *big.Int) {
	cc.sendMessage(&callRequestMessage{handler, limit})
}

func (cc *callContext) sendMessage(msg interface{}) error {
	if !cc.isInAsyncFrame() {
		return nil
	}
	cc.waiter <- msg
	return nil
}

func (cc *callContext) OnEvent(addr module.Address, indexed, data [][]byte) {
	if err := cc.addLogToFrame(addr, indexed, data); err != nil {
		cc.log.Errorf("Fail to log err=%+v", err)
	}
}

func (cc *callContext) GetBalance(addr module.Address) *big.Int {
	if ass := cc.GetAccountSnapshot(addr.ID()); ass != nil {
		return ass.GetBalance()
	} else {
		return big.NewInt(0)
	}
}

func (cc *callContext) ReserveExecutor() error {
	if cc.executor == nil {
		cc.executor = cc.EEManager().GetExecutor(cc.EEPriority())
	}
	return nil
}

func (cc *callContext) GetProxy(eeType state.EEType) eeproxy.Proxy {
	cc.ReserveExecutor()
	return cc.executor.Get(string(eeType))
}

func (cc *callContext) Dispose() {
	if cc.executor != nil {
		cc.executor.Release()
		cc.executor = nil
	}
}

func (cc *callContext) StepUsed() *big.Int {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	return cc.frame.getStepUsed()
}

func (cc *callContext) SumOfStepUsed() *big.Int {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	used := new(big.Int)
	for frame := cc.frame; frame != nil; frame = frame.parent {
		used.Add(used, frame.getStepUsed())
	}
	return used
}

func (cc *callContext) ResetStepLimit(limit *big.Int) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.frame.stepLimit = limit
	cc.frame.stepUsed.SetInt64(0)
}

func (cc *callContext) StepAvailable() *big.Int {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	steps := cc.frame.getStepAvailable()
	return steps
}

func (cc *callContext) ApplySteps(t state.StepType, n int) bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	return cc.applyStepsInLock(t, n)
}

func (cc *callContext) applyStepsInLock(t state.StepType, n int) bool {
	steps := big.NewInt(cc.StepsFor(t, n))
	ok := cc.frame.deductSteps(steps)
	cc.frame.log.TSystemf("STEP apply type=%s count=%d cost=%s total=%s", t, n, steps, &cc.frame.stepUsed)
	return ok
}

func (cc *callContext) ApplyCallSteps() error {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.calls += 1
	if cc.calls-1 > InterCallLimit {
		cc.frame.log.TSystemf("CONTEXT too many inter-calls count=%d", cc.calls-1)
		return scoreresult.IllegalFormatError.New("TooManyExternalCalls")
	}
	if ok := cc.applyStepsInLock(state.StepTypeContractCall, 1); !ok {
		return scoreresult.OutOfStepError.New("OutOfStepFor(contractCall)")
	}
	return nil
}

func (cc *callContext) DeductSteps(s *big.Int) bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	ok := cc.frame.deductSteps(s)
	cc.frame.log.TSystemf("STEP apply cost=%s total=%d", s, &cc.frame.stepUsed)
	return ok
}

func (cc *callContext) GetEventLogs(r txresult.Receipt) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.frame.getEventLogs(r)
}

func (cc *callContext) EnterQueryMode() {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.frame.enterQueryMode(cc)
}

func (cc *callContext) SetFrameCodeID(id []byte) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.frame.setCodeID(id)
}

func (cc *callContext) GetLastEIDOf(id []byte) int {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	return cc.frame.getLastEIDOf(id)
}

func (cc *callContext) NewExecution() int {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	eid := cc.nextEID
	cc.frame.newExecution(eid)
	cc.nextEID += 1
	return eid
}

func (cc *callContext) GetReturnEID() int {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	return cc.frame.getReturnEID()
}

func (cc *callContext) SetFeeProportion(addr module.Address, portion int) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	if cc.frame.fid == firstFID {
		// delegate SetFeeProportion of the first frame to the base frame
		cc.frame.parent.feePayers.SetFeeProportion(addr, portion)
	} else if cc.Revision().Has(module.MultipleFeePayers) {
		cc.frame.feePayers.SetFeeProportion(addr, portion)
	}
}

func (cc *callContext) RedeemSteps(s *big.Int) (*big.Int, error) {
	return cc.frame.feePayers.PaySteps(cc, s)
}

func (cc *callContext) GetRedeemLogs(r txresult.Receipt) bool {
	return cc.frame.feePayers.GetLogs(r)
}

func (cc *callContext) ClearRedeemLogs() {
	cc.frame.feePayers.ClearLogs()
}

func (cc *callContext) GetCustomLogs(name string, ot reflect.Type) CustomLogs {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	top, _ := cc.GetProperty(name).(CustomLogs)
	return cc.frame.getFrameData(name, ot, top)
}

func (cc *callContext) ResultFlags() ResultFlag {
	return cc.resultFlags
}
