package service

import "github.com/icon-project/goloop/module"

func NewTransaction(b []byte) (module.Transaction, error) {
	return NewTransactionV3(b)
}
