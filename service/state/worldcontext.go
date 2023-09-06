package state

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

const (
	VarStepPrice  = "step_price"
	VarStepCosts  = "step_costs"
	VarStepTypes  = "step_types"
	VarTreasury   = "treasury"
	VarGovernance = "governance"
	VarNetwork    = "network"
	VarChainID    = "chain_id"

	VarStepLimitTypes = "step_limit_types"
	VarStepLimit      = "step_limit"
	VarServiceConfig  = "serviceConfig"
	VarRevision       = "revision"
	VarMembers        = "members"
	VarDeployers      = "deployers"
	VarLicenses       = "licenses"
	VarTotalSupply    = "total_supply"

	VarTimestampThreshold = "timestamp_threshold"
	VarBlockInterval      = "block_interval"
	VarCommitTimeout      = "commit_timeout"
	VarRoundLimitFactor   = "round_limit_factor"
	VarMinimizeBlockGen   = "minimize_block_gen"
	VarTxHashToAddress    = "tx_to_address"
	VarDepositTerm        = "deposit_term"
	VarDepositIssueRate   = "deposit_issue_rate"
	VarNextBlockVersion   = "next_block_version"
	VarEnabledEETypes     = "enabled_ee_types"
	VarSystemDepositUsage = "system_deposit_usage"

	VarDSRContextHistory = "dsr_context_history"
)

const (
	DefaultNID = 1
)

const (
	SysConfigFee = 1 << iota
	SysConfigAudit
	SysConfigDeployerWhiteList
	SysConfigScorePackageValidator
	SysConfigMembership
	SysConfigFeeSharing
)

const (
	InfoBlockTimestamp = "B.timestamp"
	InfoBlockHeight    = "B.height"
	InfoTxHash         = "T.hash"
	InfoTxIndex        = "T.index"
	InfoTxTimestamp    = "T.timestamp"
	InfoTxNonce        = "T.nonce"
	InfoTxFrom         = "T.from"
	InfoRevision       = "Revision"
	InfoStepCosts      = "StepCosts"
	InfoContractOwner  = "C.owner"
)

