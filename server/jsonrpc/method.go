package jsonrpc

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

type Handler func(ctx *Context, params *Params) (result interface{}, err error)

type MethodRepository struct {
	mtx     sync.RWMutex
	methods map[string]Handler
}

func NewMethodRepository() *MethodRepository {
	return &MethodRepository{
		methods: make(map[string]Handler),
	}
}

func (mr *MethodRepository) RegisterMethod(method string, handler Handler) {
	defer mr.mtx.Unlock()
	mr.mtx.Lock()

	if method == "" || handler == nil {
		return
	}
	mr.methods[method] = handler
}

func (mr *MethodRepository) TakeMethod(r *Request) (Handler, error) {
	defer mr.mtx.RUnlock()
	mr.mtx.RLock()

	if r.Method == "" || r.Version != Version {
		return nil, ErrInvalidParams()
	}

	md, ok := mr.methods[r.Method]
	if !ok {
		return nil, ErrMethodNotFound()
	}
	return md, nil
}

func (mr *MethodRepository) InvokeMethod(c echo.Context, r *Request) (interface{}, error) {
	h := c.Get("method").(Handler)

	ctx := NewContext(c)
	param := Params{
		rawMessage: r.Params,
		validator:  c.Echo().Validator,
	}

	return h(ctx, &param)
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
