package scoreresult

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

func codeForStatus(s module.Status) errors.Code {
	return errors.Code(int(s) + int(errors.CodeSCORE))
}

func statusForCode(c errors.Code) (module.Status, bool) {
	if c <= errors.CodeSCORE || c >= (errors.CodeSCORE+1000) {
		return module.StatusSystemError, false
	} else {
		return module.Status(c - errors.CodeSCORE), true
	}
}

func NewBaseError(s module.Status, msg string) error {
	return errors.NewBase(codeForStatus(s), msg)
}

func NewError(s module.Status, m string) error {
	return errors.Errorc(codeForStatus(s), m)
}

func Errorf(s module.Status, format string, args ...interface{}) error {
	return errors.Errorcf(codeForStatus(s), format, args...)
}

func StatusAndMessageForError(s module.Status, e error) (module.Status, string) {
	if coder, ok := errors.CoderOf(e); ok {
		if status, ok := statusForCode(coder.ErrorCode()); ok {
			return status, coder.Error()
		} else {
			return s, coder.Error()
		}
	} else {
		return s, e.Error()
	}
}

func WithStatus(e error, s module.Status) error {
	return errors.WithCode(e, codeForStatus(s))
}

var (
	ErrSystemError            = NewBaseError(module.StatusSystemError, "StatusSystemError")
	ErrContractNotFound       = NewBaseError(module.StatusContractNotFound, "StatusContractNotFound")
	ErrMethodNotFound         = NewBaseError(module.StatusMethodNotFound, "StatusMethodNotFound")
	ErrMethodNotPayable       = NewBaseError(module.StatusMethodNotPayable, "StatusMethodNotPayable")
	ErrIllegalFormat          = NewBaseError(module.StatusIllegalFormat, "StatusIllegalFormat")
	ErrInvalidParameter       = NewBaseError(module.StatusInvalidParameter, "StatusInvalidParameter")
	ErrInvalidInstance        = NewBaseError(module.StatusInvalidInstance, "StatusInvalidInstance")
	ErrInvalidContainerAccess = NewBaseError(module.StatusInvalidContainerAccess, "StatusInvalidContainerAccess")
	ErrAccessDenied           = NewBaseError(module.StatusAccessDenied, "StatusAccessDenied")
	ErrOutOfStep              = NewBaseError(module.StatusOutOfStep, "StatusOutOfStep")
	ErrOutOfBalance           = NewBaseError(module.StatusOutOfBalance, "StatusOutOfBalance")
	ErrTimeout                = NewBaseError(module.StatusTimeout, "StatusTimeout")
	ErrStackOverflow          = NewBaseError(module.StatusStackOverflow, "StatusStackOverflow")
	ErrUser                   = NewBaseError(module.StatusUser, "StatusUser")
)
