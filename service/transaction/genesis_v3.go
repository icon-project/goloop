package transaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"sort"
	"strconv"

	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/service/scoredb"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type preInstalledScores struct {
	Owner       *common.Address
	ContentType string
	ContentID   string
	Params      *json.RawMessage
}
type accountInfo struct {
	Name    string              `json:"name"`
	Address common.Address      `json:"address"`
	Balance *common.HexInt      `json:"balance"`
	Score   *preInstalledScores `json:"score"`
}

type systemInfo struct {
	ConfFee                   bool
	ConfAudit                 bool
	ConfDeployerWhiteList     bool
	ConfScorePackageValidator bool
	Price                     struct {
		Step_price common.HexInt
		Step_limit *json.RawMessage
		Step_types *json.RawMessage
	}
}

type genesisV3JSON struct {
	Accounts      []accountInfo     `json:"accounts"`
	Message       string            `json:"message"`
	Validatorlist []*common.Address `json:"validatorlist"`
	SystemInfo    systemInfo
	raw           []byte
	txHash        []byte
}

func serialize(o map[string]interface{}) []byte {
	var buf = bytes.NewBuffer(nil)
	serializePart(buf, o)
	return buf.Bytes()[1:]
}

func serializePart(w io.Writer, o interface{}) {
	switch obj := o.(type) {
	case string:
		w.Write([]byte("."))
		w.Write([]byte(obj))
	case []interface{}:
		for _, v := range obj {
			serializePart(w, v)
		}
	case map[string]interface{}:
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if v, ok := obj[k]; ok {
				w.Write([]byte("."))
				w.Write([]byte(k))
				serializePart(w, v)
			}
		}
	}
}

func (g *genesisV3JSON) calcHash() ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(g.raw, &data); err != nil {
		return nil, err
	}
	bs := append([]byte("genesis_tx."), serialize(data)...)
	return crypto.SHA3Sum256(bs), nil
}

func (g *genesisV3JSON) updateTxHash() error {
	if g.txHash == nil {
		h, err := g.calcHash()
		if err != nil {
			return err
		}
		g.txHash = h
	}
	return nil
}

type genesisV3 struct {
	*genesisV3JSON
	hash []byte
}

func (g *genesisV3) From() module.Address {
	return nil
}

func (g *genesisV3) Version() int {
	return module.TransactionVersion3
}

func (g *genesisV3) Bytes() []byte {
	return g.genesisV3JSON.raw
}

func (g *genesisV3) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (g *genesisV3) Hash() []byte {
	if g.hash == nil {
		g.hash = crypto.SHA3Sum256(g.Bytes())
	}
	return g.hash
}

func (g *genesisV3) ID() []byte {
	g.updateTxHash()
	return g.txHash
}

func (g *genesisV3) ToJSON(version int) (interface{}, error) {
	var jso map[string]interface{}
	if err := json.Unmarshal(g.raw, &jso); err != nil {
		return nil, err
	}
	return jso, nil
}

func (g *genesisV3) Verify() error {
	acs := map[string]*accountInfo{}
	for _, ac := range g.genesisV3JSON.Accounts {
		acs[ac.Name] = &ac
	}
	if _, ok := acs["treasury"]; !ok {
		return errors.New("NoTreasury")
	}
	if _, ok := acs["god"]; !ok {
		return errors.New("NoGod")
	}
	return nil
}

func (g *genesisV3) PreValidate(wc state.WorldContext, update bool) error {
	if wc.BlockHeight() != 0 {
		return common.ErrInvalidState
	}
	return nil
}

func (g *genesisV3) GetHandler(contract.ContractManager) (TransactionHandler, error) {
	return g, nil
}

