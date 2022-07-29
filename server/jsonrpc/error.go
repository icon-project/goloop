package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type ErrorCode int

func (c ErrorCode) New(msg string, data ...interface{}) *Error {
	return &Error{
		Code:    c,
		Message: fmt.Sprintf("%s: %s", c.String(), msg),
		Data:    firstOf(data...),
	}
}

func (c ErrorCode) NewWithData(data interface{}) *Error {
	return &Error{
		Code:    c,
		Message: c.String(),
		Data:    data,
	}
}

func (c ErrorCode) Wrap(err error, debug bool) *Error {
	var data interface{}
	if debug {
		data = fmt.Sprintf("%+v", err)
	}
	return c.New(fmt.Sprintf("%v", err), data)
}

func (c ErrorCode) Errorf(f string, args ...interface{}) *Error {
	return c.New(fmt.Sprintf(f, args...))
}

func (c ErrorCode) String() string {
	switch c {
	case ErrorCodeJsonParse:
		return "ParseError"
	case ErrorCodeInvalidRequest:
		return "InvalidRequest"
	case ErrorCodeMethodNotFound:
		return "MethodNotFound"
	case ErrorCodeInvalidParams:
		return "InvalidParams"
	case ErrorCodeInternal:
		return "InternalError"
	case ErrorCodeServer:
		return "ServerError"
	case ErrorCodeSystem:
		return "SystemError"

	case ErrorCodeTxPoolOverflow:
		return "PoolOverflow"
	case ErrorCodePending:
		return "Pending"
	case ErrorCodeExecuting:
		return "Executing"
	case ErrorCodeNotFound:
		return "NotFound"
	case ErrorLackOfResource:
		return "LackOfResource"
	case ErrorCodeTimeout:
		return "Timeout"
	case ErrorCodeSystemTimeout:
		return "SystemTimeout"
	default:
		switch {
		case c < ErrorCodeServer && c > ErrorCodeServer-1000:
			return fmt.Sprintf("ServerError(%d)", c)
		case c < ErrorCodeSystem && c > ErrorCodeSystem-1000:
			return fmt.Sprintf("SystemError(%d)", c)
		case c <= ErrorCodeScore && c > ErrorCodeScore-1000:
			return fmt.Sprintf("SCOREError(%d)", c)
		}
		return fmt.Sprintf("UnknownError(%d)", c)
	}
}

const (
	ErrorCodeJsonParse      ErrorCode = -32700
	ErrorCodeInvalidRequest ErrorCode = -32600
	ErrorCodeMethodNotFound ErrorCode = -32601
	ErrorCodeInvalidParams  ErrorCode = -32602
	ErrorCodeInternal       ErrorCode = -32603
	ErrorCodeServer         ErrorCode = -32000
	ErrorCodeSystem         ErrorCode = -31000
	ErrorCodeScore          ErrorCode = -30000
)

const (
	ErrorCodeTxPoolOverflow ErrorCode = -31001
	ErrorCodePending        ErrorCode = -31002
	ErrorCodeExecuting      ErrorCode = -31003
	ErrorCodeNotFound       ErrorCode = -31004
	ErrorLackOfResource     ErrorCode = -31005
	ErrorCodeTimeout        ErrorCode = -31006
	ErrorCodeSystemTimeout  ErrorCode = -31007
)

type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string {
	bs, _ := json.Marshal(e.Data)
	return fmt.Sprintf("JSONRPCError(code=%d, message=%q, data=%s)", e.Code, e.Message, bs)
}

func firstOf(message ...interface{}) interface{} {
	if len(message) > 0 {
		return message[0]
	} else {
		return nil
	}
}

func ErrParse(message ...interface{}) *Error {
	return ErrorCodeJsonParse.NewWithData(firstOf(message...))
}

func ErrInvalidRequest(message ...interface{}) *Error {
	return ErrorCodeInvalidRequest.NewWithData(firstOf(message...))
}

func ErrMethodNotFound(message ...interface{}) *Error {
	return ErrorCodeMethodNotFound.NewWithData(firstOf(message...))
}

func ErrInvalidParams(message ...interface{}) *Error {
	return ErrorCodeInvalidParams.NewWithData(firstOf(message...))
}

func ErrScore(err error, debug bool) *Error {
	s, _ := scoreresult.StatusOf(err)
	code := ErrorCodeScore - ErrorCode(s)
	return code.Wrap(err, debug)
}

func ErrScoreWithStatus(s module.Status) *Error {
	code := ErrorCodeScore - ErrorCode(s)
	return code.New(s.String())
}

func ErrorHandler(re *Error, c echo.Context) {
	var res *Response
	status := 0

	res = &Response{
		Version: Version,
		Error:   re,
	}
	status = http.StatusBadRequest

	// Send response
	if !c.Response().Committed {
		err := c.JSON(status, res)
		if err != nil {
			c.Logger().Error(err)
		}
	}
}
