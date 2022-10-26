/*
 * Copyright 2022 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package errors is replacement package for standard errors package.
// It attaches error code and stack information to the error object.
package errors

import (
	serrors "errors"
	"fmt"
	"io"
	"reflect"
	"runtime"

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
	return CodeOf(e) == c
}

// New makes an error including a stack without any code
// If you want to make base error without stack
type stack []uintptr

func (s *stack) StackTrace() errors.StackTrace {
	f := make([]errors.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = errors.Frame((*s)[i])
	}
	return f
}

func callers(skip int) *stack {
	const depth = 16
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

type withStack struct {
	error
	*stack
}

func (e *withStack) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func (e *withStack) Unwrap() error {
	return Unwrap(e.error)
}

func New(msg string) error {
	return &withStack{
		error: serrors.New(msg),
		stack: callers(3),
	}
}

func Errorf(f string, args ...interface{}) error {
	return &withStack{
		error: fmt.Errorf(f, args...),
		stack: callers(3),
	}
}

type wrappedWithStack struct {
	wrapped
	*stack
}

func (e *wrappedWithStack) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func WithStack(e error) error {
	if e == nil {
		return nil
	}
	return &wrappedWithStack{
		wrapped: wrap(e),
		stack:   callers(3),
	}
}

/*------------------------------------------------------------------------------
Base error only with message and code

For general usage, you may return this directly.
*/

func formatError(e error, f fmt.State, c rune) {
	for {
		e = formatErrorOne(e, f, c)
		if e == nil {
			return
		}
		io.WriteString(f, "\nWrapping ")
	}
}

func formatErrorOne(e error, f fmt.State, c rune) error {
	switch c {
	case 's':
		io.WriteString(f, e.Error())
	case 'q':
		fmt.Fprintf(f, "%q", e.Error())
	case 'v':
		if coder, ok := CoderOf(e); ok {
			fmt.Fprintf(f, "E%04d:%s", coder.ErrorCode(), e.Error())
		} else {
			io.WriteString(f, e.Error())
		}
		if f.Flag('+') {
			if tracer, ok := StackTracerOf(e); ok {
				tracer.StackTrace().Format(f, c)
				return Unwrap(tracer.(ErrorWithStack))
			} else {
				return Unwrap(e)
			}
		}
	}
	return nil
}

type code int

func (c code) ErrorCode() Code {
	return Code(c)
}

type message string

func (e message) Error() string {
	return string(e)
}

type codedError struct {
	code
	message
}

func (e *codedError) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func NewBase(c Code, msg string) *codedError {
	return &codedError{code(c), message(msg)}
}

/*------------------------------------------------------------------------------
Coded error object
*/

type withCodeAndStack struct {
	error
	code
	*stack
}

func (e *withCodeAndStack) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func (e *withCodeAndStack) Unwrap() error {
	return Unwrap(e.error)
}

func Errorc(c Code, msg string) error {
	return &withCodeAndStack{
		error: serrors.New(msg),
		code:  code(c),
		stack: callers(3),
	}
}

func Errorcf(c Code, f string, args ...interface{}) error {
	return &withCodeAndStack{
		error: fmt.Errorf(f, args...),
		code:  code(c),
		stack: callers(3),
	}
}

type wrappedWithCode struct {
	wrapped
	code
}

func (e *wrappedWithCode) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func WithCode(err error, c Code) error {
	if err == nil {
		return nil
	}
	return &wrappedWithCode{
		wrapped: wrap(err),
		code:    code(c),
	}
}

type wrapped struct {
	error
}

func (e *wrapped) Unwrap() error {
	return e.error
}

func wrap(e error) wrapped {
	return wrapped{e}
}

type wrappedWithMessage struct {
	wrapped
	message
	*stack
}

func (e *wrappedWithMessage) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func Wrap(e error, msg string) error {
	if e == nil {
		return nil
	}
	return &wrappedWithMessage{
		wrapped: wrap(e),
		message: message(msg),
		stack:   callers(3),
	}
}

func Wrapf(e error, f string, args ...interface{}) error {
	if e == nil {
		return nil
	}
	return &wrappedWithMessage{
		wrapped: wrap(e),
		message: message(fmt.Sprintf(f, args...)),
		stack:   callers(3),
	}
}

type wrappedWithCodeMessage struct {
	wrapped
	code
	message
	*stack
}

func (e *wrappedWithCodeMessage) Format(f fmt.State, c rune) {
	formatError(e, f, c)
}

func Wrapc(e error, c Code, msg string) error {
	if e == nil {
		return nil
	}
	return &wrappedWithCodeMessage{
		wrapped: wrap(e),
		code:    code(c),
		message: message(msg),
		stack:   callers(3),
	}
}

func Wrapcf(e error, c Code, f string, args ...interface{}) error {
	if e == nil {
		return nil
	}
	return &wrappedWithCodeMessage{
		wrapped: wrap(e),
		code:    code(c),
		message: message(fmt.Sprintf(f, args...)),
		stack:   callers(3),
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

type StackTracer interface {
	StackTrace() errors.StackTrace
}

type ErrorWithStack interface {
	error
	StackTracer
}

func StackTracerOf(e error) (StackTracer, bool) {
	tr := FindCause(e, func(err error) bool {
		_, ok := err.(StackTracer)
		return ok
	})
	if tr != nil {
		return tr.(StackTracer), true
	}
	return nil, false
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

type withFormat struct {
	error
}

func (e *withFormat) Unwrap() error {
	return Unwrap(e.error)
}

func (e *withFormat) Format(f fmt.State, c rune) {
	formatError(e.error, f, c)
}

func (e *withFormat) String() string {
	return e.error.Error()
}

// Error returns wrapped error providing formatter to show error code
// and stacks properly.
func Error(e error) error {
	if e == nil {
		return nil
	}
	switch obj := e.(type) {
	case *withStack, *wrappedWithCodeMessage, *codedError, *withCodeAndStack, *wrappedWithCode, *wrappedWithMessage, *wrappedWithStack, *withFormat:
		return obj
	default:
		return &withFormat{e}
	}
}
