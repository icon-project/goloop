package contract

import (
	"github.com/icon-project/goloop/common/errors"
)

const (
	InvalidContractError = iota + errors.CodeService + 200
	PreparingContractError
)
