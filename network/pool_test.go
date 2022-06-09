package network

import (
	"container/list"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func Test_pool_PacketPool(t *testing.T) {
	p := NewPacketPool(2, 2)
	pkts := make([]*Packet, 5)
	for i := 0; i < 5; i++ {
		pkt := newPacket(module.ProtocolInfo(i), module.ProtocolInfo(i), []byte(fmt.Sprintf("test_%d", i)), nil)
		pkt.hashOfPacket = uint64(i)
		pkts[i] = pkt
	}

	for _, pkt := range pkts {
		p.Put(pkt)
	}

	assert.False(t, p.Contains(pkts[0]), "false")
	assert.False(t, p.Contains(pkts[1]), "false")
	assert.True(t, p.Contains(pkts[2]), "true")
	assert.True(t, p.Contains(pkts[3]), "true")
	assert.True(t, p.Contains(pkts[4]), "true")
}

func generateDummyPacket(s string, i int) *Packet {
	pkt := &Packet{payload: []byte(s), hashOfPacket: uint64(i)}
	return pkt
}

func Test_pool_TimestamPool(t *testing.T) {
	p := NewTimestampPool(2)
	for i := 0; i < 5; i++ {
		p.Put(i)
	}
	for i := 0; i < 5; i++ {
		assert.True(t, p.Contains(i), "true")
	}
	log.Println(time.Now().Unix())
	time.Sleep(1 * time.Second)
	log.Println(time.Now().Unix())
	for i := 0; i < 5; i++ {
		p.Put(i + 5)
	}
	for i := 0; i < 5; i++ {
		assert.True(t, p.Contains(i+5), "true")
	}
	p.RemoveBefore(1)
	for i := 0; i < 5; i++ {
		assert.False(t, p.Contains(i), "false")
	}
	for i := 0; i < 5; i++ {
		assert.True(t, p.Contains(i+5), "true")
	}
}

func Benchmark_pool_PacketPool(b *testing.B) {
	b.StopTimer()
	p := NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen)
	pkts := make([]*Packet, b.N)
	for i := 0; i < b.N; i++ {
		pkt := &Packet{}
		pkt.hashOfPacket = uint64(i)
		pkts[i] = pkt
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		pkt := pkts[i]
		p.Put(pkt)
	}
}

func Benchmark_pool_TimestamPool(b *testing.B) {
	b.StopTimer()
	// p := NewTimestampPool(10)
	l := list.New()
	for i := 0; i < b.N; i++ {
		// p.SetAndRemoveByData(i)
		l.PushBack(i)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for e := l.Front(); e != nil; e = e.Next() {
			_, ok := e.Value.(int)
			if !ok {
				b.FailNow()
			}
		}
	}
}

func Benchmark_dummy_Packet(b *testing.B) {
	pkts := make([]*Packet, b.N)
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		pkts[i] = generateDummyPacket(s, i)
	}
	//Benchmark_dummy_Packet-8   	 5000000	       282 ns/op	     144 B/op	       4 allocs/op
}
