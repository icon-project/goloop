package contract

import (
	"container/list"
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

type (
	CallContext interface {
		Context
		QueryMode() bool
		Call(ContractHandler) (error, *big.Int, *codec.TypedObj, module.Address)
		OnResult(status error, stepUsed *big.Int, result *codec.TypedObj, addr module.Address)
		OnCall(ContractHandler)
		OnEvent(addr module.Address, indexed, data [][]byte)
		GetBalance(module.Address) *big.Int
		ReserveExecutor() error
		GetProxy(eeType state.EEType) eeproxy.Proxy
		Dispose()
	}
	callResultMessage struct {
		status   error
		stepUsed *big.Int
		result   *codec.TypedObj
		addr     module.Address
	}

	callRequestMessage struct {
		handler ContractHandler
	}
)

type callContext struct {
	Context
	receipt  txresult.Receipt
	isQuery  bool
	executor *eeproxy.Executor

	timer  <-chan time.Time
	lock   sync.Mutex
	stack  list.List
	waiter chan interface{}

	log *trace.Logger
}

func NewCallContext(ctx Context, receipt txresult.Receipt, isQuery bool) CallContext {
	logger := trace.LoggerOf(ctx.Logger())
	ti := ctx.TraceInfo()
	if ti != nil {
		var info state.TransactionInfo
		if ctx.GetTransactionInfo(&info) {
			if info.Group == ti.Group && int(info.Index) == ti.Index {
				logger = trace.NewLogger(logger.Logger, ti.Callback)
			}
		}
	}

	return &callContext{
		Context: ctx,
		receipt: receipt,
		isQuery: isQuery,
		// 0-buffered channel is fine, but it sets some number just in case of
		// EE unexpectedly sends messages up to 8.
		waiter: make(chan interface{}, 8),
		log:    logger,
	}
}

func (cc *callContext) QueryMode() bool {
	return cc.isQuery
}

func (cc *callContext) Logger() log.Logger {
	return cc.log
}

type eventLog struct {
	Addr    common.Address
	Indexed [][]byte
	Data    [][]byte
}

type callFrame struct {
	handler   ContractHandler
	byOnCall  bool
	snapshot  state.WorldSnapshot
	eventLogs *list.List
}

func (f *callFrame) AddLog(addr module.Address, indexed, data [][]byte) {
	e := new(eventLog)
	e.Addr.SetBytes(addr.Bytes())
	e.Indexed = indexed
	e.Data = data
	f.eventLogs.PushBack(e)
}

func (f *callFrame) ReturnToFrame(f2 *callFrame) {
	f2.eventLogs.PushBackList(f.eventLogs)
}

func (f *callFrame) ReturnToReceipt(r txresult.Receipt) {
	for i := f.eventLogs.Front(); i != nil; i = i.Next() {
		e := i.Value.(*eventLog)
		r.AddLog(&e.Addr, e.Indexed, e.Data)
	}
}

func (cc *callContext) pushFrame(h ContractHandler, byOnCall bool) *list.Element {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	frame := &callFrame{
		handler:   h,
		byOnCall:  byOnCall,
		eventLogs: list.New(),
	}
	if !cc.isQuery {
		frame.snapshot = cc.GetSnapshot()
	}
	return cc.stack.PushBack(frame)
}

func (cc *callContext) popFrame(e *list.Element, revert bool) (*callFrame, *callFrame) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	current := cc.stack.Back()
	if current == nil {
		if e != nil {
			cc.log.Error("Fail to pop frame")
		}
		return nil, nil
	}
	if e != nil && e != current {
		cc.log.Error("Fail on onPostExecute")
	}
	cc.stack.Remove(current)

	frame := current.Value.(*callFrame)
	last := cc.stack.Back()

	if cc.isQuery {
		if last != nil {
			return frame, last.Value.(*callFrame)
		}
		return frame, nil
	}

	if !revert {
		if last != nil {
			lastFrame := last.Value.(*callFrame)
			frame.ReturnToFrame(lastFrame)
			return frame, lastFrame
		} else {
			frame.ReturnToReceipt(cc.receipt)
			return frame, nil
		}
	} else {
		if err := cc.Reset(frame.snapshot); err != nil {
			cc.log.Errorf("Fail to revert err=%+v", err)
		}
		if last != nil {
			return frame, last.Value.(*callFrame)
		}
		return frame, nil
	}
}

func (cc *callContext) isInAsyncFrame() bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	e := cc.stack.Back()
	if e == nil {
		return false
	}

	frame := e.Value.(*callFrame)
	_, ok := frame.handler.(AsyncContractHandler)
	return ok
}

func (cc *callContext) addLogToFrame(address module.Address, indexed [][]byte, data [][]byte) error {
	if cc.isQuery {
		return nil
	}

	cc.lock.Lock()
	defer cc.lock.Unlock()

	e := cc.stack.Back()
	if e == nil {
		return errors.InvalidStateError.New("Frame is Empty")
	}
	frame := e.Value.(*callFrame)
	frame.AddLog(address, indexed, data)
	return nil
}

func (cc *callContext) validateStatus(status error) error {
	if status != nil && cc.Revision() < module.Revision6 {
		code, _ := scoreresult.StatusOf(status)
		if code > module.StatusLimitRev5 && code <= module.StatusLimit {
			status = scoreresult.WithStatus(status, module.StatusLimitRev5)
		}
	}
	return status
}

