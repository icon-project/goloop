package service

import (
	"sync"

	"github.com/icon-project/goloop/common/errors"
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

		if t.canceled() {
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
		ctx = t.newContractContext(wc)

		ec.Ready()
		go func(ctx contract.Context, wc state.WorldContext, txo transaction.Transaction, cnt int, rb *txresult.Receipt) {
			wvs := ctx.WorldVirtualState()
			wvss := wvs.GetSnapshot()
			for retry := 0; ; retry++ {
				ctx.SetTransactionInfo(&state.TransactionInfo{
					Group:     txo.Group(),
					Index:     int32(cnt),
					Timestamp: txo.Timestamp(),
					Nonce:     txo.Nonce(),
					Hash:      txo.ID(),
					From:      txo.From(),
				})
				ctx.UpdateSystemInfo()
				rct, err := txh.Execute(ctx, wvss, false)
				txh.Dispose()
				if err == nil {
					err = t.plt.OnTransactionEnd(ctx, t.log, rct)
				}
				if err == nil {
					*rb = rct
					break
				}

				if !errors.ExecutionFailError.Equals(err) && !errors.CriticalRerunError.Equals(err) {
					t.log.Warnf("Fail to execute transaction err=%+v", err)
					ec.Report(err)
					break
				}

				if retry >= RetryCount {
					t.log.Warnf("Fail to execute transaction retry=%d err=%+v", retry, err)
					ec.Report(err)
					break
				}

				t.log.Warnf("RETRY TX <%#x> for err=%+v", txo.ID(), err)
				if err := wvs.Reset(wvss); err != nil {
					t.log.Errorf("Fail to revert status on rerun err=%+v", err)
					ec.Report(errors.CriticalUnknownError.Wrapf(err, "FailToResetForRetry"))
					break
				}

				txh, err = txo.GetHandler(t.cm)
				if err != nil {
					t.log.Debugf("Fail to get handler err=%+v", err)
					ec.Report(err)
					break
				}
				ctx = t.newContractContext(wc)
			}
			wvs.Commit()
			ec.Done()
		}(ctx, wc, txo, cnt, &rctBuf[cnt])

		cnt++
	}
	if wvs := ctx.WorldVirtualState(); wvs != nil {
		wvs.Realize()
	}
	return nil
}
