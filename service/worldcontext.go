package service

import (
	"github.com/icon-project/goloop/common/codec"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoredb"
)

const (
	VarStepPrice  = "step_price"
	VarStepCosts  = "step_costs"
	VarStepTypes  = "step_types"
	VarTreasury   = "treasury"
	VarGovernance = "governance"
)

var (
	SystemID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type worldContext struct {
	WorldState

	treasury   module.Address
	governance module.Address

	governanceInfo governanceStorageInfo

	blockInfo BlockInfo
	txInfo    TransactionInfo

	info map[string]interface{}

	cm ContractManager
	em eeproxy.Manager
}

func (c *worldContext) WorldVirtualState() WorldVirtualState {
	if wvs, ok := c.WorldState.(WorldVirtualState); ok {
		return wvs
	}
	return NewWorldVirtualState(c.WorldState, nil)
}

func (c *worldContext) GetFuture(lq []LockRequest) WorldContext {
	wvs := c.WorldVirtualState()
	if len(lq) == 0 {
		return c.WorldStateChanged(wvs)
	} else {
		lq2 := make([]LockRequest, len(lq)+1)
		copy(lq2, lq)
		lq2[len(lq)] = LockRequest{
			Lock: AccountReadLock,
			ID:   string(c.governance.ID()),
		}
		return c.WorldStateChanged(wvs.GetFuture(lq2))
	}
}

type governanceStorageInfo struct {
	updated      bool
	ass          AccountSnapshot
	stepPrice    *big.Int
	stepCosts    map[string]int64
	stepCostInfo *codec.TypedObj
}

func (c *worldContext) updateGovernanceInfo() {
	if !c.governanceInfo.updated {
		ass := c.GetAccountSnapshot(c.governance.ID())
		if c.governanceInfo.ass == nil || ass.StorageChangedAfter(c.governanceInfo.ass) {
			c.governanceInfo.ass = ass

			as := newAccountROState(ass)

			stepPrice := scoredb.NewVarDB(as, VarStepPrice)
			c.governanceInfo.stepPrice = stepPrice.BigInt()

			stepCosts := make(map[string]int64)
			stepTypes := scoredb.NewArrayDB(as, VarStepTypes)
			stepCostDB := scoredb.NewDictDB(as, VarStepCosts, 1)
			tcount := stepTypes.Size()
			for i := 0; i < tcount; i++ {
				tname := stepTypes.Get(i).String()
				stepCosts[tname] = stepCostDB.Get(tname).Int64()
			}
			c.governanceInfo.stepCosts = stepCosts
			c.governanceInfo.stepCostInfo = nil
		}
		c.governanceInfo.updated = true
	}
}

func (c *worldContext) StepsFor(t StepType, n int) int64 {
	c.updateGovernanceInfo()
	if v, ok := c.governanceInfo.stepCosts[string(t)]; ok {
		return v * int64(n)
	} else {
		return 0
	}
}

func (c *worldContext) StepPrice() *big.Int {
	c.updateGovernanceInfo()
	return c.governanceInfo.stepPrice
}

func (c *worldContext) BlockTimeStamp() int64 {
	return c.blockInfo.Timestamp
}

func (c *worldContext) BlockHeight() int64 {
	return c.blockInfo.Height
}

func (c *worldContext) GetBlockInfo(bi *BlockInfo) {
	*bi = c.blockInfo
}

func (c *worldContext) Treasury() module.Address {
	return c.treasury
}

func (c *worldContext) Governance() module.Address {
	return c.governance
}

func (c *worldContext) ContractManager() ContractManager {
	return c.cm
}

func (c *worldContext) EEManager() eeproxy.Manager {
	return c.em
}

func (c *worldContext) WorldStateChanged(ws WorldState) WorldContext {
	wc := &worldContext{
		WorldState:     ws,
		treasury:       c.treasury,
		governance:     c.governance,
		governanceInfo: c.governanceInfo,
		blockInfo:      c.blockInfo,

		cm: c.cm,
		em: c.em,
	}
	wc.governanceInfo.updated = false
	return wc
}

func (c *worldContext) SetTransactionInfo(ti *TransactionInfo) {
	c.txInfo = *ti
	c.info = nil
}

func (c *worldContext) GetTransactionInfo(ti *TransactionInfo) {
	*ti = c.txInfo
}

func (c *worldContext) stepCostInfo() interface{} {
	c.updateGovernanceInfo()
	if c.governanceInfo.stepCostInfo == nil {
		c.governanceInfo.stepCostInfo = common.MustEncodeAny(c.governanceInfo.stepCosts)
	}
	return c.governanceInfo.stepCostInfo
}

func (c *worldContext) GetInfo() map[string]interface{} {
	if c.info == nil {
		m := make(map[string]interface{})
		m["B.height"] = c.blockInfo.Height
		m["B.timestamp"] = c.blockInfo.Timestamp
		m["T.index"] = c.txInfo.Index
		m["T.timestamp"] = c.txInfo.Timestamp
		m["T.nonce"] = c.txInfo.Nonce
		m["StepCosts"] = c.stepCostInfo()
		c.info = m
	}
	return c.info
}

func NewWorldContext(ws WorldState, ts int64, height int64, cm ContractManager,
	em eeproxy.Manager,
) WorldContext {
	var governance, treasury module.Address
	ass := ws.GetAccountSnapshot(SystemID)
	as := newAccountROState(ass)
	if as != nil {
		treasury = scoredb.NewVarDB(as, VarTreasury).Address()
		governance = scoredb.NewVarDB(as, VarGovernance).Address()
	}
	if treasury == nil {
		treasury = common.NewAddressFromString("hx1000000000000000000000000000000000000000")
	}
	if governance == nil {
		governance = common.NewAddressFromString("cx0000000000000000000000000000000000000001")
	}
	return &worldContext{
		WorldState: ws,
		treasury:   treasury,
		governance: governance,
		blockInfo:  BlockInfo{Timestamp: ts, Height: height},

		cm: cm,
		em: em,
	}
}
