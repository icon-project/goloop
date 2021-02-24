package state

import (
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestValidatorListBasic(t *testing.T) {
	addrs := []module.Address{
		common.MustNewAddressFromString("hx0000000000000000000000000000000000000000"),
		common.MustNewAddressFromString("hx0000000000000000000000000000000000000001"),
		common.MustNewAddressFromString("hx0000000000000000000000000000000000000002"),
	}
	var validators []module.Validator
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
	vList1, err := ValidatorSnapshotFromSlice(mdb, validators)
	if err != nil {
		t.Errorf("Fail to make validatorList from slice err=%+v", err)
		return
	}
	if err := vList1.Flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
		return
	}

	vList2, err := ValidatorStateFromHash(mdb, vList1.Hash())
	if err != nil {
		t.Errorf("Fail to make validatorList from hash err=%+v", err)
		return
	}

	addrErr := common.MustNewAddressFromString("cx9999999999999999999999999999999999999999")
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

func checkEmpty(t *testing.T, vl module.ValidatorList) {
	addr := common.MustNewAddressFromString("cx7777777777777777777777777777777777777777")
	if idx := vl.IndexOf(addr); idx != -1 {
		t.Errorf("Invalid result on IndexOf() for empty list")
		return
	}

	if _, ok := vl.Get(0); ok {
		t.Errorf("Invalid result on Get(0) for empty list")
		return
	}

	if l := vl.Len(); l != 0 {
		t.Errorf("Invalid result on Len() ret=%d", l)
		return
	}

	if hash := vl.Hash(); hash != nil {
		t.Errorf("Invalid result on Hash() ret=%x", hash)
	}
}

func TestEmptyValidatorList(t *testing.T) {
	dbase := db.NewMapDB()
	if vl, err := ValidatorSnapshotFromHash(dbase, nil); vl == nil || err != nil {
		t.Errorf("Fail to make ValidatorList from nil hash err=%+v", err)
		return
	} else {
		checkEmpty(t, vl)
	}
	if vl, err := ValidatorSnapshotFromSlice(dbase, nil); vl == nil || err != nil {
		t.Errorf("Fail to make ValidatorList from nil slice err=%+v", err)
		return
	} else {
		checkEmpty(t, vl)
	}

	if vl, err := ValidatorSnapshotFromSlice(dbase, []module.Validator{}); vl == nil || err != nil {
		t.Errorf("Fail to make ValidatorList from nil slice err=%+v", err)
		return
	} else {
		checkEmpty(t, vl)
	}
}
