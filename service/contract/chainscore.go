package contract

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/scoreresult"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/service/scoreapi"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type ChainScore struct {
	from module.Address
	cc   CallContext
}

func NewChainScore(from module.Address, cc CallContext) SystemScore {
	return &ChainScore{from, cc}
}

func (s *ChainScore) GetAPI() *scoreapi.Info {
	methods := []*scoreapi.Method{
		{scoreapi.Function, "disableScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "enableScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "setRevision",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"code", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "acceptScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"txHash", scoreapi.Bytes, nil},
			},
			nil,
		},
		{scoreapi.Function, "rejectScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"txHash", scoreapi.Bytes, nil},
			},
			nil,
		},
		{scoreapi.Function, "blockScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "unblockScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "setStepPrice",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"price", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "setStepCost",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"type", scoreapi.String, nil},
				{"cost", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "setMaxStepLimit",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"contextType", scoreapi.String, nil},
				{"limit", scoreapi.Integer, nil},
			},
			nil,
		},
		// TODO add setValidators(addresses)
		{scoreapi.Function, "grantValidator",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "revokeValidator",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "addMember",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "removeMember",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "addDeployer",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "removeDeployer",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "addLicense",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"contentId", scoreapi.String, nil},
			},
			nil,
		},
		{scoreapi.Function, "removeLicense",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"contentId", scoreapi.String, nil},
			},
			nil,
		},
		{scoreapi.Function, "getRevision",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getStepPrice",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getStepCost",
			scoreapi.FlagReadOnly, 0,
			[]scoreapi.Parameter{
				{"type", scoreapi.String, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getStepCosts",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Dict,
			},
		},
		{scoreapi.Function, "getMaxStepLimit",
			scoreapi.FlagReadOnly, 0,
			[]scoreapi.Parameter{
				{"contextType", scoreapi.String, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getScoreStatus",
			scoreapi.FlagReadOnly, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Dict,
			},
		},
		{scoreapi.Function, "getMembers",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.List,
			},
		},
		{scoreapi.Function, "getValidators",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.List,
			},
		},
		{scoreapi.Function, "isDeployer",
			scoreapi.FlagReadOnly, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getServiceConfig",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
	}

	return scoreapi.NewInfo(methods)
}

type chain struct {
	AuditEnabled             common.HexInt16 `json:"auditEnabled"`
	DeployerWhiteListEnabled common.HexInt16 `json:"deployerWhiteListEnabled"`
	Fee                      struct {
		StepPrice common.HexInt    `json:"stepPrice"`
		StepLimit *json.RawMessage `json:"stepLimit"`
		StepCosts *json.RawMessage `json:"stepCosts"`
	} `json:"fee"`
	ValidatorList []*common.Address `json:"validatorList"`
	MemberList    []*common.Address `json:"memberList"`
	CommitTimeout *common.HexInt64  `json:"commitTimeout"`
}

