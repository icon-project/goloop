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
	"github.com/icon-project/goloop/service/trace"
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
	GetTraceLogger(phase module.ExecutionPhase, param interface{}) *trace.Logger
	PatchDecoder() module.PatchDecoder
	TraceInfo() *module.TraceInfo
	ChainID() int
	GetProperty(name string) interface{}
	SetProperty(name string, value interface{})
	GetEnabledEETypes() state.EETypes
}

type context struct {
	state.WorldContext
	chain     module.Chain
	cm        ContractManager
	eem       eeproxy.Manager
	ti        *module.TraceInfo
	tlog      *trace.Logger
	tlogDummy *trace.Logger
	props     map[string]interface{}
}

func NewContext(wc state.WorldContext, cm ContractManager, eem eeproxy.Manager, chain module.Chain, log log.Logger, ti *module.TraceInfo) *context {
	var cb module.TraceCallback
	if ti != nil {
		cb = ti.Callback
	}

	return &context{
		WorldContext: wc,
		cm:           cm,
		eem:          eem,
		chain:        chain,
		ti:           ti,
		tlog:         trace.NewLogger(log, cb),
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
		c.tlog.Warnf("FAIL to add sync request id=%q key=%#x err=%+v",
			id, key, err)
	}
}

func (c *context) Logger() log.Logger {
	return c.tlog.Logger
}

func (c *context) GetTraceLogger(phase module.ExecutionPhase, param interface{}) *trace.Logger {
	ti := c.TraceInfo()
	if ti != nil {
		if ti.Range == module.TraceRangeBlock {
			return c.tlog
		}

		switch phase {
		case module.EPhaseTransaction:
			if ti.Range == module.TraceRangeTransaction {
				if txInfo, ok := param.(*state.TransactionInfo); ok {
					if txInfo.Group == ti.Group && int(txInfo.Index) == ti.Index {
						return c.tlog
					}
				}
			}
		case module.EPhaseExecutionEnd:
			if ti.Range == module.TraceRangeBlockTransaction {
				return c.tlog
			}
		}
	}

	if c.tlogDummy == nil {
		c.tlogDummy = trace.LoggerOf(c.tlog.Logger)
	}
	return c.tlogDummy
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
