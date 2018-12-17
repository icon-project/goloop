package service

import (
	"container/list"
	"log"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
)

/*
var eeMngr eeproxy.Manager

func init() {
	mgr, err := eeproxy.New("unix", "/tmp/ee.socket/")
	if err != nil {
		log.Panicf("FAIL to create EE Server err=%+v", err)
	}

	eeMngr = eeproxy.Manager(mgr)
}
*/

const (
	dataTypeNone    = ""
	dataTypeMessage = "message"
	dataTypeCall    = "call"
	dataTypeDeploy  = "deploy"
)

type (
	TransactionHandler interface {
		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
		Execute(wc WorldContext) (Receipt, error)
		Dispose()
	}

	CallContext interface {
		Setup(WorldContext)
		Call(ContractHandler) (module.Status, *big.Int, []byte, module.Address)
		OnResult(status module.Status, stepUsed *big.Int, result []byte, addr module.Address)
		OnCall(ContractHandler)
		OnEvent(indexed, data [][]byte)
		GetInfo() map[string]interface{}
		GetBalance(module.Address) *big.Int
		ReserveConnection(eeType string) error
		GetConnection(eeType string) eeproxy.Proxy
		Dispose()
	}
	callResultMessage struct {
		status   module.Status
		stepUsed *big.Int
		result   []byte
		addr     module.Address
	}

	callRequestMessage struct {
		handler ContractHandler
	}
)

type transactionHandler struct {
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	dataType  string
	data      []byte

	handler ContractHandler
	cc      CallContext
	receipt Receipt
}

func NewTransactionHandler(from, to module.Address, value, stepLimit *big.Int,
	dataType string, data []byte,
) TransactionHandler {
	tc := &transactionHandler{
		from:      from,
		to:        to,
		value:     value,
		stepLimit: stepLimit,
		dataType:  dataType,
		data:      data,
	}
	ctype := ctypeNone // invalid contract type
	switch dataType {
	case dataTypeNone:
		ctype = ctypeTransfer
	case dataTypeMessage:
		ctype = ctypeTransferAndMessage
	case dataTypeDeploy:
		ctype = ctypeTransferAndDeploy
	case dataTypeCall:
		ctype = ctypeTransferAndCall
	}

	tc.receipt = NewReceipt(to)
	tc.cc = newCallContext(tc.receipt)
	tc.handler = contractMngr.GetHandler(tc.cc, from, to, value, stepLimit,
		ctype, data)
	if tc.handler == nil {
		log.Println("can't find handler:", from, to, dataType, ctype)
		return nil
	}
	return tc
}

func (th *transactionHandler) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	return th.handler.Prepare(wvs)
}

func (th *transactionHandler) Execute(wc WorldContext) (Receipt, error) {
	th.cc.Setup(wc)
	status, stepUsed, _, addr := th.cc.Call(th.handler)
	if status != module.StatusSuccess {
		stepUsed = th.stepLimit
	}
	th.receipt.SetResult(status, stepUsed, wc.StepPrice(), addr)
	return th.receipt, nil
}

func (th *transactionHandler) Dispose() {
	th.cc.Dispose()
}

type callContext struct {
	receipt Receipt
	conns   map[string]eeproxy.Proxy

	// set at Setup()
	wc    WorldContext
	info  map[string]interface{}
	timer <-chan time.Time

	lock   sync.Mutex
	stack  list.List
	waiter chan interface{}
}

func newCallContext(receipt Receipt) CallContext {
	return &callContext{
		receipt: receipt,
		waiter:  make(chan interface{}),
		conns:   make(map[string]eeproxy.Proxy),
	}
}

func (cc *callContext) Setup(wc WorldContext) {
	cc.wc = wc
	// TODO set info map

	cc.timer = time.After(transactionTimeLimit)
}

