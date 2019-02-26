package contract

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

type Context interface {
	state.WorldContext
	ContractManager() ContractManager
	EEManager() eeproxy.Manager
	GetPreInstalledScore(id string) []byte
}

type context struct {
	state.WorldContext
	chain module.Chain
	cm    ContractManager
	eem   eeproxy.Manager
}

func NewContext(wc state.WorldContext, cm ContractManager, eem eeproxy.Manager, chain module.Chain) *context {
	return &context{WorldContext: wc, cm: cm, eem: eem, chain: chain}
}
func (c *context) ContractManager() ContractManager {
	return c.cm
}

func (c *context) EEManager() eeproxy.Manager {
	return c.eem
}

func (c *context) GetPreInstalledScore(id string) []byte {
	return c.chain.GetPreInstalledScore(id)
}
