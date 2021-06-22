package block

import (
	"bytes"
	"errors"

	"github.com/icon-project/goloop/module"
)

func verifyBlock(b module.BlockData, prev module.BlockData, validators module.ValidatorList) ([]bool, error) {
	if b.Height() != prev.Height()+1 {
		return nil, errors.New("bad height")
	}
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return nil, errors.New("bad prev ID")
	}
	var voted []bool
	if vt, err := b.Votes().VerifyBlock(prev, validators); err != nil {
		return nil, err
	} else {
		voted = vt
	}

	if tcvs, ok := b.Votes().(module.TimestampedCommitVoteSet); ok {
		if b.Height() > 1 && b.Timestamp() != tcvs.Timestamp() {
			return nil, errors.New("bad timestamp")
		}
	}
	if b.Height() > 1 && prev.Timestamp() >= b.Timestamp() {
		return nil, errors.New("non-increasing timestamp")
	}
	return voted, nil
}
