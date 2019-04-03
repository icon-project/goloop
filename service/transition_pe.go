package service

import (
	"log"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type goRoutineLimiter chan struct{}

func (l goRoutineLimiter) Done() {
	l <- struct{}{}
}

func (l goRoutineLimiter) Ready() {
	<-l
}

func newLimiter(n int) goRoutineLimiter {
	ch := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		ch <- struct{}{}
	}
	return ch
}

func (t *transition) executeTxsConcurrent(level int, l module.TransactionList, ctx contract.Context, rctBuf []txresult.Receipt) bool {
	limiter := newLimiter(level)

	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if t.step == stepCanceled {
			return false
		}
		txi, _, err := i.Get()
		if err != nil {
			log.Panicf("Fail to iterate transaction list err=%+v", err)
		}
		txo := txi.(transaction.Transaction)
		txh, err := txo.GetHandler(t.cm)
		if err != nil {
			log.Panicf("Fail to handle transaction for %+v", err)
		}
		wc, err2 := txh.Prepare(ctx)
		if err2 != nil {
			log.Panicf("Fail to prepare for %+v", err2)
		}
		ctx = contract.NewContext(wc, t.cm, t.eem, t.chain)
		ctx.SetTransactionInfo(&state.TransactionInfo{
			Index:     int32(cnt),
			Timestamp: txo.Timestamp(),
			Nonce:     txo.Nonce(),
			Hash:      txo.ID(),
			From:      txo.From(),
		})

		limiter.Ready()
		go func(ctx contract.Context, rb *txresult.Receipt) {
			wvs := ctx.WorldVirtualState()
			if rct, err := txh.Execute(ctx); err != nil {
				log.Panicf("Fail to execute transaction err=%+v", err)
			} else {
				*rb = rct
			}
			txh.Dispose()
			wvs.Commit()
			limiter.Done()
		}(ctx, &rctBuf[cnt])

		cnt++
	}
	if wvs := ctx.WorldVirtualState(); wvs != nil {
		wvs.Realize()
	}
	return true
}
