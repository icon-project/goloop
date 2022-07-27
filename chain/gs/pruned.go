package gs

import (
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/service/transaction"
)

type PrunedGenesis struct {
	CID    common.HexInt32 `json:"cid"`
	NID    common.HexInt32 `json:"nid"`
	Height common.HexInt64 `json:"height"`
	Block  common.HexBytes `json:"block"`
	Votes  common.HexBytes `json:"votes"`
}

func (g *PrunedGenesis) Verify() error {
	if g.NID.Value == 0 {
		return transaction.InvalidGenesisError.New("NIDIsZero")
	}
	if g.CID.Value == 0 {
		return transaction.InvalidGenesisError.New("CIDIsZero")
	}
	if len(g.Block) != crypto.HashLen {
		return transaction.InvalidGenesisError.Errorf("InvalidBlockID(id=%x)", g.Block)
	}
	if len(g.Votes) != crypto.HashLen {
		return transaction.InvalidGenesisError.Errorf("InvalidVotes(hash=%x)", g.Votes)
	}
	return nil
}

func NewPrunedGenesis(js []byte) (*PrunedGenesis, error) {
	g := new(PrunedGenesis)
	if err := json.Unmarshal(js, g); err != nil {
		return nil, err
	}
	return g, g.Verify()
}
