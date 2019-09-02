package service

import (
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
		txh, err := txo.GetHandler(t.cm)
		if err != nil {
			t.log.Errorf("Fail to GetHandler err=%+v", err)
			return err
		}
		ctx.SetTransactionInfo(&state.TransactionInfo{
			Index:     int32(cnt),
			Timestamp: txo.Timestamp(),
			Nonce:     txo.Nonce(),
			Hash:      txo.ID(),
			From:      txo.From(),
		})
		t.log.Tracef("START TX <0x%x>", txo.ID())
		ctx.UpdateSystemInfo()
		ctx.ClearCache()
		if rct, err := txh.Execute(ctx); err != nil {
			txh.Dispose()
			t.log.Debugf("Fail to execute transaction err=%+v\n", err)
			return err
		} else {
			rctBuf[cnt] = rct
		}
		t.log.Tracef("END   TX <0x%x>", txo.ID())
		txh.Dispose()
		cnt++
	}
	return nil
}
