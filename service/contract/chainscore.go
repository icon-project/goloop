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
				{"addr", scoreapi.Address, nil},
			},
			nil,
		},
		{scoreapi.Function, "EnableScore",
			scoreapi.FlagExternal, 0,
			[]scoreapi.Parameter{
				{"addr", scoreapi.Address, nil},
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
		{scoreapi.Function, "GetServiceConfig",
			scoreapi.FlagExternal, 0,
			nil,
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
	log.Printf("method : %s, m : %v\n", method, m)
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
	log.Printf("r : c\n")
	r := m.Call(objects)
	log.Printf("r : %v\n", r)
	interfaceList := make([]interface{}, len(r))
	for i, v := range r {
		interfaceList[i] = v.Interface()
	}

	result, _ = common.EncodeAny(interfaceList)
	log.Printf("SysCall Execute : %v\n", result)
	return module.StatusSuccess, result
}

// Destroy : Allowed from score owner
func (s *ChainScore) DisableScore(addr module.Address) error {
	as := s.cc.GetAccountState(addr.ID())
	if as.ActiveContract() == nil {
		return errors.New("Not active contract")
	}
	as.SetDisable(true)
	return nil
}

func (s *ChainScore) EnableScore(addr module.Address) error {
	as := s.cc.GetAccountState(addr.ID())
	if as.ActiveContract() != nil {
		return errors.New("Not disabled contract")
	}
	as.SetDisable(false)
	return nil
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *ChainScore) SetRevision(code int64) error {
	if s.from.Equal(s.cc.Governance()) == false {
		return errors.New("Wrong")
	}
	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code <= r {
		return errors.New(fmt.Sprintf("Wrong revision. cur : %d, passed : %d\n", r, code))
	}
	return scoredb.NewVarDB(as, state.VarSysConfig).Set(code)
}

func (s *ChainScore) AcceptScore(txHash []byte) {
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	// TODO change below stepLimit
	ah := newAcceptHandler(s.from, s.to,
		nil, big.NewInt(100000000000), txHash, auditTxHash)
	// TODO check status, result
	ah.ExecuteSync(s.cc)
}

func (s *ChainScore) RejectScore(txHash []byte) {
}

// Governance score would check the verification of the address
func (s *ChainScore) BlockScore(addr module.Address) error {
	as := s.cc.GetAccountState(addr.ID())
	if as.IsBlocked() == false {
		as.SetBlock(true)
	}
	return nil
}

// Governance score would check the verification of the address
func (s *ChainScore) UnblockScore(addr module.Address) error {
	as := s.cc.GetAccountState(addr.ID())
	if as.IsBlocked() == true {
		as.SetBlock(false)
	}
	return nil
}

func (s *ChainScore) SetStepPrice(price int) {
	as := s.cc.GetAccountState(state.SystemID)
	scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *ChainScore) SetStepCost(costType string, cost int) {
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if stepCostDB.Get(costType) == nil {
		stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
		stepTypes.Put(costType)
	}
	stepCostDB.Set(costType, cost)
}

func (s *ChainScore) SetMaxStepLimit(contextType string, cost int) {
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if stepLimitDB.Get(contextType) == nil {
		stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
		stepLimitTypes.Put(contextType)
	}
	stepLimitDB.Set(contextType, cost)
}

func (s *ChainScore) AddDeployer(addr module.Address) error {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	return db.Put(addr)
}

func (s *ChainScore) RemoveDeployer(addr module.Address) error {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(addr) == true {
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

func (s *ChainScore) GetScoreStatus(addr module.Address) []byte {
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

	as := s.cc.GetAccountState(addr.ID())
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
	return result
}

func (s *ChainScore) IsDeployer(addr module.Address) int {
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployer)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(addr) == true {
			return 1
		}
	}
	return 0
}

func (s *ChainScore) GetServiceConfig() int64 {
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarSysConfig).Int64()
}

// Internal call
func (s *ChainScore) SetServiceConfig(config int64) {
	as := s.cc.GetAccountState(state.SystemID)
	scoredb.NewVarDB(as, state.VarSysConfig).Set(config)
}
