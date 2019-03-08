package state

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/pkg/errors"
)

const (
	VarStepPrice  = "step_price"
	VarStepCosts  = "step_costs"
	VarStepTypes  = "step_types"
	VarTreasury   = "treasury"
	VarGovernance = "governance"

	VarStepLimitTypes = "step_limit_types"
	VarStepLimit      = "step_limit"
	VarSysConfig      = "system_config"
	VarRevision       = "system_revision"
	VarMembers        = "members"
	VarDeployer       = "system_deployer"
)

const (
	SysConfigFee = 1 << iota
	SysConfigAudit
	SysConfigDeployerWhiteList
	SysConfigScorePackageValidator
	SysConfigMembership
)

const (
	InfoBlockTimestamp = "B.timestamp"
	InfoBlockHeight    = "B.height"
	InfoTxHash         = "T.hash"
	InfoTxIndex        = "T.index"
	InfoTxTimestamp    = "T.timestamp"
	InfoTxNonce        = "T.nonce"
	InfoTxFrom         = "T.from"
	InfoStepCosts      = "StepCosts"
	InfoContractOwner  = "C.owner"
)

const (
	SystemIDStr = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

var (
	SystemID = []byte(SystemIDStr)
)

type WorldContext interface {
	WorldState
	StepsFor(t StepType, n int) int64
	StepPrice() *big.Int
	BlockTimeStamp() int64
	GetStepLimit(t string) *big.Int
	BlockHeight() int64
	Treasury() module.Address
	Governance() module.Address
	GetInfo() map[string]interface{}
	WorldStateChanged(ws WorldState) WorldContext
	WorldVirtualState() WorldVirtualState
	GetFuture(lq []LockRequest) WorldContext
	SetTransactionInfo(ti *TransactionInfo)
	GetTransactionInfo(ti *TransactionInfo)
	SetContractInfo(si *ContractInfo)

	GetMembers() []module.Member
	IsDeployer(addr string) bool
	CfgFeeEnabled() bool
	CfgAuditEnabled() bool
	CfgDeployWhiteListEnabled() bool
	CfgPackageValidatorEnabled() bool
	CfgMembershipEnabled() bool
}

type BlockInfo struct {
	Timestamp int64
	Height    int64
}

type TransactionInfo struct {
	Index     int32
	Hash      []byte
	From      module.Address
	Timestamp int64
	Nonce     *big.Int
}

type ContractInfo struct {
	Owner module.Address
}

type worldContext struct {
	WorldState

	treasury   module.Address
	governance module.Address

	systemInfo systemStorageInfo

	blockInfo    BlockInfo
	txInfo       TransactionInfo
	contractInfo ContractInfo

	info map[string]interface{}
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
			ID:   SystemIDStr,
		}
		return c.WorldStateChanged(wvs.GetFuture(lq2))
	}
}

func (c *worldContext) SetValidators(vl []module.Validator) error {
	if c.CfgMembershipEnabled() {
		as := c.GetAccountState(SystemID)
		members := scoredb.NewArrayDB(as, VarMembers)
		size := members.Size()
		mm := make(map[module.Address]bool)
		for i := 0; i < size; i++ {
			mm[members.Get(i).Address()] = true
		}

		for _, v := range vl {
			if _, present := mm[v.Address()]; !present {
				return errors.New("validator(" + v.Address().String() + ") is not a member")
			}
		}
	}

	return c.WorldState.SetValidators(vl)
}

func (c *worldContext) GrantValidator(v module.Validator) error {
	if c.CfgMembershipEnabled() {
		as := c.GetAccountState(SystemID)
		members := scoredb.NewArrayDB(as, VarMembers)
		size := members.Size()
		found := false
		for i := 0; i < size; i++ {
			if members.Get(i).Address().Equal(v.Address()) {
				found = true
				break
			}
		}
		if !found {
			return errors.New("Should be a member before granting validator")
		}
	}
	return c.WorldState.GrantValidator(v)
}

// TODO What if some values such as deployer don't use cache here and are resolved on demand.
type systemStorageInfo struct {
	updated      bool
	ass          AccountSnapshot
	stepPrice    *big.Int
	stepCosts    map[string]int64
	stepLimit    map[string]int64
	sysConfig    int64
	stepCostInfo *codec.TypedObj
	deployer     map[string]bool
}

