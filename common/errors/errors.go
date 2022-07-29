package errors

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type Code int

const CodeSegment = 1000

const (
	CodeSCORE Code = iota * CodeSegment
	CodeGeneral
	CodeService
	CodeConsensus
	CodeNetwork
	CodeBlock
	CodeServer
	CodeCritical
)

const (
	Success      Code = 0
	UnknownError Code = CodeGeneral + iota
	IllegalArgumentError
	UnsupportedError
	InvalidStateError
	NotFoundError
	InvalidNetworkError
	TimeoutError
	ExecutionFailError
	InterruptedError
)

var (
	ErrUnknown         = NewBase(UnknownError, "UnknownError")
	ErrIllegalArgument = NewBase(IllegalArgumentError, "IllegalArgument")
	ErrInvalidState    = NewBase(InvalidStateError, "InvalidState")
	ErrUnsupported     = NewBase(UnsupportedError, "Unsupported")
	ErrNotFound        = NewBase(NotFoundError, "NotFound")
	ErrInvalidNetwork  = NewBase(InvalidNetworkError, "InvalidNetwork")
	ErrTimeout         = NewBase(TimeoutError, "Timeout")
	ErrExecutionFail   = NewBase(ExecutionFailError, "ExecutionFail")
	ErrInterrupted     = NewBase(InterruptedError, "Interrupted")
)

const (
	CriticalUnknownError = CodeCritical + iota
	CriticalIOError
	CriticalFormatError
	CriticalHashError
	CriticalRerunError
)

func IsCriticalCode(c Code) bool {
	return c >= CodeCritical && c < CodeCritical+CodeSegment
}

func IsCritical(e error) bool {
	return IsCriticalCode(CodeOf(e))
}

func (c Code) New(msg string) error {
	return Errorc(c, msg)
}

func (c Code) Errorf(f string, args ...interface{}) error {
	return Errorcf(c, f, args...)
}

func (c Code) Wrap(e error, msg string) error {
	return Wrapc(e, c, msg)
}

func (c Code) Wrapf(e error, f string, args ...interface{}) error {
	return Wrapcf(e, c, f, args...)
}

func (c Code) AttachTo(e error) error {
	return WithCode(e, c)
}

func (c Code) Equals(e error) bool {
	if e == nil {
		return false
	}
	return CodeOf(e) == c
}

/*------------------------------------------------------------------------------
Simple mapping to github.com/pkg/errors for easy stack print
*/

// New makes an error including a stack without any code
// If you want to make base error without stack
func New(msg string) error {
	return errors.New(msg)
}

func Errorf(f string, args ...interface{}) error {
	return errors.Errorf(f, args...)
}

func WithStack(e error) error {
	return errors.WithStack(e)
}

/*------------------------------------------------------------------------------
Base error only with message and code

For general usage, you may return this directly.
*/

type baseError struct {
	code Code
	msg  string
}

func (e *baseError) Error() string {
	return e.msg
}

func (e *baseError) ErrorCode() Code {
	return e.code
}

func (e *baseError) Format(f fmt.State, c rune) {
	switch c {
	case 'v', 's', 'q':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.msg)
	}
}

func (e *baseError) Equals(err error) bool {
	return CodeOf(err) == e.code
}

func NewBase(code Code, msg string) *baseError {
	return &baseError{code, msg}
}

/*------------------------------------------------------------------------------
Coded error object
*/

type codedError struct {
	code Code
	error
}

func (e *codedError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "E%04d:%+v", e.code, e.error)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.Error())
	}
}

func (e *codedError) ErrorCode() Code {
	return e.code
}

func (e *codedError) Unwrap() error {
	return e.error
}

func Errorc(code Code, msg string) error {
	return &codedError{
		code:  code,
		error: errors.New(msg),
	}
}

func Errorcf(code Code, f string, args ...interface{}) error {
	return &codedError{
		code:  code,
		error: errors.Errorf(f, args...),
	}
}

func WithCode(err error, code Code) error {
	if err == nil {
		return nil
	}
	if _, ok := CoderOf(err); ok {
		return Wrapc(err, code, err.Error())
	}
	return &codedError{
		code:  code,
		error: err,
	}
}

type messageError struct {
	error
	origin error
}

func (e *messageError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%+v", e.error)
			fmt.Fprintf(f, "\nWrapping %+v", e.origin)
			return
		}
		fallthrough
	case 's', 'q':
		fmt.Fprintf(f, "%s", e.error)
	}
}

func (e *messageError) Unwrap() error {
	return e.origin
}

func Wrap(e error, msg string) error {
	return &messageError{
		error:  errors.New(msg),
		origin: e,
	}
}

func Wrapf(e error, f string, args ...interface{}) error {
	return &messageError{
		error:  errors.Errorf(f, args...),
		origin: e,
	}
}

type wrappedError struct {
	error
	code   Code
	origin error
}

func (e *wrappedError) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "E%04d:%+v", e.code, e.error)
			fmt.Fprintf(f, "\nWrapping %+v", e.origin)
			return
		}
		fallthrough
	case 'q', 's':
		fmt.Fprintf(f, "E%04d:%s", e.code, e.error)
	}
}

func (e *wrappedError) Unwrap() error {
	return e.origin
}

func (e *wrappedError) ErrorCode() Code {
	return e.code
}

func Wrapc(e error, c Code, msg string) error {
	return &wrappedError{
		error:  errors.New(msg),
		code:   c,
		origin: e,
	}
}

func Wrapcf(e error, c Code, f string, args ...interface{}) error {
	return &wrappedError{
		error:  errors.Errorf(f, args...),
		code:   c,
		origin: e,
	}
}

type ErrorCoder interface {
	error
	ErrorCode() Code
}

func CoderOf(e error) (ErrorCoder, bool) {
	coder := FindCause(e, func(err error) bool {
		_, ok := err.(ErrorCoder)
		return ok
	})
	if coder != nil {
		return coder.(ErrorCoder), true
	} else {
		return nil, false
	}
}

func CodeOf(e error) Code {
	if e == nil {
		return Success
	}
	if coder, ok := CoderOf(e); ok {
		return coder.ErrorCode()
	}
	return UnknownError
}

// Unwrapper is interface to unwrap the error to get the origin error.
type Unwrapper interface {
	Unwrap() error
}

func Unwrap(err error) error {
	switch obj := err.(type) {
	case interface{ Unwrap() error }:
		return obj.Unwrap()
	case interface{ Cause() error }:
		return obj.Cause()
	default:
		return nil
	}
}

// Is checks whether err is caused by the target.
func Is(err, target error) bool {
	if target == nil {
		return err == target
	}
	isComparable := reflect.TypeOf(target).Comparable()
	for {
		if isComparable && err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		if err = Unwrap(err); err == nil {
			return false
		}
	}
}

func FindCause(err error, cb func(err error) bool) error {
	for {
		if err == nil {
			return nil
		}
		if cb(err) {
			return err
		}
		err = Unwrap(err)
	}
}

func ToString(e error) string {
	if e == nil {
		return ""
	} else {
		return fmt.Sprintf("%v", e)
	}
}