func (s *ChainScore) Install(param []byte) error {
	chain := chain{}
	if param != nil {
		if err := json.Unmarshal(param, &chain); err != nil {
			log.Printf("Failed to parse parameter for chainScore. err = %s", err)
			return scoreresult.Errorf(module.StatusIllegalFormat, "FailToInstallChainScore")
		}
	}
	confValue := 0
	if chain.AuditEnabled.Value != 0 {
		confValue |= state.SysConfigAudit
	}
	if chain.DeployerWhiteListEnabled.Value != 0 {
		confValue |= state.SysConfigDeployerWhiteList
	}
	if len(chain.MemberList) > 0 {
		confValue |= state.SysConfigMembership
	}
	as := s.cc.GetAccountState(state.SystemID)
	if err := scoredb.NewVarDB(as, state.VarServiceConfig).Set(confValue); err != nil {
		log.Printf("Failed to set system config. err = %s", err)
		return err
	}

	timeout := int64(1000)
	if chain.CommitTimeout != nil {
		timeout = chain.CommitTimeout.Value
	}
	if err := scoredb.NewVarDB(as, state.VarCommitTimeout).Set(timeout); err != nil {
		log.Printf("Failed to set newHeightTimeout. err = %s", err)
		return err
	}

	price := chain.Fee
	if err := scoredb.NewVarDB(as, state.VarStepPrice).Set(&price.StepPrice.Int); err != nil {
		log.Printf("Failed to set stepPrice. err = %s", err)
		return err
	}
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if price.StepLimit != nil {
		stepLimitsMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepLimit, &stepLimitsMap); err != nil {
			log.Printf("Failed to unmarshal\n")
			return err
		}
		for _, k := range state.AllStepLimitTypes {
			cost := stepLimitsMap[k]
			if err := stepLimitTypes.Put(k); err != nil {
				log.Printf("Failed to put stepLimit. err = %s", err)
				return err
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Printf("Failed to parse %s to integer. err = %s\n", cost, err)
					return err
				}
			}
			if err := stepLimitDB.Set(k, icost); err != nil {
				log.Printf("Failed to Set stepLimit. err = %s", err)
				return err
			}
		}
	} else {
		for _, k := range state.AllStepLimitTypes {
			if err := stepLimitTypes.Put(k); err != nil {
				log.Printf("Failed to put steLimitTypes. err = %s", err)
				return err
			}
			if err := stepLimitDB.Set(k, 0); err != nil {
				log.Printf("Failed to set stepLimit. err = %s", err)
				return err
			}
		}
	}

	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if price.StepCosts != nil {
		stepTypesMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepCosts, &stepTypesMap); err != nil {
			log.Printf("Failed to unmarshal\n")
			return err
		}
		for _, k := range state.AllStepTypes {
			cost := stepTypesMap[k]
			if err := stepTypes.Put(k); err != nil {
				log.Printf("Failed to put stepTypes. err = %s", err)
				return err
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Printf("Failed to parse %s to integer. err = %s\n", cost, err)
					return err
				}
			}
			if err := stepCostDB.Set(k, icost); err != nil {
				log.Printf("Failed to set stepCost. err = %s", err)
				return err
			}
		}
	} else {
		for _, k := range state.AllStepTypes {
			if err := stepTypes.Put(k); err != nil {
				log.Printf("Failed to put stepTypes. err = %s", err)
				return err
			}
			if err := stepCostDB.Set(k, 0); err != nil {
				log.Printf("Failed to set stepCost. err = %s", err)
				return err
			}
		}
	}
	validators := make([]module.Validator, len(chain.ValidatorList))
	for i, validator := range chain.ValidatorList {
		validators[i], _ = state.ValidatorFromAddress(validator)
	}
	if err := s.cc.GetValidatorState().Set(validators); err != nil {
		log.Printf("Failed to set validator. err = %s\n", err)
		return err
	}

	if len(chain.MemberList) > 0 {
		members := scoredb.NewArrayDB(as, state.VarMembers)

		vs := s.cc.GetValidatorState()
		vc := 0
		m := make(map[string]bool)
		for i, member := range chain.MemberList {
			if member == nil {
				return errors.IllegalArgumentError.Errorf(
					"Member[%d] is null", i)
			}
			if member.IsContract() {
				return errors.IllegalArgumentError.Errorf(
					"Member must be EOA(%s)", member.String())
			}
			mn := member.String()
			if _, ok := m[mn]; ok {
				return errors.IllegalArgumentError.Errorf(
					"Duplicated Member(%s)", member.String())
			}
			m[mn] = true
			if idx := vs.IndexOf(member); idx >= 0 {
				vc += 1
			}
			members.Put(member)
		}
		if vc != vs.Len() {
			return errors.IllegalArgumentError.New(
				"All Validators must be included in the members")
		}
	}
	return nil
}

func (s *ChainScore) Update(param []byte) error {
	log.Panicf("Implement me")
	return nil
}

// Destroy : Allowed from score owner
func (s *ChainScore) Ex_disableScore(address module.Address) error {
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.ErrContractNotFound
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(true)
	return nil
}

func (s *ChainScore) Ex_enableScore(address module.Address) error {
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.ErrContractNotFound
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(false)
	return nil
}

func (s *ChainScore) fromGovernance() bool {
	return s.cc.Governance().Equal(s.from)
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *ChainScore) Ex_setRevision(code *common.HexInt) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code.Int64() <= r {
		return scoreresult.Errorf(module.StatusInvalidParameter,
			"Can't set code. cur : %d, passed : %d\n", r, code)
	}
	return scoredb.NewVarDB(as, state.VarRevision).Set(code)
}

