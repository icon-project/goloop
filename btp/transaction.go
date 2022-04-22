package btp

import (
	"github.com/icon-project/goloop/module"
	sstate "github.com/icon-project/goloop/service/state"
)

type operation interface {
	EnableNetworkType(uid string, pubKeys [][]byte) (ntid int64, err error)
	DisableNetworkType(ntid int64) error
	OpenNetwork(ntid int64) (nid int64, err error)
	CloseNetwork(nid int64) error
	SetProofContext(ntid int64, pubKeys [][]byte)
	SendMessage(nid int64, msg []byte) error
}

type ChildTransaction interface {
	operation
	NewChildTransaction() ChildTransaction
	Commit()
	Discard()
}

type Transaction interface {
	operation
	NewChildTransaction() ChildTransaction
	Commit(dst sstate.AccountState) module.BTPSection
}

func NewTransaction(as sstate.AccountSnapshot) Transaction {
	return nil
}
