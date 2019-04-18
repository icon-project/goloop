package transaction

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"sort"
	"strings"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"

	"github.com/icon-project/goloop/service/scoredb"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	InvalidGenesisError = iota + errors.CodeService + 100
)

type preInstalledScores struct {
	Owner       *common.Address  `json:"owner"`
	ContentType string           `json:"contentType"`
	ContentId   string           `json:"contentId"`
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

func (g *genesisV3) GetHandler(contract.ContractManager) (TransactionHandler, error) {
	return g, nil
}

func (g *genesisV3) ValidateNetwork(nid int) bool {
	return g.NID.Value == int64(nid)
}

func (g *genesisV3) Prepare(ctx contract.Context) (state.WorldContext, error) {
	lq := []state.LockRequest{
		{state.WorldIDStr, state.AccountWriteLock},
	}
	return ctx.GetFuture(lq), nil
}

func (g *genesisV3) Execute(ctx contract.Context) (txresult.Receipt, error) {
	r := txresult.NewReceipt(common.NewContractAddress(state.SystemID))
	as := ctx.GetAccountState(state.SystemID)

	var totalSupply big.Int
	for i := range g.Accounts {
		info := g.Accounts[i]
		if info.Balance == nil {
			continue
		}
		addr := scoredb.NewVarDB(as, info.Name)
		addr.Set(&info.Address)
		ac := ctx.GetAccountState(info.Address.ID())
		ac.SetBalance(&info.Balance.Int)
		totalSupply.Add(&totalSupply, &info.Balance.Int)
	}

	var nid int64
	if g.NID == nil {
		nid = state.DefaultNID
	} else {
		nid = g.NID.Value
		if nid == 0 {
			return nil, InvalidGenesisError.Errorf("IllegalNetworkID(%d)", nid)
		}
	}
	nidVar := scoredb.NewVarDB(as, state.VarNetwork)
	nidVar.Set(nid)

	ts := scoredb.NewVarDB(as, state.VarTotalSupply)
	if err := ts.Set(&totalSupply); err != nil {
		log.Printf("Fail to store total supply err=%+v", err)
		return nil, err
	}

	if err := g.deployPreInstall(ctx, r); err != nil {
		log.Printf("Failed to install scores err=%s", err)
		return nil, err
	}
	r.SetResult(module.StatusSuccess, big.NewInt(0), big.NewInt(0), nil)
	return r, nil
}

const (
	contentIdHash = "hash:"
	contentIdCid  = "cid:"
)

func (g *genesisV3) deployChainScore(ctx contract.Context, receipt txresult.Receipt) error {
	sas := ctx.GetAccountState(state.SystemID)
	sas.InitContractAccount(nil)
	sas.DeployContract(nil, "system", state.CTAppSystem,
		nil, nil)
	if err := sas.AcceptContract(nil, nil); err != nil {
		log.Printf("Failed to accept chainScore. err = %s", err)
		return err
	}
	chainScore, err := contract.GetSystemScore(contract.CID_CHAIN, common.NewContractAddress(state.SystemID), contract.NewCallContext(ctx, receipt, false))
	if err != nil {
		log.Printf("Failed to get systemScore")
		return err
	}
	if err := contract.CheckMethod(chainScore); err != nil {
		log.Printf("Failed to check method. err = %s\n", err)
		return err
	}
	sas.SetAPIInfo(chainScore.GetAPI())
	if err := chainScore.Install(g.Chain); err != nil {
		return err
	}
	return nil
}

func (g *genesisV3) deployPreInstall(ctx contract.Context, receipt txresult.Receipt) error {
	if err := g.deployChainScore(ctx, receipt); err != nil {
		log.Printf("Filed to deploy ChainScore. err = %s\n", err)
		return err
	}
	for _, a := range g.Accounts {
		if a.Score == nil {
			continue
		}
		score := a.Score
		cc := contract.NewCallContext(ctx, receipt, false)
		if score.Content != "" {
			if strings.HasPrefix(score.Content, "0x") {
				score.Content = strings.TrimPrefix(score.Content, "0x")
			}
			data, _ := hex.DecodeString(score.Content)
			d := contract.NewDeployHandlerForPreInstall(score.Owner,
				&a.Address, score.ContentType, data, score.Params)
			status, _, _, _ := cc.Call(d)
			if status != module.StatusSuccess {
				log.Printf("Failed to install pre-installed score."+
					"status : %d, addr : %v\n", status, a.Address)
				return errors.New(fmt.Sprintf("Failed to deploy pre-installed score. status = %d", status))
			}
			cc.Dispose()
		} else if score.ContentId != "" {
			if strings.HasPrefix(score.ContentId, contentIdHash) == true {
				contentHash := strings.TrimPrefix(score.ContentId, contentIdHash)
				content, err := ctx.GetPreInstalledScore(contentHash)
				if err != nil {
					log.Printf("Fail to get PreInstalledScore for ID=%s",
						contentHash)
					return err
				}
				d := contract.NewDeployHandlerForPreInstall(score.Owner,
					&a.Address, score.ContentType, content, score.Params)
				status, _, _, _ := cc.Call(d)
				if status != module.StatusSuccess {
					log.Printf("Failed to install pre-installed score."+
						"status : %d, addr : %v, file : %s\n", status, a.Address, contentHash)
					return errors.New(fmt.Sprintf("Failed to deploy pre-installed score. status = %d", status))
				}
				cc.Dispose()
			} else if strings.HasPrefix(score.ContentId, contentIdCid) == true {
				// TODO implement for contentCid
				return errors.UnsupportedError.New("CID prefix is't Unsupported")
			} else {
				return InvalidGenesisError.Errorf("SCORE<%s> Invalid contentId=%q", &a.Address, score.ContentId)
			}
		} else {
			return InvalidGenesisError.Errorf("There is no content for score %s", &a.Address)
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