func (cc *callContext) Call(handler ContractHandler) (module.Status, *big.Int,
	[]byte, module.Address,
) {
	switch handler := handler.(type) {
	case SyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		status, stepUsed, result, scoreAddr := handler.ExecuteSync(cc.wc)

		cc.lock.Lock()
		cc.stack.Remove(e)
		cc.lock.Unlock()
		return status, stepUsed, result, scoreAddr
	case AsyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		if err := handler.ExecuteAsync(cc.wc); err != nil {
			cc.lock.Lock()
			cc.stack.Remove(e)
			cc.lock.Unlock()
			return module.StatusSystemError, handler.StepLimit(), nil, nil
		}
		return cc.waitResult(handler.StepLimit())
	default:
		log.Panicf("Unknown handler type")
		return module.StatusSystemError, handler.StepLimit(), nil, nil
	}
}

func (cc *callContext) waitResult(stepLimit *big.Int) (
	module.Status, *big.Int, []byte, module.Address,
) {
	for {
		select {
		case <-cc.timer:
			cc.lock.Lock()
			for e := cc.stack.Back(); e != nil; e = cc.stack.Back() {
				if h, ok := e.Value.(AsyncContractHandler); ok {
					h.Cancel()
				}
				cc.stack.Remove(e)
			}
			cc.lock.Unlock()
			close(cc.waiter)
			return module.StatusTimeout, stepLimit, nil, nil
		case msg, ok := <-cc.waiter:
			if ok {
				switch msg := msg.(type) {
				case *callResultMessage:
					if cc.handleResult(module.Status(msg.status), msg.stepUsed, msg.result, msg.addr) {
						continue
					}
					return module.Status(msg.status), msg.stepUsed, msg.result, nil
				case *callRequestMessage:
					switch handler := msg.handler.(type) {
					case SyncContractHandler:
						cc.lock.Lock()
						cc.stack.PushBack(handler)
						cc.lock.Unlock()
						status, used, result, addr := handler.ExecuteSync(cc.wc)
						if cc.handleResult(status, used, result, addr) {
							continue
						}
						return status, used, result, addr
					case AsyncContractHandler:
						cc.lock.Lock()
						cc.stack.PushBack(handler)
						cc.lock.Unlock()

						if err := handler.ExecuteAsync(cc.wc); err != nil {
							if cc.handleResult(module.StatusSystemError, handler.StepLimit(), nil, nil) {
								continue
							}
							return module.StatusSystemError, handler.StepLimit(), nil, nil
						}
					}
				default:
					log.Printf("Invalid message=%[1]T %[1]+v", msg)
				}
			} else {
				return module.StatusTimeout, stepLimit, nil, nil
			}
		}
	}
}

func (cc *callContext) handleResult(status module.Status,
	stepUsed *big.Int, result []byte, addr module.Address,
) bool {
	cc.lock.Lock()
	defer cc.lock.Unlock()

	// remove current frame
	e := cc.stack.Back()
	if e == nil {
		log.Panicf("Fail to handle result(it's not in frame)")
	}
	cc.stack.Remove(e)

	// back to parent frame
	e = cc.stack.Back()
	if e == nil {
		return false
	}
	switch h := e.Value.(type) {
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
		log.Panicln("Invalid contract handler type:", reflect.TypeOf(e.Value))
		return true
	}
}

func (cc *callContext) cancelCall() ContractHandler {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	e := cc.stack.Back()
	if h, ok := e.Value.(AsyncContractHandler); ok {
		h.Cancel()
	} else {
		log.Panicln("Other types than AsyncContractHandler:",
			reflect.TypeOf(e.Value))
	}
	cc.stack.Remove(e)

	return e.Value.(ContractHandler)
}

func (cc *callContext) OnResult(status module.Status, stepUsed *big.Int,
	result []byte, addr module.Address,
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
		if _, ok := e.Value.(*AsyncContractHandler); ok {
			cc.waiter <- msg
		}
	}
}

func (cc *callContext) OnEvent(indexed, data [][]byte) {
	cc.receipt.AddLog(nil, indexed, data)
}

func (cc *callContext) GetInfo() map[string]interface{} {
	return cc.info
}

func (cc *callContext) GetBalance(addr module.Address) *big.Int {
	if ass := cc.wc.GetAccountSnapshot(addr.ID()); ass != nil {
		return ass.GetBalance()
	} else {
		return big.NewInt(0)
	}
}

func (cc *callContext) ReserveConnection(eeType string) error {
	// TODO
	//tc.conns[eeType] = eeMngr.Get(eeType)
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
