package scoredb

import (
	"github.com/icon-project/goloop/common/crypto"
)

type VarDB struct {
	WritableValue
}

func NewVarDB(store StateStore, key interface{}) *VarDB {
	value := NewValueFromStore(store,
		crypto.SHA3Sum256(ToKey(VarDBPrefix, key)))
	return &VarDB{value}
}