const (
	SystemIDStr = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

var (
	SystemID      = []byte(SystemIDStr)
	SystemAddress = common.NewContractAddress(SystemID)
	ZeroAddress   = common.NewAccountAddress(SystemID)
)

var (
	defaultDepositIssueRate = big.NewInt(8)
)

type WorldContext interface {
	WorldState
	Revision() module.Revision
	ToRevision(v int) module.Revision
	StepsFor(t StepType, n int) int64
	StepPrice() *big.Int
	BlockTimeStamp() int64
	GetStepLimit(t string) *big.Int
	BlockHeight() int64
	ConsensusInfo() module.ConsensusInfo
	Treasury() module.Address
	Governance() module.Address
	GetInfo() map[string]interface{}
	WorldStateChanged(ws WorldState) WorldContext
	WorldVirtualState() WorldVirtualState
	GetFuture(lq []LockRequest) WorldContext
	SetTransactionInfo(ti *TransactionInfo)
	TransactionInfo() *TransactionInfo
	TransactionID() []byte
	NextTransactionSalt() *big.Int
	SetContractInfo(si *ContractInfo)
	DepositIssueRate() *big.Int
	FeeLimit() *big.Int
	DepositTerm() int64
	UpdateSystemInfo()

	IsDeployer(addr string) bool
	FeeEnabled() bool
	AuditEnabled() bool
	FeeSharingEnabled() bool
	DeployerWhiteListEnabled() bool
	PackageValidatorEnabled() bool
	MembershipEnabled() bool
	TransactionTimestampThreshold() int64

	EnableSkipTransaction()
	SkipTransactionEnabled() bool

	DecodeDoubleSignContext(t string, d[]byte) (module.DoubleSignContext, error)
	DecodeDoubleSignData(t string, d[]byte) (module.DoubleSignData, error)
	GetDoubleSignContextRoot() (module.DoubleSignContextRoot, error)
}

type TransactionInfo struct {
	Group     module.TransactionGroup
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
	virtualState WorldVirtualState

	treasury   module.Address
	governance module.Address

	systemInfo systemStorageInfo

	blockInfo    module.BlockInfo
	csInfo       module.ConsensusInfo
	txInfo       TransactionInfo
	contractInfo ContractInfo

	info map[string]interface{}

	skipTransaction bool

	nextTxSalt *big.Int

	platform Platform

	dsDecoder module.DoubleSignDataDecoder
}

func (c *worldContext) WorldVirtualState() WorldVirtualState {
	return c.virtualState
}

func (c *worldContext) GetFuture(lq []LockRequest) WorldContext {
	lq2 := make([]LockRequest, len(lq)+1)
	copy(lq2, lq)
	lq2[len(lq)] = LockRequest{
		Lock: AccountReadLock,
		ID:   SystemIDStr,
	}

	var wvs WorldVirtualState
	if c.virtualState != nil {
		wvs = c.virtualState.GetFuture(lq2)
	} else {
		wvs = NewWorldVirtualState(c.WorldState, lq2)
	}
	return c.WorldStateChanged(wvs)
}

// TODO What if some values such as deployer don't use cache here and are resolved on demand.
type systemStorageInfo struct {
	ass          AccountSnapshot
	stepPrice    *big.Int
	stepCosts    map[string]int64
	stepLimit    map[string]int64
	sysConfig    int64
	stepCostInfo *codec.TypedObj
	revision     module.Revision
	feeLimit     *big.Int
}

func (si *systemStorageInfo) Update(wc *worldContext) bool {
	ass := wc.GetAccountSnapshot(SystemID)
	if si.ass != nil && !ass.StorageChangedAfter(si.ass) {
		return false
	}

	si.ass = ass
	acs := wc.GetAccountState(SystemID)
	as := scoredb.NewStateStoreWith(acs)
	revision := int(scoredb.NewVarDB(as, VarRevision).Int64())
	si.revision = wc.platform.ToRevision(revision)

	stepPrice := scoredb.NewVarDB(as, VarStepPrice).BigInt()
	si.stepPrice = stepPrice

	stepCosts := make(map[string]int64)
	stepTypes := scoredb.NewArrayDB(as, VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		if value := stepCostDB.Get(tname).Int64(); value != 0 {
			stepCosts[tname] = value
		}
	}
	si.stepCosts = stepCosts
	si.stepCostInfo = nil

	stepLimit := make(map[string]int64)
	stepLimitTypes := scoredb.NewArrayDB(as, VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, VarStepLimit, 1)
	tcount = stepLimitTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepLimitTypes.Get(i).String()
		if value := stepLimitDB.Get(tname).Int64(); value != 0 {
			stepLimit[tname] = value
		}
	}
	si.stepLimit = stepLimit
	if stepPrice == nil || stepPrice.Sign() == 0 {
		si.feeLimit = new(big.Int)
	} else {
		si.feeLimit = new(big.Int).Mul(stepPrice, big.NewInt(stepLimit[StepLimitTypeInvoke]))
	}

	si.sysConfig = scoredb.NewVarDB(as, VarServiceConfig).Int64()
	return true
}

func (c *worldContext) Revision() module.Revision {
	return c.systemInfo.revision
}

func (c *worldContext) DepositIssueRate() *big.Int {
	ss := scoredb.NewStateStoreWith(c.systemInfo.ass)
	if r := scoredb.NewVarDB(ss, VarDepositIssueRate).BigInt(); r != nil {
		return r
	} else {
		return defaultDepositIssueRate
	}
}

func (c *worldContext) DepositTerm() int64 {
	ss := scoredb.NewStateStoreWith(c.systemInfo.ass)
	return scoredb.NewVarDB(ss, VarDepositTerm).Int64()
}

