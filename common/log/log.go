package log

import (
	"io"
	"os"
	"strings"

	"github.com/icon-project/goloop/common/errors"
	"github.com/sirupsen/logrus"
)

const (
	LogTimeLayout = "20060102-15:04:05.000000"
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

func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		return DebugLevel,
			errors.IllegalArgumentError.Errorf("Invalid log level str=%s", s)
	}
}

const (
	FieldKeyWallet = "wallet"
	FieldKeyModule = "module"
	FieldKeyCID    = "cid"
	FieldKeyPrefix = "prefix"
	FieldKeyEID    = "eid"
)

var systemFields = map[string]bool{
	FieldKeyWallet: true,
	FieldKeyModule: true,
	FieldKeyCID:    true,
	FieldKeyEID:    true,
	FieldKeyPrefix: true,
}

var Trace, Print, Debug, Info, Warn, Error, Panic, Fatal func(args ...interface{})
var Tracef, Printf, Debugf, Infof, Warnf, Errorf, Panicf, Fatalf func(format string, args ...interface{})
var Traceln, Println, Debugln, Infoln, Warnln, Errorln, Panicln, Fatalln func(args ...interface{})
var Must func(err error)

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

	Log(level Level, args ...interface{})
	Logf(level Level, format string, args ...interface{})
	Logln(level Level, args ...interface{})

	Must(err error)

	WithFields(Fields) Logger
	SetReportCaller(yn bool)
	SetLevel(lv Level)
	GetLevel() Level
	SetConsoleLevel(lv Level)
	GetConsoleLevel() Level
	SetModuleLevel(mod string, lv Level)
	GetModuleLevel(mod string) Level
	Writer() *io.PipeWriter
	WriterLevel(lv Level) *io.PipeWriter
	SetFileWriter(writer io.Writer) error
	SetOutput(output io.Writer)

	addHook(hook logrus.Hook)
}

type entryWrapper struct {
	*logrus.Entry
}

func (w entryWrapper) Must(err error) {
	if err != nil {
		w.Panicf("%+v", err)
	}
}

func (w entryWrapper) addHook(hook logrus.Hook) {
	w.Logger.AddHook(hook)
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

func (w entryWrapper) GetLevel() Level {
	return Level(w.Entry.Logger.GetLevel())
}

func (w entryWrapper) SetConsoleLevel(lv Level) {
	w.Logger.Formatter.(*logFilter).SetDefaultLevel(lv)
}

func (w entryWrapper) GetConsoleLevel() Level {
	return w.Logger.Formatter.(*logFilter).GetDefaultLevel()
}

func (w entryWrapper) SetModuleLevel(mod string, lv Level) {
	w.Logger.Formatter.(*logFilter).SetModuleLevel(mod, lv)
}

func (w entryWrapper) GetModuleLevel(mod string) Level {
	return w.Logger.Formatter.(*logFilter).GetModuleLevel(mod)
}

func (w entryWrapper) Writer() *io.PipeWriter {
	return w.Entry.Writer()
}

func (w entryWrapper) WriterLevel(lv Level) *io.PipeWriter {
	return w.Entry.WriterLevel(logrus.Level(lv))
}

func (w entryWrapper) Log(lv Level, args ...interface{}) {
	w.Entry.Log(logrus.Level(lv), args...)
}

func (w entryWrapper) Logln(lv Level, args ...interface{}) {
	w.Entry.Logln(logrus.Level(lv), args...)
}

func (w entryWrapper) Logf(lv Level, format string, args ...interface{}) {
	w.Entry.Logf(logrus.Level(lv), format, args...)
}

func (w entryWrapper) SetFileWriter(writer io.Writer) error {
	return w.Logger.Formatter.(*logFilter).SetFileWriter(writer)
}

func (w entryWrapper) SetOutput(output io.Writer) {
	w.Logger.SetOutput(output)
}

type loggerWrapper struct {
	*logrus.Logger
}

func (w loggerWrapper) addHook(hook logrus.Hook) {
	w.Logger.AddHook(hook)
}

func (w loggerWrapper) WithFields(fields Fields) Logger {
	return &entryWrapper{
		w.Logger.WithFields(logrus.Fields(fields)),
	}
}

func (w loggerWrapper) Must(err error) {
	if err != nil {
		w.Panicf("%+v", err)
	}
}

func (w loggerWrapper) SetLevel(lv Level) {
	w.Logger.SetLevel(logrus.Level(lv))
}

func (w loggerWrapper) GetLevel() Level {
	return Level(w.Logger.GetLevel())
}

func (w loggerWrapper) SetConsoleLevel(lv Level) {
	w.Logger.Formatter.(*logFilter).SetDefaultLevel(lv)
}

func (w loggerWrapper) GetConsoleLevel() Level {
	return w.Logger.Formatter.(*logFilter).GetDefaultLevel()
}

func (w loggerWrapper) SetModuleLevel(mod string, lv Level) {
	w.Logger.Formatter.(*logFilter).SetModuleLevel(mod, lv)
}

func (w loggerWrapper) GetModuleLevel(mod string) Level {
	return w.Logger.Formatter.(*logFilter).GetModuleLevel(mod)
}

func (w loggerWrapper) Writer() *io.PipeWriter {
	return w.Logger.Writer()
}

func (w loggerWrapper) WriterLevel(lv Level) *io.PipeWriter {
	return w.Logger.WriterLevel(logrus.Level(lv))
}

func (w loggerWrapper) Log(lv Level, args ...interface{}) {
	w.Logger.Log(logrus.Level(lv), args...)
}

func (w loggerWrapper) Logln(lv Level, args ...interface{}) {
	w.Logger.Logln(logrus.Level(lv), args...)
}

func (w loggerWrapper) Logf(lv Level, format string, args ...interface{}) {
	w.Logger.Logf(logrus.Level(lv), format, args...)
}

func (w loggerWrapper) SetFileWriter(writer io.Writer) error {
	return w.Logger.Formatter.(*logFilter).SetFileWriter(writer)
}

func (w loggerWrapper) SetOutput(output io.Writer) {
	w.Logger.SetOutput(output)
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

	Must = logger.Must
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
