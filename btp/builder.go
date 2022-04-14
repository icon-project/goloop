package btp

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type Transaction interface {
	BeginTransaction() Transaction
	Commit()
	Discard()
	SendMessage(nid int32, msg []byte)
	SetNextProofContext(ntid int32, pc []byte)
	AddNetwork(ntid int32, nid int32)
	AddNetworkType(netType string, ntid int32, pc []byte)
}

type SectionBuilder interface {
	BeginTransaction() Transaction
	Build() module.BTPSection
}

func NewSectionBuilder(initialState state.AccountSnapshot) SectionBuilder {
	return nil
}
