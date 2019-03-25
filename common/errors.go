package common

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

var (
	ErrUnknown         = BaseError(ErrorCodeUnknown, "Unknown error")
	ErrIllegalArgument = BaseError(ErrorCodeIllegalArgument, "Illegal argument")
	ErrInvalidState    = BaseError(ErrorCodeInvalidState, "Invalid state")
	ErrUnsupported     = BaseError(ErrorCodeUnsupported, "Unsupported")
	ErrNotFound        = BaseError(ErrorCodeNotFound, "Not found")
)

type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeIllegalArgument
	ErrorCodeInvalidState
	ErrorCodeUnsupported
	ErrorCodeNotFound
)

func (c ErrorCode) Error(msg ...interface{}) error {
	e := BaseError(c, msg...)
	return errors.WithStack(e)
}

func (c ErrorCode) Errorf(str string, args ...interface{}) error {
	e := BaseErrorf(c, str, args...)
	return errors.WithStack(e)
}

func codePrefix(c ErrorCode) string {
	return fmt.Sprintf("E%04d:", c)
}

type baseError struct {
	code ErrorCode
	msg  string
}

func (e *baseError) ErrorCode() ErrorCode {
	return e.code
}

func (e *baseError) Error() string {
	return e.msg
}

func (e *baseError) WithStack() error {
	return errors.WithStack(e)
}

func BaseError(code ErrorCode, msg ...interface{}) *baseError {
	return &baseError{
		code: code,
		msg:  codePrefix(code) + fmt.Sprint(msg...),
	}
}

func Error(code ErrorCode, msg ...interface{}) error {
	err := BaseError(code, msg...)
	return errors.WithStack(err)
}

func BaseErrorf(code ErrorCode, format string, arg ...interface{}) error {
	err := &baseError{
		code: code,
		msg:  codePrefix(code) + fmt.Sprintf(format, arg...),
	}
	return errors.WithStack(err)
}

func Errorf(code ErrorCode, format string, arg ...interface{}) error {
	err := BaseErrorf(code, format, arg...)
	return errors.WithStack(err)
}

type wrappedError struct {
	code ErrorCode
	error
}

func (e *wrappedError) ErrorCode() ErrorCode {
	return e.code
}

func (e *wrappedError) Cause() error {
	return e.error
}

func (e *wrappedError) Error() string {
	return codePrefix(e.code) + e.error.Error()
}

func (e *wrappedError) Format(f fmt.State, c rune) {
	if formatter, ok := e.error.(fmt.Formatter); ok {
		io.WriteString(f, codePrefix(e.code))
		formatter.Format(f, c)
	} else {
		switch c {
		case 'v', 's', 'q':
			io.WriteString(f, codePrefix(e.code)+e.error.Error())
		}
	}
}

func Wrap(code ErrorCode, err error) error {
	return &wrappedError{
		code:  code,
		error: err,
	}
}

func Cause(err error) error {
	type causer interface {
		Cause() error
	}
	for {
		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else {
			return err
		}
	}
}

func ErrorCodeOf(err error) ErrorCode {
	type errorCoder interface {
		ErrorCode() ErrorCode
	}

	type causer interface {
		Cause() error
	}

	for {
		if base, ok := err.(errorCoder); ok {
			return base.ErrorCode()
		}
		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else {
			return ErrorCodeUnknown
		}
	}
}
