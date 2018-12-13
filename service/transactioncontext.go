package service

import (
	"container/list"
	"log"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
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

	ContractCallContext interface {
		GetContract(common.Address) []byte
		ReserveConnection(eeType string) error
		GetConnection(eeType string) eeproxy.Proxy
		AddEvent(idxcnt uint16, msgs [][]byte)
	}

	CallResultMessage struct {
		status   module.Status
		stepUsed *big.Int
		result   []byte
	}

	CallRequestMessage struct {
		from      module.Address
		to        module.Address
		value     *big.Int
		stepLimit *big.Int
		params    []byte
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
		receipt:   NewReceipt(to),
	}
	// TODO check type of data
	ctype := -1 // invalid contract type
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
	cc := newContractCallContext()
	tc.handler = contractMngr.GetHandler(cc, from, to, value, stepLimit, ctype, data)
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
	cc := newContractCallContext()
	return cc.Call(th.handler, wc)
}

func (th *transactionHandler) Dispose() {
	// TODO clean up all resources just in case of not calling Execute()
	panic("implement me")
}

type contractCallContext struct {
	// set at Call()
	wc             WorldContext
	initialHandler ContractHandler

	lock  sync.Mutex
	timer <-chan time.Time

	//stepPrice *big.Int
	//info map[string]interface{}

	stack list.List
	conns map[string]eeproxy.Proxy
}

func newContractCallContext() *contractCallContext {
	return &contractCallContext{
		conns: make(map[string]eeproxy.Proxy),
	}
}

func (cc *contractCallContext) Call(handler ContractHandler, wc WorldContext,
) (Receipt, error) {
	cc.wc = wc
	cc.initialHandler = handler

	// TODO create receipt
	//r := NewReceipt(handler.To())

	cc.timer = time.After(transactionTimeLimit)
	cc.handleCall(handler)

	return nil, nil
}

func (cc *contractCallContext) handleCall(handler ContractHandler,
) (module.Status, *big.Int, module.Address) {
	switch handler := handler.(type) {
	case SyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		status, stepUsed, scoreAddr := handler.ExecuteSync(cc.wc)

		cc.lock.Lock()
		cc.stack.Remove(e)
		cc.lock.Unlock()
		return status, stepUsed, scoreAddr
	case AsyncContractHandler:
		cc.lock.Lock()
		e := cc.stack.PushBack(handler)
		cc.lock.Unlock()

		exec, err := handler.ExecuteAsync(cc.wc)
		if err != nil {
			cc.lock.Lock()
			cc.stack.Remove(e)
			cc.lock.Unlock()
			return module.StatusSystemError, handler.StepLimit(), nil
		}
		status, stepUsed, _, scoreAddr := cc.waitResult(exec)
		return status, stepUsed, scoreAddr
	default:
		log.Panicf("Unknown handler type")
		return module.StatusSystemError, handler.StepLimit(), nil
	}
}

func (cc *contractCallContext) waitResult(ch <-chan interface{}) (
	module.Status, *big.Int, []byte, module.Address) {
	for {
		select {
		case <-cc.timer:
			cc.lock.Lock()
			for e := cc.stack.Back(); e != nil; e = cc.stack.Back() {
				if _, ok := e.Value.(AsyncContractHandler); ok {
					// actually all value is an instance of AsyncContractHandler
					e.Value.(AsyncContractHandler).Cancel()
				}
				cc.stack.Remove(e)
			}
			cc.lock.Unlock()
			return module.StatusTimeout, cc.initialHandler.StepLimit(), nil, nil
		case msg := <-ch:
			switch msg := msg.(type) {
			case *CallResultMessage:
				cc.handleResult(msg.status, msg.stepUsed, msg.result, nil)
				return msg.status, msg.stepUsed, msg.result, nil
			case *CallRequestMessage:
				h := contractMngr.GetHandler(
					cc, msg.from, msg.to, msg.value, msg.stepLimit,
					ctypeCall, msg.params,
				).(AsyncContractHandler)

				status, stepLimit, addr := cc.handleCall(h)
				return status, stepLimit, nil, addr
			default:
				log.Printf("Invalid message=%[1]T %[1]+v", msg)
			}
		}
	}
}

func (cc *contractCallContext) handleResult(status module.Status,
	stepUsed *big.Int, result []byte, addr module.Address,
) {
	cc.lock.Lock()

	// remove current frame
	e := cc.stack.Back()
	if e == nil {
		log.Panicf("Fail to handle result(it's not in frame)")
	}
	cc.stack.Remove(e)

	// back to parent frame
	e = cc.stack.Back()
	if e == nil {
		return
	}
	cc.lock.Unlock()
	switch h := e.Value.(type) {
	case AsyncContractHandler:
		if conn := cc.conns[h.EEType()]; conn != nil {
			conn.SendResult(h, uint16(status), stepUsed, result)
		} else {
			log.Println("Unexpected contract handling: no IPC connection")
			// go to parent frame with same result
			cc.handleResult(module.StatusSystemError, stepUsed, nil, addr)
		}
	case SyncContractHandler:

	default:
		// It can't be happened
		log.Println("Invalid contract handler type:", reflect.TypeOf(e.Value))
	}
}

func (cc *contractCallContext) GetContract(addr common.Address) []byte {
	// TODO contract addr로 contract code 받아오기
	panic("implement me")
}

func (cc *contractCallContext) ReserveConnection(eeType string) error {
	// TODO
	//tc.conns[eeType] = eeMngr.Get(eeType)
	return nil
}

func (cc *contractCallContext) GetConnection(eeType string) eeproxy.Proxy {
	conn := cc.conns[eeType]
	// Conceptually, it should return nil when it's not reserved in advance.
	// But currently it doesn't assume it should be reserved, so retry to reserve here.
	if conn == nil {
		cc.ReserveConnection(eeType)
	}
	return cc.conns[eeType]
}

func (cc *contractCallContext) AddEvent(idxcnt uint16, msgs [][]byte) {
	// TODO parameter 정리 필요
}
