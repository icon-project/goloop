package tx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
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

type accountInfo struct {
	Name    string         `json:"name"`
	Address common.Address `json:"address"`
	Balance common.HexInt  `json:"balance"`
}

type genesisV3JSON struct {
	Accounts      []accountInfo     `json:"accounts"`
	Message       string            `json:"message"`
	Validatorlist []*common.Address `json:"validatorlist"`
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

func (g *genesisV3) setDefaultSystemInfo(as state.AccountState) {
	sysConfig := "./systemInfo.json"
	var stepPrice int64 = 10000000
	var stepCosts map[string]int64
	var stepLimit map[string]int64
	if _, err := os.Stat(sysConfig); !os.IsNotExist(err) {
		info, err := ioutil.ReadFile(sysConfig)
		if err != nil {
			log.Panicf("Fail to open genesis file=%s err=%+v", info, err)
		}
		var infoMap = make(map[string]interface{})
		err = json.Unmarshal(info, &infoMap)
		if err != nil {
			log.Panicf("error : %s\n", err)
		}
		for k, v := range infoMap {
			switch k {
			case state.VarStepTypes:
				stepTypesMap := v.(map[string]interface{})
				stepCosts = make(map[string]int64)
				for sk, sv := range stepTypesMap {
					stepCosts[sk], _ = strconv.ParseInt(sv.(string), 10, 64)
				}
			case state.VarStepPrice:
				stepPrice, _ = strconv.ParseInt(v.(string), 10, 64)
			case state.VarStepLimit:
				stepLimitMap := v.(map[string]interface{})
				stepLimit = make(map[string]int64)
				for sk, sv := range stepLimitMap {
					stepLimit[sk], _ = strconv.ParseInt(sv.(string), 10, 64)
				}
			}
		}
	} else {
		stepCosts = map[string]int64{
			"default":          0x186a0,
			"contractCall":     0x61a8,
			"contractCreate":   0x3b9aca00,
			"contractUpdate":   0x5f5e1000,
			"contractDestruct": -0x11170,
			"contractSet":      0x7530,
			"get":              0x0,
			"set":              0x140,
			"replace":          0x50,
			"delete":           -0xf0,
			"input":            0xc8,
			"eventLog":         0x64,
			"apiCall":          0x0,
		}

		stepLimit = map[string]int64{
			state.LimitTypeInvoke: 0x9502f900,
			state.LimitTypeCall:   0x2faf080,
		}
	}

	scoredb.NewVarDB(as, state.VarStepPrice).Set(big.NewInt(stepPrice))
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	for _, k := range state.AllStepTypes {
		if v, ok := stepCosts[k]; ok {
			stepTypes.Put(k)
			stepCostDB.Set(k, v)
		}
	}

	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	for _, k := range state.AllLimitTypes {
		if v, ok := stepLimit[k]; ok {
			stepLimitTypes.Put(k)
			stepLimitDB.Set(k, v)
		}
	}
}

func (g *genesisV3) Execute(ctx contract.Context) (txresult.Receipt, error) {
	r := txresult.NewReceipt(common.NewAccountAddress([]byte{}))
	as := ctx.GetAccountState(state.SystemID)
	for _, info := range g.Accounts {
		addr := scoredb.NewVarDB(as, info.Name)
		addr.Set(&info.Address)
		ac := ctx.GetAccountState(info.Address.ID())
		ac.SetBalance(&info.Balance.Int)
	}
	g.setDefaultSystemInfo(as)
	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	validators := make([]module.Validator, len(g.Validatorlist))
	for i, validator := range g.Validatorlist {
		validators[i], _ = state.ValidatorFromAddress(validator)
	}
	ctx.SetValidators(validators)
	return r, nil
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
