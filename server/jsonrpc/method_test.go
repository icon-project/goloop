package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func prepare(reqJson string) (echo.Context, *httptest.ResponseRecorder, error){
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(reqJson), &raw); err != nil {
		return nil, nil, err
	}

	e := echo.New()
	e.Validator = NewValidator()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqJson))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	//server/server.go:179 srv.IncludeDebug()
	c.Set("includeDebug", false)
	//server/middleware.go:16 JsonRpc()
	c.Set("raw", raw)
	return c, rec, nil
}

func TestMethodRepository(t *testing.T) {
	mr := NewMethodRepository()
	mr.RegisterMethod("hello", hello)

	helloReq := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001"}`
	helloResp := `{"jsonrpc":"2.0","result":"hello, icon","id":"1001"}`
	invalidRequestResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":"1001"}`
	methodNotFoundResp := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"MethodNotFound"},"id":"1001"}`
	invalidParamsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams"},"id":"1001"}`

	//Handle
	c, rec, err := prepare(helloReq)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, helloResp+"\n", rec.Body.String())

	invalidMethodType:= `{"jsonrpc":"2.0","method":1,"params":"bar","id":"1001"}`
	c, rec, err = prepare(invalidMethodType)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, invalidRequestResp+"\n", rec.Body.String())

	unknownFields := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001","foo":"boo"}`
	c, rec, err = prepare(unknownFields)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, invalidRequestResp+"\n", rec.Body.String())

	emitMethod := `{"jsonrpc":"2.0","params":"bar","id":"1001"}`
	c, rec, err = prepare(emitMethod)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, invalidRequestResp+"\n", rec.Body.String())

	emptyMethod := `{"jsonrpc":"2.0","method":"","params":"bar","id":"1001"}`
	c, rec, err = prepare(emptyMethod)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, methodNotFoundResp+"\n", rec.Body.String())

	methodNotFound := `{"jsonrpc":"2.0","method":"mustNotExists","id":"1001"}`
	c, rec, err = prepare(methodNotFound)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, methodNotFoundResp+"\n", rec.Body.String())

	unknownFieldsInParams := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon","unknown":"unknown"},"id":"1001"}`
	c, rec, err = prepare(unknownFieldsInParams)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, invalidParamsResp+"\n", rec.Body.String())

	emptyBatch := `[]`
	emptyBatchResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null}`
	c, rec, err = prepare(emptyBatch)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, emptyBatchResp+"\n", rec.Body.String())

	invalidBatch := `[1,2,3]`
	invalidBatchResp := `[{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null},{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null},{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null}]`
	c, rec, err = prepare(invalidBatch)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, invalidBatchResp+"\n", rec.Body.String())

	mixedBatch := `[{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001"},{"foo":"boo"},{"jsonrpc":"2.0","method":"mustNotExists","id":"1001"}]`
	mixedBatchResp := `[{"jsonrpc":"2.0","result":"hello, icon","id":"1001"},{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null},{"jsonrpc":"2.0","error":{"code":-32601,"message":"MethodNotFound"},"id":"1001"}]`
	c, rec, err = prepare(mixedBatch)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, mixedBatchResp+"\n", rec.Body.String())

	exceedLimitBatch := "["
	exceedLimitBatchResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest","data":"too many request"},"id":null}`
	for i := 0; i <= LimitOfBatch; i++ {
		if i != 0 {
			exceedLimitBatch += ","
		}
		exceedLimitBatch += fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":\"hello\",\"params\":{\"name\":\"icon\"},\"id\":\"%d\"}",
			i+1001)
	}
	exceedLimitBatch += "]"
	c, rec, err = prepare(exceedLimitBatch)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, exceedLimitBatchResp+"\n", rec.Body.String())
}

type HelloParam struct {
	Name string `json:"name"`
}

func hello(ctx *Context, params *Params) (interface{}, error) {
	var param HelloParam
	if err := params.Convert(&param); err != nil {
		return nil, ErrInvalidParams()
	}
	return "hello, " + param.Name, nil
}