func (c *worldContext) ToRevision(value int) module.Revision {
	return c.platform.ToRevision(value)
}

func (c *worldContext) StepsFor(t StepType, n int) int64 {
	if v, ok := c.systemInfo.stepCosts[string(t)]; ok {
		return v * int64(n)
	} else {
		return 0
	}
}

func (c *worldContext) StepPrice() *big.Int {
	return c.systemInfo.stepPrice
}

func (c *worldContext) FeeLimit() *big.Int {
	return c.systemInfo.feeLimit
}

func (c *worldContext) GetStepLimit(t string) *big.Int {
	if v, ok := c.systemInfo.stepLimit[t]; ok {
		return big.NewInt(v)
	} else {
		return big.NewInt(0)
	}
}

func (c *worldContext) FeeEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigFee) != 0
}

func (c *worldContext) AuditEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigAudit) != 0
}

func (c *worldContext) FeeSharingEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigFeeSharing) != 0
}

func (c *worldContext) DeployerWhiteListEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigDeployerWhiteList) != 0
}

func (c *worldContext) PackageValidatorEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigScorePackageValidator) != 0
}

func (c *worldContext) MembershipEnabled() bool {
	return (c.systemInfo.sysConfig & SysConfigMembership) != 0
}

func (c *worldContext) TransactionTimestampThreshold() int64 {
	ass := c.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tshInMS := scoredb.NewVarDB(as, VarTimestampThreshold).Int64()
	return tshInMS * 1000
}

func (c *worldContext) IsDeployer(addr string) bool {
	ass := c.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	db := scoredb.NewArrayDB(as, VarDeployers)
	if db.Size() > 0 {
		for i := 0; i < db.Size(); i++ {
			if addr == db.Get(i).Address().String() {
				return true
			}
		}
	}
	return false
}

func (c *worldContext) BlockTimeStamp() int64 {
	return c.blockInfo.Timestamp()
}

func (c *worldContext) BlockHeight() int64 {
	return c.blockInfo.Height()
}

func (c *worldContext) ConsensusInfo() module.ConsensusInfo {
	return c.csInfo
}

func (c *worldContext) Treasury() module.Address {
	return c.treasury
}

func (c *worldContext) Governance() module.Address {
	return c.governance
}

func tryVirtualState(ws WorldState) WorldVirtualState {
	wvs, _ := ws.(WorldVirtualState)
	return wvs
}

func (c *worldContext) WorldStateChanged(ws WorldState) WorldContext {
	wc := &worldContext{
		WorldState:   ws,
		virtualState: tryVirtualState(ws),
		treasury:     c.treasury,
		governance:   c.governance,
		systemInfo:   c.systemInfo,
		blockInfo:    c.blockInfo,
		csInfo:       c.csInfo,
		platform:     c.platform,
	}
	return wc
}

func (c *worldContext) SetTransactionInfo(ti *TransactionInfo) {
	c.txInfo = *ti
	c.info = nil
	c.nextTxSalt = nil
}

func (c *worldContext) TransactionInfo() *TransactionInfo {
	if c.txInfo.Hash != nil {
		info := c.txInfo
		return &info
	}
	return nil
}

func (c *worldContext) TransactionID() []byte {
	return c.txInfo.Hash
}

// TransactionSalt returns index value to be used as salt.
// On the first call in a transaction, it returns nil.
// Then it returns 1 to N in a sequence.
func (c *worldContext) NextTransactionSalt() *big.Int {
	salt := c.nextTxSalt
	if c.nextTxSalt == nil {
		c.nextTxSalt = intconv.BigIntOne
	} else {
		c.nextTxSalt = new(big.Int).Add(c.nextTxSalt, intconv.BigIntOne)
	}
	return salt
}

func (c *worldContext) SetContractInfo(si *ContractInfo) {
	c.contractInfo = *si
	c.info = nil
}

