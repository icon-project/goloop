package trace

import (
	"fmt"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type Logger struct {
	log.Logger
	isTrace bool
	onLog   func(lv module.TraceLevel, msg string)
}

func (l *Logger) TLog(lv module.TraceLevel, a ...interface{}) {
	l.onLog(lv, fmt.Sprint(a...))
}

func (l *Logger) IsTrace() bool {
	return l.isTrace
}

func (l *Logger) TLogln(lv module.TraceLevel, f string, a ...interface{}) {
	l.onLog(lv, fmt.Sprint(a...))
}

func (l *Logger) TLogf(lv module.TraceLevel, f string, a ...interface{}) {
	l.onLog(lv, fmt.Sprintf(f, a...))
}

func (l *Logger) TDebug(a ...interface{}) {
	l.onLog(module.TDebugLevel, fmt.Sprint(a...))
}

func (l *Logger) TDebugln(a ...interface{}) {
	l.onLog(module.TDebugLevel, fmt.Sprint(a...))
}

func (l *Logger) TDebugf(f string, a ...interface{}) {
	l.onLog(module.TDebugLevel, fmt.Sprintf(f, a...))
}

func (l *Logger) TTrace(a ...interface{}) {
	l.onLog(module.TTraceLevel, fmt.Sprint(a...))
}

func (l *Logger) TTraceln(a ...interface{}) {
	l.onLog(module.TTraceLevel, fmt.Sprint(a...))
}

func (l *Logger) TTracef(f string, a ...interface{}) {
	l.onLog(module.TTraceLevel, fmt.Sprintf(f, a...))
}

func (l *Logger) TSystem(a ...interface{}) {
	l.onLog(module.TSystemLevel, fmt.Sprint(a...))
}

func (l *Logger) TSystemln(a ...interface{}) {
	l.onLog(module.TSystemLevel, fmt.Sprint(a...))
}

func (l *Logger) TSystemf(f string, a ...interface{}) {
	l.onLog(module.TSystemLevel, fmt.Sprintf(f, a...))
}

func (l *Logger) WithFields(f log.Fields) log.Logger {
	return &Logger{
		Logger:  l.Logger.WithFields(f),
		isTrace: l.isTrace,
		onLog:   l.onLog,
	}
}

func dummyLog(lv module.TraceLevel, msg string) {
	// do nothing
}

func NewLogger(l log.Logger, t module.TraceCallback) *Logger {
	if t != nil {
		return &Logger{
			Logger:  l,
			isTrace: true,
			onLog:   t.OnLog,
		}
	} else {
		return &Logger{
			Logger: l,
			onLog:  dummyLog,
		}
	}
}

func LoggerOf(l log.Logger) *Logger {
	if logger, ok := l.(*Logger); ok {
		return logger
	}
	return NewLogger(l, nil)
}
