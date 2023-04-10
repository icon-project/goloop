package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func newDummyValidators(size int) []module.Validator {
	validators := make([]module.Validator, size)
	for i := 0; i < size; i++ {
		validators[i] = newDummyValidator(i)
	}
	return validators
}

func newDummyValidator(id int) module.Validator {
	addr := newDummyAddress(id)
	v, _ := ValidatorFromAddress(addr)
	return v
}

func newDummyAddress(id int) module.Address {
	b := make([]byte, 21)
	for i := 20; id > 0 && i > 0; i-- {
		b[i] = byte(id & 0xff)
		id >>= 8
	}
	return common.MustNewAddress(b)
}

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

func TestValidatorState_Add(t *testing.T) {
	size := 5
	var nv module.Validator
	validators := newDummyValidators(size)

	dbase := db.NewMapDB()
	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	vs := ValidatorStateFromSnapshot(vss)
	assert.Equal(t, size, vs.Len())
	assert.Equal(t, vs.GetSnapshot(), vss)

	// Error case: add already existent validator
	nv = newDummyValidator(0)
	err = vs.Add(nv)
	assert.NoError(t, err)
	assert.Equal(t, size, vs.Len())
	assert.Equal(t, vs.GetSnapshot(), vss)

	// Success case
	idx := 5
	nv = newDummyValidator(100)
	err = vs.Add(nv)
	assert.NoError(t, err)
	assert.Equal(t, idx, vs.IndexOf(nv.Address()))
	v, ok := vs.Get(idx)
	assert.True(t, ok)
	assert.True(t, v.Address().Equal(nv.Address()))
	assert.True(t, vss != vs.GetSnapshot())

	for i, v := range validators {
		v2, ok := vs.Get(i)
		assert.True(t, ok)
		assert.True(t, v.Address().Equal(v2.Address()))
	}
}

func TestValidatorState_Remove(t *testing.T) {
	size := 5
	var idx int
	validators := newDummyValidators(size)

	dbase := db.NewMapDB()
	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	vs := ValidatorStateFromSnapshot(vss)
	assert.Equal(t, size, vs.Len())

	v := newDummyValidator(100)
	err = vs.Add(v)
	assert.NoError(t, err)
	validators = append(validators, v)

	v = validators[2]
	idx = vs.IndexOf(v.Address())
	assert.Equal(t, 2, idx)

	ok := vs.Remove(v)
	assert.True(t, ok)
	assert.Equal(t, size, vs.Len())

	expIdx := 0
	for i, v := range validators {
		if i == 2 {
			continue
		}
		if i < 2 {
			expIdx = i
		} else {
			expIdx = i - 1
		}
		idx = vs.IndexOf(validators[i].Address())
		assert.Equal(t, expIdx, idx)

		v2, ok := vs.Get(idx)
		assert.True(t, ok)
		assert.True(t, v2.Address().Equal(v.Address()))
	}

	assert.True(t, vss != vs.GetSnapshot())
}

func TestValidatorState_Replace(t *testing.T) {
	size := 5
	var ov, nv module.Validator
	validators := newDummyValidators(size)

	dbase := db.NewMapDB()
	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	vs := ValidatorStateFromSnapshot(vss)
	assert.Equal(t, size, vs.Len())

	// Error case: Replace a non-existent validator
	ov = newDummyValidator(1234)
	nv = newDummyValidator(101)
	err = vs.Replace(ov, nv)
	assert.Error(t, err)
	assert.Equal(t, vss, vs.GetSnapshot())

	for i := 0; i < size; i++ {
		ev := validators[i]
		v, ok := vs.Get(i)
		assert.True(t, ok)
		assert.True(t, v.Address().Equal(ev.Address()))
	}

	// Replace a validator with the same one
	err = vs.Replace(validators[2], validators[2])
	assert.NoError(t, err)
	for i := 0; i < size; i++ {
		ev := validators[i]
		v, ok := vs.Get(i)
		assert.True(t, ok)
		assert.True(t, v.Address().Equal(ev.Address()))
	}
	assert.Equal(t, vss, vs.GetSnapshot())

	// Success case: Replace the validator indicated by idx with new one
	idx := 1
	ov, _ = vs.Get(idx)
	err = vs.Replace(ov, nv)
	assert.NoError(t, err)
	v, ok := vs.Get(idx)
	assert.True(t, ok)
	assert.True(t, v.Address().Equal(nv.Address()))
	assert.True(t, vs.IndexOf(ov.Address()) < 0)

	for i := 0; i < vs.Len(); i++ {
		v, ok = vs.Get(i)
		assert.True(t, ok)

		ev := validators[i]
		if i == 1 {
			ev = nv
		}

		assert.True(t, v.Address().Equal(ev.Address()))
	}
}

func TestValidatorState_SetAt(t *testing.T) {
	size := 5
	validators := newDummyValidators(size)
	nv := newDummyValidator(1234)

	dbase := db.NewMapDB()
	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	vs := ValidatorStateFromSnapshot(vss)
	assert.Equal(t, size, vs.Len())

	// Error case: Index out of range
	err = vs.SetAt(-2, nv)
	assert.Error(t, err)
	assert.Equal(t, vss, vs.GetSnapshot())
	err = vs.SetAt(size, nv)
	assert.Error(t, err)
	assert.Equal(t, vss, vs.GetSnapshot())

	// Error case: already existent validator
	err = vs.SetAt(0, validators[1])
	assert.Error(t, err)
	assert.Equal(t, vss, vs.GetSnapshot())

	// Success case: replace a validator with the same one
	for i := 0; i < size; i++ {
		err = vs.SetAt(i, validators[i])
		assert.NoError(t, err)
		assert.Equal(t, vss, vs.GetSnapshot())
	}

	// Success case: replace a validator with new one
	for i := 0; i < size; i++ {
		ov, ok := vs.Get(i)
		nv = newDummyValidator(i + 100)
		err = vs.SetAt(i, nv)
		assert.NoError(t, err)

		v, ok := vs.Get(i)
		assert.True(t, ok)
		assert.True(t, v.Address().Equal(nv.Address()))

		assert.Equal(t, i, vs.IndexOf(nv.Address()))
		assert.True(t, vs.IndexOf(ov.Address()) < 0)
	}
}
