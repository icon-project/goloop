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
		return module.StatusUnknownFailure, false
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

func Wrapf(e error, s module.Status, format string, args ...interface{}) error {
	return errors.Wrapcf(e, codeForStatus(s), format, args...)
}

func Wrap(e error, s module.Status, msg string) error {
	return errors.Wrapc(e, codeForStatus(s), msg)
}

func Validate(e error) error {
	if s, ok := StatusOf(e); !ok {
		return WithStatus(e, s)
	}
	return e
}

func IsValid(e error) bool {
	_, ok := StatusOf(e)
	return ok
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

const (
	Success = errors.CodeSCORE + errors.Code(module.StatusSuccess) + iota
	UnknownFailureError
	ContractNotFoundError
	MethodNotFoundError
	MethodNotPayableError
	IllegalFormatError
	InvalidParameterError
	InvalidInstanceError
	InvalidContainerAccessError
	AccessDeniedError
	OutOfStepError
	OutOfBalanceError
	TimeoutError
	StackOverflowError
	SkipTransactionError
	InvalidPackageError
	RevertedError = errors.CodeSCORE + errors.Code(module.StatusReverted)
)

const (
	// InvalidRequestError is used by ICON
	InvalidRequestError = IllegalFormatError
)

var (
	ErrUnknownFailure         = errors.NewBase(UnknownFailureError, "UnknownFailure")
	ErrContractNotFound       = errors.NewBase(ContractNotFoundError, "ContractNotFound")
	ErrMethodNotFound         = errors.NewBase(MethodNotFoundError, "MethodNotFound")
	ErrMethodNotPayable       = errors.NewBase(MethodNotPayableError, "MethodNotPayable")
	ErrIllegalFormat          = errors.NewBase(IllegalFormatError, "IllegalFormat")
	ErrInvalidParameter       = errors.NewBase(InvalidParameterError, "InvalidParameter")
	ErrInvalidInstance        = errors.NewBase(InvalidInstanceError, "InvalidInstance")
	ErrInvalidContainerAccess = errors.NewBase(InvalidContainerAccessError, "InvalidContainerAccess")
	ErrAccessDenied           = errors.NewBase(AccessDeniedError, "AccessDenied")
	ErrOutOfStep              = errors.NewBase(OutOfStepError, "OutOfStep")
	ErrOutOfBalance           = errors.NewBase(OutOfBalanceError, "OutOfBalance")
	ErrTimeout                = errors.NewBase(TimeoutError, "Timeout")
	ErrStackOverflow          = errors.NewBase(StackOverflowError, "StackOverflow")
	ErrSkipTransaction        = errors.NewBase(SkipTransactionError, "SkipTransaction")
	ErrInvalidPackage         = errors.NewBase(InvalidPackageError, "InvalidPackage")
	ErrReverted               = errors.NewBase(RevertedError, "Reverted")
)
