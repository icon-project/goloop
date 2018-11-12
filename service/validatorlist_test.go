package service

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"testing"
)

func TestValidatorListBasic(t *testing.T) {
	addrs := []module.Address{
		common.NewAddressFromString("cx0000000000000000000000000000000000000000"),
		common.NewAddressFromString("hx0000000000000000000000000000000000000001"),
		common.NewAddressFromString("cx0000000000000000000000000000000000000002"),
	}
	validators := []module.Validator{}
	for _, a := range addrs {
		v, err := ValidatorFromAddress(a)
		if err != nil {
			t.Errorf("Fail to make validator addr=%s", a.String())
			return
		}
		validators = append(validators, v)
	}

	_, pubKey := crypto.GenerateKeyPair()
	v1, err := ValidatorFromPublicKey(pubKey.SerializeCompressed())
	if err != nil {
		t.Errorf("Fail to make validator from public key err=%+v", err)
		return
	}
	a1 := common.NewAccountAddressFromPublicKey(pubKey)
	addrs = append(addrs, a1)
	validators = append(validators, v1)

	mdb := db.NewMapDB()
	vList1, err := ValidatorListFromSlice(mdb, validators)
	if err != nil {
		t.Errorf("Fail to make validatorList from slice err=%+v", err)
		return
	}
	if err := vList1.Flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
		return
	}

	vList2, err := ValidatorListFromHash(mdb, vList1.Hash())
	if err != nil {
		t.Errorf("Fail to make validatorList from hash err=%+v", err)
		return
	}

	addrErr := common.NewAddressFromString("cx9999999999999999999999999999999999999999")
	if ridx := vList2.IndexOf(addrErr); ridx >= 0 {
		t.Errorf("Invalid index value(%d) with invalid address", ridx)
		return
	}
	for i, a := range addrs {
		ridx := vList2.IndexOf(a)
		if ridx != i {
			t.Errorf("Invalid index ret=%d exp=%d", ridx, i)
			return
		}
	}

	if v, ok := vList2.Get(-1); ok {
		t.Errorf("Accessing invalid index returns valid obj=%v", v)
		return
	}
	if v, ok := vList2.Get(vList2.Len()); ok {
		t.Errorf("Accessing invalid index returns valid obj=%v", v)
		return
	}
	for i := 0; i < vList2.Len(); i++ {
		v, ok := vList2.Get(i)
		if !ok {
			t.Errorf("Fail to get validator from list for idx=%d", i)
			return
		}
		if !v.Address().Equal(addrs[i]) {
			t.Errorf("Returned validator has different address")
			return
		}
	}
}
