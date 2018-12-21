package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/icon-project/goloop/service/scoredb"
	"io"
	"math/big"
	"sort"

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

func (g *genesisV3) PreValidate(wc WorldContext, update bool) error {
	if wc.BlockHeight() != 0 {
		return common.ErrInvalidState
	}
	return nil
}

func (g *genesisV3) GetHandler(cm ContractManager) (TransactionHandler, error) {
	return g, nil
}

func (g *genesisV3) Prepare(wc WorldContext) (WorldContext, error) {
	lq := []LockRequest{
		{"", AccountWriteLock},
	}
	return wc.WorldStateChanged(wc.WorldVirtualState().GetFuture(lq)), nil
}

func (g *genesisV3) Execute(wc WorldContext) (Receipt, error) {
	r := NewReceipt(common.NewAccountAddress([]byte{}))
	as := wc.GetAccountState(SystemID)
	for _, info := range g.Accounts {
		addr := scoredb.NewVarDB(as, info.Name)
		addr.Set(&info.Address)
		ac := wc.GetAccountState(info.Address.ID())
		ac.SetBalance(&info.Balance.Int)
	}
	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	validators := make([]module.Validator, len(g.Validatorlist))
	for i, validator := range g.Validatorlist {
		validators[i], _ = ValidatorFromAddress(validator)
	}
	wc.SetValidators(validators)
	return r, nil
}

func (g *genesisV3) Dispose() {
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
