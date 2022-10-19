package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitArray_PickRandom(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		len    int
		ones   []int
		repeat int
		expect int
	}{
		{10, []int{}, 2, -1},
		{10, []int{0}, 2, 0},
		{10, []int{1}, 2, 1},
		{10, []int{9}, 2, 9},
		{65, []int{0}, 2, 0},
		{65, []int{62}, 2, 62},
		{65, []int{63}, 2, 63},
		{65, []int{64}, 2, 64},
	}
	for _, c := range testCases {
		ba := newBitArray(c.len)
		for i := 0; i < len(c.ones); i++ {
			ba.Set(c.ones[i])
			assert.True(ba.Get(c.ones[i]))
		}
		for i := 0; i < c.repeat; i++ {
			v := ba.PickRandom()
			assert.Equal(c.expect, v)
		}
	}
}

func TestBitArray_PickRandom2(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		len  int
		ones []int
	}{
		{100, []int{}},
		{100, []int{1}},
		{100, []int{31}},
		{100, []int{63}},
		{100, []int{1, 2, 3, 4, 5}},
		{100, []int{1, 2, 3, 4, 5, 63, 64, 77}},
		{500, []int{64, 77, 399}},
	}
	for _, c := range testCases {
		ba := newBitArray(c.len)
		for i := 0; i < len(c.ones); i++ {
			ba.Set(c.ones[i])
			assert.True(ba.Get(c.ones[i]))
		}
		baCopy := ba.Copy()
		ba2 := newBitArray(c.len)
		for i := 0; i < len(c.ones); i++ {
			v := ba.PickRandom()
			assert.True(ba.Get(v))
			assert.False(ba2.Get(v))
			ba.Unset(v)
			ba2.Set(v)
		}
		assert.Equal(-1, ba.PickRandom())
		assert.True(baCopy.Equal(ba2))
	}
}

func TestBitArray_Basics(t *testing.T) {
	assert := assert.New(t)
	const bits = 10
	ba := newBitArray(bits)

	assert.EqualValues(bits, ba.Len())
	assert.EqualValues(false, ba.Get(2))
	ba.Put(2, true)
	assert.EqualValues(true, ba.Get(2))
	ba.Put(2, false)
	assert.EqualValues(false, ba.Get(2))
	assert.EqualValues(false, ba.Get(11))

	ba2 := newBitArray(bits + 1)
	ba2.AssignAnd(ba)
	assert.EqualValues(bits, ba2.Len())

	ba2 = newBitArray(bits + 1)
	assert.False(ba.Equal(ba2))

	ba2 = ba.Copy()
	assert.True(ba.Equal(ba2))
	ba2.Set(3)
	assert.False(ba.Equal(ba2))
}
