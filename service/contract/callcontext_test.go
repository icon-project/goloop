package contract

import (
	"github.com/icon-project/goloop/common/log"
	"math/big"
	"sync"
	"testing"

	"github.com/icon-project/goloop/common/db"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
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
		{
			name:   "Sync(OnCall(Sync))",
			call:   newHandler(true, false, sh, tcc),
			result: "error",
		},
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
				err := recover()
				if err != nil {
					if test.result != "error" {
						t.Errorf("Error occurred")
					}
				} else {
					if test.result != "error" && test.result != tcc.trail {
						t.Errorf("trail(must:%s,cur:%s)\n", test.result, tcc.trail)
					} else if test.result == "error" {
						t.Errorf("It must be failed")
					}
				}
				wg.Done()
			}()

			tcc.Reset()
			tcc.Call(test.call)
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

func newCallContext() CallContext {
	dbo, _ := db.Open("", string(db.MapDBBackend), "map")
	return NewCallContext(
		NewContext(
			state.NewWorldContext(
				state.NewWorldState(dbo, nil, nil),
				&blockInfo{},
			),
			nil,
			nil,
			nil,
			log.New(),
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

type asyncHandler struct {
	*commonHandler
	cc *testCallContext
}

func (h *asyncHandler) ExecuteAsync(cc CallContext) error {
	h.cc.trail += "a"
	if h.subcall != nil {
		if h.callSync {
			cc.Call(h.subcall)
			h.cc.trail += "a"
		} else {
			cc.OnCall(h.subcall)
			return nil
		}
	}
	cc.OnResult(module.StatusSuccess, big.NewInt(0), nil, nil)
	return nil
}

func (h *asyncHandler) SendResult(status module.Status, steps *big.Int, result *codec.TypedObj) error {
	if h.subcall != nil && !h.callSync {
		h.cc.trail += "a"
		h.cc.OnResult(status, steps, result, nil)
	} else {
	}
	return nil
}

func (h *asyncHandler) Dispose() {
}

func (h *asyncHandler) EEType() string {
	panic("implement me")
}

func (h *asyncHandler) GetValue(key []byte) ([]byte, error) {
	panic("implement me")
}

func (h *asyncHandler) SetValue(key, value []byte) error {
	panic("implement me")
}

func (h *asyncHandler) DeleteValue(key []byte) error {
	panic("implement me")
}

func (h *asyncHandler) GetInfo() *codec.TypedObj {
	panic("implement me")
}

func (h *asyncHandler) GetBalance(addr module.Address) *big.Int {
	panic("implement me")
}

func (h *asyncHandler) OnEvent(addr module.Address, indexed, data [][]byte) {
	panic("implement me")
}

func (h *asyncHandler) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	panic("implement me")
}

func (h *asyncHandler) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	panic("implement me")
}

func (h *asyncHandler) OnAPI(status uint16, obj *scoreapi.Info) {
	panic("implement me")
}

type syncHandler struct {
	*commonHandler
	cc *testCallContext
}

func (h *syncHandler) ExecuteSync(cc CallContext) (module.Status, *big.Int, *codec.TypedObj, module.Address) {
	h.cc.trail += "s"
	if h.subcall != nil {
		if h.callSync {
			cc.Call(h.subcall)
			h.cc.trail += "s"
		} else {
			// Actually it's not supported
			cc.OnCall(h.subcall)
		}
	}
	return module.StatusSuccess, big.NewInt(0), nil, nil
}

type blockInfo struct {
}

func (bi *blockInfo) Height() int64 {
	return 0
}

func (bi *blockInfo) Timestamp() int64 {
	return 0
}
