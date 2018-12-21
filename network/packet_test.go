package network

import (
	"bytes"
	"encoding/binary"
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
	return newPacket(protocolInfo(0x0000), b)
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
	_, err = hash.Write(hb)
	assert.NoError(t, err, "hash.Write(hb) NoError")
	_, err = hash.Write(payload)
	assert.NoError(t, err, "hash.Write(payload) NoError")
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
	pkt := newPacket(protocolInfo(0), []byte("test"))
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
