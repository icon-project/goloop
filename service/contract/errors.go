package contract

import (
	"github.com/icon-project/goloop/common/errors"
)

const (
	PreparingContractError = iota + errors.CodeService + 200
	NoAvailableProxy
)
