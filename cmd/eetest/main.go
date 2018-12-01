package main

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/eeproxy"
	"log"
	"math/big"
	"time"
)

var mgr eeproxy.Manager

type callContext struct {
	bk    db.Bucket
	proxy eeproxy.Proxy
}

func (cc *callContext) GetInfo() map[string]interface{} {
	m := make(map[string]interface{})
	m["T.Index"] = int(1)
	m["B.Height"] = int(1)
	m["B.Timestamp"] = int(1)
	fmt.Printf("CallContext.GetInfo() -> %+v\n", m)
	return m
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

func (cc *callContext) OnEvent(idxcnt uint16, msgs [][]byte) {
	fmt.Printf("CallContext.OnEvent()\n")
}

func (cc *callContext) OnResult(status uint16, steps *big.Int, result []byte) {
	fmt.Printf("CallContext.OnResult(%d,%s,[% x])\n",
		status, steps.String(), result)
}

func (cc *callContext) OnCall(from, to module.Address, value, limit *big.Int, params []byte) {
	fmt.Printf("CallContext.OnCall(%s,%s,%s,%s,[% x])\n",
		from, to, value, limit, params)
}

func makeTransactions(cc *callContext, mgr eeproxy.Manager) {
	for {
		proxy := mgr.Get("python")
		cc.proxy = proxy
		proxy.Invoke(cc, "score/",
			common.NewAddressFromString("cx9999999999999999999999999999999999999999"),
			common.NewAddressFromString("hx3333333333333333333333333333333333333333"),
			big.NewInt(10), big.NewInt(service.GIGA),
			"test",
			[]byte("{ \"param1\": \"0x1\"}"))
		proxy.Release()
		time.Sleep(time.Second)
	}
}

func main() {
	var err error
	mgr, err = eeproxy.New("unix", "/tmp/ee.socket")
	if err != nil {
		log.Panicf("Fail to make EEProxy err=%+v", err)
	}

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
