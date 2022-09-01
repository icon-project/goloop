package trace

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type Logger struct {
	log.Logger
	prefix     string
	traceMode  module.TraceMode
	traceBlock module.TraceBlock
	cb         module.TraceCallback
}

func (l *Logger) TraceMode() module.TraceMode {
	if l.cb != nil {
		return l.traceMode
	}
	return module.TraceModeNone
}

func (l *Logger) onLog(lv module.TraceLevel, msg string) {
	if l.TraceMode() == module.TraceModeInvoke {
		l.cb.OnLog(lv, msg)
	}
}

func (l *Logger) TLog(lv module.TraceLevel, a ...interface{}) {
	l.onLog(lv, l.prefix+fmt.Sprint(a...))
}

func (l *Logger) TLogln(lv module.TraceLevel, a ...interface{}) {
	l.onLog(lv, l.prefix+fmt.Sprint(a...))
}

func (l *Logger) TLogf(lv module.TraceLevel, f string, a ...interface{}) {
	l.onLog(lv, l.prefix+fmt.Sprintf(f, a...))
}

func (l *Logger) TDebug(a ...interface{}) {
	l.TLog(module.TDebugLevel, a...)
}

func (l *Logger) TDebugln(a ...interface{}) {
	l.TLogln(module.TDebugLevel, a...)
}

func (l *Logger) TDebugf(f string, a ...interface{}) {
	l.TLogf(module.TDebugLevel, f, a...)
}

func (l *Logger) TTrace(a ...interface{}) {
	l.TLog(module.TTraceLevel, a...)
}

func (l *Logger) TTraceln(a ...interface{}) {
	l.TLogln(module.TTraceLevel, a...)
}

func (l *Logger) TTracef(f string, a ...interface{}) {
	l.TLogf(module.TTraceLevel, f, a...)
}

func (l *Logger) TSystem(a ...interface{}) {
	l.TLog(module.TSystemLevel, a...)
}

func (l *Logger) TSystemln(a ...interface{}) {
	l.TLogln(module.TSystemLevel, a...)
}

func (l *Logger) TSystemf(f string, a ...interface{}) {
	l.TLogf(module.TSystemLevel, f, a...)
}

func (l *Logger) WithFields(f log.Fields) log.Logger {
	return &Logger{
		Logger:     l.Logger.WithFields(f),
		prefix:     l.prefix,
		traceMode:  l.traceMode,
		traceBlock: l.traceBlock,
		cb:         l.cb,
	}
}

func (l *Logger) TPrefix() string {
	return l.prefix
}

func (l *Logger) WithTPrefix(prefix string) *Logger {
	return &Logger{
		Logger:     l.Logger,
		prefix:     prefix,
		traceMode:  l.traceMode,
		traceBlock: l.traceBlock,
		cb:         l.cb,
	}
}

func (l *Logger) OnTransactionStart(txIndex int32, txHash []byte) {
	if l.TraceMode() == module.TraceModeBalanceChange {
		if txHash == nil {
			// nil txHash means that it is a virtual transaction to trace balance changes caused by unstaking
			// blockHash is used for txHash of this virtual transaction
			txHash = l.traceBlock.ID()
		}

		if err := l.cb.OnTransactionStart(txIndex, txHash); err != nil {
			l.Warnf("OnTransactionStart() error: txIndex=%d txHash=%#x err=%#v",
				txIndex, txHash, err)
		}
	}
}

func (l *Logger) OnTransactionRerun(txIndex int32, txHash []byte) {
	if l.TraceMode() == module.TraceModeBalanceChange {
		if err := l.cb.OnTransactionRerun(txIndex, txHash); err != nil {
			l.Warnf("OnTransactionRerun() error: txIndex=%d txHash=%#x err=%#v",
				txIndex, txHash, err)
		}
	}
}

