package jsonrpc

import (
	"net/http"
	"sync"

	"github.com/labstack/echo"
)

type Handler func(ctx *Context, params *Params) (result interface{}, err error)

type MethodRepository struct {
	mtx     sync.RWMutex
	methods map[string]Handler
}

func NewMethodRepository() *MethodRepository {
	return &MethodRepository{
		mtx:     sync.RWMutex{},
		methods: map[string]Handler{},
	}
}

func (mr *MethodRepository) RegisterMethod(method string, handler Handler) {
	if method == "" || handler == nil {
		return
	}
	mr.mtx.Lock()
	mr.methods[method] = handler
	mr.mtx.Unlock()
	return
}

func (mr *MethodRepository) TakeMethod(r *Request) (Handler, error) {
	if r.Method == "" || r.Version != Version {
		return nil, ErrInvalidParams()
	}
	mr.mtx.RLock()
	md, ok := mr.methods[r.Method]
	mr.mtx.RUnlock()
	if !ok {
		return nil, ErrMethodNotFound()
	}
	return md, nil
}

func (mr *MethodRepository) InvokeMethod(c echo.Context, r *Request) (interface{}, error) {
	h := c.Get("method").(Handler)

	ctx := Context{c}
	param := Params{
		rawMessage: r.Params,
		validator:  c.Echo().Validator,
	}

	return h(&ctx, &param)
}

func (mr *MethodRepository) Handle(c echo.Context) (err error) {
	r := c.Get("request").(*Request)

	result, err := mr.InvokeMethod(c, r)
	if err != nil {
		return err
	}

	res := &Response{
		ID:      r.ID,
		Version: Version,
		Result:  result,
	}

	return c.JSON(http.StatusOK, res)
}
