package contract

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/trace"
)

func TestCallContext_Call(t *testing.T) {
	tcc := &testCallContext{CallContext: newCallContext(), trail: ""}
	ah := newHandlerWithNoCall(false, tcc)
	sh := newHandlerWithNoCall(true, tcc)
	tests := []struct {
		name   string
		call   ContractHandler
		result string
	}{
		{
			name:   "Async(OnCall(Async))",
			call:   newHandler(false, false, ah, tcc),
			result: "aaa",
		},
		{
			name:   "Async(OnCall(Sync))",
			call:   newHandler(false, false, sh, tcc),
			result: "asa",
		},
		{
			name:   "Sync(Call(Async))",
			call:   newHandler(true, true, ah, tcc),
			result: "sas",
		},
		{
			name:   "Sync(Call(Sync))",
			call:   newHandler(true, true, sh, tcc),
			result: "sss",
		},
		{
			name:   "Async(Call(Async))",
			call:   newHandler(false, true, ah, tcc),
			result: "aaa",
		},
		// {
		// 	name:   "Sync(OnCall(Sync))",
		// 	call:   newHandler(true, false, sh, tcc),
		// 	result: "error",
		// },
		{
			name:   "Async(OnCall(Sync(Call(Async))))",
			call:   newHandler(false, false, newHandler(true, true, ah, tcc), tcc),
			result: "asasa",
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(tests))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if test.result == "error" {
					err := recover()
					if err == nil {
						t.Errorf("It must be failed")
					}
					wg.Done()
					return
				}
				if test.result != tcc.trail {
					t.Errorf("trail(must:%s,cur:%s)\n", test.result, tcc.trail)
				}
				wg.Done()
			}()

			tcc.Reset()
			tcc.Call(test.call, nil)
		})
	}
	wg.Wait()
}

func newHandlerWithNoCall(sync bool, cc *testCallContext) ContractHandler {
	return newHandler(sync, false, nil, cc)
}

func newHandler(sync bool, callSync bool, targetCall ContractHandler, cc *testCallContext) ContractHandler {
	ch := &commonHandler{subcall: targetCall, callSync: callSync}
	if sync {
		return &syncHandler{commonHandler: ch, cc: cc}
	} else {
		return &asyncHandler{commonHandler: ch, cc: cc}
	}
}

type testCallContext struct {
	CallContext
	trail string
}

func (tcc *testCallContext) Reset() {
	tcc.CallContext = newCallContext()
	tcc.trail = ""
}

type dummyPlatformType struct{}

func (d dummyPlatformType) ToRevision(value int) module.Revision {
	return module.LatestRevision
}

type dummyChain struct {
	module.Chain
}

func (c *dummyChain) TransactionTimeout() time.Duration {
	return 5 * time.Second
}

func newDummyChain() module.Chain {
	return &dummyChain{}
}

func newCallContext() CallContext {
	dbo, _ := db.Open("", string(db.MapDBBackend), "map")
	return NewCallContext(
		NewContext(
			state.NewWorldContext(
				state.NewWorldState(dbo, nil, nil, nil, nil),
				common.NewBlockInfo(0, 0),
				nil,
				dummyPlatformType{},
			),
			nil,
			nil,
			newDummyChain(),
			log.New(),
			nil,
			eeproxy.ForTransaction,
		),
		nil,
		false,
	)
}

type commonHandler struct {
	callSync bool
	subcall  ContractHandler
}

func (h *commonHandler) StepLimit() *big.Int {
	return big.NewInt(0)
}

func (h *commonHandler) StepUsed() *big.Int {
	panic("implement me")
}

func (h *commonHandler) DeductSteps(*big.Int) bool {
	panic("implement me")
}

func (h *commonHandler) ApplySteps(state.WorldContext, state.StepType, int) bool {
	panic("implement me")
}

func (h *commonHandler) ResetSteps(*big.Int) {
	panic("implement me")
}

func (h *commonHandler) Prepare(ctx Context) (state.WorldContext, error) {
	panic("implement me")
}

func (h *commonHandler) SetTraceLogger(logger *trace.Logger) {
	// do nothing
}

func (h *commonHandler) TraceLogger() *trace.Logger {
	return nil
}

func (h *commonHandler) Logger() log.Logger {
	return log.GlobalLogger()
}

type asyncHandler struct {
	*commonHandler
	cc *testCallContext
}

func (h *asyncHandler) ExecuteAsync(cc CallContext) error {
	h.cc.trail += "a"
	if h.subcall != nil {
		if h.callSync {
			cc.Call(h.subcall, nil)
			h.cc.trail += "a"
		} else {
			cc.OnCall(h.subcall, nil)
			return nil
		}
	}
	cc.OnResult(nil, 0, big.NewInt(0), nil, nil)
	return nil
}

func (h *asyncHandler) SendResult(status error, steps *big.Int, result *codec.TypedObj) error {
	if h.subcall != nil && !h.callSync {
		h.cc.trail += "a"
		h.cc.OnResult(status, 0, steps, result, nil)
	}
	return nil
}

func (h *asyncHandler) Dispose() {
}

func (h *asyncHandler) EEType() state.EEType {
	panic("implement me")
}

func (h *asyncHandler) GetValue(key []byte) ([]byte, error) {
	panic("implement me")
}

func (h *asyncHandler) SetValue(key []byte, value []byte) ([]byte, error) {
	panic("implement me")
}

func (h *asyncHandler) DeleteValue(key []byte) ([]byte, error) {
	panic("implement me")
}

func (h *asyncHandler) ArrayDBContains(prefix, value []byte, limit int64) (bool, int, int, error) {
	panic("implement me")
}

func (h *asyncHandler) GetInfo() *codec.TypedObj {
	panic("implement me")
}

func (h *asyncHandler) GetBalance(addr module.Address) *big.Int {
	panic("implement me")
}

func (h *asyncHandler) OnEvent(addr module.Address, indexed, data [][]byte) error {
	panic("implement me")
}

func (h *asyncHandler) OnResult(status error, flag int, steps *big.Int, result *codec.TypedObj) {
	panic("implement me")
}

func (h *asyncHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	panic("implement me")
}

func (h *asyncHandler) OnAPI(status error, info *scoreapi.Info) {
	panic("implement me")
}

func (h *asyncHandler) OnSetFeeProportion(portion int) {
	panic("implement me")
}

func (h *asyncHandler) SetCode(code []byte) error {
	panic("implement me")
}

func (h *asyncHandler) GetObjGraph(bool) (int, []byte, []byte, error) {
	panic("implement me")
}

func (h *asyncHandler) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	panic("implement me")
}

type syncHandler struct {
	*commonHandler
	cc *testCallContext
}

func (h *syncHandler) ExecuteSync(cc CallContext) (error, *codec.TypedObj, module.Address) {
	h.cc.trail += "s"
	if h.subcall != nil {
		if h.callSync {
			cc.Call(h.subcall, nil)
			h.cc.trail += "s"
		} else {
			// Actually it's not supported
			cc.OnCall(h.subcall, nil)
		}
	}
	return nil, nil, nil
}
