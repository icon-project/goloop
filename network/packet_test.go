package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func generatePacket(b []byte, len int) *Packet {
	if b == nil {
		if len < 0 {
			b = make([]byte, 1)
		} else {
			b = make([]byte, len)
		}
	} else {
		if len > 0 {
			b = b[:len]
		}
	}
	return NewPacket(protocolInfo(0x0000), b)
}

func Test_packet_PacketReader(t *testing.T) {
	//TODO test with TCPConn
	b := bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
	b.Reset()
	pr := NewPacketReader(b)
	_, _, err := pr.ReadPacket()
	assert.Error(t, io.EOF, "ReadPacket EOF")

	hb := make([]byte, packetHeaderSize)
	payload := []byte("test")
	fb := make([]byte, packetFooterSize)
	binary.BigEndian.PutUint32(hb[packetHeaderSize-4:], uint32(len(payload)))
	hash := fnv.New64a()
	hash.Write(hb)
	hash.Write(payload)
	binary.BigEndian.PutUint64(fb, hash.Sum64())

	b.Write(hb)
	b.Write(payload)
	b.Write(fb)
	pkt, h, err := pr.ReadPacket()
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, hash.Sum64(), pkt.hashOfPacket, "ReadPacket Invalid hash")
	assert.Equal(t, h.Sum64(), pkt.hashOfPacket, "ReadPacket Invalid hash")
}

func Test_packet_PacketReadWriter(t *testing.T) {
	prw := NewPacketReadWriter()
	pkt := NewPacket(protocolInfo(0), []byte("test"))
	pkt.src = generatePeerID()
	assert.NoError(t, prw.WritePacket(pkt), "WritePacket fail")
	rpkt, err := prw.ReadPacket()
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, pkt, rpkt, "ReadPacket")
	rpkt, err = prw.ReadPacket()
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, pkt, rpkt, "ReadPacket")
	prw.Reset()
	rpkt, err = prw.ReadPacket()
	assert.Error(t, err, "ReadPacket must fail(io.EOF) after Reset")

	//prw.rd.WriteTo()
}
func Test_packet_PacketPool(t *testing.T) {
	pp := NewPacketPool(2, 2)
	pkts := make([]*Packet, 5)
	for i := 0; i < 5; i++ {
		pkt := NewPacket(protocolInfo(i), []byte(fmt.Sprintf("test_%d", i)))
		pkt.hashOfPacket = uint64(i)
		pkts[i] = pkt
	}

	for _, pkt := range pkts {
		pp.Put(pkt)
	}

	assert.Equal(t, false, pp.Contains(pkts[0]), "false")
	assert.Equal(t, false, pp.Contains(pkts[1]), "false")
	assert.Equal(t, true, pp.Contains(pkts[2]), "true")
	assert.Equal(t, true, pp.Contains(pkts[3]), "true")
	assert.Equal(t, true, pp.Contains(pkts[4]), "true")
}

func generateDummyPacket(s string, i int) *Packet {
	pkt := &Packet{payload: []byte(s), hashOfPacket: uint64(i)}
	return pkt
}

func Benchmark_packet_PacketPool(b *testing.B) {
	b.StopTimer()
	pp := NewPacketPool(DefaultPacketPoolNumBucket, DefaultPacketPoolBucketLen)
	pkts := make([]*Packet, b.N)
	for i := 0; i < b.N; i++ {
		pkt := &Packet{}
		pkt.hashOfPacket = uint64(i)
		pkts[i] = pkt
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		pkt := pkts[i]
		pp.Put(pkt)
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
