package state

import (
	"github.com/icon-project/goloop/common/errors"
)

var (
	ErrNotContractAddress = errors.NewBase(NotContractAddressError, "NotContractAddress")
	ErrNoActiveContract   = errors.NewBase(NoActiveContractError, "NoActiveContract")
	ErrNotEnoughBalance   = errors.NewBase(NotEnoughBalanceError, "NotEnoughBalance")
	ErrNotEnoughStep      = errors.NewBase(NotEnoughStepError, "NotEnoughStep")
	ErrAccessDenied       = errors.NewBase(AccessDeniedError, "AccessDenied")
)

const (
	NotContractAddressError = iota + errors.CodeService + 300
	NoActiveContractError
	NotEnoughBalanceError
	NotEnoughStepError
	AccessDeniedError
)
