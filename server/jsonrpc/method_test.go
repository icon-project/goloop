package jsonrpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/icon-project/goloop/server/metric"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func prepare(reqJson string) (echo.Context, *httptest.ResponseRecorder, error) {
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

func invokeTest(t *testing.T, mr *MethodRepository, req, resp string, status int) {
	c, rec, err := prepare(req)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, status, rec.Code)
	if resp != "" {
		assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
		resp += "\n"
	}
	assert.Equal(t, resp, rec.Body.String())
}

func invokeBatchTest(t *testing.T, mr *MethodRepository, req, resp []string) {
	batchReq := "[" + strings.Join(req, ",") + "]"
	c, rec, err := prepare(batchReq)
	assert.NoError(t, err)
	err = mr.Handle(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, echo.MIMEApplicationJSONCharsetUTF8, rec.Header().Get(echo.HeaderContentType))
	batchResp := "[" + strings.Join(resp, ",") + "]" + "\n"
	assert.Equal(t, batchResp, rec.Body.String())
}

func TestMethodRepository(t *testing.T) {
	mtr := metric.NewJsonrpcMetric(metric.DefaultJsonrpcDurationsExpire, metric.DefaultJsonrpcDurationsSize, true)
	mr := NewMethodRepository(mtr)
	mr.RegisterMethod("hello", hello)
	mr.RegisterMethod("noArgs", noArgs)

	helloReq := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001"}`
	helloResp := `{"jsonrpc":"2.0","result":"hello, icon","id":"1001"}`
	invokeTest(t, mr, helloReq, helloResp, http.StatusOK)

	noArgsReq := `{"jsonrpc":"2.0","method":"noArgs","id":"1001"}`
	noArgsResp := `{"jsonrpc":"2.0","result":"noArgs","id":"1001"}`
	invokeTest(t, mr, noArgsReq, noArgsResp, http.StatusOK)

	//unknownFields of request
	unknownFieldsOnly := `{"foo":"bar"}`
	unknownFieldsOnlyResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to unmarshal, unknown field 'foo'"},"id":null}`
	invokeTest(t, mr, unknownFieldsOnly, unknownFieldsOnlyResp, http.StatusBadRequest)
	unknownFields := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001","foo":"bar"}`
	unknownFieldsResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to unmarshal, unknown field 'foo'"},"id":"1001"}`
	invokeTest(t, mr, unknownFields, unknownFieldsResp, http.StatusBadRequest)

	//'jsonrpc' of request
	requiredVersionResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to validate, required('jsonrpc')"},"id":"1001"}`
	omitVersion := `{"method":"noArgs","id":"1001"}`
	invokeTest(t, mr, omitVersion, requiredVersionResp, http.StatusBadRequest)
	nullVersion := `{"jsonrpc":null,"method":"noArgs","id":"1001"}`
	invokeTest(t, mr, nullVersion, requiredVersionResp, http.StatusBadRequest)

	invalidVersion := `{"jsonrpc":"2.1","method":"noArgs","id":"1001"}`
	invalidVersionResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to validate, version('jsonrpc')"},"id":"1001"}`
	invokeTest(t, mr, invalidVersion, invalidVersionResp, http.StatusBadRequest)

	invalidVersionType := `{"jsonrpc":2.0,"method":"noArgs","id":"1001"}`
	invalidVersionTypeResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to unmarshal, 'jsonrpc' must be string type"},"id":"1001"}`
	invokeTest(t, mr, invalidVersionType, invalidVersionTypeResp, http.StatusBadRequest)

	//'method' of request
	requiredMethodResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to validate, required('method')"},"id":"1001"}`
	omitMethod := `{"jsonrpc":"2.0","params":"bar","id":"1001"}`
	invokeTest(t, mr, omitMethod, requiredMethodResp, http.StatusBadRequest)
	nullMethod := `{"jsonrpc":"2.0","method":null,"params":"bar","id":"1001"}`
	invokeTest(t, mr, nullMethod, requiredMethodResp, http.StatusBadRequest)

	methodNotFoundResp := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"MethodNotFound"},"id":"1001"}`
	mustNotFound := `{"jsonrpc":"2.0","method":"mustNotFound","id":"1001"}`
	invokeTest(t, mr, mustNotFound, methodNotFoundResp, http.StatusBadRequest)
	emptyMethod := `{"jsonrpc":"2.0","method":"","params":"bar","id":"1001"}`
	invokeTest(t, mr, emptyMethod, methodNotFoundResp, http.StatusBadRequest)

	invalidMethod := `{"jsonrpc":"2.0","method":"` + strings.Repeat("0", 256) + `","id":"1001"}`
	invokeTest(t, mr, invalidMethod, methodNotFoundResp, http.StatusBadRequest)

	invalidMethodType := `{"jsonrpc":"2.0","method":1,"id":"1001"}`
	invalidMethodTypeResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to unmarshal, 'method' must be string type"},"id":"1001"}`
	invokeTest(t, mr, invalidMethodType, invalidMethodTypeResp, http.StatusBadRequest)

	//'id' of request
	emptyResp := ""
	notification := `{"jsonrpc":"2.0","method":"noArgs"}`
	invokeTest(t, mr, notification, emptyResp, http.StatusOK)
	nullId := `{"jsonrpc":"2.0","method":"noArgs","id":null}`
	invokeTest(t, mr, nullId, emptyResp, http.StatusOK)

	emptyId := `{"jsonrpc":"2.0","method":"noArgs","id":""}`
	emptyIdResp := `{"jsonrpc":"2.0","result":"noArgs","id":""}`
	invokeTest(t, mr, emptyId, emptyIdResp, http.StatusOK)

	numberId := `{"jsonrpc":"2.0","method":"noArgs","id":0}`
	numberIdResp := `{"jsonrpc":"2.0","result":"noArgs","id":0}`
	invokeTest(t, mr, numberId, numberIdResp, http.StatusOK)

	fractionalPartsId := `{"jsonrpc":"2.0","method":"noArgs","id":0.1}`
	fractionalPartsIdResp := `{"jsonrpc":"2.0","result":"noArgs","id":0.1}`
	invokeTest(t, mr, fractionalPartsId, fractionalPartsIdResp, http.StatusOK)

	invalidIdType := `{"jsonrpc":"2.0","method":"noArgs","id":true}`
	invalidIdTypeResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to validate, id('id')"},"id":true}`
	invokeTest(t, mr, invalidIdType, invalidIdTypeResp, http.StatusBadRequest)

	//'params' of request
	requiredParamsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to unmarshal, 'params' of request is required "},"id":"1001"}`
	omitParams := `{"jsonrpc":"2.0","method":"hello","id":"1001"}`
	invokeTest(t, mr, omitParams, requiredParamsResp, http.StatusBadRequest)
	nullParams := `{"jsonrpc":"2.0","method":"hello","params":null,"id":"1001"}`
	invokeTest(t, mr, nullParams, requiredParamsResp, http.StatusBadRequest)

	invalidParamsType := `{"jsonrpc":"2.0","method":"hello","params":0,"id":"1001"}`
	invalidParamsTypeResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to unmarshal, 'params' of request must be object type"},"id":"1001"}`
	invokeTest(t, mr, invalidParamsType, invalidParamsTypeResp, http.StatusBadRequest)

	unknownFieldsInParams := `{"jsonrpc":"2.0","method":"hello","params":{"name":"icon","foo":"bar"},"id":"1001"}`
	unknownFieldsInParamsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to unmarshal, unknown field 'foo'"},"id":"1001"}`
	invokeTest(t, mr, unknownFieldsInParams, unknownFieldsInParamsResp, http.StatusBadRequest)

	emptyParamInParams := `{"jsonrpc":"2.0","method":"hello","params":{"name":""},"id":"1001"}`
	emptyParamInParamsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to validate, required('name')"},"id":"1001"}`
	invokeTest(t, mr, emptyParamInParams, emptyParamInParamsResp, http.StatusBadRequest)

	invalidParamTypeInParams := `{"jsonrpc":"2.0","method":"hello","params":{"name":0},"id":"1001"}`
	invalidParamTypeInParamsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to unmarshal, 'name' must be string type"},"id":"1001"}`
	invokeTest(t, mr, invalidParamTypeInParams, invalidParamTypeInParamsResp, http.StatusBadRequest)

	unknownFieldsInParamsForNoArgs := `{"jsonrpc":"2.0","method":"noArgs","params":{"foo":"bar"},"id":"1001"}`
	unknownFieldsInParamsForNoArgsResp := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"InvalidParams: fail to unmarshal, unknown field 'foo'"},"id":"1001"}`
	invokeTest(t, mr, unknownFieldsInParamsForNoArgs, unknownFieldsInParamsForNoArgsResp, http.StatusBadRequest)

	//batch
	emptyBatch := `[]`
	emptyBatchResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest"},"id":null}`
	invokeTest(t, mr, emptyBatch, emptyBatchResp, http.StatusBadRequest)

	invalidRequestTypeResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest: fail to unmarshal, request must be object type"},"id":null}`
	invalidBatch := []string{`1`, `2`, `3`}
	invalidBatchResp := []string{
		invalidRequestTypeResp,
		invalidRequestTypeResp,
		invalidRequestTypeResp,
	}
	invokeBatchTest(t, mr, invalidBatch, invalidBatchResp)

	mixedBatch := []string{
		`{"jsonrpc":"2.0","method":"hello","params":{"name":"icon"},"id":"1001"}`,
		notification,
		`{"jsonrpc":"2.0","method":"noArgs","id":"1002"}`,
		unknownFieldsOnly,
		`{"jsonrpc":"2.0","method":"mustNotFound","id":"1005"}`,
		`{"jsonrpc":"2.0","method":"hello","params":{"name":"world"},"id":"1009"}`,
	}
	mixedBatchResp := []string{
		`{"jsonrpc":"2.0","result":"hello, icon","id":"1001"}`,
		`{"jsonrpc":"2.0","result":"noArgs","id":"1002"}`,
		unknownFieldsOnlyResp,
		`{"jsonrpc":"2.0","error":{"code":-32601,"message":"MethodNotFound"},"id":"1005"}`,
		`{"jsonrpc":"2.0","result":"hello, world","id":"1009"}`,
	}
	invokeBatchTest(t, mr, mixedBatch, mixedBatchResp)

	exceedLimitBatch := "[" + strings.Repeat(","+notification, DefaultBatchLimit+1)[1:] + "]"
	exceedLimitBatchResp := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"InvalidRequest","data":"too many request"},"id":null}`
	invokeTest(t, mr, exceedLimitBatch, exceedLimitBatchResp, http.StatusServiceUnavailable)

	assert.Panics(t, func() {
		mr.RegisterMethod("panicFunc", func(*Context, *Params) (interface{}, error) {
			panic("panic")
		})
		panicBatch := []string{
			`{"jsonrpc":"2.0","method":"panicFunc","params":0,"id":"1001"}`,
		}
		invokeBatchTest(t, mr, panicBatch, nil)
	})
}

type HelloParam struct {
	Name string `json:"name" validate:"required"`
}

func hello(ctx *Context, params *Params) (interface{}, error) {
	var param HelloParam
	if err := params.Convert(&param); err != nil {
		return nil, ErrorCodeInvalidParams.Wrap(err, false)
	}
	return "hello, " + param.Name, nil
}

func noArgs(ctx *Context, params *Params) (interface{}, error) {
	var param struct{}
	if err := params.Convert(&param); err != nil {
		return nil, ErrorCodeInvalidParams.Wrap(err, false)
	}
	return "noArgs", nil
}
