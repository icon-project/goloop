package service

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

func (t *transition) executeTxsSequential(l module.TransactionList, ctx contract.Context, rctBuf []txresult.Receipt) error {
	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.step == stepCanceled {
			return ErrTransitionInterrupted
		}
		txi, _, err := i.Get()
		if err != nil {
			t.log.Errorf("Fail to iterate transaction list err=%+v", err)
			return err
		}
		txo := txi.(transaction.Transaction)
		t.log.Tracef("START TX <0x%x>", txo.ID())
		for trial := 0; ; trial++ {
			txh, err := txo.GetHandler(t.cm)
			if err != nil {
				t.log.Errorf("Fail to GetHandler err=%+v", err)
				return err
			}
			ctx.SetTransactionInfo(&state.TransactionInfo{
				Group:     txo.Group(),
				Index:     int32(cnt),
				Timestamp: txo.Timestamp(),
				Nonce:     txo.Nonce(),
				Hash:      txo.ID(),
				From:      txo.From(),
			})
			ctx.UpdateSystemInfo()
			rct, err := txh.Execute(ctx, false)
			txh.Dispose()
			if err == nil {
				rctBuf[cnt] = rct
				break
			}
			if !errors.ExecutionFailError.Equals(err) {
				t.log.Warnf("Fail to execute transaction err=%+v", err)
				return err
			}
			if trial == RetryCount {
				t.log.Warnf("Fail to execute transaction retry=%d err=%+v", trial, err)
				return err
			}
			t.log.Warnf("RETRY TX <%#x> for err=%+v", txo.ID(), err)
		}
		t.log.Tracef("END   TX <0x%x>", txo.ID())
		cnt++
	}
	return nil
}
