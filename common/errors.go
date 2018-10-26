package common

import "errors"

var (
	ErrUnknown         = errors.New("Unknown error")
	ErrIllegalArgument = errors.New("Illegal argument")
	ErrInvalidState    = errors.New("Invalid state")
	ErrUnsupported     = errors.New("Unsupported")
	ErrNotFound        = errors.New("Not found")
)
