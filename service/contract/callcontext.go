package contract

import (
	"math/big"
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

type (
	CallContext interface {
		Context
		QueryMode() bool
		Call(handler ContractHandler, limit *big.Int) (error, *big.Int, *codec.TypedObj, module.Address)
		OnResult(status error, stepUsed *big.Int, result *codec.TypedObj, addr module.Address)
		OnCall(handler ContractHandler, limit *big.Int)
		OnEvent(addr module.Address, indexed, data [][]byte)
		GetBalance(module.Address) *big.Int
		ReserveExecutor() error
		GetProxy(eeType state.EEType) eeproxy.Proxy
		Dispose()
		StepUsed() *big.Int
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
		SetFeeProportion(addr module.Address, portion int)
		RedeemSteps(s *big.Int) (*big.Int, error)
		GetRedeemLogs(r txresult.Receipt) bool
		ClearRedeemLogs()
		DoIOTask(func())
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
	isQuery  bool
	executor *eeproxy.Executor
	nextEID  int
	nextFID  int

	lock   sync.Mutex
	frame  *callFrame
	waiter chan interface{}
	calls  int64

	timer   <-chan time.Time
	ioStart *time.Time
	ioTime  time.Duration

	payers *stepPayers

	log *trace.Logger
}

func NewCallContext(ctx Context, limit *big.Int, isQuery bool) CallContext {
	logger := trace.LoggerOf(ctx.Logger())
	ti := ctx.TraceInfo()
	if ti != nil {
		if info := ctx.TransactionInfo(); info != nil {
			if info.Group == ti.Group && int(info.Index) == ti.Index {
				logger = trace.NewLogger(logger.Logger, ti.Callback)
			}
		}
	}

	return &callContext{
		Context: ctx,
		isQuery: isQuery,
		nextEID: initialEID,
		nextFID: firstFID,
		frame:   NewFrame(nil, nil, limit, isQuery),

		waiter: make(chan interface{}, 8),
		log:    logger,
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
	handler.Init(cc.nextFID, cc.Logger())
	frame := NewFrame(cc.frame, handler, limit, false)
	if !frame.isQuery {
		frame.snapshot = cc.GetSnapshot()
	}
	cc.log.TSystemf("FRAME[%d] START parent=FRAME[%d]", cc.nextFID, cc.frame.fid)
	frame.fid = cc.nextFID
	cc.nextFID += 1
	cc.frame = frame
	return frame
}

func (cc *callContext) popFrame(success bool) *callFrame {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	frame := cc.frame
	cc.log.TSystemf("FRAME[%d] END success=%v steps=%d", frame.fid, success, &frame.stepUsed)
	if !frame.isQuery {
		if success {
			frame.parent.pushBackEventLogsOf(frame)
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

func (cc *callContext) enterQueryMode() {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.frame.enterQueryMode(cc)
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
	cc.log.Warnf("cleanUpFrames() TX=<%#x> err=%+v", cc.GetInfo()[state.InfoTxHash], err)
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
			cc.OnResult(err, parent.getStepAvailable(), nil, nil)
		}
		return true
	} else {
		return false
	}
}

func (cc *callContext) OnResult(status error, stepUsed *big.Int, result *codec.TypedObj, addr module.Address) {
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
	cc.log.TSystemf("FRAME[%d] EVENT score=%s sig=%s indexed=%v data=%v",
		cc.FrameID(), addr, indexed[0],
		common.SliceOfHexBytes(indexed[1:]),
		common.SliceOfHexBytes(data))
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
		priority := eeproxy.ForTransaction
		if cc.isQuery {
			priority = eeproxy.ForQuery
		}
		cc.executor = cc.EEManager().GetExecutor(priority)
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
	}
}

func (cc *callContext) StepUsed() *big.Int {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	return cc.frame.getStepUsed()
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
	cc.log.TSystemf("FRAME[%d] STEP apply type=%s count=%d cost=%s total=%s", cc.frame.fid, t, n, steps, &cc.frame.stepUsed)
	return ok
}

func (cc *callContext) ApplyCallSteps() error {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	cc.calls += 1
	if cc.calls-1 > InterCallLimit {
		cc.log.TSystemf("FRAME[%d] too many inter-calls count=%d", cc.frame.fid, cc.calls-1)
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
	cc.log.TSystemf("FRAME[%d] STEP apply cost=%s total=%d", cc.frame.fid, s, &cc.frame.stepUsed)
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
		if portion == 0 {
			cc.payers = nil
		} else {
			cc.payers = &stepPayers{
				payer: addr, portion: portion,
			}
		}
	}
}

func (cc *callContext) RedeemSteps(s *big.Int) (*big.Int, error) {
	if cc.payers != nil {
		return cc.payers.PaySteps(cc, s)
	}
	return nil, nil
}

func (cc *callContext) GetRedeemLogs(r txresult.Receipt) bool {
	if cc.payers != nil {
		return cc.payers.GetLogs(r)
	}
	return false
}

func (cc *callContext) ClearRedeemLogs() {
	if cc.payers != nil {
		cc.payers.ClearLogs()
	}
}

type stepPayers struct {
	payer   module.Address
	portion int
	payed   *big.Int
}

func (p *stepPayers) PaySteps(cc CallContext, s *big.Int) (*big.Int, error) {
	sp := new(big.Int).SetInt64(int64(p.portion))
	sp.Mul(sp, s).Div(sp, big.NewInt(100))
	as := cc.GetAccountState(p.payer.ID())
	payed, err := as.PaySteps(cc, sp)
	if err != nil {
		return nil, err
	}
	if payed != nil && payed.Sign() > 0 {
		p.payed = payed
	}
	return payed, nil
}

func (p *stepPayers) GetLogs(r txresult.Receipt) bool {
	if p.payed != nil {
		r.AddPayment(p.payer, p.payed)
		return true
	}
	return false
}

func (p *stepPayers) ClearLogs() {
	p.payed = nil
}
