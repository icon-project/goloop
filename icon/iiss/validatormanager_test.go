package iiss

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

func newAddress(idx int) module.Address {
	value := fmt.Sprintf("hx%02x", idx)
	return common.MustNewAddressFromString(value)
}

func newDummyValidatorManager(t *testing.T, size int) *ValidatorManager {
	vm := NewValidatorManager()

	for i := 0; i < size; i++ {
		address := newAddress(i)
		err := vm.add(address, true)
		assert.NoError(t, err)

		vi, _ := vm.Get(i)
		assert.True(t, vi.IsAdded())
	}
	assert.Equal(t, size, vm.Len())

	return vm
}

func TestNewValidatorManager(t *testing.T) {
	vm := NewValidatorManager()
	assert.Zero(t, vm.Len())
}

func TestValidatorManager_add(t *testing.T) {
	size := 10
	vm := newDummyValidatorManager(t, size)
	assert.Equal(t, size, vm.Len())
}

func TestValidatorManager_GetValidators(t *testing.T) {
	size := 22
	vm := newDummyValidatorManager(t, size)

	vs, err := vm.GetValidators()
	assert.NoError(t, err)
	assert.Equal(t, size, len(vs))

	for i := 0; i < size; i++ {
		vi, _ := vm.Get(i)
		assert.True(t, vs[i].Address().Equal(vi.Address()))
	}
}

func TestValidatorManager_Remove(t *testing.T) {
	var err error
	size := 22
	vm := newDummyValidatorManager(t, size)

	idx := size / 2
	address := newAddress(idx)

	v, ok := vm.Get(idx)
	assert.True(t, ok)
	assert.True(t, address.Equal(v.Address()))

	i := vm.IndexOf(address)
	assert.Equal(t, idx, i)

	err = vm.Remove(address)
	assert.NoError(t, err)
	assert.Equal(t, size-1, vm.Len())

	idx = vm.IndexOf(address)
	assert.True(t, idx < 0)

	err = vm.Remove(address)
	assert.Error(t, err)

	idx = vm.IndexOf(address)
	assert.True(t, idx < 0)
}

func TestValidatorManager_Replace(t *testing.T) {
	size := 22
	vm := newDummyValidatorManager(t, size)
	assert.Equal(t, size, vm.Len())

	for i := 0; i < size; i++ {
		v, ok := vm.Get(i)
		assert.NotNil(t, v)
		assert.True(t, ok)

		newAddr := newAddress(i)
		err := vm.Replace(v.Address(), newAddr)
		assert.Error(t, err)
	}

	for i := 0; i < size; i++ {
		v, ok := vm.Get(i)
		assert.NotNil(t, v)
		assert.True(t, ok)

		newAddr := newAddress(i + 100)
		err := vm.Replace(v.Address(), newAddr)
		assert.NoError(t, err)

		idx := vm.IndexOf(newAddr)
		assert.Equal(t, i, idx)

		idx = vm.IndexOf(v.Address())
		assert.True(t, idx < 0)

		v, ok = vm.Get(i)
		assert.True(t, ok)
		assert.True(t, newAddr.Equal(v.Address()))
	}
}

func TestValidatorManager_Clear(t *testing.T) {
	size := 22
	vm := newDummyValidatorManager(t, size)
	assert.Equal(t, size, vm.Len())

	err := vm.Clear()
	assert.False(t, vm.IsUpdated())
	assert.NoError(t, err)
	assert.Zero(t, vm.Len())
}
