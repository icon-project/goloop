package service

import (
	"log"
	"math/big"
	"reflect"
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

type (
	TransactionHandler interface {
		Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
		Execute(wc WorldContext) (Receipt, error)
		Dispose()
	}

	TransactionContext interface {
		GetContract(common.Address) []byte
		ReserveConnection(eeType string) error
		GetConnection(eeType string) eeproxy.Proxy
		GetValue(key []byte) ([]byte, error)
		SetValue(key, value []byte) error
		GetInfo() map[string]interface{}
		AddEvent(idxcnt uint16, msgs [][]byte)
	}

	CallResultMessage struct {
		status   uint16
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

type contractStack struct {
	handlers []ContractHandler
}

func newContractStack() *contractStack {
	return &contractStack{make([]ContractHandler, 0)}
}
func (s *contractStack) push(v ContractHandler) {
	s.handlers = append(s.handlers, v)
}

func (s *contractStack) pop() ContractHandler {
	l := len(s.handlers)
	if l > 0 {
		e := s.handlers[l-1]
		s.handlers = s.handlers[:l-1]
		return e
	}
	return nil
}

func (s *contractStack) peek() ContractHandler {
	l := len(s.handlers)
	if l > 0 {
		return s.handlers[l-1]
	}
	return nil
}

func (s *contractStack) popIfValid(h ContractHandler) bool {
	l := len(s.handlers)
	if l > 0 {
		e := s.handlers[l-1]
		if e == h {
			s.handlers = s.handlers[:l-1]
			return true
		}
	}
	return false
}

type transactionContext struct {
	from      module.Address
	to        module.Address
	value     *big.Int
	stepLimit *big.Int
	dataType  string
	// TODO data type 검토
	data interface{}

	conns   map[string]eeproxy.Proxy
	handler ContractHandler
	receipt Receipt
}

func NewTransactionHandler(from, to module.Address, value, stepLimit *big.Int, dataType string, data interface{}) TransactionHandler {
	tc := &transactionContext{
		from:      from,
		to:        to,
		value:     value,
		stepLimit: stepLimit,
		dataType:  dataType,
		data:      data,
		conns:     make(map[string]eeproxy.Proxy),
		receipt:   NewReceipt(to),
	}
	// TODO check type of data
	tc.handler = contractMngr.GetHandler(tc, from, to, value, stepLimit, dataType, data)
	if tc.handler == nil {
		log.Println("can't find handler:", from, to, value, stepLimit, dataType, data)
		return nil
	}
	return tc
}

func (tc *transactionContext) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	return tc.handler.Prepare(wvs)
}

func (tc *transactionContext) Execute(wc WorldContext) (Receipt, error) {
	// TODO handle transfer (TransferCall과 구현을 통합할 방법 고민)

	switch tc.handler.(type) {
	case SyncContractHandler:
		h := tc.handler.(SyncContractHandler)
		code, stepUsed := h.ExecuteSync(wc)
		return tc.makeReceipt(code, stepUsed), nil
	case AsyncContractHandler:
		callStack := newContractStack()
		callStack.push(tc.handler)
		curCall := tc.handler

		h := tc.handler.(AsyncContractHandler)
		exec := h.ExecuteAsync(wc)
		timer := time.After(transactionTimeLimit)
		for curCall != nil {
			select {
			case <-timer:
				for curCall = callStack.pop(); curCall != nil; curCall = callStack.pop() {
					curCall.Cancel()
				}
			case result := <-exec:
				switch result.(type) {
				case *CallResultMessage:
					msg := result.(*CallResultMessage)
					callStack.pop()
					curCall = callStack.peek()
					if curCall != nil {
						tc.conns[h.EEType()].SendResult(h, msg.status, msg.stepUsed, msg.result)
					}
				case *CallRequestMessage:
					msg := result.(*CallRequestMessage)
					h = contractMngr.GetHandler(tc, msg.from, msg.to, msg.value, msg.stepLimit, dataTypeCall, msg.params).(AsyncContractHandler)
					if h != nil {
						callStack.push(h)
						curCall = h
						exec = h.ExecuteAsync(wc)
					} else {
						log.Println("can't find handler:", msg.from, msg.to, msg.value, msg.stepLimit, dataTypeCall, msg.params)
					}
				default:
					log.Println("unknown message type:", reflect.TypeOf(result))
				}
			}
		}
		// TODO create receipt for async
		return nil, nil
	default:
		log.Println("unknown contract handler type:", reflect.TypeOf(tc.handler))
		return nil, nil
	}
}

func (tc *transactionContext) makeReceipt(code int, stepUsed *big.Int) Receipt {
	// TODO make a receipt
	return nil
}

func (tc *transactionContext) Dispose() {
	// TODO clean up all resources just in case of not calling Execute()
	panic("implement me")
}

func (tc *transactionContext) GetContract(addr common.Address) []byte {
	// TODO contract addr로 contract code 받아오기
	panic("implement me")
}

func (tc *transactionContext) ReserveConnection(eeType string) error {
	// TODO
	//tc.conns[eeType] = eeMngr.Get(eeType)
	return nil
}

func (tc *transactionContext) GetConnection(eeType string) eeproxy.Proxy {
	conn := tc.conns[eeType]
	// Conceptually, it should return nil when it's not reserved in advance.
	// But currently it doesn't assume it should be reserved, so retry to reserve here.
	if conn == nil {
		tc.ReserveConnection(eeType)
	}
	return tc.conns[eeType]
}

func (tc *transactionContext) GetValue(key []byte) ([]byte, error) {
	// TODO
	panic("implement me")
}

func (tc *transactionContext) SetValue(key, value []byte) error {
	// TODO
	panic("implement me")
}

func (tc *transactionContext) GetInfo() map[string]interface{} {
	// TODO
	panic("implement me")
}

func (tc *transactionContext) AddEvent(idxcnt uint16, msgs [][]byte) {
	// TODO parameter 정리 필요
}
