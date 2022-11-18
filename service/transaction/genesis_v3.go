package transaction

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"sort"
	"strings"

	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/service/scoredb"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type preInstalledScores struct {
	Owner       *common.Address  `json:"owner"`
	ContentType string           `json:"contentType"`
	ContentID   string           `json:"contentId"`
	Content     string           `json:"content"`
	Params      *json.RawMessage `json:"params"`
}
type accountInfo struct {
	Name    string              `json:"name"`
	Address common.Address      `json:"address"`
	Balance *common.HexInt      `json:"balance"`
	Score   *preInstalledScores `json:"score"`
}

type genesisV3JSON struct {
	Accounts []accountInfo    `json:"accounts"`
	Message  string           `json:"message"`
	Chain    json.RawMessage  `json:"chain"`
	NID      *common.HexInt64 `json:"nid"`
	raw      []byte
	txHash   []byte
}

const (
	ICONMainNetGenesisID = "5aa2453a84ba2fb1e3394b9e3471f5dcebc6225fc311a97ca505728153b9d246"
	ICONMainNetCID       = 1
)

func legacySerializeValue(w io.Writer, o interface{}) error {
	switch obj := o.(type) {
	case string:
		w.Write([]byte("."))
		w.Write([]byte(obj))
	case []interface{}:
		for _, v := range obj {
			if err := legacySerializeValue(w, v); err != nil {
				return err
			}
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
				if err := legacySerializeValue(w, v); err != nil {
					return err
				}
			}
		}
	default:
		return errors.IllegalArgumentError.Errorf("InvalidType(%T)", o)
	}
	return nil
}

func legacySerializeMap(jso map[string]interface{}) ([]byte, error) {
	var buf = bytes.NewBuffer(nil)
	if err := legacySerializeValue(buf, jso); err != nil {
		return nil, err
	}
	return buf.Bytes()[1:], nil
}

// If there is "chain" field, we assume that it's enterprise
// genesis. Otherwise, we assume that it's ICON genesis.
func useNormalGenesisSerialize(jso map[string]interface{}) bool {
	if _, ok := jso["chain"]; ok {
		return true
	}
	if _, ok := jso["nid"]; ok {
		return true
	}
	return false
}

func serializeGenesis(bs []byte) ([]byte, error) {
	var jso map[string]interface{}
	if err := json.Unmarshal(bs, &jso); err != nil {
		return nil, err
	}
	if useNormalGenesisSerialize(jso) {
		return SerializeMap(jso, nil, nil)
	}
	return legacySerializeMap(jso)
}

func (g *genesisV3JSON) calcHash() ([]byte, error) {
	bs, err := serializeGenesis(g.raw)
	if err != nil {
		return nil, err
	}
	bs = append([]byte("genesis_tx."), bs...)
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
	nid  int
	cid  int
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

func (g *genesisV3) ToJSON(version module.JSONVersion) (interface{}, error) {
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
		return InvalidGenesisError.New("NoTreasuryAccount")
	}
	if _, ok := acs["god"]; !ok {
		return InvalidGenesisError.New("NoGodAccount")
	}
	return nil
}

func (g *genesisV3) PreValidate(wc state.WorldContext, update bool) error {
	if wc.BlockHeight() != 0 {
		return errors.ErrInvalidState
	}
	return nil
}

func (g *genesisV3) GetHandler(contract.ContractManager) (Handler, error) {
	return g, nil
}

var ICONMainNetGenesisIDBytes, _ = hex.DecodeString(ICONMainNetGenesisID)

func CIDForGenesisTransactionID(txid []byte) int {
	if bytes.Equal(txid, ICONMainNetGenesisIDBytes) {
		return ICONMainNetCID
	}
	return int(txid[2]) | int(txid[1])<<8 | int(txid[0])<<16
}

func (g *genesisV3) NID() int {
	if g.nid == 0 {
		if g.genesisV3JSON.NID == nil {
			g.nid = g.CID()
		} else {
			g.nid = int(g.genesisV3JSON.NID.Value)
		}
	}
	return g.nid
}

func (g *genesisV3) CID() int {
	if g.cid == 0 {
		g.cid = CIDForGenesisTransactionID(g.ID())
	}
	return g.cid
}

func (g *genesisV3) ValidateNetwork(nid int) bool {
	return g.NID() == nid
}

