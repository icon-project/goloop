package jsonrpc

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type ErrorCode int

func (c ErrorCode) Wrap(err error, debug bool) *Error {
	return NewError(c, err, debug)
}

func (c ErrorCode) New(msg string) *Error {
	return &Error{
		Code:    c,
		Message: msg,
	}
}

func (c ErrorCode) NewWithData(msg string, data interface{}) *Error {
	return &Error{
		Code:    c,
		Message: msg,
		Data:    data,
	}
}

func (c ErrorCode) Errorf(f string, args ...interface{}) *Error {
	return &Error{
		Code:    c,
		Message: fmt.Sprintf(f, args...),
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
	return fmt.Sprintf("jsonrpc: code: %d, message: %s, data: %+v", e.Code, e.Message, e.Data)
}

func ErrParse(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeJsonParse,
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

func AttachDebug(je *Error, err error) {
	je.Data = fmt.Sprintf("%+v", err)
}

func NewError(code ErrorCode, err error, debug bool) *Error {
	re := &Error{
		Code:    code,
		Message: fmt.Sprintf("%s", err),
	}
	if debug {
		AttachDebug(re, err)
	}
	return re
}

func ErrScore(err error, debug bool) *Error {
	s, _ := scoreresult.StatusOf(err)
	return NewError(ErrorCodeScore-ErrorCode(s), err, debug)
}

func ErrScoreWithStatus(s module.Status) *Error {
	return &Error{
		Code:    ErrorCodeScore - ErrorCode(s),
		Message: s.String(),
	}
}

func ErrServer(message ...interface{}) *Error {
	re := &Error{
		Code:    ErrorCodeServer,
		Message: fmt.Sprint(message...),
	}
	return re
}

func ErrorHandler(re *Error, c echo.Context) {
	var res *ErrorResponse
	status := 0

	if re.Code == ErrorCodeJsonParse {
		res = &ErrorResponse{
			Version: Version,
			Error:   re,
		}
		status = http.StatusBadRequest
	} else {
		req := c.Get("request").(*Request)
		res = &ErrorResponse{
			ID:      req.ID,
			Version: Version,
			Error:   re,
		}
		switch re.Code {
		case ErrorCodeInvalidRequest, ErrorCodeInvalidParams:
			status = http.StatusBadRequest
		case ErrorCodeMethodNotFound, ErrorCodeNotFound:
			status = http.StatusNotFound
		case ErrorCodeServer, ErrorCodeInternal:
			status = http.StatusInternalServerError
		default:
			switch {
			case re.Code <= ErrorCodeScore && re.Code > ErrorCode(ErrorCodeScore-100):
				// status = http.StatusOK
				status = http.StatusInternalServerError
			default:
				status = http.StatusInternalServerError
			}
		}
	}

	// Send response
	if !c.Response().Committed {
		err := c.JSON(status, res)
		if err != nil {
			c.Logger().Error(err)
		}
	}
}
