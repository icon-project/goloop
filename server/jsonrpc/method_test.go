package jsonrpc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodRepository(t *testing.T) {
	mr := NewMethodRepository()
	mr.RegisterMethod("hello", hello)

	message := []byte(`{"id":"1001","jsonrpc":"2.0","method":"hello","params":{"name":"icon"}}`)

	// jsonrpc()
	req := &Request{}
	err := json.Unmarshal(message, req)
	h, err := mr.TakeMethod(req)

	// mr.InvokeMethod()
	ctx := &Context{}
	param := &Params{
		rawMessage: req.Params,
		validator:  NewValidator(),
	}

	result, err := h(ctx, param)

	res := &Response{
		ID:      req.ID,
		Version: req.Version,
		Result:  result,
	}

	// c.JSON()
	js, err := json.Marshal(res)
	if err != nil {
		assert.Error(t, err)
	}

	fmt.Println(string(js))
}

type HelloParam struct {
	Name string `json:"name"`
}

func hello(ctx *Context, params *Params) (result interface{}, err error) {
	var param HelloParam
	if err := params.Convert(&param); err != nil {
		return nil, ErrInvalidParams()
	}
	return "hello, " + param.Name, nil
}
