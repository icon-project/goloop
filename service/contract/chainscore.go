package contract

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"

	"github.com/icon-project/goloop/common"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/service/scoreapi"

	"github.com/go-errors/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type ChainScore struct {
	from, to module.Address
	cc       CallContext
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
				{"costType", scoreapi.String, nil},
				{"cost", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "setMaxStepLimit",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"contextType", scoreapi.String, nil},
				{"cost", scoreapi.Integer, nil},
			},
			nil,
		},
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
		{scoreapi.Function, "getValidators",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.List,
			},
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
				{"t", scoreapi.String, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "getStepCosts",
			scoreapi.FlagReadOnly, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.String,
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
				scoreapi.String,
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
		{scoreapi.Function, "setServiceConfig",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"config", scoreapi.Integer, nil},
			},
			nil,
		},
	}

	return scoreapi.NewInfo(methods)
}

func (s *ChainScore) Invoke(method string, paramObj *codec.TypedObj) (
	status module.Status, result *codec.TypedObj) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Failed to sysCall method[%s]. err = %s\n", method, err)
			status = module.StatusSystemError
		}
	}()
	m := reflect.ValueOf(s).MethodByName(FUNC_PREFIX + method)
	if m.IsValid() == false {
		return module.StatusMethodNotFound, nil
	}
	params, _ := common.DecodeAny(paramObj)
	numIn := m.Type().NumIn()
	objects := make([]reflect.Value, numIn)
	if l, ok := params.([]interface{}); ok == true {
		if len(l) != numIn {
			return module.StatusInvalidParameter, nil
		}
		for i, v := range l {
			objects[i] = reflect.ValueOf(v)
		}
	}
	r := m.Call(objects)
	resultLen := len(r)
	var interfaceList []interface{}
	if resultLen > 1 {
		interfaceList = make([]interface{}, resultLen-1)
	}

	// first output type in chain score method is error.
	status = module.StatusSuccess
	for i, v := range r {
		if resultLen == i+1 {
			if err := v.Interface(); err != nil {
				log.Printf("Failed to invoke %s on chain score. %s\n", method, err.(error))
			}
			continue
		} else {
			interfaceList[i] = v.Interface()
		}
	}

	result, _ = common.EncodeAny(interfaceList)
	return module.StatusSuccess, result
}

type chain struct {
	AuditEnabled             bool `json:"auditEnabled"`
	DeployerWhiteListEnabled bool `json:"deployerWhiteListEnabled"`
	Fee                      struct {
		StepPrice common.HexInt    `json:"stepPrice"`
		StepLimit *json.RawMessage `json:"stepLimit"`
		StepCosts *json.RawMessage `json:"stepCosts"`
	} `json:"fee"`
}

func (s *ChainScore) Install(param []byte) error {
	chainCfg := chain{}
	if param != nil {
		if err := json.Unmarshal(param, &chainCfg); err != nil {
			log.Panicf("Failed to parse parameter for chainScore. err = %s", err)
		}
	}
	confValue := 0
	if chainCfg.AuditEnabled == true {
		confValue |= state.SysConfigAudit
	}
	if chainCfg.DeployerWhiteListEnabled == true {
		confValue |= state.SysConfigDeployerWhiteList
	}
	as := s.cc.GetAccountState(state.SystemID)
	if err := scoredb.NewVarDB(as, state.VarSysConfig).Set(confValue); err != nil {
		log.Panicf("Failed to set system config. err = %s", err)
	}

	price := chainCfg.Fee
	if err := scoredb.NewVarDB(as, state.VarStepPrice).Set(&price.StepPrice.Int); err != nil {
		log.Panicf("Failed to set stepPrice. err = %s", err)
	}
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if price.StepLimit != nil {
		stepLimitsMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepLimit, &stepLimitsMap); err != nil {
			log.Panicf("Failed to unmarshal\n")
		}
		for _, k := range state.AllStepLimitTypes {
			cost := stepLimitsMap[k]
			if err := stepLimitTypes.Put(k); err != nil {
				log.Panicf("Failed to put stepLimit. err = %s", err)
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Panicf("Failed to parse %s to integer. err = %s\n", cost, err)
				}
			}
			if err := stepLimitDB.Set(k, icost); err != nil {
				log.Panicf("Failed to Set stepLimit. err = %s", err)
			}
		}
	} else {
		for _, k := range state.AllStepLimitTypes {
			if err := stepLimitTypes.Put(k); err != nil {
				log.Panicf("Failed to put steLimitTypes. err = %s", err)
			}
			if err := stepLimitDB.Set(k, 0); err != nil {
				log.Panicf("Failed to set stepLimit. err = %s", err)
			}
		}
	}

	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if price.StepCosts != nil {
		stepTypesMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepCosts, &stepTypesMap); err != nil {
			log.Panicf("Failed to unmarshal\n")
		}
		for _, k := range state.AllStepTypes {
			cost := stepTypesMap[k]
			if err := stepTypes.Put(k); err != nil {
				log.Panicf("Failed to put stepTypes. err = %s", err)
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Panicf("Failed to parse %s to integer. err = %s\n", cost, err)
				}
			}
			if err := stepCostDB.Set(k, icost); err != nil {
				log.Panicf("Failed to set stepCost. err = %s", err)
			}
		}
	} else {
		for _, k := range state.AllStepTypes {
			if err := stepTypes.Put(k); err != nil {
				log.Panicf("Failed to put stepTypes. err = %s", err)
			}
			if err := stepCostDB.Set(k, 0); err != nil {
				log.Panicf("Failed to set stepCost. err = %s", err)
			}
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
		return errors.New("Not contract")
	}
	if as.IsContractOwner(s.from) == false {
		return errors.New("Not Contract owner")
	}
	as.SetDisable(true)
	return nil
}

