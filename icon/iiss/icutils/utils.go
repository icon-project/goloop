package icutils

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

var BigIntICX = big.NewInt(1_000_000_000_000_000_000)

func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	size := len(maps)
	if size == 0 {
		return nil
	}

	ret := maps[0]
	for i := 1; i < size; i++ {
		for k, v := range maps[i] {
			ret[k] = v
		}
	}

	return ret
}

func ToKey(o interface{}) string {
	switch o := o.(type) {
	case module.Address:
		return string(o.Bytes())
	case []byte:
		return string(o)
	default:
		panic(errors.Errorf("Unsupported type: %v", o))
	}
}

func EqualAddress(a1 module.Address, a2 module.Address) bool {
	if a1 == a2 {
		return true
	}

	if a1 != nil {
		return a1.Equal(a2)
	} else if a2 != nil {
		return a2.Equal(a1)
	}

	return false
}

func GetTotalSupply(ws state.WorldState) *big.Int {
	wss := ws.GetSnapshot()
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := tsVar.BigInt()
	return ts
}

func Min(value1, value2 int) int {
	if value1 < value2 {
		return value1
	} else {
		return value2
	}
}


func BigInt2HexInt(value *big.Int) *common.HexInt {
	h := new(common.HexInt)
	h.Set(value)
	return h
}
