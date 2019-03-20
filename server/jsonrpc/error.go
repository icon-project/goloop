package jsonrpc

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

type ErrorCode int

const (
	ErrorCodeParse          ErrorCode = -32700
	ErrorCodeInvalidRequest ErrorCode = -32600
	ErrorCodeMethodNotFound ErrorCode = -32601
	ErrorCodeInvalidParams  ErrorCode = -32602
	ErrorCodeInternal       ErrorCode = -32603
	ErrorCodeServer         ErrorCode = -32000
	ErrorCodeScore          ErrorCode = -32100
)

type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc: code: %d, message: %s, data: %+v", e.Code, e.Message, e.Data)
}

func ErrParse(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeParse,
		Message: "Parse error",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrInvalidRequest(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeInvalidRequest,
		Message: "Invalid Request",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrMethodNotFound(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeMethodNotFound,
		Message: "Method not found",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrInvalidParams(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeInvalidParams,
		Message: "Invalid params",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrInternal(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeInternal,
		Message: "Internal error",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrServer(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeServer,
		Message: "Server error",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrScore(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeScore,
		Message: "Score error",
	}
	if len(message) > 0 {
		re.Data = message[0]
	}
	return re
}

func ErrorHandler(err error, c echo.Context) {
	re, ok := err.(*Error)
	if !ok {
		// if err is not jsonrpc.Error, delegate to DefaultHTTPErrorHandler
		c.Echo().DefaultHTTPErrorHandler(err, c)
		return
	}

	req := c.Get("request").(*Request)
	res := &Response{
		ID:      req.ID,
		Version: Version,
		Error:   re,
	}

	status := 0
	switch re.Code {
	case ErrorCodeParse, ErrorCodeInvalidRequest, ErrorCodeInvalidParams:
		status = http.StatusBadRequest
	case ErrorCodeMethodNotFound:
		status = http.StatusNotFound
	case ErrorCodeScore:
		status = http.StatusOK
	case ErrorCodeServer, ErrorCodeInternal:
		status = http.StatusInternalServerError
	default:
		status = http.StatusInternalServerError
	}

	// Send response
	if !c.Response().Committed {
		err = c.JSON(status, res)
		if err != nil {
			c.Logger().Error(err)
		}
	}
}
