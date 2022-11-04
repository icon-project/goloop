package network

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

const (
	packetTestProtocolInfo = module.ProtocolInfo(0x0000)
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
	return newPacket(packetTestProtocolInfo, packetTestProtocolInfo, b, nil)
}

func Test_packet_PacketReader(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
	b.Reset()
	pr := NewPacketReader(b)
	_, err := pr.ReadPacket()
	assert.Error(t, io.EOF, "ReadPacket EOF")

	hb := make([]byte, packetHeaderSize)
	payload := []byte("test")
	fb := make([]byte, packetFooterSize)
	binary.BigEndian.PutUint32(hb[packetHeaderSize-4:], uint32(len(payload)))
	hash := fnv.New64a()
	_, err = hash.Write(hb)
	assert.NoError(t, err, "footer.Write(hb) NoError")
	_, err = hash.Write(payload)
	assert.NoError(t, err, "footer.Write(payload) NoError")
	binary.BigEndian.PutUint64(fb, hash.Sum64())

	b.Write(hb)
	b.Write(payload)
	b.Write(fb)
	pkt, err := pr.ReadPacket()
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, hash.Sum64(), pkt.hashOfPacket, "ReadPacket Invalid footer")
}

func Test_packet_PacketReadWriter(t *testing.T) {
	prw := NewPacketReadWriter()
	pkt := newPacket(packetTestProtocolInfo, packetTestProtocolInfo, []byte("test"), generatePeerID())
	pkt.forceSend = false
	pkt.timestamp = time.Now()
	assert.NoError(t, prw.WritePacket(pkt), "WritePacket fail")
	rpkt, err := prw.ReadPacket()
	rpkt.timestamp = pkt.timestamp
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, pkt, rpkt, "ReadPacket")
	rpkt, err = prw.ReadPacket()
	assert.NoError(t, err, "ReadPacket fail")
	assert.Equal(t, pkt, rpkt, "ReadPacket")
	prw.Reset(prw.b, prw.b)
	rpkt, err = prw.ReadPacket()
	assert.Error(t, err, "ReadPacket must fail(io.EOF) after Reset")

	//prw.rd.WriteTo()
}

func FuzzPacketReadFrom(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		buf := bytes.NewBuffer(data)
		var pk Packet
		n, _ := pk.ReadFrom(buf)
		assert.LessOrEqual(t, n, int64(len(data)))
	})
}
