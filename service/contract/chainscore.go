package contract

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"reflect"

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
		{scoreapi.Function, "DisableScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "EnableScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "SetRevision",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"code", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "AcceptScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"txHash", scoreapi.Bytes, nil},
			},
			nil,
		},
		{scoreapi.Function, "RejectScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"txHash", scoreapi.Bytes, nil},
			},
			nil,
		},
		{scoreapi.Function, "BlockScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "UnblockScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "SetStepPrice",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"price", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "SetStepCost",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"costType", scoreapi.String, nil},
				{"cost", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "SetMaxStepLimit",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"contextType", scoreapi.String, nil},
				{"cost", scoreapi.Integer, nil},
			},
			nil,
		},
		{scoreapi.Function, "AddDeployer",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "RemoveDeployer",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "GetRevision",
			scoreapi.FlagExternal, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "GetStepPrice",
			scoreapi.FlagExternal, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "GetStepCost",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"t", scoreapi.String, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "GetStepCosts",
			scoreapi.FlagExternal, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Dict,
			},
		},
		{scoreapi.Function, "GetMaxStepLimit",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"t", scoreapi.String, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "GetScoreStatus",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Bytes,
			},
		},
		{scoreapi.Function, "IsDeployer",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"address", scoreapi.Address, nil},
			},
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "GetServiceConfig",
			scoreapi.FlagExternal, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Integer,
			},
		},
		{scoreapi.Function, "SetServiceConfig",
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
			log.Printf("Failed to sysCall. err = %s\n", err)
			status = module.StatusSystemError
		}
	}()
	m := reflect.ValueOf(s).MethodByName(method)
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
	interfaceList := make([]interface{}, len(r)-1)

	// first output type in chain score method is error.
	status = module.StatusSuccess
	for i, v := range r {
		if i == 0 {
			if err := v.Interface(); err != nil {
				log.Printf("Failed to invoke %s on chain score. %s\n", method, err.(error))
			}
			continue
		}
		interfaceList[i-1] = v.Interface()
	}

	result, _ = common.EncodeAny(interfaceList)
	return module.StatusSuccess, result
}

// Destroy : Allowed from score owner
func (s *ChainScore) DisableScore(address module.Address) error {
	as := s.cc.GetAccountState(address.ID())
	if as.ActiveContract() == nil {
		return errors.New("Not active contract")
	}
	if as.IsContractOwner(s.from) == false {
		return errors.New("Not Contract owner")
	}
	as.SetDisable(true)
	return nil
}

func (s *ChainScore) EnableScore(address module.Address) error {
	as := s.cc.GetAccountState(address.ID())
	if as.ActiveContract() != nil {
		return errors.New("Not disabled contract")
	}
	if as.IsContractOwner(s.from) == false {
		return errors.New("Not Contract owner")
	}
	as.SetDisable(false)
	return nil
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *ChainScore) SetRevision(code int64) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code <= r {
		return errors.New(fmt.Sprintf("Wrong revision. cur : %d, passed : %d\n", r, code))
	}
	return scoredb.NewVarDB(as, state.VarSysConfig).Set(code)
}

func (s *ChainScore) AcceptScore(txHash []byte) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	// TODO change below stepLimit
	ah := newAcceptHandler(s.from, s.to,
		nil, big.NewInt(100000000000), txHash, auditTxHash)
	// TODO check status, result
	ah.ExecuteSync(s.cc)
	return nil
}

func (s *ChainScore) RejectScore(txHash []byte) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	return nil
}

// Governance score would check the verification of the address
func (s *ChainScore) BlockScore(address module.Address) error {
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
func (s *ChainScore) UnblockScore(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true {
		as.SetBlock(false)
	}
	return nil
}

func (s *ChainScore) SetStepPrice(price int) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *ChainScore) SetStepCost(costType string, cost int) error {
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

func (s *ChainScore) SetMaxStepLimit(contextType string, cost int) error {
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

func (s *ChainScore) AddDeployer(address module.Address) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("No permission to call this method.")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	return db.Put(address)
}

func (s *ChainScore) RemoveDeployer(address module.Address) error {
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
func (s *ChainScore) GetRevision() int64 {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64()
}

func (s *ChainScore) GetStepPrice() int64 {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Int64()
}

func (s *ChainScore) GetStepCost(t string) int64 {
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	return stepCostDB.Get(t).Int64()
}

func (s *ChainScore) GetStepCosts() map[string]string {
	return nil
}

func (s *ChainScore) GetMaxStepLimit(t string) int64 {
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	return stepLimitDB.Get(t).Int64()
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

func (s *ChainScore) GetScoreStatus(address module.Address) (error, []byte) {
	stringStatus := func(s state.ContractStatus) string {
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
		case state.CSBlocked:
			status = "blacklist"
		case state.CSDisabled:
			status = "disable"
		default:

		}
		return status
	}

	as := s.cc.GetAccountState(address.ID())
	scoreStatus := scoreStatus{}
	if cur := as.Contract(); cur == nil {
		return errors.New("SCORE not found"), nil
	} else {
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
	return nil, result
}

func (s *ChainScore) IsDeployer(address module.Address) (error, int) {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return nil, 1
		}
	}
	return nil, 0
}

func (s *ChainScore) GetServiceConfig() (error, int64) {
	as := s.cc.GetAccountState(state.SystemID)
	return nil, scoredb.NewVarDB(as, state.VarSysConfig).Int64()
}

// Internal call
func (s *ChainScore) SetServiceConfig(config int64) error {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarSysConfig).Set(config)
}
