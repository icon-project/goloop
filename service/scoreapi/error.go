package scoreapi

import "github.com/icon-project/goloop/common/errors"

const (
	errorBase        = errors.CodeService + 400
	NoSignatureError = errorBase + iota
	IllegalEventError
)

var (
	ErrNoSignature  = errors.NewBase(NoSignatureError, "NoSignatureError")
	ErrIllegalEvent = errors.NewBase(IllegalEventError, "IllegalEventError")
)