func (cc *callContext) Call(handler ContractHandler) (error, *big.Int, *codec.TypedObj, module.Address) {
	handler.ResetLogger(cc.Logger())

	switch handler := handler.(type) {
	case SyncContractHandler:
		e := cc.pushFrame(handler, false)

		status, stepUsed, result, scoreAddr := handler.ExecuteSync(cc)
		status = cc.validateStatus(status)

		cc.popFrame(e, status != nil)
		return status, stepUsed, result, scoreAddr
	case AsyncContractHandler:
		e := cc.pushFrame(handler, false)

		if err := handler.ExecuteAsync(cc); err != nil {
			cc.popFrame(e, true)
			handler.Dispose()
			err = cc.validateStatus(err)
			return err, handler.StepLimit(), nil, nil
		}
		return cc.waitResult(handler.StepLimit())
	default:
		cc.log.Panicln("Unknown handler type:", reflect.TypeOf(handler))
		return nil, nil, nil, nil
	}
}

func (cc *callContext) waitResult(stepLimit *big.Int) (error, *big.Int, *codec.TypedObj, module.Address) {
	// It checks transaction timeout after the first call to EE
	if cc.timer == nil {
		cc.timer = time.After(transactionTimeLimit)
	}

	for {
		select {
		case <-cc.timer:
			cc.cleanUpFrames(scoreresult.ErrTimeout)
			return scoreresult.ErrTimeout, stepLimit, nil, nil
		case msg := <-cc.waiter:
			switch msg := msg.(type) {
			case *callResultMessage:
				status := cc.validateStatus(msg.status)
				if cc.handleResult(status, msg.stepUsed,
					msg.result, msg.addr) {
					continue
				}
				return status, msg.stepUsed, msg.result, msg.addr
			case *callRequestMessage:
				msg.handler.ResetLogger(cc.log)

				switch handler := msg.handler.(type) {
				case SyncContractHandler:
					cc.pushFrame(handler, true)
					status, used, result, addr := handler.ExecuteSync(cc)
					status = cc.validateStatus(status)
					if cc.handleResult(status, used, result, addr) {
						continue
					}
					return status, used, result, addr
				case AsyncContractHandler:
					cc.pushFrame(handler, true)
					if err := handler.ExecuteAsync(cc); err != nil {
						err = cc.validateStatus(err)
						if cc.handleResult(err, handler.StepLimit(), nil, nil) {
							continue
						}
						return err, handler.StepLimit(), nil, nil
					} else {
						continue
					}
				}
			default:
				cc.log.Panicf("Invalid message=%[1]T %+[1]v", msg)
			}
		}
	}
}

func (cc *callContext) cleanUpFrames(err error) {
	cc.log.Warnf("cleanUpFrames() TX=<%#x> err=%+v", cc.GetInfo()[state.InfoTxHash], err)
	l := common.Lock(&cc.lock)
	var frame *callFrame
	achs := make([]AsyncContractHandler, 0, cc.stack.Len())
	for e := cc.stack.Back(); e != nil; e = cc.stack.Back() {
		frame = e.Value.(*callFrame)
		if h, ok := frame.handler.(AsyncContractHandler); ok {
			achs = append(achs, h)
		}
		cc.stack.Remove(e)
	}
	l.Unlock()

	if frame != nil {
		cc.Reset(frame.snapshot)
	}
	for _, h := range achs {
		h.Dispose()
	}

	if cc.executor != nil {
		cc.executor.Kill()
		cc.executor = nil
	}
}

func (cc *callContext) handleResult(status error, stepUsed *big.Int, result *codec.TypedObj, addr module.Address) bool {
	if code := errors.CodeOf(status); code == scoreresult.TimeoutError ||
		code == errors.ExecutionFailError || errors.IsCriticalCode(code) {
		cc.cleanUpFrames(status)
		return false
	}

	currentFrame, lastFrame := cc.popFrame(nil, status != nil)
	if currentFrame == nil {
		cc.log.Error("Fail to pop frame")
	}

	if ach, ok := currentFrame.handler.(AsyncContractHandler); ok {
		ach.Dispose()
	}
	if lastFrame == nil {
		return false
	}

	if currentFrame.byOnCall {
		// SyncContractHandler can't be queued by OnCall(), so don't consider it.
		h := lastFrame.handler.(AsyncContractHandler)
		if err := h.SendResult(status, stepUsed, result); err != nil {
			cc.OnResult(err, h.StepUsed(), nil, nil)
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

func (cc *callContext) OnCall(handler ContractHandler) {
	cc.sendMessage(&callRequestMessage{handler})
}

func (cc *callContext) sendMessage(msg interface{}) error {
	if !cc.isInAsyncFrame() {
		return nil
	}
	cc.waiter <- msg
	return nil
}

func (cc *callContext) OnEvent(addr module.Address, indexed, data [][]byte) {
	cc.log.TSystemf("EVENT score=%s sig=%s indexed=%v data=%v",
		addr, indexed[0],
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

func (cc *callContext) KillExecutor() {
	if cc.executor != nil {
		cc.executor.Kill()
		cc.executor = nil
	}
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