func (s *ChainScore) Ex_acceptScore(txHash []byte) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)

	v, err := s.Ex_getMaxStepLimit(state.StepLimitTypeInvoke)
	if err != nil {
		return scoreresult.WithStatus(err, module.StatusSystemError)
	}
	ah := newAcceptHandler(s.from, common.NewAddress(state.SystemID),
		nil, big.NewInt(v), txHash, auditTxHash)
	status, _, _, _ := ah.ExecuteSync(s.cc)
	if status != module.StatusSuccess {
		return scoreresult.New(status, "Fail to execute acceptHandler")
	}
	return nil
}

func (s *ChainScore) Ex_rejectScore(txHash []byte) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}

	sysAs := s.cc.GetAccountState(state.SystemID)
	varDb := scoredb.NewVarDB(sysAs, txHash)
	scoreAddr := varDb.Address()
	if scoreAddr == nil {
		return scoreresult.Errorf(module.StatusInvalidParameter,
			"Fail to find score by txHash[%x]\n", txHash)
	}
	scoreAs := s.cc.GetAccountState(scoreAddr.ID())
	// NOTE : cannot change from reject to accept state because data with address mapped txHash is deleted from DB
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	if err := varDb.Delete(); err != nil {
		return scoreresult.WithStatus(err, module.StatusSystemError)
	}
	return scoreAs.RejectContract(txHash, auditTxHash)
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_blockScore(address module.Address) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == false {
		as.SetBlock(true)
	}
	return nil
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_unblockScore(address module.Address) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true {
		as.SetBlock(false)
	}
	return nil
}

func (s *ChainScore) Ex_setStepPrice(price *common.HexInt) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *ChainScore) Ex_setStepCost(costType string, cost *common.HexInt) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if stepCostDB.Get(costType) == nil {
		stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
		if err := stepTypes.Put(costType); err != nil {
			return scoreresult.WithStatus(err, module.StatusSystemError)
		}
	}
	return stepCostDB.Set(costType, cost)
}

func (s *ChainScore) Ex_setMaxStepLimit(contextType string, cost *common.HexInt) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if stepLimitDB.Get(contextType) == nil {
		stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
		if err := stepLimitTypes.Put(contextType); err != nil {
			return scoreresult.WithStatus(err, module.StatusSystemError)
		}
	}
	return stepLimitDB.Set(contextType, cost)
}

func (s *ChainScore) Ex_grantValidator(address module.Address) error {
	if address.IsContract() {
		return scoreresult.New(module.StatusInvalidParameter, "address should be EOA")
	}
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	if v, err := state.ValidatorFromAddress(address); err == nil {
		return s.cc.GetValidatorState().Add(v)
	} else {
		return err
	}
}

func (s *ChainScore) Ex_revokeValidator(address module.Address) error {
	if address.IsContract() {
		return scoreresult.New(module.StatusInvalidParameter, "address should be EOA")
	}
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	if v, err := state.ValidatorFromAddress(address); err == nil {
		s.cc.GetValidatorState().Remove(v)
		return nil
	} else {
		return err
	}
}

func (s *ChainScore) Ex_getValidators() ([]interface{}, error) {
	vs := s.cc.GetValidatorState()
	validators := make([]interface{}, vs.Len())
	for i := 0; i < vs.Len(); i++ {
		if v, ok := vs.Get(i); ok {
			validators[i] = v.Address()
		} else {
			return nil, scoreresult.New(module.StatusSystemError, "Unexpected access failure")
		}
	}
	return validators, nil
}

func (s *ChainScore) Ex_addMember(address module.Address) error {
	if address.IsContract() {
		return scoreresult.New(module.StatusInvalidParameter, "address should be EOA")
	}
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return nil
		}
	}
	return db.Put(address)
}

func (s *ChainScore) Ex_removeMember(address module.Address) error {
	if address.IsContract() {
		return scoreresult.New(module.StatusInvalidParameter, "address should be EOA")
	}

	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}

	// If membership system is on, first check if the member is not a validator
	if s.cc.MembershipEnabled() {
		if s.cc.GetValidatorState().IndexOf(address) >= 0 {
			return scoreresult.New(module.StatusSystemError, "Should revoke validator before removing the member")
		}
	}

	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size() { // addr is not rAddr
				if err := db.Set(i, rAddr); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (s *ChainScore) Ex_addDeployer(address module.Address) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return nil
		}
	}
	return db.Put(address)
}