func (c *worldContext) updateSystemInfo() {
	if !c.systemInfo.updated {
		ass := c.GetAccountSnapshot(SystemID)
		if c.systemInfo.ass == nil || ass.StorageChangedAfter(c.systemInfo.ass) {
			c.systemInfo.ass = ass

			as := newAccountROState(ass)

			stepPrice := scoredb.NewVarDB(as, VarStepPrice).BigInt()
			c.systemInfo.stepPrice = stepPrice

			stepCosts := make(map[string]int64)
			stepTypes := scoredb.NewArrayDB(as, VarStepTypes)
			stepCostDB := scoredb.NewDictDB(as, VarStepCosts, 1)
			tcount := stepTypes.Size()
			for i := 0; i < tcount; i++ {
				tname := stepTypes.Get(i).String()
				stepCosts[tname] = stepCostDB.Get(tname).Int64()
			}
			c.systemInfo.stepCosts = stepCosts
			c.systemInfo.stepCostInfo = nil

			stepLimit := make(map[string]int64)
			stepLimitTypes := scoredb.NewArrayDB(as, VarStepLimitTypes)
			stepLimitDB := scoredb.NewDictDB(as, VarStepLimit, 1)
			tcount = stepLimitTypes.Size()
			for i := 0; i < tcount; i++ {
				tname := stepLimitTypes.Get(i).String()
				stepLimit[tname] = stepLimitDB.Get(tname).Int64()
			}
			c.systemInfo.stepLimit = stepLimit

			c.systemInfo.sysConfig = scoredb.NewVarDB(as, VarSysConfig).Int64()
			db := scoredb.NewArrayDB(as, VarDeployer)
			if db.Size() > 0 {
				c.systemInfo.deployer = make(map[string]bool)
				for i := 0; i < db.Size(); i++ {
					addr := db.Get(i).Address().String()
					c.systemInfo.deployer[addr] = true
				}
			}
		}
		c.systemInfo.updated = true
	}
}

func (c *worldContext) StepsFor(t StepType, n int) int64 {
	c.updateSystemInfo()
	if v, ok := c.systemInfo.stepCosts[string(t)]; ok {
		return v * int64(n)
	} else {
		return 0
	}
}

func (c *worldContext) StepPrice() *big.Int {
	c.updateSystemInfo()
	return c.systemInfo.stepPrice
}

func (c *worldContext) GetStepLimit(t string) *big.Int {
	c.updateSystemInfo()
	if v, ok := c.systemInfo.stepLimit[t]; ok {
		return big.NewInt(v)
	} else {
		return big.NewInt(0)
	}
}

func (c *worldContext) CfgFeeEnabled() bool {
	c.updateSystemInfo()
	if c.systemInfo.sysConfig&SysConfigFee == 0 {
		return false
	}
	return true
}

func (c *worldContext) CfgAuditEnabled() bool {
	c.updateSystemInfo()
	if c.systemInfo.sysConfig&SysConfigAudit == 0 {
		return false
	}
	return true
}

func (c *worldContext) CfgDeployWhiteListEnabled() bool {
	c.updateSystemInfo()
	if c.systemInfo.sysConfig&SysConfigDeployerWhiteList == 0 {
		return false
	}
	return true
}

func (c *worldContext) CfgPackageValidatorEnabled() bool {
	c.updateSystemInfo()
	if c.systemInfo.sysConfig&SysConfigScorePackageValidator == 0 {
		return false
	}
	return true
}

func (c *worldContext) CfgMembershipEnabled() bool {
	c.updateSystemInfo()
	if c.systemInfo.sysConfig&SysConfigMembership == 0 {
		return false
	}
	return true
}

func (c *worldContext) GetMembers() []module.Member {
	as := c.GetAccountState(SystemID)
	members := scoredb.NewArrayDB(as, VarMembers)
	size := members.Size()
	ml := make([]module.Member, size)
	for i := 0; i < size; i++ {
		ml[i] = members.Get(i).Address().(module.Member)
	}
	return ml
}

func (c *worldContext) IsDeployer(addr string) bool {
	if c.systemInfo.deployer == nil {
		return false
	}
	_, ok := c.systemInfo.deployer[addr]
	return ok
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

func (c *worldContext) WorldStateChanged(ws WorldState) WorldContext {
	wc := &worldContext{
		WorldState: ws,
		treasury:   c.treasury,
		governance: c.governance,
		systemInfo: c.systemInfo,
		blockInfo:  c.blockInfo,
	}
	wc.systemInfo.updated = false
	return wc
}

func (c *worldContext) SetTransactionInfo(ti *TransactionInfo) {
	c.txInfo = *ti
	c.info = nil
}

func (c *worldContext) GetTransactionInfo(ti *TransactionInfo) {
	*ti = c.txInfo
}

func (c *worldContext) SetContractInfo(si *ContractInfo) {
	c.contractInfo = *si
	c.info = nil
}

func (c *worldContext) stepCostInfo() interface{} {
	c.updateSystemInfo()
	if c.systemInfo.stepCostInfo == nil {
		c.systemInfo.stepCostInfo = common.MustEncodeAny(c.systemInfo.stepCosts)
	}
	return c.systemInfo.stepCostInfo
}

func (c *worldContext) GetInfo() map[string]interface{} {
	if c.info == nil {
		m := make(map[string]interface{})
		m[InfoBlockHeight] = c.blockInfo.Height
		m[InfoBlockTimestamp] = c.blockInfo.Timestamp
		m[InfoTxHash] = c.txInfo.Hash
		m[InfoTxIndex] = c.txInfo.Index
		m[InfoTxTimestamp] = c.txInfo.Timestamp
		m[InfoTxNonce] = c.txInfo.Nonce
		m[InfoTxFrom] = c.txInfo.From
		m[InfoStepCosts] = c.stepCostInfo()
		m[InfoContractOwner] = c.contractInfo.Owner
		c.info = m
	}
	return c.info
}

func NewWorldContext(ws WorldState, bi module.BlockInfo) WorldContext {
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
		blockInfo:  BlockInfo{Timestamp: bi.Timestamp(), Height: bi.Height()},
	}
}
