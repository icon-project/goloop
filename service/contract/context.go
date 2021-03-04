package contract

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

const (
	PropInitialSnapshot = "transition.initialSnapshot"
)

type Context interface {
	state.WorldContext
	TransactionTimeout() time.Duration
	ContractManager() ContractManager
	EEManager() eeproxy.Manager
	GetPreInstalledScore(id string) ([]byte, error)
	Logger() log.Logger
	PatchDecoder() module.PatchDecoder
	TraceInfo() *module.TraceInfo
	ChainID() int
	GetProperty(name string) interface{}
}

type context struct {
	state.WorldContext
	chain module.Chain
	cm    ContractManager
	eem   eeproxy.Manager
	log   log.Logger
	ti    *module.TraceInfo
	props map[string]interface{}
}

func NewContext(wc state.WorldContext, cm ContractManager, eem eeproxy.Manager, chain module.Chain, log log.Logger, ti *module.TraceInfo) *context {
	return &context{WorldContext: wc, cm: cm, eem: eem, chain: chain, log: log, ti: ti, props: make(map[string]interface{})}
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
	return c.chain.GenesisStorage().Get(hash)
}

func (c *context) Logger() log.Logger {
	return c.log
}

func (c *context) TraceInfo() *module.TraceInfo {
	return c.ti
}

func (c *context) ChainID() int {
	return c.chain.CID()
}

func (c *context) TransactionTimeout() time.Duration {
	return c.chain.TransactionTimeout()
}

func (c *context) SetProperty(name string, value interface{}) {
	c.props[name] = value
}

func (c *context) GetProperty(name string) interface{} {
	return c.props[name]
}
