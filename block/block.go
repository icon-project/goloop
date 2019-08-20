package block

import (
	"bytes"
	"errors"

	"github.com/icon-project/goloop/module"
)

func verifyBlock(b module.BlockData, prev module.BlockData, validators module.ValidatorList) error {
	if b.Height() != prev.Height()+1 {
		return errors.New("bad height")
	}
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return errors.New("bad prev ID")
	}
	if err := b.Votes().Verify(prev, validators); err != nil {
		return err
	}

	if b.Height() > 1 && b.Timestamp() != b.Votes().Timestamp() {
		return errors.New("bad timestamp")
	}
	return nil
}
