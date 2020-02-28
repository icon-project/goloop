package service

import "github.com/icon-project/goloop/common/errors"

const (
	DuplicateTransactionError errors.Code = iota + errors.CodeService
	TransactionPoolOverflowError
	ExpiredTransactionError
	FutureTransactionError
	TransitionInterruptedError
	InvalidTransactionError
	InvalidQueryError
	InvalidResultError
	NoActiveContractError
	NotContractAddressError
	InvalidPatchDataError
	CommittedTransactionError
)

var (
	ErrDuplicateTransaction    = errors.NewBase(DuplicateTransactionError, "DuplicateTransaction")
	ErrTransactionPoolOverFlow = errors.NewBase(TransactionPoolOverflowError, "TransactionPoolOverFlow")
	ErrExpiredTransaction      = errors.NewBase(ExpiredTransactionError, "ExpiredTransaction")
	ErrTransitionInterrupted   = errors.NewBase(TransitionInterruptedError, "TransitionInterrupted")
	ErrInvalidTransaction      = errors.NewBase(InvalidTransactionError, "InvalidTransaction")
	ErrCommittedTransaction    = errors.NewBase(CommittedTransactionError, "CommittedTransaction")
)