func (c *worldContext) stepCostInfo() interface{} {
	if c.systemInfo.stepCostInfo == nil {
		stepCosts := make(map[string]interface{})
		for k, v := range c.systemInfo.stepCosts {
			switch k {
			case StepTypeDefault:
			case StepTypeContractCall:
			case StepTypeContractCreate:
			case StepTypeContractUpdate:
			case StepTypeContractDestruct:
			case StepTypeContractSet:
			case StepTypeInput:
				continue
			default:
				stepCosts[k] = v
			}
		}
		c.systemInfo.stepCostInfo = common.MustEncodeAny(stepCosts)
	}
	return c.systemInfo.stepCostInfo
}

func (c *worldContext) GetInfo() map[string]interface{} {
	if c.info == nil {
		m := make(map[string]interface{})
		m[InfoBlockHeight] = c.BlockHeight()
		m[InfoBlockTimestamp] = c.BlockTimeStamp()
		m[InfoTxHash] = c.txInfo.Hash
		m[InfoTxIndex] = c.txInfo.Index
		m[InfoTxTimestamp] = c.txInfo.Timestamp
		m[InfoTxNonce] = c.txInfo.Nonce
		m[InfoTxFrom] = c.txInfo.From
		m[InfoRevision] = int(c.Revision())
		m[InfoStepCosts] = c.stepCostInfo()
		m[InfoContractOwner] = c.contractInfo.Owner
		c.info = m
	}
	return c.info
}

func (c *worldContext) EnableSkipTransaction() {
	c.skipTransaction = true
}

func (c *worldContext) SkipTransactionEnabled() bool {
	return c.skipTransaction
}

func (c *worldContext) UpdateSystemInfo() {
	if c.systemInfo.Update(c) {
		c.info = nil
	}
}

func (c *worldContext) DecodeDoubleSignData(t string, d []byte) (module.DoubleSignData, error) {
	if c.dsDecoder == nil {
		return nil, errors.UnsupportedError.New("NoDoubleSignDataDecoder")
	}
	return c.dsDecoder(t, d)
}

type Platform interface {
	ToRevision(value int) module.Revision
}

type PlatformWithDoubleSignDataDecoder interface {
	Platform
	DoubleSignDataDecoder() module.DoubleSignDataDecoder
}

func getDoubleSignDataDecoder(plt Platform) module.DoubleSignDataDecoder {
	if p, ok := plt.(PlatformWithDoubleSignDataDecoder) ; ok {
		return p.DoubleSignDataDecoder()
	} else {
		return nil
	}
}

func (c *worldContext) DecodeDoubleSignContext(t string, d []byte) (module.DoubleSignContext, error) {
	return decodeDoubleSignContext(t, d)
}

func (c *worldContext) GetDoubleSignContextRoot() (module.DoubleSignContextRoot, error) {
	c.UpdateSystemInfo()
	return getDoubleSignContextRootOf(c, c.Revision())
}


func NewWorldContext(ws WorldState, bi module.BlockInfo, csi module.ConsensusInfo, plt Platform) WorldContext {
	var governance, treasury module.Address
	ass := ws.GetAccountSnapshot(SystemID)
	as := scoredb.NewStateStoreWith(ass)
	if as != nil {
		treasury = scoredb.NewVarDB(as, VarTreasury).Address()
		governance = scoredb.NewVarDB(as, VarGovernance).Address()
	}
	if treasury == nil {
		treasury = common.MustNewAddressFromString("hx1000000000000000000000000000000000000000")
	}
	if governance == nil {
		governance = common.MustNewAddressFromString("cx0000000000000000000000000000000000000001")
	}
	wc := &worldContext{
		WorldState:   ws,
		virtualState: tryVirtualState(ws),
		treasury:     treasury,
		governance:   governance,
		blockInfo:    bi,
		csInfo:       csi,
		platform:     plt,
		dsDecoder: 	  getDoubleSignDataDecoder(plt),
	}
	ws.EnableAccountNodeCache(SystemID)
	wc.UpdateSystemInfo()
	return wc
}
