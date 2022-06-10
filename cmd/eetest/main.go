package main

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/common/log"
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

func (cc *callContext) SetValue(key []byte, value []byte) ([]byte, error) {
	old, err := cc.bk.Get(key)
	if err != nil {
		return nil, err
	}
	err = cc.bk.Set(key, value)
	if err != nil {
		fmt.Printf("CallContext.SetValue([% x],[% x]) -> %+v\n",
			key, value, err)
	} else {
		fmt.Printf("CallContext.SetValue([% x],[% x]) -> SUCCESS\n",
			key, value)
	}
	return old, err
}

func (cc *callContext) ArrayDBContains(prefix, value []byte, limit int64) (bool, int, int, error) {
	adb := containerdb.NewArrayDB(cc, containerdb.NewHashKey(prefix))

	var count int
	var size int
	items := adb.Size()
	for i := 0; i < items; i++ {
		v := adb.Get(i)
		count += 1
		if v != nil {
			bs := v.Bytes()
			size += len(bs)
			if bytes.Equal(value, bs) {
				return true, count, size, nil
			}
		}
	}
	return false, count, size, nil
}

func (cc *callContext) DeleteValue(key []byte) ([]byte, error) {
	old, err := cc.bk.Get(key)
	if err != nil {
		return nil, err
	}
	return old, cc.bk.Delete(key)
}

func (cc *callContext) GetBalance(addr module.Address) *big.Int {
	return big.NewInt(state.GIGA)
}

func (cc *callContext) OnEvent(addr module.Address, indexed, data [][]byte) error {
	fmt.Printf("CallContext.OnEvent(%s,%+v,%+v)\n", addr, indexed, data)
	return nil
}

func (cc *callContext) OnResult(status error, flag int, steps *big.Int, result *codec.TypedObj) {
	fmt.Printf("CallContext.OnResult(%d,%s,[%+v])\n",
		status, steps.String(), common.MustDecodeAny(result))
}

func (cc *callContext) OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) {
	fmt.Printf("CallContext.OnCall(%s,%s,%s,%s,%s,[% x])\n",
		from, to, value, limit, method, params)
}

func (cc *callContext) OnAPI(status error, info *scoreapi.Info) {
	fmt.Printf("CallContext.OnAPI(%d,%+v)\n", status, info)
}

func (cc *callContext) OnSetFeeProportion(portion int) {
	fmt.Printf("CallContext.OnSetPortion(portion=%d)", portion)
}

func (cc *callContext) SetCode(code []byte) error {
	fmt.Println("CallContext.SetCode")
	return nil
}

func (cc *callContext) GetObjGraph(flags bool) (int, []byte, []byte, error) {
	fmt.Printf("CallContext.GetObjGraph(%t)\n", flags)
	return 0, nil, nil, nil
}

func (cc *callContext) SetObjGraph(flags bool, nextHash int, objGraph []byte) error {
	fmt.Printf("CallContext.SetObjGraph(%t,%d,%#x)\n", flags, nextHash, objGraph)
	return nil
}

func (cc *callContext) Logger() log.Logger {
	return log.GlobalLogger()
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
			common.MustNewAddressFromString("cx9999999999999999999999999999999999999999"),
			common.MustNewAddressFromString("hx3333333333333333333333333333333333333333"),
			big.NewInt(10), big.NewInt(state.GIGA), "test", paramAny, nil, 1, nil)
		executor.Release()
		time.Sleep(time.Second)
	}
}

type pythonEngine struct {
}

func (e *pythonEngine) OnConnect(conn ipc.Connection, version uint16) error {
	return errors.New("NotSupported")
}

func (e *pythonEngine) OnClose(conn ipc.Connection) bool {
	return false
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

func (e *pythonEngine) OnEnd(uid string) bool {
	fmt.Printf("PythonEngine.OnEnd(uid=%s)\n", uid)
	return true
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

	logger := log.New()
	mgr, err := eeproxy.NewManager("unix", "/tmp/ee.socket", logger, new(pythonEngine))
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
