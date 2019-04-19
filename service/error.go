package service

import "github.com/icon-project/goloop/common/errors"

const (
	DuplicateTransactionError errors.Code = iota + errors.CodeService
	TransactionPoolOverflowError
	ExpiredTransactionError
	TransitionInterruptedError
	IllegalTransactionTypeError
	InvalidTransactionError
)

var (
	ErrDuplicateTransaction    = errors.NewBase(DuplicateTransactionError, "DuplicateTransaction")
	ErrTransactionPoolOverFlow = errors.NewBase(TransactionPoolOverflowError, "TransactionPoolOverFlow")
	ErrExpiredTransaction      = errors.NewBase(ExpiredTransactionError, "ExpiredTransaction")
	ErrTransitionInterrupted   = errors.NewBase(TransitionInterruptedError, "TransitionInterrupted")
	ErrIllegalTransactionType  = errors.NewBase(IllegalTransactionTypeError, "IllegalTransactionType")
	ErrInvalidTransaction      = errors.NewBase(InvalidTransactionError, "InvalidTransaction")
)
