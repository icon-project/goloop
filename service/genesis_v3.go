package service

import (
	"encoding/json"
	"errors"
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
	Accounts []accountInfo `json:"accounts"`
	Message  string        `json:"message"`
	raw      []byte
}

type genesisV3 struct {
	*genesisV3JSON
	id, hash []byte
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
	// TODO need to follow loopchain implementation.
	panic("implement me")
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
	// TODO Implement PreValidate
	panic("implement me")
}

func (g *genesisV3) Prepare(wvs WorldVirtualState) (WorldVirtualState, error) {
	// TODO Implement PrePare
	panic("implement me")
}

func (g *genesisV3) Execute(wc WorldContext) (Receipt, error) {
	// TODO Implement PreValidate
	panic("implement me")
}

func (g *genesisV3) Timestamp() int64 {
	return 0
}
