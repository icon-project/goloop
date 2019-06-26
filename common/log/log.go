package log

import (
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	TextFormatter = iota
	JSONFormatter
	LogrusText
)

const (
	LogTimeLayout = "15:04:05.000000"
)

type Level int

const (
	TraceLevel = Level(logrus.TraceLevel)
	DebugLevel = Level(logrus.DebugLevel)
	InfoLevel  = Level(logrus.InfoLevel)
	WarnLevel  = Level(logrus.WarnLevel)
	ErrorLevel = Level(logrus.ErrorLevel)
	FatalLevel = Level(logrus.FatalLevel)
	PanicLevel = Level(logrus.PanicLevel)
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	default:
		return "unknown"
	}
}

const (
	FieldKeyWallet = "wallet"
	FieldKeyModule = "module"
	FieldKeyNID    = "nid"
)

var systemFields = map[string]bool{
	FieldKeyWallet: true,
	FieldKeyModule: true,
	FieldKeyNID:    true,
}

var Trace, Print, Debug, Info, Warn, Error, Panic, Fatal func(args ...interface{})
var Tracef, Printf, Debugf, Infof, Warnf, Errorf, Panicf, Fatalf func(format string, args ...interface{})
var Traceln, Println, Debugln, Infoln, Warnln, Errorln, Panicln, Fatalln func(args ...interface{})

type Fields logrus.Fields

type Logger interface {
	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})

	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	Traceln(args ...interface{})

	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Debugln(args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Panicln(args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})

	WithFields(Fields) Logger
	SetReportCaller(yn bool)
	SetLevel(lv Level)
	SetConsoleLevel(lv Level)
	SetModuleLevel(mod string, lv Level)
	Writer() *io.PipeWriter
}

type entryWrapper struct {
	*logrus.Entry
}

func (w entryWrapper) WithFields(fields Fields) Logger {
	return &entryWrapper{
		w.Entry.WithFields(logrus.Fields(fields)),
	}
}

func (w entryWrapper) SetReportCaller(yn bool) {
	w.Entry.Logger.SetReportCaller(yn)
}

func (w entryWrapper) SetLevel(lv Level) {
	w.Entry.Logger.SetLevel(logrus.Level(lv))
}

func (w entryWrapper) SetConsoleLevel(lv Level) {
	w.Logger.Formatter.(*logFilter).SetDefaultLevel(lv)
}

func (w entryWrapper) SetModuleLevel(mod string, lv Level) {
	w.Logger.Formatter.(*logFilter).SetModuleLevel(mod, lv)
}

func (w entryWrapper) Writer() *io.PipeWriter {
	return w.Entry.Writer()
}

type loggerWrapper struct {
	*logrus.Logger
}

func (w loggerWrapper) WithFields(fields Fields) Logger {
	return &entryWrapper{
		w.Logger.WithFields(logrus.Fields(fields)),
	}
}

func (w loggerWrapper) SetLevel(lv Level) {
	w.Logger.SetLevel(logrus.Level(lv))
}

func (w loggerWrapper) SetConsoleLevel(lv Level) {
	w.Logger.Formatter.(*logFilter).SetDefaultLevel(lv)
}

func (w loggerWrapper) SetModuleLevel(mod string, lv Level) {
	w.Logger.Formatter.(*logFilter).SetModuleLevel(mod, lv)
}

func (w loggerWrapper) Writer() *io.PipeWriter {
	return w.Logger.Writer()
}

func getPackageName(f string) string {
	lastSlash := strings.LastIndex(f, "/")
	if lastSlash >= 0 {
		f = f[lastSlash+1:]
	}

	firstPeriod := strings.Index(f, ".")
	if firstPeriod > 0 {
		f = f[0:firstPeriod]
	}
	return f
}

var globalLogger Logger

func SetGlobalLogger(logger Logger) {
	globalLogger = logger

	Print = logger.Print
	Printf = logger.Printf
	Println = logger.Println

	Trace = logger.Trace
	Tracef = logger.Tracef
	Traceln = logger.Traceln

	Debug = logger.Debug
	Debugf = logger.Debugf
	Debugln = logger.Debugln

	Info = logger.Info
	Infof = logger.Infof
	Infoln = logger.Infoln

	Warn = logger.Warn
	Warnf = logger.Warnf
	Warnln = logger.Warnln

	Error = logger.Error
	Errorf = logger.Errorf
	Errorln = logger.Errorln

	Panic = logger.Panic
	Panicf = logger.Panicf
	Panicln = logger.Panicln

	Fatal = logger.Fatal
	Fatalf = logger.Fatalf
	Fatalln = logger.Fatalln
}

func WithFields(fields Fields) Logger {
	return globalLogger.WithFields(fields)
}

func GlobalLogger() Logger {
	return globalLogger
}

func New() Logger {
	logger := logrus.New()
	logger.Out = os.Stderr
	logger.Level = logrus.DebugLevel
	logger.SetReportCaller(true)
	logger.SetFormatter(newLogFilter(customFormatter{}))
	return &loggerWrapper{
		Logger: logger,
	}
}

func init() {
	logger := New()
	SetGlobalLogger(logger)
}
