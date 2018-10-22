package common

import "errors"

var (
	ErrorUnknown         = errors.New("Unknown error")
	ErrorIllegalArgument = errors.New("Illegal argument")
	ErrorInvalidState    = errors.New("Invalid state")
	ErrorUnsupported     = errors.New("Unsupported")
)
