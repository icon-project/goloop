package jsonrpc

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common/errors"
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
	if err := UnmarshalWithValidate(raw, req, ctx.Validator()); err != nil {
		resp.ID = req.ID
		resp.Error = ErrorCodeInvalidRequest.Wrap(err, debug)
		return resp
	}
	resp.ID = req.ID
	if req.Method == nil {
		err := errors.New(ValidateFailPrefix + "required('method')")
		resp.Error = ErrorCodeInvalidRequest.Wrap(err, debug)
		return resp
	}
	method := mr.GetMethod(*req.Method)
	if method == nil {
		resp.Error = ErrMethodNotFound()
		return resp
	}

	if req.ID == nil && !mr.IsAllowedNotification(*req.Method) {
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
		if res == nil {
			resp.Result = json.RawMessage("null")
		} else {
			resp.Result = res
		}
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
		limitOfBatch, err := strconv.Atoi(os.Getenv("GOLOOP_LIMIT_OF_BATCH"))
		if err != nil || limitOfBatch < 1 {
			limitOfBatch = LimitOfBatch
		}
		if n > limitOfBatch {
			resp := &Response{
				Version: Version,
				Error:   ErrInvalidRequest("too many request"),
			}
			return c.JSON(http.StatusServiceUnavailable, resp)
		}
		var wg sync.WaitGroup
		wg.Add(n)
		rs := make([]*Response, len(raws))
		for i, r := range raws {
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