func (s *ChainScore) Ex_enableScore(address module.Address) error {
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return errors.New("Not contract")
	}
	if as.IsContractOwner(s.from) == false {
		return errors.New("Not Contract owner")
	}
	as.SetDisable(false)
	return nil
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *ChainScore) Ex_setRevision(code *common.HexInt) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code.Int64() <= r {
		return errors.New(fmt.Sprintf("Wrong revision. cur : %d, passed : %d\n", r, code))
	}
	return scoredb.NewVarDB(as, state.VarRevision).Set(code)
}

func (s *ChainScore) Ex_acceptScore(txHash []byte) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)

	v, err := s.Ex_getMaxStepLimit(state.StepLimitTypeInvoke)
	if err != nil {
		return err
	}
	ah := newAcceptHandler(s.from, s.to,
		nil, big.NewInt(v), txHash, auditTxHash)
	status, _, _, _ := ah.ExecuteSync(s.cc)
	if status != module.StatusSuccess {
		return errors.New(fmt.Sprintf("Failed to  execute acceptHandler. status = %d", status))
	}
	return nil
}

func (s *ChainScore) Ex_rejectScore(txHash []byte) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}

	sysAs := s.cc.GetAccountState(state.SystemID)
	varDb := scoredb.NewVarDB(sysAs, txHash)
	scoreAddr := varDb.Address()
	if scoreAddr == nil {
		return errors.New(fmt.Sprintf("Faile d to find score by txHash[%x]\n", txHash))
	}
	scoreAs := s.cc.GetAccountState(scoreAddr.ID())
	// NOTE : cannot change from reject to accept state because data with address mapped txHash is deleted from DB
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	if err := varDb.Delete(); err != nil {
		log.Printf("Failed to delete scoreAddr. %s", scoreAddr.String())
		return err
	}
	return scoreAs.RejectContract(txHash, auditTxHash)
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_blockScore(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == false {
		as.SetBlock(true)
	}
	return nil
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_unblockScore(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true {
		as.SetBlock(false)
	}
	return nil
}

func (s *ChainScore) Ex_setStepPrice(price *common.HexInt) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *ChainScore) Ex_setStepCost(costType string, cost *common.HexInt) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if stepCostDB.Get(costType) == nil {
		stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
		if err := stepTypes.Put(costType); err != nil {
			return err
		}
	}
	return stepCostDB.Set(costType, cost)
}

func (s *ChainScore) Ex_setMaxStepLimit(contextType string, cost *common.HexInt) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if stepLimitDB.Get(contextType) == nil {
		stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
		if err := stepLimitTypes.Put(contextType); err != nil {
			return err
		}
	}
	return stepLimitDB.Set(contextType, cost)
}

