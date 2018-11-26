package network

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

func Test_set_PeerSet(t *testing.T) {
	s := NewPeerSet()
	_, pubK1 := crypto.GenerateKeyPair()
	_, pubK2 := crypto.GenerateKeyPair()
	_, pubK3 := crypto.GenerateKeyPair()
	p1 := NewPeerIDFromPublicKey(pubK1)
	p2 := NewPeerIDFromPublicKey(pubK2)
	p2_1 := NewPeerIDFromPublicKey(pubK2)
	p3 := NewPeerIDFromPublicKey(pubK3)
	n1 := NetAddress("127.0.0.1:8080")
	n2 := NetAddress("127.0.0.1:8081")
	n2_1 := NetAddress("127.0.0.1:8081")
	n3 := NetAddress("127.0.0.1:8082")
	v1 := &Peer{id: p1, netAddress: n1}
	v2 := &Peer{id: p2, netAddress: n2}
	v2_1 := &Peer{id: p2_1, netAddress: n2}
	v2_2 := &Peer{id: p2, netAddress: n3}
	v2_3 := &Peer{id: p2, netAddress: n2_1, incomming: true}
	v3 := &Peer{id: p3, netAddress: n3}

	assert.Equal(t, true, s.IsEmpty(), "true")
	assert.Equal(t, true, s.Add(v1), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.Equal(t, false, s.IsEmpty(), "false")
	assert.Equal(t, true, s.Add(v2), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, false, s.Add(v2_1), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, false, s.Add(v2_2), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.Equal(t, true, s.Add(v2_3), "true")
	assert.Equal(t, 3, s.Len(), "3")
	assert.Equal(t, true, s.Contains(v1), "true")
	assert.Equal(t, false, s.Contains(v3), "false")

	assert.Equal(t, true, s.HasNetAddresse(n1), "true")
	assert.Equal(t, false, s.HasNetAddresse(n3), "false")
	t.Log(s.NetAddresses())

	t.Log(s.Array())
}

func Test_set_NetAddressSet(t *testing.T) {
	s := NewNetAddressSet()
	v1 := NetAddress("127.0.0.1:8080")
	v2 := NetAddress("127.0.0.1:8081")
	v2_1 := NetAddress("127.0.0.1:8081")
	v3 := NetAddress("127.0.0.1:8082")

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

	v4 := NetAddress("127.0.0.1:8084")
	s.Merge(v2, v2_1, v3, v4)
	assert.Equal(t, 4, s.Len(), "4")
	assert.Equal(t, true, s.Contains(v3), "true")
	assert.Equal(t, true, s.Contains(v4), "true")
	t.Log(s.Array())
}

func Test_set_PeerIdSet(t *testing.T) {
	s := NewPeerIdSet()

	_, pubK1 := crypto.GenerateKeyPair()
	_, pubK2 := crypto.GenerateKeyPair()
	_, pubK3 := crypto.GenerateKeyPair()
	v1 := NewPeerIDFromPublicKey(pubK1)
	v2 := NewPeerIDFromPublicKey(pubK2)
	v2_1 := NewPeerIDFromPublicKey(pubK2)
	v3 := NewPeerIDFromPublicKey(pubK3)

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

	_, pubK4 := crypto.GenerateKeyPair()
	v4 := NewPeerIDFromPublicKey(pubK4)
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

func newDummyPeerID() *module.PeerID {
	return nil
}

func Benchmark_set_PeerSet(b *testing.B) {
	b.StopTimer()
	s := NewPeerSet()
	_, pubK := crypto.GenerateKeyPair()
	p := &Peer{id: NewPeerIDFromPublicKey(pubK), netAddress: "127.0.0.1:8080"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Add(p)
	}
}
