package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/module"
)

const Version = "2.0"

const (
	APIVersion2    = 2
	APIVersion3    = 3
	APIVersionLast = APIVersion3
)

type Request struct {
	Version string          `json:"jsonrpc" validate:"required,version"`
	Method  string          `json:"method" validate:"required"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type Response struct {
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

const (
	HeaderKeyIconOptions = "Icon-Options"
	IconOptionsDebug     = "debug"
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
	ctx := &Context{Context: c, opts: NewIconOptionsByHeader(c.Request().Header)}
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

type Params struct {
	rawMessage json.RawMessage
	validator  echo.Validator
}

func (p *Params) Convert(v interface{}) error {
	if p.rawMessage == nil {
		return errors.New("params message is null")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("v is not pointer type or v is nil")
	}
	if err := json.Unmarshal(p.rawMessage, v); err != nil {
		return err
	}
	if err := p.validator.Validate(v); err != nil {
		return err
	}
	return nil
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
