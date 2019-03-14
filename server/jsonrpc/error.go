package jsonrpc

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

const (
	// ErrorCodeParse is parse error code.
	ErrorCodeParse ErrorCode = -32700
	// ErrorCodeInvalidRequest is invalid request error code.
	ErrorCodeInvalidRequest ErrorCode = -32600
	// ErrorCodeMethodNotFound is method not found error code.
	ErrorCodeMethodNotFound ErrorCode = -32601
	// ErrorCodeInvalidParams is invalid params error code.
	ErrorCodeInvalidParams ErrorCode = -32602
	// ErrorCodeInternal is internal error code.
	ErrorCodeInternal ErrorCode = -32603
)

type ErrorCode int

type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc: code: %d, message: %s, data: %+v", e.Code, e.Message, e.Data)
}

// ErrParse returns parse error.
func ErrParse() *Error {
	return &Error{
		Code:    ErrorCodeParse,
		Message: "Parse error",
	}
}

// ErrInvalidRequest returns invalid request error.
func ErrInvalidRequest() *Error {
	return &Error{
		Code:    ErrorCodeInvalidRequest,
		Message: "Invalid Request",
	}
}

// ErrMethodNotFound returns method not found error.
func ErrMethodNotFound() *Error {
	return &Error{
		Code:    ErrorCodeMethodNotFound,
		Message: "Method not found",
	}
}

// ErrInvalidParams returns invalid params error.
func ErrInvalidParams() *Error {
	return &Error{
		Code:    ErrorCodeInvalidParams,
		Message: "Invalid params",
	}
}

// ErrInternal returns internal error.
func ErrInternal() *Error {
	return &Error{
		Code:    ErrorCodeInternal,
		Message: "Internal error",
	}
}

func ErrorHandler(err error, c echo.Context) {
	re, ok := err.(*Error)
	if !ok {
		re = ErrInternal()
		// if err is not jsonrpc.Error, delegate to DefaultHTTPErrorHandler
		// c.Echo().DefaultHTTPErrorHandler(err, c)
		// return
	}

	// TODO : request is nil
	req := c.Get("request").(*Request)
	res := &Response{
		ID:      req.ID,
		Version: req.Version,
		Error:   re,
	}

	status := 0
	switch re.Code {
	case ErrorCodeParse, ErrorCodeInvalidRequest, ErrorCodeInvalidParams:
		status = http.StatusBadRequest
	case ErrorCodeMethodNotFound:
		status = http.StatusNotFound
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
