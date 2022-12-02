package network

import (
	"bytes"
	"container/list"
	"fmt"
	"math/rand"
	"testing"

	"github.com/icon-project/goloop/common"
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
	v2_1 := &Peer{id: v2.ID(), netAddress: v2.NetAddress()}
	v2_2 := &Peer{id: v2.ID(), netAddress: v2.NetAddress(), in: true}
	v3 := generatePeer()

	assert.True(t, s.IsEmpty(), "true")
	assert.True(t, s.Add(v1), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.False(t, s.IsEmpty(), "false")
	assert.True(t, s.Add(v2), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.False(t, s.Add(v2_1), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.True(t, s.Remove(v2_1), "true") // removed v2
	assert.Equal(t, 1, s.Len(), "1")
	assert.True(t, s.Add(v2_1), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.True(t, s.Add(v2_2), "true")
	assert.Equal(t, 3, s.Len(), "3")
	assert.True(t, s.Contains(v1), "true")
	assert.False(t, s.Contains(v3), "false")

	assert.True(t, s.HasNetAddress(v2.NetAddress()), "true")
	assert.False(t, s.HasNetAddress(v3.NetAddress()), "false")
	t.Log(s.NetAddresses())


	s.Remove(v2_2)
	s.Add(v3)
	l := s.Len()
	arr := s.Array()
	for i:=0;i<l;i++{
		v := arr[i]
		t.Log(i,v.ID(),v.NetAddress())
	}

	for i:=0;i<100;i++ {
		tarr := s.Array()
		for ti := 0;ti<l;ti++{
			if arr[ti].NetAddress() != tarr[ti].NetAddress(){
				t.Log(i,ti,"Not equal",tarr[ti].NetAddress(), arr[ti].NetAddress())
			}
		}
	}
}

func Test_set_NetAddressSet(t *testing.T) {
	s := NewNetAddressSet()
	v1 := generatePeer()
	v1_1 := &Peer{id: v1.ID(), netAddress: generateNetAddress()}
	v2 := &Peer{id: generatePeerID(), netAddress: v1_1.NetAddress()}

	// assert.True(t, s.IsEmpty(), "true")
	// assert.True(t, s.Add(v1.netAddress), "true")
	// assert.Equal(t, 1, s.Len(), "1")
	// assert.False(t, s.IsEmpty(), "false")
	// assert.Equal(t, "", s.Map()[v1.netAddress], "empty string")
	// assert.False(t, s.Add(v1.netAddress), "false")
	// assert.Equal(t, 1, s.Len(), "1")
	// t.Log(s.Map())

	//When Peer connected
	o, r := s.SetAndRemoveByData(v1.NetAddress(), v1.ID().String())
	assert.EqualValues(t, []interface{}{"", NetAddress("")}, []interface{}{o, r}, "empty NetAddress")
	assert.True(t, s.Map()[v1.NetAddress()] == v1.ID().String(), v1.ID().String())
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	//Update NetAddress, NetAddressSet.PutByPeer returns old NetAddress
	o, r = s.SetAndRemoveByData(v1_1.NetAddress(), v1_1.ID().String())
	assert.EqualValues(t, []interface{}{"", v1.NetAddress()}, []interface{}{o, r}, "empty NetAddress")
	assert.Equal(t, v1_1.ID().String(), s.Map()[v1_1.NetAddress()],"equal data")
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	//When Peer connected with same NetAddress, NetAddressSet.PutByPeer returns conflict PeerID
	o, r = s.SetAndRemoveByData(v2.NetAddress(), v2.ID().String())
	assert.EqualValues(t, []interface{}{v1_1.ID().String(), NetAddress("")}, []interface{}{o, r}, "empty NetAddress")
	assert.Equal(t, 1, s.Len(), "1")
	t.Log(s.Map())

	assert.True(t, s.Remove(v2.NetAddress()), "true")
	assert.False(t, s.ContainsWithData(v2.NetAddress(), v2.ID().String()), "false")
	assert.Equal(t, 0, s.Len(), "0")
	t.Log(s.Map())
	//
	//v2_1 := &Peer{id: v2.id, netAddress: v1.netAddress}

	na1 := NetAddress("localhost:1234")
	na2 := NetAddress("localhost:1234")
	s.Add(na1)
	assert.True(t, s.Contains(na2))
}

func Test_set_PeerIDSet(t *testing.T) {
	s := NewPeerIDSet()

	v1 := generatePeerID()
	v2 := generatePeerID()
	v2_1 := &peerID{
		Address: common.NewAccountAddress(v2.Bytes()),
	}
	v3 := generatePeerID()

	assert.True(t, s.IsEmpty(), "true")
	assert.True(t, s.Add(v1), "true")
	assert.Equal(t, 1, s.Len(), "1")
	assert.False(t, s.IsEmpty(), "false")
	assert.True(t, s.Add(v2), "true")
	assert.Equal(t, 2, s.Len(), "2")
	assert.False(t, s.Add(v2_1), "false")
	assert.Equal(t, 2, s.Len(), "2")
	assert.True(t, s.Contains(v1), "true")
	assert.False(t, s.Contains(v3), "false")

	v4 := generatePeerID()
	s.Merge(v2, v2_1, v3, v4)
	assert.Equal(t, 4, s.Len(), "4")
	assert.True(t, s.Contains(v3), "true")
	assert.True(t, s.Contains(v4), "true")
	t.Log(s.Array())
}

type dummyPeerID struct {
	s string
	b []byte
}

func newDummyPeerID(s string) module.PeerID         { return &dummyPeerID{s: s, b: []byte(s)} }
func (pi *dummyPeerID) String() string              { return pi.s }
func (pi *dummyPeerID) Bytes() []byte               { return pi.b }
func (pi *dummyPeerID) Equal(a module.PeerID) bool { return bytes.Equal(pi.b, a.Bytes()) }

func generateDummyPeer(s string) *Peer {
	p := &Peer{id: newDummyPeerID(s), netAddress: NetAddress(s)}
	return p
}

func Benchmark_set_PeerSet(b *testing.B) {
	b.StopTimer()
	s := NewPeerSet()
	pArr := make([]*Peer, b.N)
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		pArr[i] = generateDummyPeer(s)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		p := pArr[i]
		s.Add(p)
	}
}

func Benchmark_dummy_Peer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		generateDummyPeer(s)
	}
	//Benchmark_dummy_Peer-8   	20000000	        97.1 ns/op	      16 B/op	       2 allocs/op
}



func Benchmark_golang_slice(b *testing.B) {
	b.StopTimer()
	s := make([]interface{}, b.N)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s[i] = i
	}
	for i := 0; i < b.N; i++ {
		s[i] = nil
	}
}

func Benchmark_golang_map(b *testing.B) {
	b.StopTimer()
	m := make(map[int]int, b.N)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
	for i := 0; i < b.N; i++ {
		delete(m, i)
	}
}

func Benchmark_golang_map_remove(b *testing.B) {
	b.StopTimer()
	m := make(map[int]int, b.N)
	for i := 0; i < b.N; i++ {
		m[i] = i
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		delete(m, i)
	}
}

func Benchmark_golang_list(b *testing.B) {
	b.StopTimer()
	l := list.New()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		l.PushBack(i)
	}
	e := l.Front()
	for e != nil{
		n := e.Next()
		l.Remove(e)
		e = n
	}
}

func Benchmark_golang_list_remove(b *testing.B) {
	b.StopTimer()
	l := list.New()
	m := make(map[int]*list.Element, b.N)
	for i := 0; i < b.N; i++ {
		m[i] = l.PushBack(i)
	}
	b.StartTimer()
	for _, v := range m {
		l.Remove(v)
	}

}