package contract

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoredb"
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
	AddSyncRequest(id db.BucketID, key []byte)
	Logger() log.Logger
	PatchDecoder() module.PatchDecoder
	TraceInfo() *module.TraceInfo
	ChainID() int
	GetProperty(name string) interface{}
	SetProperty(name string, value interface{})
	GetEnabledEETypes() state.EETypes
	EEPriority() eeproxy.RequestPriority
}

type context struct {
	state.WorldContext
	chain module.Chain
	cm    ContractManager
	eem   eeproxy.Manager
	eep   eeproxy.RequestPriority
	log   log.Logger
	ti    *module.TraceInfo
	props map[string]interface{}
}

func NewContext(
	wc state.WorldContext,
	cm ContractManager,
	eem eeproxy.Manager,
	chain module.Chain,
	log log.Logger,
	ti *module.TraceInfo,
	eep eeproxy.RequestPriority,
) *context {
	if ti != nil {
		eep = eeproxy.ForQuery
	}
	return &context{
		WorldContext: wc,
		cm:           cm,
		eem:          eem,
		chain:        chain,
		log:          log,
		ti:           ti,
		eep:          eep,
		props:        make(map[string]interface{}),
	}
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

func (c *context) AddSyncRequest(id db.BucketID, key []byte) {
	err := c.chain.ServiceManager().AddSyncRequest(id, key)
	if err != nil {
		c.log.Warnf("FAIL to add sync request id=%q key=%#x err=%+v",
			id, key, err)
	}
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

func (c *context) GetEnabledEETypes() state.EETypes {
	as := c.GetAccountState(state.SystemID)
	s := scoredb.NewVarDB(as, state.VarEnabledEETypes).String()
	if len(s) > 0 {
		if ets, err := state.ParseEETypes(s); err == nil {
			return ets
		}
	}
	return c.cm.DefaultEnabledEETypes()
}

func (c *context) EEPriority() eeproxy.RequestPriority {
	return c.eep
}
