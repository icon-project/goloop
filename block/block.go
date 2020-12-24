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
	if vt, err := b.Votes().Verify(prev, validators); err != nil {
		return nil, err
	} else {
		voted = vt
	}

	if b.Height() > 1 && b.Timestamp() != b.Votes().Timestamp() {
		return nil, errors.New("bad timestamp")
	}
	return voted, nil
}
