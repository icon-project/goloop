package jsonrpc

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/server/metric"
)

type Handler func(ctx *Context, params *Params) (result interface{}, err error)

type MethodRepository struct {
	mtx     sync.RWMutex
	methods map[string]Handler
	allowed map[string]bool
	v       *Validator
	mtr     *metric.JsonrpcMetric
}

func NewMethodRepository(mtr *metric.JsonrpcMetric) *MethodRepository {
	return &MethodRepository{
		methods: make(map[string]Handler),
		allowed: make(map[string]bool),
		v:       NewValidator(),
		mtr:     mtr,
	}
}

func (mr *MethodRepository) Validator() *Validator {
	return mr.v
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
	start := time.Now()
	defer func() {
		method := ""
		if req.Method != nil {
			method = *req.Method
		}
		var err error
		if resp.Error != nil {
			err = resp.Error
		}
		mr.mtr.OnHandle(ctx.MetricContext(), method, start, err)
	}()
	if err := UnmarshalWithValidate(raw, req, mr.v); err != nil {
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
		//Ignore not-allowed notification request
		resp.Error = ErrorCodeInvalidRequest.Wrap(
			errors.Errorf("not allowed notification request"), debug)
		return nil
	}

	p := &Params{
		rawMessage: req.Params,
		validator:  mr.v,
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
			mr.mtr.OnHandle(ctx.MetricContext(), "", time.Now(), resp.Error)
			return c.JSON(http.StatusBadRequest, resp)
		}
		if n > ctx.BatchLimit() {
			resp := &Response{
				Version: Version,
				Error:   ErrInvalidRequest("too many request"),
			}
			mr.mtr.OnHandle(ctx.MetricContext(), "", time.Now(), resp.Error)
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
