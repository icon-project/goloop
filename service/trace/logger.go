package trace

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type Logger struct {
	log.Logger
	prefix string
	cb     module.TraceCallback
}

func (l *Logger) GetTraceMode() module.TraceMode {
	if l.cb != nil {
		return l.cb.TraceMode()
	}
	return module.TraceModeNone
}

func (l *Logger) onLog(lv module.TraceLevel, msg string) {
	if l.cb != nil {
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
		Logger: l.Logger.WithFields(f),
		prefix: l.prefix,
		cb:     l.cb,
	}
}

func (l *Logger) TPrefix() string {
	return l.prefix
}

func (l *Logger) WithTPrefix(prefix string) *Logger {
	return &Logger{
		Logger: l.Logger,
		prefix: prefix,
		cb:     l.cb,
	}
}

func (l *Logger) OnTransactionStart(txInfo *state.TransactionInfo) {
	if l.cb == nil {
		return
	}
	if txInfo == nil {
		l.Warnf("OnTransactionStart() error: invalid txInfo")
		return
	}

	txHashHex := hex.EncodeToString(txInfo.Hash)

	if err := l.cb.OnTransactionStart(txInfo.Index, txInfo.Hash); err != nil {
		l.Warnf("OnTransactionStart() error: txIndex=%d txHash=%s err=%#v",
			txInfo.Index, txHashHex, err)
	}
}

func (l *Logger) OnTransactionRerun(txInfo *state.TransactionInfo) {
	if l.cb == nil {
		return
	}
	if txInfo == nil {
		l.Warnf("OnTransactionRerun() error: invalid txInfo")
		return
	}

	txHashHex := hex.EncodeToString(txInfo.Hash)

	if err := l.cb.OnTransactionRerun(txInfo.Index, txInfo.Hash); err != nil {
		l.Warnf("OnTransactionRerun() error: txIndex=%d txHash=%s err=%#v",
			txInfo.Index, txHashHex, err)
	}
}

func (l *Logger) OnTransactionEnd(txInfo *state.TransactionInfo, traceRct module.Receipt, treasury module.Address) {
	if l.cb == nil {
		return
	}
	if txInfo == nil || treasury == nil {
		l.Warnf("OnTransactionEnd() error")
		return
	}

	finalRct := l.cb.GetReceipt(int(txInfo.Index))
	l.checkReceipts(traceRct, finalRct)

	txHashHex := hex.EncodeToString(txInfo.Hash)
	l.onFee(txInfo, treasury, finalRct)

	if err := l.cb.OnTransactionEnd(txInfo.Index, txInfo.Hash); err != nil {
		l.Warnf("OnTransactionEnd() error: txIndex=%d txHash=%s err=%#v",
			txInfo.Index, txHashHex, err)
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

func (l *Logger) onFee(txInfo *state.TransactionInfo, to module.Address, rct module.Receipt) {
	if rct == nil {
		l.Warnf("Receipt is null: txIndex=%d txHash=%s from=%s",
			txInfo.Index, txInfo.Hash, txInfo.From)
		return
	}

	stepPrice := rct.StepPrice()
	feePayerCnt := 0
	for it := rct.FeePaymentIterator(); it.Has(); log.Must(it.Next()) {
		feePayment, _ := it.Get()
		l.OnBalanceChange(
			module.Fee, txInfo.From, to,
			new(big.Int).Mul(stepPrice, feePayment.Amount()))
		feePayerCnt++
	}
	if feePayerCnt == 0 {
		l.OnBalanceChange(
			module.Fee, txInfo.From, to,
			new(big.Int).Mul(rct.StepPrice(), rct.StepUsed()))
	}
}

func (l *Logger) OnEnter(frameId int) {
	if l.cb == nil {
		return
	}

	l.TSystemf("START parent=FRAME[%d]", frameId)
	if err := l.cb.OnEnter(); err != nil {
		l.Warnf("OnEnter() error: err=%#v", err)
	}
}

func (l *Logger) OnLeave(success bool, stepUsed *big.Int) {
	if l.cb == nil {
		return
	}
	if stepUsed == nil {
		l.Warnf("OnLeave() error: invalid stepUsed")
		return
	}

	l.TSystemf("END success=%v steps=%d", success, stepUsed)
	if err := l.cb.OnLeave(success); err != nil {
		l.Warnf("OnLeave() error: success=%t err=%#v", success, err)
	}
}

func (l *Logger) OnBalanceChange(opType module.OpType, from, to module.Address, amount *big.Int) {
	if l.cb == nil {
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

func NewLogger(l log.Logger, cb module.TraceCallback) *Logger {
	return &Logger{
		Logger: l,
		cb:     cb,
	}
}

func LoggerOf(l log.Logger) *Logger {
	if logger, ok := l.(*Logger); ok {
		return logger
	}
	return NewLogger(l, nil)
}
