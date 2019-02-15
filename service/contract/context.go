package contract

import (
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

type Context interface {
	state.WorldContext
	ContractManager() ContractManager
	EEManager() eeproxy.Manager
}
type context struct {
	state.WorldContext
	cm  ContractManager
	eem eeproxy.Manager
}

func NewContext(wc state.WorldContext, cm ContractManager, eem eeproxy.Manager) *context {
	return &context{WorldContext: wc, cm: cm, eem: eem}
}
func (c *context) ContractManager() ContractManager {
	return c.cm
}

func (c *context) EEManager() eeproxy.Manager {
	return c.eem
}
