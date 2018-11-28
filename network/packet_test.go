package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
