package contract

import (
	"container/list"
	"log"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/txresult"
)

type (
	CallContext interface {
		Setup(Context)
		QueryMode() bool
		Call(ContractHandler) (module.Status, *big.Int, *codec.TypedObj, module.Address)
		OnResult(status module.Status, stepUsed *big.Int, result *codec.TypedObj, addr module.Address)
		OnCall(ContractHandler)
		OnEvent(addr module.Address, indexed, data [][]byte)
		GetInfo() map[string]interface{}
		GetBalance(module.Address) *big.Int
		ReserveConnection(eeType string) error
		GetConnection(eeType string) eeproxy.Proxy
		Dispose()
	}
	callResultMessage struct {
		status   module.Status
		stepUsed *big.Int
		result   *codec.TypedObj
		addr     module.Address
	}

	callRequestMessage struct {
		handler ContractHandler
	}
)

type callContext struct {
	receipt txresult.Receipt
	isQuery bool
	conns   map[string]eeproxy.Proxy

	// set at Setup()
	ctx   Context
	info  map[string]interface{}
	timer <-chan time.Time

	lock   sync.Mutex
	stack  list.List
	waiter chan interface{}
}

func NewCallContext(receipt txresult.Receipt, isQuery bool) CallContext {
	return &callContext{
		receipt: receipt,
		isQuery: isQuery,
		// 0-buffered channel is fine, but it sets some number just in case of
		// EE unexpectedly sends messages up to 8.
		waiter: make(chan interface{}, 8),
		conns:  make(map[string]eeproxy.Proxy),
	}
}

func (cc *callContext) Setup(ctx Context) {
	cc.ctx = ctx
}

func (cc *callContext) QueryMode() bool {
	return cc.isQuery
}

func (cc *callContext) Call(handler ContractHandler) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	switch handler := handler.(type) {
	case SyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		status, stepUsed, result, scoreAddr := handler.ExecuteSync(cc.ctx)

		cc.lock.Lock()
		cc.stack.Remove(e)
		cc.lock.Unlock()
		return status, stepUsed, result, scoreAddr
	case AsyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		if err := handler.ExecuteAsync(cc.ctx); err != nil {
			cc.lock.Lock()
			cc.stack.Remove(e)
			cc.lock.Unlock()
			handler.Dispose()
			return module.StatusSystemError, handler.StepLimit(), nil, nil
		}
		return cc.waitResult(handler.StepLimit())
	default:
		log.Panicln("Unknown handler type:", reflect.TypeOf(handler))
		return module.StatusSystemError, handler.StepLimit(), nil, nil
	}
}

func (cc *callContext) waitResult(stepLimit *big.Int) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	// It checks transaction timeout after the first call to EE
	if cc.timer == nil {
		cc.timer = time.After(transactionTimeLimit)
	}

	for {
		select {
		case <-cc.timer:
			cc.handleTimeout()
			return module.StatusTimeout, stepLimit, nil, nil
		case msg := <-cc.waiter:
			switch msg := msg.(type) {
			case *callResultMessage:
				if cc.handleResult(module.Status(msg.status), msg.stepUsed,
					msg.result, msg.addr) {
					continue
				}
				return msg.status, msg.stepUsed, msg.result, nil
			case *callRequestMessage:
				switch handler := msg.handler.(type) {
				case SyncContractHandler:
					cc.lock.Lock()
					cc.stack.PushBack(handler)
					cc.lock.Unlock()
					status, used, result, addr := handler.ExecuteSync(cc.ctx)
					if cc.handleResult(status, used, result, addr) {
						continue
					}
					return status, used, result, addr
				case AsyncContractHandler:
					cc.lock.Lock()
					cc.stack.PushBack(handler)
					cc.lock.Unlock()

					if err := handler.ExecuteAsync(cc.ctx); err != nil {
						if cc.handleResult(module.StatusSystemError,
							handler.StepLimit(), nil, nil) {
							continue
						}
						return module.StatusSystemError, handler.StepLimit(), nil, nil
					} else {
						continue
					}
				}
			default:
				log.Printf("Invalid message=%[1]T %+[1]v", msg)
			}
		}
	}
}

