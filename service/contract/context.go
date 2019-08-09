package contract

import (
	"encoding/hex"
	"strings"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

type Context interface {
	state.WorldContext
	ContractManager() ContractManager
	EEManager() eeproxy.Manager
	GetPreInstalledScore(id string) ([]byte, error)
	Logger() log.Logger
	PatchDecoder() module.PatchDecoder
}

type context struct {
	state.WorldContext
	chain module.Chain
	cm    ContractManager
	eem   eeproxy.Manager
	log   log.Logger
}

func NewContext(wc state.WorldContext, cm ContractManager, eem eeproxy.Manager, chain module.Chain, log log.Logger) *context {
	return &context{WorldContext: wc, cm: cm, eem: eem, chain: chain, log: log}
}
func (c *context) ContractManager() ContractManager {
	return c.cm
}

func (c *context) EEManager() eeproxy.Manager {
	return c.eem
}

func (c *context) PatchDecoder() module.PatchDecoder {
	return c.chain.PatchDecoder()
}

func (c *context) GetPreInstalledScore(id string) ([]byte, error) {
	if strings.HasPrefix(id, "0x") == true {
		id = strings.TrimPrefix(id, "0x")
	}
	hash, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}
	return c.chain.GetGenesisData(hash)
}

func (c *context) Logger() log.Logger {
	return c.log
}
