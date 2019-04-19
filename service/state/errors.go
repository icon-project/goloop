package state

import (
	"github.com/icon-project/goloop/common/errors"
)

var (
	ErrIllegalType = errors.NewBase(IllegalTypeError, "IllegalType")
)

const (
	IllegalTypeError = iota + errors.CodeService + 300
)