func (cc *callContext) handleTimeout() {
	cc.lock.Lock()
	achs := make([]AsyncContractHandler, 0, cc.stack.Len())
	for e := cc.stack.Back(); e != nil; e = cc.stack.Back() {
		if h, ok := e.Value.(AsyncContractHandler); ok {
			achs = append(achs, h)
		}
		cc.stack.Remove(e)
	}
	cc.lock.Unlock()

	for _, h := range achs {
		h.Dispose()
	}

	// kill EE; It'll restart by itself
	for name, conn := range cc.conns {
		if err := conn.Kill(); err != nil {
			log.Println("FAIL: conn[", name, "].Kill() (", err.Error(), ")")
		}
	}
	cc.conns = nil
}

func (cc *callContext) handleResult(status module.Status,
	stepUsed *big.Int, result *codec.TypedObj, addr module.Address,
) bool {
	if status == module.StatusTimeout {
		cc.lock.Lock()
		e := cc.stack.Back()
		cc.lock.Unlock()
		if e != nil {
			log.Println("Unexpected: StatusTimeout is thrown by another code than callcontext!")
			cc.handleTimeout()
		}
		return false
	}

	cc.lock.Lock()
	// remove current frame
	current := cc.stack.Back()
	if current == nil {
		log.Panicf("Fail to handle result(it's not in frame)")
	}
	cc.stack.Remove(current)

	// back to parent frame
	parent := cc.stack.Back()
	cc.lock.Unlock()

	if ach, ok := current.Value.(AsyncContractHandler); ok {
		ach.Dispose()
	}
	if parent == nil {
		return false
	}

	switch h := parent.Value.(type) {
	case AsyncContractHandler:
		if err := h.SendResult(status, stepUsed, result); err != nil {
			log.Println("FAIL to SendResult(): ", err)
			cc.OnResult(module.StatusSystemError, h.StepLimit(), nil, nil)
		}
		return true
	case SyncContractHandler:
		// do nothing
		return false
	default:
		// It can't be happened
		log.Panicln("Invalid contract handler type:", reflect.TypeOf(parent.Value))
		return true
	}
}

func (cc *callContext) OnResult(status module.Status, stepUsed *big.Int,
	result *codec.TypedObj, addr module.Address,
) {
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

func (cc *callContext) sendMessage(msg interface{}) {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	if e := cc.stack.Back(); e != nil {
		if _, ok := e.Value.(AsyncContractHandler); ok {
			cc.waiter <- msg
		}
	}
}

func (cc *callContext) OnEvent(addr module.Address, indexed, data [][]byte) {
	cc.receipt.AddLog(addr, indexed, data)
}

func (cc *callContext) GetInfo() map[string]interface{} {
	return cc.ctx.GetInfo()
}

func (cc *callContext) GetBalance(addr module.Address) *big.Int {
	if ass := cc.ctx.GetAccountSnapshot(addr.ID()); ass != nil {
		return ass.GetBalance()
	} else {
		return big.NewInt(0)
	}
}

func (cc *callContext) ReserveConnection(eeType string) error {
	conn := cc.ctx.EEManager().Get(eeType)
	if conn == nil {
		log.Panicln("Fails to get connection of eetype(" + eeType + ")")
	}
	cc.conns[eeType] = conn
	return nil
}

func (cc *callContext) GetConnection(eeType string) eeproxy.Proxy {
	conn := cc.conns[eeType]
	// Conceptually, it should return nil when it's not reserved in advance.
	// But currently it doesn't assume it should be reserved, so retry to reserve here.
	if conn == nil {
		cc.ReserveConnection(eeType)
	}
	return cc.conns[eeType]
}

func (cc *callContext) Dispose() {
	for _, v := range cc.conns {
		v.Release()
	}
}