func (g *genesisV3) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{"", state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (g *genesisV3) setSystemInfo(as state.AccountState) {
	info := g.SystemInfo
	confValue := 0
	if info.ConfFee == true {
		confValue |= state.SysConfigFee
	}
	if info.ConfAudit == true {
		confValue |= state.SysConfigAudit
	}
	if info.ConfDeployerWhiteList == true {
		confValue |= state.SysConfigDeployerWhiteList
	}
	if info.ConfScorePackageValidator == true {
		confValue |= state.SysConfigScorePackageValidator
	}
	scoredb.NewVarDB(as, state.VarSysConfig).Set(confValue)

	price := info.Price
	scoredb.NewVarDB(as, state.VarStepPrice).Set(&price.Step_price.Int)
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if price.Step_limit != nil {
		stepLimitsMap := make(map[string]string)
		if err := json.Unmarshal(*price.Step_limit, &stepLimitsMap); err != nil {
			log.Panicf("Failed to unmarshal\n")
		}
		for _, k := range state.AllStepLimitTypes {
			cost := stepLimitsMap[k]
			stepLimitTypes.Put(k)
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Panicf("Failed to parse %s to integer. err = %s\n")
				}
			}
			stepLimitDB.Set(k, icost)
		}
	} else {
		for _, k := range state.AllStepLimitTypes {
			stepLimitTypes.Put(k)
			stepLimitDB.Set(k, 0)
		}
	}

	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if price.Step_types != nil {
		stepTypesMap := make(map[string]string)
		if err := json.Unmarshal(*price.Step_types, &stepTypesMap); err != nil {
			log.Panicf("Failed to unmarshal\n")
		}
		for _, k := range state.AllStepTypes {
			cost := stepTypesMap[k]
			stepTypes.Put(k)
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					log.Panicf("Failed to parse %s to integer. err = %s\n")
				}
			}
			stepCostDB.Set(k, icost)
		}
	} else {
		for _, k := range state.AllStepTypes {
			stepTypes.Put(k)
			stepCostDB.Set(k, 0)
		}
	}
}

func (g *genesisV3) Execute(ctx contract.Context) (txresult.Receipt, error) {
	r := txresult.NewReceipt(common.NewAccountAddress([]byte{}))
	as := ctx.GetAccountState(state.SystemID)
	for i := range g.Accounts {
		info := g.Accounts[i]
		if info.Balance == nil {
			continue
		}
		addr := scoredb.NewVarDB(as, info.Name)
		addr.Set(&info.Address)
		ac := ctx.GetAccountState(info.Address.ID())
		ac.SetBalance(&info.Balance.Int)
	}
	g.setSystemInfo(as)

	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	validators := make([]module.Validator, len(g.Validatorlist))
	for i, validator := range g.Validatorlist {
		validators[i], _ = state.ValidatorFromAddress(validator)
	}
	ctx.SetValidators(validators)
	g.deployPreInstall(ctx)
	return r, nil
}

func (g *genesisV3) deployPreInstall(ctx contract.Context) {
	// first install chainScore.
	sas := ctx.GetAccountState(state.SystemID)
	sas.InitContractAccount(nil)
	sas.DeployContract(nil, "system", state.CTAppSystem,
		nil, nil)
	sas.AcceptContract(nil, nil)
	chainScore := contract.GetSystemScore(nil, common.NewContractAddress(state.SystemID), nil)
	if contract.CheckMethod(chainScore) == false {
		log.Panicf("Failed to check method. wrong method info\n")
	}
	sas.SetAPIInfo(chainScore.GetAPI())

	// TODO add map table for static system score.

	for _, a := range g.Accounts {
		if a.Score == nil {
			continue
		}
		score := a.Score
		cc := contract.NewCallContext(ctx, nil, false)
		content := ctx.GetPreInstalledScore(score.ContentID)
		d := contract.NewDeployHandlerForPreInstall(score.Owner,
			&a.Address, score.ContentType, content, score.Params)
		status, _, _, _ := cc.Call(d)
		if status != module.StatusSuccess {
			log.Panicf("Failed to install pre-installed score."+
				"status : %d, addr : %v, file : %s\n", status, a.Address, score.ContentID)
		}
		cc.Dispose()
	}
}

func (g *genesisV3) Dispose() {
}

func (g *genesisV3) Query(wc state.WorldContext) (module.Status, interface{}) {
	return module.StatusSuccess, nil
}

func (g *genesisV3) Timestamp() int64 {
	return 0
}

func (g *genesisV3) MarshalJSON() ([]byte, error) {
	return g.raw, nil
}

func (g *genesisV3) Nonce() *big.Int {
	return nil
}
