package jsonrpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

type Handler func(ctx *Context, params *Params) (result interface{}, err error)

type MethodRepository struct {
	mtx     sync.RWMutex
	methods map[string]Handler
	allowed map[string]bool
}

func NewMethodRepository() *MethodRepository {
	return &MethodRepository{
		methods: make(map[string]Handler),
		allowed: make(map[string]bool),
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

func (mr *MethodRepository) GetMethod(method string) Handler {
	defer mr.mtx.RUnlock()
	mr.mtx.RLock()

	return mr.methods[method]
}

func (mr *MethodRepository) SetAllowedNotification(method string) {
	defer mr.mtx.Unlock()
	mr.mtx.Lock()

	mr.allowed[method] = true
}

func (mr *MethodRepository) IsAllowedNotification(method string) bool {
	defer mr.mtx.RUnlock()
	mr.mtx.RLock()

	allowed, ok := mr.allowed[method]
	return ok && allowed
}

func (mr *MethodRepository) handle(ctx *Context, raw json.RawMessage) *Response {
	debug := ctx.IncludeDebug()

	resp := &Response{Version: Version}
	req := new(Request)
	jd := json.NewDecoder(bytes.NewBuffer(raw))
	jd.DisallowUnknownFields()
	if err := jd.Decode(req); err != nil {
		resp.ID = req.ID
		resp.Error = ErrInvalidRequest()
		return resp
	}
	resp.ID = req.ID
	if err := ctx.Validate(req); err != nil || req.Method == nil {
		resp.Error = ErrInvalidRequest()
		return resp
	}
	method := mr.GetMethod(*req.Method)
	if method == nil {
		resp.Error = ErrMethodNotFound()
		return resp
	}

	if req.ID == nil && !mr.IsAllowedNotification(*req.Method){
		return nil
	}

	p := &Params{
		rawMessage: req.Params,
		validator:  ctx.Validator(),
	}
	res, err := method(ctx, p)
	if err != nil {
		if je, ok := err.(*Error); ok {
			resp.Error = je
		} else {
			resp.Error = ErrorCodeInternal.Wrap(err, debug)
		}
	} else {
		resp.Result = res
	}
	if req.ID == nil {
		return nil
	}
	return resp
}

func (mr *MethodRepository) Handle(c echo.Context) error {
	ctx := NewContext(c)
	raw := c.Get("raw").(json.RawMessage)
	var raws []json.RawMessage
	if err := json.Unmarshal(raw, &raws); err == nil {
		n := len(raws)
		if n == 0 {
			resp := &Response{
				Version: Version,
				Error:   ErrInvalidRequest(),
			}
			return c.JSON(http.StatusBadRequest, resp)
		}
		if n > LimitOfBatch {
			resp := &Response{
				Version: Version,
				Error:   ErrInvalidRequest("too many request"),
			}
			return c.JSON(http.StatusServiceUnavailable, resp)
		}
		var wg sync.WaitGroup
		wg.Add(n)
		rs := make([]*Response, len(raws))
		for i, r := range raws{
			go func(r json.RawMessage, rs []*Response, i int) {
				rs[i] = mr.handle(ctx, r)
				wg.Done()
			}(r, rs, i)
		}
		wg.Wait()

		resps := make([]*Response, 0)
		for _, r := range rs {
			if r != nil {
				resps = append(resps, r)
			}
		}
		return c.JSON(http.StatusOK, resps)
	} else {
		resp := mr.handle(ctx, raw)
		if resp != nil {
			if resp.Error != nil {
				return c.JSON(http.StatusBadRequest, resp)
			} else {
				return c.JSON(http.StatusOK, resp)
			}
		} else {
			return c.NoContent(http.StatusOK)
		}
	}
}
