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

func NewBase(s module.Status, msg string) error {
	return errors.NewBase(codeForStatus(s), msg)
}

func New(s module.Status, m string) error {
	return errors.Errorc(codeForStatus(s), m)
}

func Errorf(s module.Status, format string, args ...interface{}) error {
	return errors.Errorcf(codeForStatus(s), format, args...)
}

func StatusOf(e error) (module.Status, bool) {
	if e == nil {
		return module.StatusSuccess, true
	}
	return statusForCode(errors.CodeOf(e))
}

func WithStatus(e error, s module.Status) error {
	return errors.WithCode(e, codeForStatus(s))
}

var (
	ErrSystemError            = NewBase(module.StatusSystemError, "StatusSystemError")
	ErrContractNotFound       = NewBase(module.StatusContractNotFound, "StatusContractNotFound")
	ErrMethodNotFound         = NewBase(module.StatusMethodNotFound, "StatusMethodNotFound")
	ErrMethodNotPayable       = NewBase(module.StatusMethodNotPayable, "StatusMethodNotPayable")
	ErrIllegalFormat          = NewBase(module.StatusIllegalFormat, "StatusIllegalFormat")
	ErrInvalidParameter       = NewBase(module.StatusInvalidParameter, "StatusInvalidParameter")
	ErrInvalidInstance        = NewBase(module.StatusInvalidInstance, "StatusInvalidInstance")
	ErrInvalidContainerAccess = NewBase(module.StatusInvalidContainerAccess, "StatusInvalidContainerAccess")
	ErrAccessDenied           = NewBase(module.StatusAccessDenied, "StatusAccessDenied")
	ErrOutOfStep              = NewBase(module.StatusOutOfStep, "StatusOutOfStep")
	ErrOutOfBalance           = NewBase(module.StatusOutOfBalance, "StatusOutOfBalance")
	ErrTimeout                = NewBase(module.StatusTimeout, "StatusTimeout")
	ErrStackOverflow          = NewBase(module.StatusStackOverflow, "StatusStackOverflow")
	ErrUser                   = NewBase(module.StatusUser, "StatusUser")
)
