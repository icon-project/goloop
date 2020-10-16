package block

import "github.com/icon-project/goloop/common/errors"

const (
	ResultNotFinalizedError = errors.CodeBlock + iota
)

var (
	ErrResultNotFinalized = errors.NewBase(ResultNotFinalizedError, "ResultNotFinalized")
)
