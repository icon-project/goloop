package service

import (
	"sync"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type executionContext struct {
	waiter    chan struct{}
	lastError error
	lock      sync.Mutex
}

func (c *executionContext) Done() {
	c.waiter <- struct{}{}
}

func (c *executionContext) Ready() {
	<-c.waiter
}

func (c *executionContext) Error() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lastError
}

func (c *executionContext) Report(e error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.lastError != nil {
		c.lastError = e
	}
}

func newExecutionContext(n int) *executionContext {
	ch := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		ch <- struct{}{}
	}
	return &executionContext{waiter: ch}
}

func (t *transition) executeTxsConcurrent(level int, l module.TransactionList, ctx contract.Context, rctBuf []txresult.Receipt) error {
	ec := newExecutionContext(level)

	cnt := 0
	for i := l.Iterator(); i.Has(); i.Next() {
		if err := ec.Error(); err != nil {
			return err
		}

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
			t.log.Debugf("Fail to handle transaction for %+v", err)
			return err
		}
		wc, err2 := txh.Prepare(ctx)
		if err2 != nil {
			t.log.Debugf("Fail to prepare for %+v", err2)
			return err2
		}
		ctx = contract.NewContext(wc, t.cm, t.eem, t.chain, t.log)
		ctx.SetTransactionInfo(&state.TransactionInfo{
			Index:     int32(cnt),
			Timestamp: txo.Timestamp(),
			Nonce:     txo.Nonce(),
			Hash:      txo.ID(),
			From:      txo.From(),
		})

		ec.Ready()
		go func(ctx contract.Context, rb *txresult.Receipt) {
			wvs := ctx.WorldVirtualState()
			if rct, err := txh.Execute(ctx); err != nil {
				t.log.Debugf("Fail to execute transaction err=%+v", err)
				ec.Report(err)
			} else {
				*rb = rct
			}
			txh.Dispose()
			wvs.Commit()
			ec.Done()
		}(ctx, &rctBuf[cnt])

		cnt++
	}
	if wvs := ctx.WorldVirtualState(); wvs != nil {
		wvs.Realize()
	}
	return nil
}
