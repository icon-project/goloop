package network

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

func generatePeer() *Peer {
	id := generatePeerID()
	na := generateNetAddress()
	return &Peer{id: id, netAddress: na}
}

func generatePeerID() module.PeerID {
	_, pubK := crypto.GenerateKeyPair()
	return NewPeerIDFromPublicKey(pubK)
}

func generateNetAddress() NetAddress {
	return NetAddress(fmt.Sprintf("127.0.0.1:%d", rand.Intn(65536)))
}

func Test_set_PeerSet(t *testing.T) {
	s := NewPeerSet()

	v1 := generatePeer()
	v2 := generatePeer()
	v2_1 := &Peer{id: v2.id, netAddress: v2.netAddress}
	v2_2 := &Peer{id: v2.id, netAddress: v2.netAddress, incomming: true}
	v3 := generatePeer()

	assert.Equal(t, true, s.IsEmpty(), "true")
	assert.Equal(t, true, s.Add(v1), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.Equal(t, false, s.IsEmpty(), "false")
	assert.Equal(t, true, s.Add(v2), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, false, s.Add(v2_1), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, true, s.Add(v2_2), "true")
	assert.Equal(t, 3, s.Len(), "3")
	assert.Equal(t, true, s.Contains(v1), "true")
	assert.Equal(t, false, s.Contains(v3), "false")

	assert.Equal(t, true, s.HasNetAddresse(v2.netAddress), "true")
	assert.Equal(t, false, s.HasNetAddresse(v3.netAddress), "false")
	t.Log(s.NetAddresses())

	t.Log(s.Array())
}

func Test_set_NetAddressSet(t *testing.T) {
	s := NewNetAddressSet()
	v1 := generatePeer()
	v1_1 := &Peer{id: v1.id, netAddress: generateNetAddress()}
	v2 := &Peer{id: generatePeerID(), netAddress: v1_1.netAddress}

	// assert.Equal(t, true, s.IsEmpty(), "true")
	// assert.Equal(t, true, s.Add(v1.netAddress), "true")
	// assert.Equal(t, 1, s.Len(), "1")
	// assert.Equal(t, false, s.IsEmpty(), "false")
	// assert.Equal(t, "", s.Map()[v1.netAddress], "empty string")
	// assert.Equal(t, false, s.Add(v1.netAddress), "false")
	// assert.Equal(t, 1, s.Len(), "1")
	// t.Log(s.Map())

	//When Peer connected
	o, r := s.PutByPeer(v1)
	assert.EqualValues(t, []interface{}{"", NetAddress("")}, []interface{}{o, r}, "empty NetAddress")
	assert.Equal(t, true, s.Map()[v1.netAddress] == v1.id.String(), v1.id.String())
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	//Update NetAddress, NetAddressSet.PutByPeer returns old NetAddress
	o, r = s.PutByPeer(v1_1)
	assert.EqualValues(t, []interface{}{"", v1.netAddress}, []interface{}{o, r}, "empty NetAddress")
	assert.Equal(t, v1_1.id.String(), s.Map()[v1_1.netAddress], v1_1.id.String())
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	//When Peer connected with same NetAddress, NetAddressSet.PutByPeer returns conflict PeerID
	o, r = s.PutByPeer(v2)
	assert.EqualValues(t, []interface{}{v1_1.id.String(), NetAddress("")}, []interface{}{o, r}, "empty NetAddress")
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	assert.Equal(t, true, s.RemoveByPeer(v2), "true")
	assert.Equal(t, false, s.ContainsByPeer(v2), "false")
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())
	//
	//v2_1 := &Peer{id: v2.id, netAddress: v1.netAddress}
}

func Test_set_PeerIDSet(t *testing.T) {
	s := NewPeerIDSet()

	v1 := generatePeerID()
	v2 := generatePeerID()
	v2_1 := NewPeerIDFromAddress(v2)
	v3 := generatePeerID()

	assert.Equal(t, true, s.IsEmpty(), "true")
	assert.Equal(t, true, s.Add(v1), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.Equal(t, false, s.IsEmpty(), "false")
	assert.Equal(t, true, s.Add(v2), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, false, s.Add(v2_1), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, true, s.Contains(v1), "true")
	assert.Equal(t, false, s.Contains(v3), "false")

	v4 := generatePeerID()
	s.Merge(v2, v2_1, v3, v4)
	assert.Equal(t, 4, s.Len(), "4")
	assert.Equal(t, true, s.Contains(v3), "true")
	assert.Equal(t, true, s.Contains(v4), "true")
	t.Log(s.Array())
}

func Test_set_RoleSet(t *testing.T) {
	s := NewRoleSet()
	assert.Equal(t, true, s.IsEmpty(), "true")
	assert.Equal(t, true, s.Add(module.ROLE_SEED), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.Equal(t, false, s.IsEmpty(), "false")
	assert.Equal(t, true, s.Add(module.ROLE_VALIDATOR), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, false, s.Add(module.ROLE_VALIDATOR), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, true, s.Contains(module.ROLE_SEED), "true")
	assert.Equal(t, false, s.Contains(module.Role("test")), "false")

	s.Merge(module.ROLE_SEED, module.Role("test"))
	assert.Equal(t, 3, s.Len(), "3")
	assert.Equal(t, true, s.Contains(module.Role("test")), "true")
	t.Log(s.Array())
}

type dummyPeerID struct {
	s string
	b []byte
}

func newDummyPeerID(s string) module.PeerID         { return &dummyPeerID{s: s, b: []byte(s)} }
func (pi *dummyPeerID) String() string              { return pi.s }
func (pi *dummyPeerID) Bytes() []byte               { return pi.b }
func (pi *dummyPeerID) ID() []byte                  { return pi.b }
func (pi *dummyPeerID) IsContract() bool            { return false }
func (pi *dummyPeerID) Equal(a module.Address) bool { return bytes.Equal(pi.b, a.ID()) }
func (pi *dummyPeerID) Copy(b []byte)               { copy(b, pi.b) }

func Benchmark_set_PeerSet(b *testing.B) {
	b.StopTimer()
	s := NewPeerSet()
	pArr := make([]*Peer, b.N)
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		p := &Peer{id: newDummyPeerID(s), netAddress: NetAddress(s)}
		pArr[i] = p
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		p := pArr[i]
		s.Add(p)
	}
}
