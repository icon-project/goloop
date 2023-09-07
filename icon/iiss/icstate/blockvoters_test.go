package icstate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

func _newDummyVoters(size int, reverse bool) []module.Address {
	voters := make([]module.Address, size)
	for i := 0; i < size; i++ {
		value := i
		if reverse {
			value = size - i
		}
		bs := make([]byte, common.AddressIDBytes)
		for j := common.AddressIDBytes - 1; value > 0 || j >= 0; j-- {
			bs[j] = byte(value & 0xff)
			value >>= 8
		}
		voters[i] = common.MustNewAddress(bs)
	}
	return voters
}
func newDummyVoters(size int) []module.Address {
	return _newDummyVoters(size, false)
}

func newDummyVotersReverse(size int) []module.Address {
	return _newDummyVoters(size, true)
}

func TestBlockVotersData_init(t *testing.T) {
	bbvd := newBlockVotersData(nil)
	assert.Zero(t, bbvd.Len())
	assert.Nil(t, bbvd.Get(0))
	assert.True(t, bbvd.IndexOf(common.MustNewAddressFromString("hx1")) < 0)
}

func TestBlockVotersData_equal(t *testing.T) {
	voters0 := newDummyVoters(5)
	voters1 := newDummyVoters(5)
	voters2 := newDummyVoters(3)
	voters2r := newDummyVotersReverse(3)

	bvd0 := newBlockVotersData(voters0)
	bvd1 := newBlockVotersData(voters1)
	bvd2 := newBlockVotersData(voters2)
	bvd2r := newBlockVotersData(voters2r)
	bvd3 := newBlockVotersData(nil)

	assert.True(t, bvd0.equal(bvd0))
	assert.True(t, bvd1.equal(bvd1))
	assert.True(t, bvd2.equal(bvd2))

	assert.True(t, bvd0.equal(bvd1))
	assert.True(t, bvd1.equal(bvd0))

	assert.False(t, bvd0.equal(bvd2))
	assert.False(t, bvd2.equal(bvd0))
	assert.False(t, bvd1.equal(bvd2))
	assert.False(t, bvd2.equal(bvd1))

	assert.False(t, bvd2.equal(bvd2r))
	assert.False(t, bvd2r.equal(bvd2))

	assert.False(t, bvd3.equal(bvd0))
	assert.False(t, bvd0.equal(bvd3))
}

func TestBlockVotersData_Get(t *testing.T) {
	size := 5
	voters := newDummyVoters(size)
	bvd := newBlockVotersData(voters)

	for i := 0; i < size; i++ {
		assert.NotNil(t, bvd.Get(i))
		assert.True(t, voters[i].Equal(bvd.Get(i)))
	}

	assert.Nil(t, bvd.Get(bvd.Len()))
	assert.Nil(t, bvd.Get(-1))
}

func TestBlockVotersData_Len(t *testing.T) {
	size := 5
	voters := newDummyVoters(size)
	bvd := newBlockVotersData(voters)
	assert.Equal(t, size, bvd.Len())

	bvd = newBlockVotersData(nil)
	assert.Zero(t, bvd.Len())
}

func TestBlockVotersData_IndexOf(t *testing.T) {
	size := 5
	voters := newDummyVoters(size)
	bvd := newBlockVotersData(voters)
	emptyBvd := newBlockVotersData(nil)

	for i := 0; i < size; i++ {
		assert.Equal(t, i, bvd.IndexOf(voters[i]))
		assert.True(t, emptyBvd.IndexOf(voters[i]) < 0)
	}

	invalidVoter := common.MustNewAddressFromString("hx1234")
	assert.True(t, bvd.IndexOf(invalidVoter) < 0)
}

func TestBlockVotersSnapshot_Equal(t *testing.T) {
	size := 3
	voters0 := newDummyVoters(size)
	bvs0 := NewBlockVotersSnapshot(voters0)

	voters1 := newDummyVoters(size)
	bvs1 := NewBlockVotersSnapshot(voters1)

	assert.True(t, bvs0.Equal(bvs0))
	assert.True(t, bvs1.Equal(bvs1))
	assert.True(t, bvs0.Equal(bvs1))
	assert.True(t, bvs1.Equal(bvs0))

	voters2 := newDummyVoters(2)
	bvs2 := NewBlockVotersSnapshot(voters2)

	assert.False(t, bvs0.Equal(bvs2))
	assert.False(t, bvs2.Equal(bvs0))

	bvs3 := NewBlockVotersSnapshot(nil)
	assert.False(t, bvs3.Equal(bvs0))
	assert.False(t, bvs0.Equal(bvs3))

	voters0r := newDummyVotersReverse(size)
	bvs0r := NewBlockVotersSnapshot(voters0r)
	assert.False(t, bvs0.Equal(bvs0r))
	assert.False(t, bvs0r.Equal(bvs0))

	bvs0r = nil
	assert.False(t, bvs0.Equal(bvs0r))
	bvs0 = nil
	assert.True(t, bvs0.Equal(bvs0r))
	assert.False(t, bvs0.Equal(bvs1))
}

func TestBlockVotersSnapshot_RLPDecodeFields(t *testing.T) {
	size := 5
	voters := newDummyVoters(size)
	bvs0 := NewBlockVotersSnapshot(voters)
	o0 := icobject.New(TypeBlockVoters, bvs0)

	buf := bytes.NewBuffer(nil)
	e := codec.BC.NewEncoder(buf)

	err := o0.RLPEncodeSelf(e)
	assert.NoError(t, err)

	err = e.Close()
	assert.NoError(t, err)

	tag := icobject.MakeTag(TypeBlockVoters, 0)
	bvs1 := NewBlockVotersWithTag(tag)
	o1 := icobject.New(TypeBlockVoters, bvs1)

	d := codec.BC.NewDecoder(buf)
	err = o1.RLPDecodeSelf(d, NewObjectImpl)
	assert.NoError(t, err)

	assert.True(t, o0.Equal(o1))
	assert.True(t, o1.Equal(o0))

	real0 := o0.Real().(*BlockVotersSnapshot)
	real1 := o1.Real().(*BlockVotersSnapshot)
	for i := 0; i < size; i++ {
		voter := voters[i]
		assert.True(t, voter.Equal(real0.Get(i)))
		assert.True(t, voter.Equal(real1.Get(i)))
	}
}
