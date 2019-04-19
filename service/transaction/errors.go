package transaction

import (
	"github.com/icon-project/goloop/common/errors"
)

var (
	ErrInvalidSignature = errors.NewBase(InvalidSignatureError, "InvalidSignature")
	ErrInvalidFormat    = errors.NewBase(InvalidFormat, "InvalidFormat")
)

const (
	InvalidGenesisError = iota + errors.CodeService + 100
	InvalidSignatureError
	InvalidVersion
	InvalidTxValue
	InvalidTxTime
	InvalidFormat
)