func (l *Logger) OnTransactionEnd(
	txIndex int32, txHash []byte, from module.Address, traceRct module.Receipt, treasury module.Address) {
	if l.TraceMode() == module.TraceModeBalanceChange {
		if txHash != nil {
			// Common transaction
			finalRct := l.traceBlock.GetReceipt(int(txIndex))
			l.checkReceipts(traceRct, finalRct)
			l.onFee(from, treasury, finalRct)
		} else {
			// nil txHash means that it is a virtual transaction to trace balance changes caused by unstaking timer
			// blockHash is used for txHash of this virtual transaction
			txHash = l.traceBlock.ID()
		}

		if err := l.cb.OnTransactionEnd(txIndex, txHash); err != nil {
			l.Warnf("OnTransactionEnd() error: txIndex=%d txHash=%#x err=%#v",
				txIndex, txHash, err)
		}
	}
}

func (l *Logger) checkReceipts(traceRct, finalRct module.Receipt) {
	if traceRct == nil {
		l.Errorf("Trace receipt is nil in Logger.OnTransactionEnd()")
		return
	}
	if finalRct == nil {
		l.Errorf("Final receipt is nil in Logger.OnTransactionEnd()")
		return
	}
	if traceRct.Status() != finalRct.Status() {
		l.Errorf("Different status between trace and final receipts: trace=%d final=%d",
			traceRct.Status(), finalRct.Status())
	}
}

func (l *Logger) onFee(from, to module.Address, rct module.Receipt) {
	if rct != nil {
		stepPrice := rct.StepPrice()
		feePayerCnt := 0
		for it := rct.FeePaymentIterator(); it.Has(); log.Must(it.Next()) {
			feePayment, _ := it.Get()
			l.OnBalanceChange(module.Fee, from, to, new(big.Int).Mul(stepPrice, feePayment.Amount()))
			feePayerCnt++
		}
		if feePayerCnt == 0 {
			l.OnBalanceChange(module.Fee, from, to, new(big.Int).Mul(stepPrice, rct.StepUsed()))
		}
	}
}

func (l *Logger) OnFrameEnter(frameId int) {
	if l.cb == nil {
		return
	}

	l.TSystemf("START parent=FRAME[%d]", frameId)
	if err := l.cb.OnFrameEnter(); err != nil {
		l.Warnf("OnFrameEnter() error: err=%#v", err)
	}
}

func (l *Logger) OnFrameExit(success bool, stepUsed *big.Int) {
	if l.TraceMode() == module.TraceModeNone {
		return
	}
	if stepUsed == nil {
		l.Warnf("OnFrameExit() error: invalid stepUsed")
		return
	}

	l.TSystemf("END success=%v steps=%d", success, stepUsed)
	if err := l.cb.OnFrameExit(success); err != nil {
		l.Warnf("OnFrameExit() error: success=%t err=%#v", success, err)
	}
}

func (l *Logger) OnBalanceChange(opType module.OpType, from, to module.Address, amount *big.Int) {
	if l.TraceMode() == module.TraceModeNone {
		return
	}
	if from == nil && to == nil {
		l.Warnf("OnBalanceChange() error: invalid addresses")
		return
	}

	if amount == nil || amount.Sign() <= 0 {
		l.Infof("Invalid amount in OnBalanceChange(): amount=%v", amount)
		return
	}

	l.TSystemf("BALANCECHANGE opType=%d from=%s to=%s amount=%d", opType, from, to, amount)
	if err := l.cb.OnBalanceChange(opType, from, to, amount); err != nil {
		l.Warnf("OnBalanceChange() error: opType=%d from=%s to=%s amount=%d err=%#v",
			opType, from, to, amount, err)
	}
}

func NewLogger(l log.Logger, ti *module.TraceInfo) *Logger {
	tlog := &Logger{
		Logger: l,
	}
	if ti != nil {
		tlog.traceMode = ti.TraceMode
		tlog.traceBlock = ti.TraceBlock
		tlog.cb = ti.Callback
	}
	return tlog
}

func LoggerOf(l log.Logger) *Logger {
	if logger, ok := l.(*Logger); ok {
		return logger
	}
	return NewLogger(l, nil)
}
