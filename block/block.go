package block

import (
	"bytes"
	"errors"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/module"
)

func verifyBlock(b module.BlockData, prev module.BlockData, prevVoters module.ValidatorList) ([]bool, error) {
	if b.Height() != prev.Height()+1 {
		return nil, errors.New("bad height")
	}
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return nil, errors.New("bad prev ID")
	}
	var voted []bool
	if vt, err := b.Votes().VerifyBlock(prev, prevVoters); err != nil {
		return nil, err
	} else {
		voted = vt
	}

	if err := b.(base.BlockVersionSpec).VerifyTimestamp(prev, prevVoters); err != nil {
		return nil, err
	}
	return voted, nil
}
