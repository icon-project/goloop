package v3

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/trace"
)

type traceCallback struct {
	lock    sync.Mutex
	logs    []interface{}
	last    error
	ts      time.Time
	channel chan interface{}
	bt      *trace.BalanceTracer
}

type traceLog struct {
	Level module.TraceLevel `json:"level"`
	Msg   string            `json:"msg"`
	Ts    int64             `json:"ts"`
}

func (t *traceCallback) OnLog(level module.TraceLevel, msg string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	ts := time.Now()
	if len(t.logs) == 0 {
		t.ts = ts
	}
	dur := ts.Sub(t.ts) / time.Microsecond
	t.logs = append(t.logs, traceLog{level, msg, int64(dur)})
}

func (t *traceCallback) OnEnd(e error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.last = e

	t.channel <- e
	close(t.channel)
}

func (t *traceCallback) invokeTraceToJSON() interface{} {
	t.lock.Lock()
	defer t.lock.Unlock()

	result := map[string]interface{}{
		"logs": t.logs,
	}
	if t.last == nil {
		result["status"] = "0x1"
	} else {
		result["status"] = "0x0"
		status, _ := scoreresult.StatusOf(t.last)
		result["failure"] = map[string]interface{}{
			"code":    status,
			"message": t.last.Error(),
		}
	}
	return result
}

func (t *traceCallback) balanceChangeToJSON(blk module.Block) interface{} {
	t.lock.Lock()
	defer t.lock.Unlock()

	result := map[string]interface{}{
		"blockHash":     "0x" + hex.EncodeToString(blk.ID()),
		"prevBlockHash": "0x" + hex.EncodeToString(blk.PrevID()),
		"blockHeight":   fmt.Sprintf("%#x", blk.Height()),
		"timestamp":     fmt.Sprintf("%#x", blk.Timestamp()),
	}

	balanceChanges := t.bt.ToJSON()
	if balanceChanges != nil {
		result["balanceChanges"] = balanceChanges
	}
	return result
}

func (t *traceCallback) OnTransactionStart(txIndex int, txHash []byte, isBlockTx bool) error {
	if t.bt != nil {
		t.lock.Lock()
		defer t.lock.Unlock()
		return t.bt.OnTransactionStart(txIndex, txHash, isBlockTx)
	}
	return nil
}

func (t *traceCallback) OnTransactionReset() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.logs = nil
	if t.bt != nil {
		return t.bt.OnTransactionReset()
	}
	return nil
}

func (t *traceCallback) OnTransactionEnd(txIndex int, txHash []byte) error {
	if t.bt != nil {
		t.lock.Lock()
		defer t.lock.Unlock()
		return t.bt.OnTransactionEnd(txIndex, txHash)
	}
	return nil
}

func (t *traceCallback) OnFrameEnter() error {
	if t.bt != nil {
		t.lock.Lock()
		defer t.lock.Unlock()
		return t.bt.OnFrameEnter()
	}
	return nil
}

func (t *traceCallback) OnFrameExit(success bool) error {
	if t.bt != nil {
		t.lock.Lock()
		defer t.lock.Unlock()
		return t.bt.OnFrameExit(success)
	}
	return nil
}

func (t *traceCallback) OnBalanceChange(opType module.OpType, from, to module.Address, amount *big.Int) error {
	if t.bt != nil {
		t.lock.Lock()
		defer t.lock.Unlock()
		return t.bt.OnBalanceChange(opType, from, to, amount)
	}
	return nil
}
