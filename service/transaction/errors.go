package transaction

import (
	"github.com/icon-project/goloop/common/errors"
)

const (
	InvalidGenesisError = iota + errors.CodeService + 100
	InvalidSignatureError
	InvalidVersion
	InvalidTxValue
	InvalidFormat
	NotEnoughStepError
	NotEnoughBalanceError
	ContractNotUsable
	AccessDeniedError
)
