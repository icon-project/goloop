package log

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/bshuster-repo/logrus-logstash-hook"
	"github.com/evalphobia/logrus_fluent"
	"github.com/sirupsen/logrus"
)

const (
	HookVendorFluentd  = "fluentd"
	HookVendorLogstash = "logstash"
)

type ForwarderConfig struct {
	Vendor     string                 `json:"vendor"`
	Address    string                 `json:"address"`
	Level      string                 `json:"level"`
	Name       string                 `json:"name"`
	TimeFormat string                 `json:"time_format,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

func (c *ForwarderConfig) UnmarshalByOptions(v interface{}) error {
	if len(c.Options) > 0 {
		b, err := json.Marshal(c.Options)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, v); err != nil {
			return err
		}
	}
	return nil
}

func (c *ForwarderConfig) NetworkAndHostPort(defaultNet string) (network string, hostPort string, err error) {
	addr := c.Address
	if !strings.Contains(addr, "://") {
		addr = defaultNet + "://" + addr
	}
	fmt.Println(c.Address)
	var u *url.URL
	u, err = url.Parse(addr)
	if err != nil {
		return
	}
	hostPort = u.Host
	if hostPort == "" {
		err = fmt.Errorf("not exists hostPort value")
		return
	}
	network = u.Scheme
	if network == "unix" {
		err = fmt.Errorf("invalid network")
		return
	}
	return network, u.Host, nil
}

func parseHostPort(hostPort string) (string, int) {
	idx := strings.Index(hostPort, ":")
	if idx > 0 {
		port, _ := strconv.Atoi(hostPort[idx+1:])
		return hostPort[:idx], port
	}
	return hostPort, 0
}

func (c *ForwarderConfig) HookLevels() ([]logrus.Level, error) {
	lv, err := ParseLevel(c.Level)
	if err != nil {
		return nil, err
	}
	lvs := make([]logrus.Level, 0)
	switch lv {
	case TraceLevel:
		lvs = append(lvs, logrus.TraceLevel)
		fallthrough
	case DebugLevel:
		lvs = append(lvs, logrus.DebugLevel)
		fallthrough
	case InfoLevel:
		lvs = append(lvs, logrus.InfoLevel)
		fallthrough
	case WarnLevel:
		lvs = append(lvs, logrus.WarnLevel)
		fallthrough
	case ErrorLevel:
		lvs = append(lvs, logrus.ErrorLevel)
		fallthrough
	case FatalLevel:
		lvs = append(lvs, logrus.FatalLevel)
		fallthrough
	case PanicLevel:
		lvs = append(lvs, logrus.PanicLevel)
	}
	return lvs, nil
}

type HookWrapper struct {
	h   logrus.Hook
	lvs []logrus.Level
	c   *ForwarderConfig
}

func (h *HookWrapper) Levels() []logrus.Level {
	return h.lvs
}

func (h *HookWrapper) Fire(e *logrus.Entry) error {
	d := e.Data
	defer func() {
		e.Data = d
	}()
	e.Data = make(map[string]interface{}, len(d)+2)
	for k, v := range d {
		e.Data[k] = v
	}

	e.Data["logtime"] = e.Time.UnixNano()

	var hasModule bool
	if e.Caller != nil {
		if _, hasModule = e.Data[FieldKeyModule]; !hasModule {
			e.Data[FieldKeyModule] = getPackageName(e.Caller.Function)
		}
		e.Data["src"] = fmt.Sprintf("%s:%d", path.Base(e.Caller.File), e.Caller.Line)
	}
	err := h.h.Fire(e)
	return err
}

type HookCreater func(c *ForwarderConfig) (logrus.Hook, error)

func AddForwarder(c *ForwarderConfig) error {
	if c.Level == "" {
		c.Level = "info"
	}
	if c.Name == "" {
		c.Name = "goloop"
	}
	if c.TimeFormat == "" {
		c.TimeFormat = time.RFC3339Nano
	}

	var h logrus.Hook
	var err error
	switch c.Vendor {
	case HookVendorFluentd:
		h, err = newHook(c, fluentHookCreater)
	case HookVendorLogstash:
		h, err = newHook(c, logstashHookCreater)
	default:
		return fmt.Errorf("not supported forwarder %s", c.Vendor)
	}
	if err != nil {
		return err
	}
	globalLogger.addHook(h)
	return nil
}

func newHook(c *ForwarderConfig, f HookCreater) (logrus.Hook, error) {
	if c == nil || f == nil {
		return nil, fmt.Errorf("arguments cannot be nil")
	}
	lvs, err := c.HookLevels()
	if err != nil {
		return nil, err
	}
	h, err := f(c)
	if err != nil {
		return nil, err
	}
	return &HookWrapper{h, lvs, c}, nil
}

func fluentHookCreater(c *ForwarderConfig) (logrus.Hook, error) {
	network, hostPort, err := c.NetworkAndHostPort("tcp")
	if err != nil {
		return nil, err
	}
	host, port := parseHostPort(hostPort)
	lvs, err := c.HookLevels()
	if err != nil {
		return nil, err
	}
	opt := struct {
		Timeout      time.Duration `json:"timeout"`
		WriteTimeout time.Duration `json:"write_timeout"`
		RetryWait    int           `json:"retry_wait"`
		MaxRetry     int           `json:"max_retry"`
	}{
		//default values from https://github.com/fluent/fluent-logger-golang/blob/master/fluent/fluent.go
		Timeout:      3 * time.Second,
		WriteTimeout: time.Duration(0), // Write() will not time out
		RetryWait:    500,
		MaxRetry:     13,
	}
	if err = c.UnmarshalByOptions(&opt); err != nil {
		return nil, err
	}
	fc := logrus_fluent.Config{
		DefaultTag:    c.Name,
		FluentNetwork: network,
		Host:          host,
		Port:          port,
		LogLevels:     lvs,
		Timeout:       opt.Timeout,
		WriteTimeout:  opt.WriteTimeout,
		RetryWait:     opt.RetryWait,
		MaxRetry:      opt.MaxRetry,
		//Default
		DefaultMessageField: "message",
		SubSecondPrecision:  true,
	}
	return logrus_fluent.NewWithConfig(fc)
}

func logstashHookCreater(c *ForwarderConfig) (logrus.Hook, error) {
	network, hostPort, err := c.NetworkAndHostPort("tcp")
	if err != nil {
		return nil, err
	}

	if h, err := logrustash.NewHook(network, hostPort, c.Name); err != nil {
		return nil, err
	} else {
		h.TimeFormat = c.TimeFormat
		return h, nil
	}
}