func (g *genesisV3) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (g *genesisV3) Execute(ctx contract.Context, wcs state.WorldSnapshot, estimate bool) (txresult.Receipt, error) {
	cc := contract.NewCallContext(ctx, ctx.GetStepLimit(state.StepLimitTypeInvoke), false)
	defer cc.Dispose()

	as := cc.GetAccountState(state.SystemID)

	var totalSupply big.Int
	for i := range g.Accounts {
		info := g.Accounts[i]
		if info.Balance == nil {
			continue
		}
		balance := &info.Balance.Int
		addr := scoredb.NewVarDB(as, info.Name)
		addr.Set(&info.Address)
		ac := ctx.GetAccountState(info.Address.ID())
		ac.SetBalance(balance)
		totalSupply.Add(&totalSupply, balance)

		cc.FrameLogger().OnBalanceChange(module.Genesis, nil, &info.Address, balance)
	}

	nid := g.NID()
	nidVar := scoredb.NewVarDB(as, state.VarNetwork)
	nidVar.Set(nid)

	ts := scoredb.NewVarDB(as, state.VarTotalSupply)
	if err := ts.Set(&totalSupply); err != nil {
		ctx.Logger().Errorf("Fail to store total supply err=%+v\n", err)
		return nil, err
	}

	if err := contract.DeployAndInstallSystemSCORE(cc,
		contract.CID_CHAIN, nil, state.SystemAddress, g.Chain, nil); err != nil {
		return nil, InvalidGenesisError.Wrapf(err, "FAIL to deploy ChainScore")
	}

	cc.UpdateSystemInfo()
	cc.ResetStepLimit(cc.GetStepLimit(state.StepLimitTypeInvoke))
	r := txresult.NewReceipt(cc.Database(), cc.Revision(), cc.ContractManager().GenesisTo())
	if err := g.installContracts(cc); err != nil {
		ctx.Logger().Warnf("Fail to install scores err=%+v\n", err)
		return nil, err
	}
	cc.GetEventLogs(r)
	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	return r, nil
}

const (
	contentIdHash = "hash:"
	contentIdCid  = "cid:"
)

func (g *genesisV3) installContracts(cc contract.CallContext) error {
	for _, acc := range g.Accounts {
		if acc.Score == nil {
			continue
		}
		score := acc.Score
		if score.Content != "" {
			data, err := hex.DecodeString(strings.TrimPrefix(score.Content, "0x"))
			if err != nil {
				return InvalidGenesisError.Wrapf(err,
					"Invalid content(content=%s,addr=%s)", score.Content, &acc.Address)
			}
			handler := contract.NewDeployHandlerForPreInstall(score.Owner,
				&acc.Address, score.ContentType, data, score.Params, cc.Logger())
			status, _, _, _ := cc.Call(handler, cc.StepAvailable())
			if status != nil {
				return InvalidGenesisError.Wrapf(status,
					"FAIL to install pre-installed score. addr=%s", &acc.Address)
			}
		} else if score.ContentID != "" {
			if strings.HasPrefix(score.ContentID, contentIdHash) == true {
				contentHash := strings.TrimPrefix(score.ContentID, contentIdHash)
				content, err := cc.GetPreInstalledScore(contentHash)
				if err != nil {
					return InvalidGenesisError.Wrapf(err,
						"Fail to get PreInstalledScore for ID=%s", contentHash)
				}
				handler := contract.NewDeployHandlerForPreInstall(score.Owner,
					&acc.Address, score.ContentType, content, score.Params, cc.Logger())
				status, _, _, _ := cc.Call(handler, cc.StepAvailable())
				if status != nil {
					return InvalidGenesisError.Wrapf(status,
						"FAIL to install pre-installed score. addr=%s", &acc.Address)
				}
			} else if strings.HasPrefix(score.ContentID, contentIdCid) == true {
				// TODO implement for contentCid
				return InvalidGenesisError.New("CID prefix is't Unsupported")
			} else {
				return InvalidGenesisError.Errorf("SCORE<%s> Invalid contentId=%q", &acc.Address, score.ContentID)
			}
		} else {
			return InvalidGenesisError.Errorf("There is no content for score %s", &acc.Address)
		}
	}
	return nil
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

func (g *genesisV3) To() module.Address {
	if g.CID() == ICONMainNetCID {
		return state.ZeroAddress
	} else {
		return state.SystemAddress
	}
}

func (g *genesisV3) IsSkippable() bool {
	return false
}

func checkV3Genesis(jso map[string]interface{}) bool {
	_, hasAccounts := jso["accounts"]
	return hasAccounts
}

func parseV3Genesis(js []byte, raw bool) (Transaction, error) {
	genjs := new(genesisV3JSON)
	if err := json.Unmarshal(js, genjs); err != nil {
		return nil, errors.IllegalArgumentError.Wrapf(err, "Invalid json for genesis(%s)", string(js))
	}
	if len(genjs.Accounts) != 0 {
		genjs.raw = js
		tx := &genesisV3{genesisV3JSON: genjs}
		if err := tx.updateTxHash(); err != nil {
			return nil, InvalidGenesisError.Wrap(err, "FailToMakeTxHash")
		}
		return tx, nil
	}
	return nil, errors.IllegalArgumentError.New("NoAccounts")
}

func init() {
	RegisterFactory(&Factory{
		Priority:  10,
		CheckJSON: checkV3Genesis,
		ParseJSON: parseV3Genesis,
	})
}
