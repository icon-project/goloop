package icstate

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type ownerToNodeMapper struct {
	o2n map[string]module.Address
	n2o map[string]module.Address
}

func (m *ownerToNodeMapper) add(owner module.Address, node module.Address) {
	if owner == nil || node == nil {
		return
	}
	if m.o2n == nil {
		m.o2n = make(map[string]module.Address)
	}
	if m.n2o == nil {
		m.n2o = make(map[string]module.Address)
	}
	m.o2n[icutils.ToKey(owner)] = node
	m.o2n[icutils.ToKey(node)] = owner
}

func (m *ownerToNodeMapper) GetNodeByOwner(owner module.Address) module.Address {
	node, ok := m.o2n[icutils.ToKey(owner)]
	if !ok {
		return nil
	}
	return node
}

func (m *ownerToNodeMapper) GetOwnerByNode(node module.Address) module.Address {
	owner, ok := m.n2o[icutils.ToKey(node)]
	if !ok {
		return nil
	}
	return owner
}

func newDummyOwnerToNodeMapper(size int) OwnerToNodeMappable {
	m := new(ownerToNodeMapper)
	for i := 0; i < size; i++ {
		owner := newDummyAddress(i)
		node := newDummyAddress(i + 100)
		m.add(owner, node)
	}
	return m
}

func newDummyValidatorsData(size int) *validatorsData {
	snapshots := newDummyPRepSnapshots(size)
	m := newDummyOwnerToNodeMapper(size)
	vd := new(validatorsData)
	vd.init(snapshots, m, size)
	return vd
}

func newDummyValidatorSnapshot(size int) *ValidatorsSnapshot {
	nodes := make([]module.Address, size)
	for i := 0; i < size; i++ {
		nodes[i] = newDummyAddress(i)
	}

	vd := newValidatorsData(nodes)
	return &ValidatorsSnapshot{
		validatorsData: vd,
	}
}

func TestValidatorsData_init(t *testing.T) {
	size := 10
	snapshots := newDummyPRepSnapshots(size)
	m := newDummyOwnerToNodeMapper(size)

	type a struct {
		m int
		n string
	}

	b := a{1, "hello"}
	c := a{2, "w"}
	b.m = c.m
	b.n = c.n

	vd := validatorsData{}
	vd.init(snapshots, m, size)
	assert.Equal(t, size, vd.Len())

	for i, snapshot := range snapshots {
		node := m.GetNodeByOwner(snapshot.Owner())

		node2 := vd.Get(i)
		assert.True(t, node.Equal(node2))
		assert.Equal(t, i, vd.IndexOf(node))
	}

	assert.Equal(t, size, vd.NextPRepSnapshotIndex())
}

func TestValidatorsData_clone(t *testing.T) {
	size := 22
	vd := newDummyValidatorsData(size)
	vd2 := vd.clone()
	assert.True(t, vd.equal(&vd2))
}

func TestValidatorsSnapshot_RLPEncodeDecode(t *testing.T) {
	state := newDummyState(false)

	size := 10
	vss := newDummyValidatorSnapshot(size)
	assert.Equal(t, size, vss.Len())

	err := state.SetValidatorsSnapshot(vss)
	assert.NoError(t, err)

	state = flushAndNewState(state, false)

	vss2 := state.GetValidatorsSnapshot()
	assert.NotNil(t, vss2)

	assert.True(t, vss.Equal(vss2))
	assert.Equal(t, size, vss2.Len())
}

func TestNewValidatorStateWithSnapshot(t *testing.T) {
	var snapshot *ValidatorsSnapshot
	vs := NewValidatorsStateWithSnapshot(snapshot)
	assert.Zero(t, vs.Len())
	assert.False(t, vs.IsDirty())
}

func TestValidatorsState_Set(t *testing.T) {
	size := 22
	nextPssIdx := size
	bh := int64(100)
	vss := newDummyValidatorSnapshot(size)
	vs := NewValidatorsStateWithSnapshot(vss)
	assert.False(t, vs.IsDirty())

	for i := 0; i < size; i++ {
		node := vs.Get(i)
		assert.NotNil(t, node)
		vs.Set(bh, i, nextPssIdx, node)
		assert.False(t, vs.IsDirty())
	}

	for i := 0; i < size; i++ {
		newNode := newDummyAddress(999 + i)
		oldNode := vs.Get(i)
		assert.False(t, oldNode.Equal(newNode))

		vs.Set(bh, i, -1, newNode)
		assert.True(t, vs.IsDirty())
		assert.True(t, vs.Get(i).Equal(newNode))

		vss2 := vs.GetSnapshot()
		assert.True(t, vss2.IsUpdated(bh))
		assert.False(t, vss2.IsUpdated(bh+1))
		assert.Equal(t, size, vss2.Len())
		assert.Panicsf(t, func() { vss2.IsUpdated(bh - 1) }, "ValidatorsState.IsUpdate() did not panic")

	}

	for i := 0; i < size; i++ {
		idx := rand.Intn(vs.Len())
		vs.Set(bh, idx, -1, nil)
		assert.Equal(t, size-i-1, vs.Len())

		vss2 := vs.GetSnapshot()
		assert.True(t, vss2.IsUpdated(bh))
		assert.False(t, vss2.IsUpdated(bh+1))
		assert.Panicsf(t, func() { vss2.IsUpdated(bh - 1) }, "ValidatorsState.IsUpdate() did not panic")
	}
}
