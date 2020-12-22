package icstate

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"math/big"
)

var termVarPrefix = containerdb.ToKey(containerdb.RawBuilder, "term")

type Term struct {
	icobject.NoDatabase
	StateAndSnapshot

	sequence        int
	startHeight     int64
	period          int
	irep            int
	totalSupply     *big.Int
	totalDelegation *big.Int
	prepSnapshots   []*PRepSnapshot
}

type PRepSnapshot struct {
	owner     module.Address
	delegated *big.Int
}

func (term *Term) Version() int {
	return 0
}

func (term *Term) RLPDecodeFields(decoder codec.Decoder) error {
	return nil
}

func (term* Term) RLPEncodeFields(encoder codec.Encoder) error {
	return nil
}

func (term* Term) Equal(o icobject.Impl) bool {
	return false
}
