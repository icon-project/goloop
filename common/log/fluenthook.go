package log

import (
	"bytes"
	"github.com/evalphobia/logrus_fluent"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	defaultTag          = "fluent.tag"
	defaultMessageField = "msg"
	defaultFluentLevel  = "info"
	defaultMaxRetry     = 1

	defaultHost         = "127.0.0.1"
	defaultPort         = 24224
	defaultTimeout      = 3 * time.Second
	defaultWriteTimeout = time.Duration(0) // Write() will not time out
	defaultBufferLimit  = 8 * 1024
	defaultRetryWait    = 500
	defaultMaxRetryWait = 60000
)

type GoLoopFluentConfig struct {
	Level               string `json:"level"`
	DefaultTag          string `json:"tag"`
	DefaultMessageField string `json:"msg_filed"`

	Port         int           `json:"port"`
	Host         string        `json:"host"`
	Timeout      time.Duration `json:"timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	RetryWait    int           `json:"retry_wait"`
	MaxRetry     int           `json:"max_retry"`
}

var logLevels = map[int]logrus.Level{
	0: logrus.TraceLevel,
	1: logrus.DebugLevel,
	2: logrus.InfoLevel,
	3: logrus.WarnLevel,
	4: logrus.ErrorLevel,
	5: logrus.FatalLevel,
	6: logrus.PanicLevel,
}

var logLevelsToInt = map[string]int{
	"trace": 0,
	"debug": 1,
	"info":  2,
	"warn":  3,
	"error": 4,
	"fatal": 5,
	"panic": 6,
}

func SetFluentConfig(fluent map[string]string, cfg *GoLoopFluentConfig) error {

	cfg.Level = defaultFluentLevel
	cfg.DefaultTag = defaultTag
	cfg.DefaultMessageField = defaultMessageField
	cfg.Port = defaultPort
	cfg.Host = defaultHost
	cfg.Timeout = defaultTimeout
	cfg.WriteTimeout = defaultWriteTimeout
	cfg.RetryWait = defaultRetryWait
	cfg.MaxRetry = defaultMaxRetry

	if host, ok := fluent["host"]; ok { // 127.0.0.1
		cfg.Host = host
	}
	if port, ok := fluent["port"]; ok { // 24224
		cfg.Port, _ = strconv.Atoi(port)
	}
	if level, ok := fluent["level"]; ok { // info
		cfg.Level = level
	}
	if tag, ok := fluent["tag"]; ok { // fluent.tag
		cfg.DefaultTag = tag
	}
	if msgFiled, ok := fluent["msg_filed"]; ok { // msg
		cfg.DefaultMessageField = msgFiled
	}
	if maxRetry, ok := fluent["max_retry"]; ok { // 1
		cfg.MaxRetry, _ = strconv.Atoi(maxRetry)
	}
	if timeout, ok := fluent["timeout"]; ok { // 3 * time.Second
		cfg.Timeout, _ = time.ParseDuration(timeout)
	}
	if writeTimeout, ok := fluent["write_timeout"]; ok { // time.Duration(0)
		cfg.WriteTimeout, _ = time.ParseDuration(writeTimeout)
	}
	if retryWait, ok := fluent["retry_wait"]; ok { // 500
		cfg.RetryWait, _ = strconv.Atoi(retryWait)
	}

	return nil
}

func SetReFluentConfig(fluent map[string]string, cfg *GoLoopFluentConfig) error {

	if host, ok := fluent["host"]; ok { // 127.0.0.1
		cfg.Host = host
	}
	if port, ok := fluent["port"]; ok { // 24224
		cfg.Port, _ = strconv.Atoi(port)
	}
	if level, ok := fluent["level"]; ok { // info
		cfg.Level = level
	}
	if tag, ok := fluent["tag"]; ok { // fluent.tag
		cfg.DefaultTag = tag
	}
	if msgFiled, ok := fluent["msg_filed"]; ok { // msg
		cfg.DefaultMessageField = msgFiled
	}
	if maxRetry, ok := fluent["max_retry"]; ok { // 1
		cfg.MaxRetry, _ = strconv.Atoi(maxRetry)
	}
	if timeout, ok := fluent["timeout"]; ok { // 3 * time.Second
		cfg.Timeout, _ = time.ParseDuration(timeout)
	}
	if writeTimeout, ok := fluent["write_timeout"]; ok { // time.Duration(0)
		cfg.WriteTimeout, _ = time.ParseDuration(writeTimeout)
	}
	if retryWait, ok := fluent["retry_wait"]; ok { // 500
		cfg.RetryWait, _ = strconv.Atoi(retryWait)
	}

	return nil
}

func SetFluentHook(cfg *GoLoopFluentConfig) error {

	fConfig := logrus_fluent.Config{

		DisableConnectionPool: false,
		RequestAck:            false,

		// Secret
		FluentNetwork:      "", // "tcp"/"unix", default: tcp
		FluentSocketPath:   "", // only FluentNetwork "unix" use
		SubSecondPrecision: false,

		DefaultIgnoreFields: nil,
		DefaultFilters:      nil,
		TagPrefix:           "",

		// Deprecated: Use Async instead
		AsyncConnect:  false,
		MarshalAsJSON: false,

		// BufferLimit	:0,  // only Async use, default : 8 * 1024
	}

	fLv := logLevelsToInt[cfg.Level]

	setLogLevels := make([]logrus.Level, 7-fLv)

	for i := fLv; i < 7; i++ {
		setLogLevels[i-fLv] = logLevels[i]
	}

	fConfig.LogLevels = setLogLevels
	fConfig.DefaultTag = cfg.DefaultTag
	fConfig.DefaultMessageField = cfg.DefaultMessageField
	fConfig.Port = cfg.Port
	fConfig.Host = cfg.Host
	fConfig.Timeout = cfg.Timeout
	fConfig.WriteTimeout = cfg.WriteTimeout
	fConfig.RetryWait = cfg.RetryWait
	fConfig.MaxRetry = cfg.MaxRetry

	hook, err := logrus_fluent.NewWithConfig(fConfig)

	if err != nil {
		//panic(err)
		return err
	}

	if hook != nil {

		// ADD : logtime, source
		hook.AddCustomizer(func(e *logrus.Entry, data logrus.Fields) {

			//e.Time.Format(LogTimeLayout)
			data["logtime"] = e.Time.Format(LogTimeLayout)

			var buf bytes.Buffer
			if e.HasCaller() {

				buf.WriteString(getPackageName(e.Caller.Function))
				buf.WriteString(".go:")
				buf.WriteString(strconv.Itoa(e.Caller.Line))
				data["source"] = buf.String()
			}
		})
	}

	globalLogger.addHook(hook)

	return nil
}
