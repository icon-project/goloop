package consensus

import "github.com/icon-project/goloop/module"

type consensus struct {
	bm module.BlockManager
}

func NewConsensus(manager module.BlockManager) module.Consensus {
	return &consensus{
		bm: manager,
	}
}

func (cs *consensus) Start() {
	// TODO Implement consensus loop
	return
}
