package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gopkg.in/go-playground/validator.v9"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

const (
	Version           = "2.0"
	DefaultBatchLimit = 10
)

type Request struct {
	Version string          `json:"jsonrpc" validate:"required,version"`
	Method  *string         `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id" validate:"optional,id"`
}

type Response struct {
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

const (
	HeaderKeyIconOptions = "Icon-Options"
	IconOptionsDebug     = "debug"
	IconOptionsTimeout   = "timeout"
)

type IconOptions map[string]string

func (opts IconOptions) Set(key, value string) {
	opts[key] = value
}
func (opts IconOptions) Get(key string) string {
	if opts == nil {
		return ""
	}
	v := opts[key]
	if len(v) == 0 {
		return ""
	}
	return v
}
func (opts IconOptions) Del(key string) {
	delete(opts, key)
}
func (opts IconOptions) SetBool(key string, value bool) {
	opts.Set(key, strconv.FormatBool(value))
}
func (opts IconOptions) GetBool(key string) (bool, error) {
	return strconv.ParseBool(opts.Get(key))
}
func (opts IconOptions) SetInt(key string, v int64) {
	opts.Set(key, strconv.FormatInt(v, 10))
}
func (opts IconOptions) GetInt(key string) (int64, error) {
	return strconv.ParseInt(opts.Get(key), 10, 64)
}
func (opts IconOptions) ToHeaderValue() string {
	if opts == nil {
		return ""
	}
	strs := make([]string, len(opts))
	i := 0
	for k, v := range opts {
		strs[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	return strings.Join(strs, ",")
}

func NewIconOptionsByHeader(h http.Header) IconOptions {
	s := h.Get(HeaderKeyIconOptions)
	if s != "" {
		kvs := strings.Split(s, ",")
		m := make(map[string]string)
		for _, kv := range kvs {
			if kv != "" {
				idx := strings.Index(kv, "=")
				if idx > 0 {
					m[kv[:idx]] = kv[(idx + 1):]
				} else {
					m[kv] = ""
				}
			}
		}
		return m
	}
	return nil
}

type Context struct {
	echo.Context
	opts IconOptions
}

func NewContext(c echo.Context) *Context {
	ctx := &Context{
		Context: c,
		opts:    NewIconOptionsByHeader(c.Request().Header),
	}
	return ctx
}

func (ctx *Context) Chain() (module.Chain, error) {
	chain, ok := ctx.Get("chain").(module.Chain)
	if chain == nil || !ok {
		return nil, errors.New("chain is not contained in this context")
	}
	return chain, nil
}

func (ctx *Context) IncludeDebug() bool {
	serverDebug := ctx.Get("includeDebug").(bool)
	v, _ := ctx.opts.GetBool(IconOptionsDebug)
	return v && serverDebug
}

func (ctx *Context) BatchLimit() int {
	batchLimit, ok := ctx.Get("batchLimit").(int)
	if !ok {
		batchLimit = DefaultBatchLimit
	}
	return batchLimit
}

func (ctx *Context) GetTimeout(t time.Duration) time.Duration {
	if v, err := ctx.opts.GetInt(IconOptionsTimeout); err != nil {
		return t
	} else {
		return time.Duration(v) * time.Millisecond
	}
}

func (ctx *Context) Validator() echo.Validator {
	return ctx.Echo().Validator
}

func (ctx *Context) MetricContext() context.Context {
	if c, _ := ctx.Chain(); c == nil {
		return metric.DefaultMetricContext()
	} else {
		return c.MetricContext()
	}
}

type Params struct {
	rawMessage json.RawMessage
	validator  echo.Validator
}

func (p *Params) Convert(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("v is not pointer type or v is nil")
	}
	if p.rawMessage == nil || string(p.rawMessage) == "null" {
		rve := rv.Elem()
		if rve.Kind() == reflect.Ptr {
			rve.Set(reflect.Zero(rve.Type()))
			return nil
		}
		nf := rve.NumField()
		if nf > 0 {
			return errors.New(UnmarshalFailPrefix + "'params' of request is required ")
		} else {
			return nil
		}
	} else {
		rve := rv.Elem()
		if rve.Kind() == reflect.Ptr {
			value := reflect.New(rve.Type().Elem())
			if err := UnmarshalWithValidate(p.rawMessage, value.Interface(), p.validator); err != nil {
				return err
			} else {
				rve.Set(value)
				return nil
			}
		}
	}
	return UnmarshalWithValidate(p.rawMessage, v, p.validator)
}

func (p *Params) RawMessage() []byte {
	bs, _ := p.rawMessage.MarshalJSON()
	return bs
}

func (p *Params) IsEmpty() bool {
	if p.rawMessage == nil {
		return true
	}
	return false
}

const (
	UnmarshalFailPrefix = "fail to unmarshal, "
	ValidateFailPrefix  = "fail to validate, "
	JsonErrorPrefix     = "json: "
)

func UnmarshalWithValidate(data []byte, v interface{}, vd echo.Validator) error {
	jd := json.NewDecoder(bytes.NewBuffer(data))
	jd.DisallowUnknownFields()
	if err := jd.Decode(v); err != nil {
		var msg string
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			if ute.Field == "" {
				switch v.(type) {
				case *Request:
					msg = "request must be object type"
				default:
					msg = "'params' of request must be object type"
				}
			} else {
				msg = fmt.Sprintf("'%s' must be %s type", ute.Field, ute.Type)
			}
		} else {
			msg = err.Error()
			if strings.HasPrefix(msg, JsonErrorPrefix) {
				msg = strings.ReplaceAll(msg[len(JsonErrorPrefix):], "\"", "'")
			}
		}
		return errors.Wrap(err, UnmarshalFailPrefix+msg)
	}
	if err := vd.Validate(v); err != nil {
		var msg string
		if ve, ok := err.(validator.ValidationErrors); ok {
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Ptr && !val.IsNil() {
				val = val.Elem()
			}
			vt := val.Type()
			m := make(map[string][]string)
			for _, fe := range ve {
				sf, _ := vt.FieldByName(fe.StructField())
				jt := sf.Tag.Get("json")
				if jt == "" {
					jt = fe.Field()
				}
				if idx := strings.Index(jt, ","); idx >= 0 {
					jt = jt[:idx]
				}
				jt = "'" + jt + "'"
				l, has := m[fe.Tag()]
				if !has {
					l = make([]string, 0)
				}
				l = append(l, jt)
				m[fe.Tag()] = l
			}
			sl := make([]string, len(m))
			idx := 0
			for k, l := range m {
				sl[idx] = fmt.Sprintf("%s(%s)", k, strings.Join(l, ","))
			}
			msg = strings.Join(sl, ",")
		} else {
			fmt.Printf(ValidateFailPrefix+"err:%T %+v", err, err)
			msg = err.Error()
		}
		return errors.Wrap(err, ValidateFailPrefix+msg)
	}
	return nil
}
