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
	mode    module.TraceMode
	lock    sync.Mutex
	logs    []interface{}
	last    error
	ts      time.Time
	channel chan interface{}
	bt      module.BalanceTracer
	rl      module.ReceiptList
}

type traceLog struct {
	Level module.TraceLevel `json:"level"`
	Msg   string            `json:"msg"`
	Ts    int64             `json:"ts"`
}

func (t *traceCallback) TraceMode() module.TraceMode {
	return t.mode
}

func (t *traceCallback) OnLog(level module.TraceLevel, msg string) {
	if t.mode != module.TraceModeInvoke {
		return
	}

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

func (t *traceCallback) result(blk module.Block) interface{} {
	t.lock.Lock()
	defer t.lock.Unlock()

	result := map[string]interface{}{}

	if t.mode == module.TraceModeInvoke {
		result["logs"] = t.logs
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
	} else if t.mode == module.TraceModeBalanceChange {
		result["blockHash"] = "0x" + hex.EncodeToString(blk.ID())
		result["prevBlockHash"] = "0x" + hex.EncodeToString(blk.PrevID())
		result["blockHeight"] = fmt.Sprintf("%#x", blk.Height())
		result["timestamp"] = fmt.Sprintf("%#x", blk.Timestamp())
		balanceChanges := trace.BalanceTracerToJSON(t.bt)
		if balanceChanges != nil {
			result["balanceChanges"] = balanceChanges
		}
	}

	return result
}

func (t *traceCallback) OnTransactionStart(txIndex int32, txHash []byte) error {
	if t.mode != module.TraceModeBalanceChange {
		return nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.bt.OnTransactionStart(txIndex, txHash)
}

func (t *traceCallback) OnTransactionRerun(txIndex int32, txHash []byte) error {
	var err error
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.mode == module.TraceModeInvoke {
		t.logs = nil
	} else if t.mode == module.TraceModeBalanceChange {
		err = t.bt.OnTransactionRerun(txIndex, txHash)
	}
	return err
}

func (t *traceCallback) OnTransactionEnd(txIndex int32, txHash []byte) error {
	if t.mode != module.TraceModeBalanceChange {
		return nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.bt.OnTransactionEnd(txIndex, txHash)
}

func (t *traceCallback) OnEnter() error {
	if t.mode != module.TraceModeBalanceChange {
		return nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.bt.OnEnter()
}

func (t *traceCallback) OnLeave(success bool) error {
	if t.mode != module.TraceModeBalanceChange {
		return nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.bt.OnLeave(success)
}

func (t *traceCallback) OnBalanceChange(opType module.OpType, from, to module.Address, amount *big.Int) error {
	if t.mode != module.TraceModeBalanceChange {
		return nil
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.bt.OnBalanceChange(opType, from, to, amount)
}

func (t *traceCallback) GetReceipt(txIndex int) module.Receipt {
	if t.rl == nil {
		return nil
	}
	rct, _ := t.rl.Get(txIndex)
	return rct
}
