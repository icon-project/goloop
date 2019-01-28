package scoreresult

import (
	"fmt"

	"github.com/icon-project/goloop/module"
)

type baseError struct {
	status module.Status
	msg    string
}

func (e *baseError) Error() string {
	return e.msg
}

func (e *baseError) Status() module.Status {
	return e.status
}

func NewError(s module.Status, m string) error {
	return &baseError{s, m}
}

func NewDefaultError(s module.Status) error {
	return &baseError{s, string(s)}
}

func Errorf(s module.Status, m string, args ...interface{}) error {
	return &baseError{s, fmt.Sprintf(m, args...)}
}

func StatusAndMessageForError(s module.Status, e error) (module.Status, string) {
	if be, ok := e.(*baseError); ok {
		return be.Status(), be.Error()
	}
	return s, e.Error()
}

func Error(e error, s module.Status) error {
	if e == nil {
		return nil
	}
	if _, ok := e.(*baseError); ok {
		return e
	}
	return NewError(s, e.Error())
}

var (
	ErrSystemError            = NewDefaultError(module.StatusSystemError)
	ErrContractNotFound       = NewDefaultError(module.StatusContractNotFound)
	ErrMethodNotFound         = NewDefaultError(module.StatusMethodNotFound)
	ErrMethodNotPayable       = NewDefaultError(module.StatusMethodNotPayable)
	ErrIllegalFormat          = NewDefaultError(module.StatusIllegalFormat)
	ErrInvalidParameter       = NewDefaultError(module.StatusInvalidParameter)
	ErrInvalidInstance        = NewDefaultError(module.StatusInvalidInstance)
	ErrInvalidContainerAccess = NewDefaultError(module.StatusInvalidContainerAccess)
	ErrAccessDenied           = NewDefaultError(module.StatusAccessDenied)
	ErrOutOfStep              = NewDefaultError(module.StatusOutOfStep)
	ErrOutOfBalance           = NewDefaultError(module.StatusOutOfBalance)
	ErrTimeout                = NewDefaultError(module.StatusTimeout)
	ErrStackOverflow          = NewDefaultError(module.StatusStackOverflow)
	ErrUser                   = NewDefaultError(module.StatusUser)
)
