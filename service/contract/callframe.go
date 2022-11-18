package contract

import (
	"container/list"
	"math/big"
	"reflect"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
	"github.com/icon-project/goloop/service/txresult"
)

type eventLog struct {
	Addr    common.Address
	Indexed [][]byte
	Data    [][]byte
}

type callFrame struct {
	parent      *callFrame
	fid         int
	eid         int
	code        string
	isQuery     bool
	snapshot    state.WorldSnapshot
	handler     ContractHandler
	log         *trace.Logger
	stepUsed    big.Int
	stepLimit   *big.Int
	eventLogs   list.List
	btpMessages list.List
	code2EID    map[string]int
	logsMap     map[string]CustomLogs
	feePayers   FeePayerInfo
}

func NewFrame(p *callFrame, h ContractHandler, l *big.Int, q bool, logger *trace.Logger) *callFrame {
	frame := &callFrame{
		parent:    p,
		isQuery:   (p != nil && p.isQuery) || q,
		handler:   h,
		stepLimit: l,
		code2EID:  make(map[string]int),
		eid:       unknownEID,
		fid:       baseFID,
		log:       logger,
	}
	frame.eventLogs.Init()
	frame.btpMessages.Init()
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

func (f *callFrame) addLog(addr module.Address, indexed, data [][]byte) {
	if f.isQuery {
		return
	}
	e := new(eventLog)
	e.Addr.Set(addr)
	e.Indexed = indexed
	e.Data = data
	f.eventLogs.PushBack(e)
}

func (f *callFrame) applyFrameLogsOf(frame *callFrame) {
	if f != nil {
		f.eventLogs.PushBackList(&frame.eventLogs)
		for name, data := range frame.logsMap {
			f.logsMap[name].Apply(data)
		}
	}
}

func (f *callFrame) getEventLogs(r txresult.Receipt) {
	for i := f.eventLogs.Front(); i != nil; i = i.Next() {
		e := i.Value.(*eventLog)
		r.AddLog(&e.Addr, e.Indexed, e.Data)
	}
}

func (f *callFrame) addBTPMessage(nid int64, message []byte) {
	if f.isQuery {
		return
	}
	bm := state.NewBTPMsg(nid, message)
	f.btpMessages.PushBack(bm)
}

func (f *callFrame) applyBTPMessagesOf(frame *callFrame) {
	if f != nil {
		f.btpMessages.PushBackList(&frame.btpMessages)
	}
}

func (f *callFrame) getBTPMessages(r txresult.Receipt) {
	r.AddBTPMessages(f.btpMessages)
}

func (f *callFrame) enterQueryMode(cc *callContext) {
	if !f.isQuery {
		cc.Reset(f.snapshot)
		f.snapshot = nil
		f.eventLogs.Init()
		f.btpMessages.Init()
		f.isQuery = true
	}
}

func (f *callFrame) getLastEIDOf(id []byte) int {
	code := string(id)
	for ptr := f; ptr != nil; ptr = ptr.parent {
		if id, ok := ptr.code2EID[code]; ok {
			return id
		}
		if code == ptr.code && ptr.eid != unknownEID {
			return ptr.eid
		}
	}
	return unknownEID
}

func (f *callFrame) setCodeID(id []byte) {
	f.code = string(id)
}

func (f *callFrame) newExecution(eid int) {
	f.eid = eid
	delete(f.code2EID, f.code)
}

func (f *callFrame) mergeLastEIDMap(f2 *callFrame) {
	for code, id := range f2.code2EID {
		f.code2EID[code] = id
	}
	if f2.code != "" && f2.eid != unknownEID {
		f.code2EID[f2.code] = f2.eid
	}
}

func (f *callFrame) getReturnEID() int {
	if eid, ok := f.code2EID[f.code]; ok {
		return eid
	}
	return f.eid
}

func (f *callFrame) getFrameData(name string, ot reflect.Type, top CustomLogs) CustomLogs {
	if f == nil {
		return top
	}
	if f.logsMap == nil {
		f.logsMap = make(map[string]CustomLogs)
	}
	if obj, ok := f.logsMap[name]; ok {
		return obj
	} else {
		parent := f.parent.getFrameData(name, ot, top)
		obj = reflect.New(ot).Interface().(CustomLogs)
		obj.Init(parent)
		f.logsMap[name] = obj
		return obj
	}
}

func (f *callFrame) applyFeePayerInfoOf(frame *callFrame) {
	f.feePayers.Apply(frame.feePayers, frame.getStepUsed())
}
