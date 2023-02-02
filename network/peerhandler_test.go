package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type peerFunc func(*Peer)

type testPeerHandler struct {
	*peerHandler
	onPeerFunc   peerFunc
	onPacketFunc packetCbFunc
	onCloseFunc  peerFunc
	setNextFunc  func(PeerHandler)

	nextOnPeerFunc peerFunc
}

func (ph *testPeerHandler) onPeer(p *Peer) {
	if ph.onPeerFunc != nil {
		ph.onPeerFunc(p)
	} else {
		ph.peerHandler.onPeer(p)
	}
}

func (ph *testPeerHandler) onPacket(pkt *Packet, p *Peer) {
	if ph.onPacketFunc != nil {
		ph.onPacketFunc(pkt, p)
	}
	ph.peerHandler.onPacket(pkt, p)
}

func (ph *testPeerHandler) onClose(p *Peer) {
	if ph.onCloseFunc != nil {
		ph.onCloseFunc(p)
	} else {
		ph.peerHandler.onClose(p)
	}
}

func (ph *testPeerHandler) setNext(next PeerHandler) {
	if ph.setNextFunc != nil {
		ph.setNextFunc(next)
	} else {
		ph.peerHandler.setNext(next)
	}
}

func (ph *testPeerHandler) nextOnPeer(p *Peer) {
	if ph.nextOnPeerFunc != nil {
		ph.nextOnPeerFunc(p)
	} else {
		ph.peerHandler.nextOnPeer(p)
	}
}

func newTestPeerHandler(id module.PeerID, l log.Logger) *testPeerHandler {
	return &testPeerHandler{
		peerHandler: newPeerHandler(id, l),
	}
}

func assertEqualPacket(t *testing.T, expected, actual *Packet) {
	assert.Equal(t, expected.header, actual.header)
	assert.Equal(t, expected.payload, actual.payload)
	assert.Equal(t, expected.footer, actual.footer)
	assert.Equal(t, expected.ext, actual.ext)
}

func Test_PeerHandler(t *testing.T) {
	p, conn := newPeerWithFakeConn(false)
	p.setMetric(metric.NewNetworkMetric(metric.DefaultMetricContext()))

	ch := make(chan interface{}, 1)
	id := generatePeerID()
	tph := newTestPeerHandler(id, testLogger())
	tph.onPeerFunc = func(p *Peer) {
		ch <- p
	}
	ph := newPeerHandler(id, testLogger())
	ph.setNext(tph)

	ph.onPeer(p)
	v := <-ch
	assert.Equal(t, p, v.(*Peer))

	tph.onPacketFunc = func(pkt *Packet, p *Peer) {
		ch <- pkt
	}
	msg := "test"
	expected := newPacket(ProtoTestNetwork, ProtoTestNetwork, codec.MP.MustMarshalToBytes(msg), id)
	err := conn.WritePacket(expected)
	assert.NoError(t, err)
	v = <-ch
	actual := v.(*Packet)
	assertEqualPacket(t, expected, actual)

	tph.onCloseFunc = func(p *Peer) {
		ch <- p
	}
	conn.errWrite = fmt.Errorf("test error")
	tph.sendMessage(expected.protocol, expected.subProtocol, msg, p)
	v = <-ch
	assert.True(t, v.(*Peer).IsClosed())
}
