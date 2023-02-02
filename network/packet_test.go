package network

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func FuzzPacketReadFrom(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		buf := bytes.NewBuffer(data)
		var pk Packet
		n, _ := pk.ReadFrom(buf)
		assert.LessOrEqual(t, n, int64(len(data)))
	})
}
