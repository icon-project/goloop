package contract

import (
	"container/list"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type eventLog struct {
	Addr    common.Address
	Indexed [][]byte
	Data    [][]byte
}

type callFrame struct {
	parent    *callFrame
	isQuery   bool
	snapshot  state.WorldSnapshot
	handler   ContractHandler
	stepUsed  big.Int
	stepLimit *big.Int
	eventLogs list.List
}

func NewFrame(p *callFrame, h ContractHandler, l *big.Int, q bool) *callFrame {
	frame := &callFrame{
		parent:    p,
		isQuery:   (p != nil && p.isQuery) || q,
		handler:   h,
		stepLimit: l,
	}
	frame.eventLogs.Init()
	return frame
}

func (f *callFrame) deductSteps(steps *big.Int) bool {
	f.stepUsed.Add(&f.stepUsed, steps)
	if f.stepLimit == nil {
		return true
	}
	if f.stepUsed.Cmp(f.stepLimit) > 0 {
		f.stepUsed.Set(f.stepLimit)
		return false
	} else {
		return true
	}
}

func (f *callFrame) getStepUsed() *big.Int {
	tmp := new(big.Int)
	return tmp.Set(&f.stepUsed)
}

func (f *callFrame) getStepAvailable() *big.Int {
	if f.stepLimit == nil {
		return nil
	}
	tmp := new(big.Int)
	return tmp.Sub(f.stepLimit, &f.stepUsed)
}

func (f *callFrame) getStepLimit() *big.Int {
	return f.stepLimit
}

func (f *callFrame) addLog(addr module.Address, indexed, data [][]byte) {
	if f.isQuery {
		return
	}
	e := new(eventLog)
	e.Addr.SetBytes(addr.Bytes())
	e.Indexed = indexed
	e.Data = data
	f.eventLogs.PushBack(e)
}

func (f *callFrame) pushBackEventLogsOf(frame *callFrame) {
	if f != nil {
		f.eventLogs.PushBackList(&frame.eventLogs)
	}
}

func (f *callFrame) getEventLogs(r txresult.Receipt) {
	for i := f.eventLogs.Front(); i != nil; i = i.Next() {
		e := i.Value.(*eventLog)
		r.AddLog(&e.Addr, e.Indexed, e.Data)
	}
}

func (f *callFrame) enterQueryMode(cc *callContext) {
	if !f.isQuery {
		cc.Reset(f.snapshot)
		f.snapshot = nil
		f.eventLogs.Init()
		f.isQuery = true
	}
}
