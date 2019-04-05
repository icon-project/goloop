package main

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/state"
)

var mgr eeproxy.Manager

const (
	ApplicationType = "python"
)

type callContext struct {
	bk    db.Bucket
	proxy eeproxy.Proxy
}

func (cc *callContext) GetInfo() *codec.TypedObj {
	m := make(map[string]interface{})
	m["T.Index"] = int(1)
	m["B.Height"] = int(1)
	m["B.Timestamp"] = int(1)
	fmt.Printf("CallContext.GetInfo() -> %+v\n", m)
	return common.MustEncodeAny(m)
}

func (cc *callContext) GetValue(key []byte) ([]byte, error) {
	ret, err := cc.bk.Get(key)
	if err != nil {
		fmt.Printf("CallContext.GetValue([% x]) --> %+v\n", key, err)
		return nil, err
	}
	fmt.Printf("CallContext.GetValue([% x]) --> [% x]\n", key, ret)
	return ret, err
}

func (cc *callContext) SetValue(key, value []byte) error {
	err := cc.bk.Set(key, value)
	if err != nil {
		fmt.Printf("CallContext.SetValue([% x],[% x]) -> %+v\n",
			key, value, err)
	} else {
		fmt.Printf("CallContext.SetValue([% x],[% x]) -> SUCCESS\n",
			key, value)
	}
	return err
}

func (cc *callContext) DeleteValue(key []byte) error {
	return cc.bk.Delete(key)
}

func (cc *callContext) GetBalance(addr module.Address) *big.Int {
	return big.NewInt(state.GIGA)
}

func (cc *callContext) OnEvent(score module.Address, indexed, data [][]byte) {
	fmt.Printf("CallContext.OnEvent(%s,%+v,%+v)\n", score, indexed, data)
}

func (cc *callContext) OnResult(status uint16, steps *big.Int, result *codec.TypedObj) {
	fmt.Printf("CallContext.OnResult(%d,%s,[%+v])\n",
		status, steps.String(), common.MustDecodeAny(result))
}

func (cc *callContext) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	fmt.Printf("CallContext.OnCall(%s,%s,%s,%s,%s,[% x])\n",
		from, to, value, limit, method, params)
}

func (cc *callContext) OnAPI(status uint16, obj *scoreapi.Info) {
	fmt.Printf("CallContext.OnAPI(%d,%+v)\n", status, obj)
}

func makeTransactions(cc *callContext, mgr eeproxy.Manager) {
	paramObj := []interface{}{"Test"}
	paramAny := common.MustEncodeAny(paramObj)
	for {
		executor := mgr.GetExecutor(eeproxy.ForTransaction)
		proxy := executor.Get(ApplicationType)
		cc.proxy = proxy
		proxy.GetAPI(cc, "score/")
		proxy.Invoke(cc, "score/", false,
			common.NewAddressFromString("cx9999999999999999999999999999999999999999"),
			common.NewAddressFromString("hx3333333333333333333333333333333333333333"),
			big.NewInt(10), big.NewInt(state.GIGA), "test", paramAny)
		executor.Release()
		time.Sleep(time.Second)
	}
}

type pythonEngine struct {
}

func (e *pythonEngine) Type() string {
	return ApplicationType
}

func (e *pythonEngine) Init(net, addr string) error {
	// do nothing
	return nil
}

func (e *pythonEngine) SetInstances(n int) error {
	fmt.Printf("PythonEngine.SetInstances(n=%d)\n", n)
	return nil
}

func (e *pythonEngine) OnAttach(uid string) bool {
	fmt.Printf("PythonEngine.OnAttach(uid=%s)\n", uid)
	return true
}

func (e *pythonEngine) Kill(uid string) (bool, error) {
	// do nothing and return success.
	return true, nil
}

func main() {
	var err error

	mgr, err := eeproxy.NewManager("unix", "/tmp/ee.socket", new(pythonEngine))
	if err != nil {
		log.Panicf("Fail to make EEProxy err=%+v", err)
	}
	mgr.SetInstances(1, 1, 1)

	dbase := db.NewMapDB()
	bk, err := dbase.GetBucket("")
	if err != nil {
		log.Panicf("Fail to make bucket from dbase err=%+v", err)
	}

	cc := &callContext{
		bk: bk,
	}
	go makeTransactions(cc, mgr)
	mgr.Loop()
}