func (s *ChainScore) Ex_grantValidator(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	if v, err := state.ValidatorFromAddress(address); err == nil {
		return s.cc.GrantValidator(v)
	} else {
		return err
	}
}

func (s *ChainScore) Ex_revokeValidator(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	if v, err := state.ValidatorFromAddress(address); err == nil {
		_, err = s.cc.RevokeValidator(v)
		return err
	} else {
		return err
	}
}

func (s *ChainScore) Ex_getValidators() ([]module.Address, error) {
	vl := s.cc.GetValidators()
	validators := make([]module.Address, vl.Len())
	for i := 0; i < vl.Len(); i++ {
		if v, ok := vl.Get(i); ok {
			validators[i] = v.Address()
		} else {
			return nil, errors.New("Unexpected access failure")
		}
	}
	return validators, nil
}

func (s *ChainScore) Ex_addMember(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	return db.Put(address)
}

func (s *ChainScore) Ex_removeMember(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}

	// If membership system is on, first check if the member is not a validator
	if s.cc.CfgMembershipEnabled() {
		if s.cc.GetValidators().IndexOf(address) >= 0 {
			return errors.New("Should revoke validator before removing the member")
		}
	}

	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size()-1 { // addr is not rAddr
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
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	return db.Put(address)
}

func (s *ChainScore) Ex_removeDeployer(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size()-1 { // addr is not rAddr
				if err := db.Set(i, rAddr); err != nil {
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

func (s *ChainScore) Ex_getStepCosts() (string, error) {
	as := s.cc.GetAccountState(state.SystemID)

	stepCosts := make(map[string]string)
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		stepCosts[tname] = fmt.Sprintf("%d", stepCostDB.Get(tname).Int64())
	}
	result, err := json.Marshal(stepCosts)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (s *ChainScore) Ex_getMaxStepLimit(contextType string) (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	return stepLimitDB.Get(contextType).Int64(), nil
}

type curScore struct {
	Status       string `json:"status"`
	DeployTxHash string `json:"deployTxHash"`
	AuditTxHash  string `json:"auditTxHash"`
}

type nextScore struct {
	Status       string `json:"status"`
	DeployTxHash string `json:"deployTxHash"`
}

type scoreStatus struct {
	Current  *curScore  `json:"current,omitempty"`
	Next     *nextScore `json:"next,omitempty"`
	Blocked  string     `json:"blocked"`
	Disabled string     `json:"disabled"`
}

func (s *ChainScore) Ex_getScoreStatus(address module.Address) (string, error) {
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
			status = "reject"
		default:
			log.Printf("GetScoreStatus - string : %v\n", s)
		}
		return status
	}

	as := s.cc.GetAccountState(address.ID())
	scoreStatus := scoreStatus{}
	if cur := as.Contract(); cur != nil {
		current := &curScore{}
		current.Status = stringStatus(cur.Status())
		current.DeployTxHash = fmt.Sprintf("%x", cur.DeployTxHash())
		current.AuditTxHash = fmt.Sprintf("%x", cur.AuditTxHash())
		scoreStatus.Current = current
	}

	if next := as.NextContract(); next != nil {
		nextContract := &nextScore{}
		nextContract.Status = stringStatus(next.Status())
		nextContract.DeployTxHash = fmt.Sprintf("%x", next.DeployTxHash())
		scoreStatus.Next = nextContract
	}

	// blocked
	if as.IsBlocked() == true {
		scoreStatus.Blocked = "0x01"
	} else {
		scoreStatus.Blocked = "0x00"
	}

	// disabled
	if as.IsDisabled() == true {
		scoreStatus.Disabled = "0x01"
	} else {
		scoreStatus.Disabled = "0x00"
	}
	result, err := json.Marshal(scoreStatus)
	if err != nil {
		log.Panicf("err : %s\n", err)
	}
	return string(result), nil
}

func (s *ChainScore) Ex_isDeployer(address module.Address) (int, error) {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return 1, nil
		}
	}
	return 0, nil
}

func (s *ChainScore) Ex_getServiceConfig() (int64, error) {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarSysConfig).Int64(), nil
}

// Internal call
func (s *ChainScore) SetServiceConfig(config int64) error {
	// TODO If membership system get enabled from disabled, it should ensure all validators should be members.
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarSysConfig).Set(config)
}
