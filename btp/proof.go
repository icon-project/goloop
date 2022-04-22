package btp

import (
	"github.com/icon-project/goloop/module"
)

func NewProofContextFromBytes(netType string, bs []byte) (module.BTPProofContext, error) {
	return nil, nil
}

func NewProofContext(netType string, pubKeys [][]byte) (module.BTPProofContext, error) {
	return nil, nil
}

func NewProofContextsMap(result []byte) ProofContextMap {
	return nil
}

type ProofContextMap interface {
	ProofContextFor(ntid int64) module.BTPProofContext
	Update(btpSection module.BTPSection)
}
