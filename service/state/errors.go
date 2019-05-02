package state

import (
	"github.com/icon-project/goloop/common/errors"
)

var (
	ErrIllegalType        = errors.NewBase(IllegalTypeError, "IllegalType")
	ErrNotContractAccount = errors.NewBase(NotContractAccountError, "NotContractAccount")
	ErrNoActiveContract   = errors.NewBase(NoActiveContractError, "NoActiveContract")
	ErrNotEnoughBalance   = errors.NewBase(NotEnoughBalanceError, "NotEnoughBalance")
	ErrTimeOut            = errors.NewBase(TimeOutError, "TimeOut")
	ErrFutureTransaction  = errors.NewBase(FutureTransactionError, "FutureTransaction")
	ErrInvalidValueValue  = errors.NewBase(InvalidValueValueError, "InvalidValueValue")
	ErrInvalidFeeValue    = errors.NewBase(InvalidFeeValueError, "InvalidFeeValue")
	ErrInvalidDataValue   = errors.NewBase(InvalidDataValueError, "InvalidDataValue")
	ErrNotEnoughStep      = errors.NewBase(NotEnoughStepError, "NotEnoughStep")
	ErrContractIsRequired = errors.NewBase(ContractIsRequiredError, "ContractIsRequired")
	ErrInvalidHashValue   = errors.NewBase(InvalidHashValueError, "InvalidHashValue")
	ErrNotEOA             = errors.NewBase(NotEOAError, "NotEOA")
	ErrNotContractOwner   = errors.NewBase(NotContractOwnerError, "NotContractOwner")
	ErrBlockedContract    = errors.NewBase(BlockedContractError, "BlockedContract")
	ErrDisabledContract   = errors.NewBase(DisabledContractError, "DisabledContract")
	ErrInvalidMethod      = errors.NewBase(InvalidMethodError, "InvalidMethod")
)

const (
	IllegalTypeError = iota + errors.CodeService + 300
	NotContractAccountError
	NoActiveContractError
	NotEnoughBalanceError
	TimeOutError
	FutureTransactionError
	InvalidValueValueError
	InvalidFeeValueError
	InvalidDataValueError
	NotEnoughStepError
	ContractIsRequiredError
	InvalidHashValueError
	NotEOAError
	NotContractOwnerError
	BlockedContractError
	DisabledContractError
	InvalidMethodError
)
