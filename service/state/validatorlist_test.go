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
	return newDummyValidatorsFrom(0, size)
}

func newDummyValidatorsFrom(from, size int) []module.Validator {
	validators := make([]module.Validator, size)
	for i := 0; i < size; i++ {
		validators[i] = newDummyValidator(from+i)
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
	t.Logf("Validators: %s", vList2)

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

	// Error case: Replace non-existent validator with same
	ov = newDummyValidator(1234)
	nv = newDummyValidator(1234)
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

func TestValidatorState_Set(t *testing.T) {
	size := 5
	validators := newDummyValidators(size)

	dbase := db.NewMapDB()
	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	for i := 0; i < size ; i++ {
		assert.Equal(t, i, vss.IndexOf(validators[i].Address()))
	}

	vs := ValidatorStateFromSnapshot(vss)

	validators2 := newDummyValidatorsFrom(size, size)
	err = vs.Set(validators2)
	assert.NoError(t, err)

	for i := 0; i < size ; i++ {
		assert.Equal(t, i, vs.IndexOf(validators2[i].Address()))
	}

	err = vs.Set(validators)
	assert.NoError(t, err)

	for i := 0; i < size ; i++ {
		assert.Equal(t, -1, vs.IndexOf(validators2[i].Address()))
		assert.Equal(t, i, vs.IndexOf(validators[i].Address()))
	}

	vss2 := vs.GetSnapshot()
	assert.NoError(t, vss2.Flush())

	vss3, err := ValidatorSnapshotFromHash(dbase, vss2.Hash())
	assert.NoError(t, err)

	for i := 0; i < size ; i++ {
		assert.Equal(t, -1, vss3.IndexOf(validators2[i].Address()))
		assert.Equal(t, i, vss3.IndexOf(validators[i].Address()))
	}

	// test failure for duplicated validators
	validators3 := newDummyValidators(size)
	validators3 = append(validators3, validators3[1])

	err = vs.Set(validators3)
	assert.Error(t, err)

	for i := 0; i < size ; i++ {
		assert.Equal(t, -1, vs.IndexOf(validators2[i].Address()))
		assert.Equal(t, i, vs.IndexOf(validators[i].Address()))
	}
}

func TestValidatorState_GetSnapshot(t *testing.T) {
	size := 5
	validators := newDummyValidators(size)
	dbase := db.NewMapDB()

	vss, err := ValidatorSnapshotFromSlice(dbase, validators)
	assert.NoError(t, err)
	assert.Equal(t, size, vss.Len())

	vs := ValidatorStateFromSnapshot(vss)

	assert.True(t, vss == vs.GetSnapshot())
	for i, v := range validators {
		assert.Equal(t, i, vs.IndexOf(v.Address()))
		assert.Equal(t, i, vss.IndexOf(v.Address()))
	}

	validator1 := newDummyValidator(size)
	err = vs.Add(validator1)
	assert.NoError(t, err)

	validator2 := newDummyValidator(size+1)
	assert.False(t, vs.Remove(validator2))
	assert.NoError(t, vs.Add(validator2))

	// making new validators (but same content)
	validators2 := newDummyValidators(size+2)

	vss2 := vs.GetSnapshot()
	assert.False(t, vss == vss2)

	for i, v := range validators2 {
		err = vs.SetAt(i, v)
		assert.NoError(t, err)
		assert.True(t, vss2 == vs.GetSnapshot())
	}

	for _, v := range validators2 {
		err = vs.Replace(v, v)
		assert.NoError(t, err)
		assert.True(t, vss2 == vs.GetSnapshot())
	}

	err = vs.Set(validators2)
	assert.NoError(t, err)
	assert.True(t, vss2 == vs.GetSnapshot())

	vs.Reset(vss)
	assert.True(t, vss == vs.GetSnapshot())
	vs.Reset(vss)
	assert.True(t, vss == vs.GetSnapshot())
}

func TestNewValidatorListFromBytes(t *testing.T) {
	dbase := db.NewMapDB()
	vs := newDummyValidatorsFrom(100, 20)
	vss, err := ValidatorSnapshotFromSlice(dbase, vs)
	assert.NoError(t, err)

	vl1, err := NewValidatorListFromBytes(vss.Bytes())
	assert.NoError(t, err)
	for i, v := range vs {
		i2 := vl1.IndexOf(v.Address())
		assert.Equal(t, i, i2)
		v2, ok := vl1.Get(i)
		assert.True(t, ok)
		assert.Equal(t, v.Address(), v2.Address())
	}
	assert.Equal(t, vss.Bytes(), vl1.Bytes())

	vl2, err := ToValidatorList(vss)
	assert.NoError(t, err)
	for i, v := range vs {
		i2 := vl1.IndexOf(v.Address())
		assert.Equal(t, i, i2)
	}
	assert.Equal(t, vss.Bytes(), vl2.Bytes())
	assert.Equal(t, vss.Hash(), vl2.Hash())

	_, err = NewValidatorListFromBytes([]byte{0x00,0x12})
	assert.Error(t, err)

	_, err = NewValidatorListFromBytes([]byte{0xC0,0x12})
	assert.Error(t, err)
}

func TestToValidatorList(t *testing.T) {
	dbase := db.NewMapDB()

	// nil validator list conversion
	vl0, err := ToValidatorList(nil)
	assert.NoError(t, err)
	assert.Nil(t, vl0)

	// empty validator list conversion
	vss, err := ValidatorSnapshotFromSlice(dbase, nil)
	assert.NoError(t, err)

	vl1, err := ToValidatorList(vss)
	assert.NoError(t, err)
	assert.Equal(t, 0, vl1.Len())
	assert.Equal(t, vss.Hash(), vl1.Hash())

	// normal validator list conversion
	vs := newDummyValidatorsFrom(100, 20)
	vss, err = ValidatorSnapshotFromSlice(dbase, vs)
	assert.NoError(t, err)

	// may differ but not required.
	vl2, err := ToValidatorList(vss)
	assert.NoError(t, err)

	// should be same
	vl3, err := ToValidatorList(vl2)
	assert.NoError(t, err)
	assert.Equal(t, vl2, vl3)
}

func TestValidatorSnapshotFromList(t *testing.T) {
	dbase := db.NewMapDB()

	vss0, err := ValidatorSnapshotFromList(dbase, nil)
	assert.NoError(t, err)
	assert.Nil(t, vss0)

	vs := newDummyValidatorsFrom(100, 20)
	vss1, err := ValidatorSnapshotFromSlice(dbase, vs)
	assert.NoError(t, err)

	vl0, err := ToValidatorList(vss1)
	assert.NoError(t, err)
	assert.NotNil(t, vl0)

	vss2, err := ValidatorSnapshotFromList(dbase, vl0)
	err = vss2.Flush()
	assert.NoError(t, err)

	vss3, err := ValidatorSnapshotFromHash(dbase, vss2.Hash())
	assert.NoError(t, err)

	assert.EqualValues(t, vl0.Len(), vss1.Len(), vss2.Len(), vss3.Len())
	for i:=0 ; i < vl0.Len() ; i++ {
		v0, ok := vl0.Get(i)
		assert.True(t, ok)
		v1, ok := vss1.Get(i)
		assert.True(t, ok)
		v2, ok := vss2.Get(i)
		assert.True(t, ok)
		v3, ok := vss3.Get(i)
		assert.True(t, ok)

		assert.EqualValues(t, v0, v1, v2, v3)
	}

	vss4, err := ValidatorSnapshotFromList(dbase, vss3)
	assert.NoError(t, err)
	assert.Equal(t, vss3, vss4)
}