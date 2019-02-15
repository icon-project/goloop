package service

import "github.com/pkg/errors"

var (
	ErrDuplicateTransaction    = errors.New("DuplicateTransaction")
	ErrTransactionPoolOverFlow = errors.New("TransactionPoolOverFlow")
	ErrExpiredTransaction      = errors.New("ExpiredTransaction")
)