func (s *ChainScore) Ex_removeDeployer(address module.Address) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size() { // addr is not rAddr
				if err := db.Set(i, rAddr); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (s *ChainScore) Ex_addLicense(contentId string) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarLicenses)
	for i := 0; i < db.Size(); i++ {
		if strings.Compare(db.Get(i).String(), contentId) == 0 {
			return nil
		}
	}
	return db.Put(contentId)
}

func (s *ChainScore) Ex_removeLicense(contentId string) error {
	if !s.fromGovernance() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarLicenses)
	for i := 0; i < db.Size(); i++ {
		if strings.Compare(db.Get(i).String(), contentId) == 0 {
			id := db.Pop().String()
			if i < db.Size() { // id is not contentId
				if err := db.Set(i, id); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// User calls icx_call : Functions which can be called by anyone.
func (s *ChainScore) Ex_getRevision() (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64(), nil
}

func (s *ChainScore) Ex_getStepPrice() (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Int64(), nil
}

func (s *ChainScore) Ex_getStepCost(t string) (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	return stepCostDB.Get(t).Int64(), nil
}

func (s *ChainScore) Ex_getStepCosts() (map[string]interface{}, error) {
	as := s.cc.GetAccountState(state.SystemID)

	stepCosts := make(map[string]interface{})
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		stepCosts[tname] = stepCostDB.Get(tname).Int64()
	}
	return stepCosts, nil
}

func (s *ChainScore) Ex_getMaxStepLimit(contextType string) (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	tcount := stepLimitTypes.Size()
	found := false
	for i := 0; i < tcount; i++ {
		if stepLimitTypes.Get(i).String() == contextType {
			found = true
			break
		}
	}

	if found == false {
		return 0, nil
	}

	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	return stepLimitDB.Get(contextType).Int64(), nil
}

func (s *ChainScore) Ex_getScoreStatus(address module.Address) (map[string]interface{}, error) {
	stringStatus := func(s state.ContractState) string {
		var status string
		switch s {
		case state.CSInactive:
			status = "inactive"
		case state.CSActive:
			status = "active"
		case state.CSPending:
			status = "pending"
		case state.CSRejected:
			status = "rejected"
		default:
			log.Printf("GetScoreStatus - string : %v\n", s)
		}
		return status
	}

	as := s.cc.GetAccountState(address.ID())
	if as == nil {
		return nil, scoreresult.ErrContractNotFound
	}
	scoreStatus := make(map[string]interface{})
	if cur := as.Contract(); cur != nil {
		curContract := make(map[string]interface{})
		curContract["status"] = stringStatus(cur.Status())
		curContract["deployTxHash"] = fmt.Sprintf("%x", cur.DeployTxHash())
		curContract["auditTxHash"] = fmt.Sprintf("%x", cur.AuditTxHash())
		scoreStatus["current"] = curContract
	}

	if next := as.NextContract(); next != nil {
		nextContract := make(map[string]interface{})
		nextContract["status"] = stringStatus(next.Status())
		nextContract["deployTxHash"] = fmt.Sprintf("%x", next.DeployTxHash())
		scoreStatus["next"] = nextContract
	}

	// blocked
	if as.IsBlocked() == true {
		scoreStatus["blocked"] = "0x1"
	} else {
		scoreStatus["blocked"] = "0x0"
	}

	// disabled
	if as.IsDisabled() == true {
		scoreStatus["disabled"] = "0x1"
	} else {
		scoreStatus["disabled"] = "0x0"
	}
	return scoreStatus, nil
}

func (s *ChainScore) Ex_isDeployer(address module.Address) (int, error) {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return 1, nil
		}
	}
	return 0, nil
}

func (s *ChainScore) Ex_getServiceConfig() (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarServiceConfig).Int64(), nil
}

func (s *ChainScore) Ex_getMembers() ([]interface{}, error) {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	members := make([]interface{}, db.Size())
	for i := 0; i < db.Size(); i++ {
		members[i] = db.Get(i).Address()
	}
	return members, nil
}
