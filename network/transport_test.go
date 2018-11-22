package network

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	testChannel                  = "testchannel"
	testTransportAddress         = "127.0.0.1:8081"
	PayloadTestTransportRequest  = "Hello"
	PayloadTestTransportResponse = "World"
)

var (
	ProtoTestTransportRequest  module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestTransportResponse module.ProtocolInfo = protocolInfo(0x0400)
)

type TestPeerHandler struct {
	*peerHandler
	t  *testing.T
	wg *sync.WaitGroup
}

type TestTransportRequest struct {
	Message string
}

type TestTransportResponse struct {
	Message string
}

func (ph *TestPeerHandler) onPeer(p *Peer) {
	ph.log.Println("onPeer", p)
	p.setPacketCbFunc(ph.onPacket)
	if !p.incomming {
		ph.wg.Add(1)
		m := &TestTransportRequest{Message: "Hello"}
		ph.sendPacket(ProtoTestTransportRequest, m, p)
		ph.log.Println("sendProtoTestTransportRequest", m, p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (ph *TestPeerHandler) onError(err error, p *Peer) {
	ph.log.Println("onError", err, p)
	ph.peerHandler.onError(err, p)
	assert.Fail(ph.t, "TestPeerHandler.onError", err, p)
}

func (ph *TestPeerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.log.Println("onPacket", pkt, p)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case ProtoTestTransportRequest:
			rm := &TestTransportRequest{}
			ph.decode(pkt.payload, rm)
			ph.log.Println("handleProtoTestTransportRequest", rm, p)

			m := &TestTransportResponse{Message: "World"}
			ph.sendPacket(ProtoTestTransportResponse, m, p)

			ph.nextOnPeer(p)
		case ProtoTestTransportResponse:
			rm := &TestTransportResponse{}
			ph.decode(pkt.payload, rm)
			ph.log.Println("handleProtoTestTransportResponse", rm, p)

			ph.nextOnPeer(p)
			ph.wg.Done()
		}
	}
}

func newTestTransport(channel string, address string) (*PeerDispatcher, *Listener, *Dialer) {
	priK, pubK := crypto.GenerateKeyPair()
	pd := newPeerDispatcher(NewPeerIDFromPublicKey(pubK),
		newChannelNegotiator(NetAddress(address)),
		newAuthenticator(priK))
	l := newListener(address, pd.onAccept)
	d := newDialer(channel, pd.onConnect)
	return pd, l, d
}

func getTransport(channel string) (*PeerDispatcher, *Listener, *Dialer) {
	pd := GetPeerDispatcher()
	l := GetListener()
	d := GetDialer(channel)
	return pd, l, d
}
func Test_transport(t *testing.T) {
	var wg sync.WaitGroup
	pd, l, _ := getTransport(testChannel)
	tpd, tl, td := newTestTransport(testChannel, testTransportAddress)

	tph1 := &TestPeerHandler{newPeerHandler(newLogger("TestPeerHandler1", "")), t, &wg}
	tph2 := &TestPeerHandler{newPeerHandler(newLogger("TestPeerHandler2", "")), t, &wg}
	pd.registPeerHandler(tph1)
	tpd.registPeerHandler(tph2)

	assert.Nil(t, l.Listen(), "Listener.Listen fail")
	assert.Nil(t, tl.Listen(), "Listener.Listen fail")

	assert.Nil(t, td.Dial(l.address), "Dialer.Dial fail")

	wg.Wait()
	time.Sleep(1 * time.Second)
	l.Close()
}
