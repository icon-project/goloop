package block

import (
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/module"
)

type blockCandidate struct {
	base.Block
	m *manager
}

func (bc *blockCandidate) Dispose() {
	if bc != nil {
		bc.m.DisposeBlockCandidate(bc)
	}
}

func (bc *blockCandidate) Dup() module.BlockCandidate {
	return bc.m.DupBlockCandidate(bc)
}
