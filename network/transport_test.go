package network

import (
	"sync"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	testChannel                  = "testchannel"
	PayloadTestTransportRequest  = "Hello"
	PayloadTestTransportResponse = "World"
	// PROTO_TEST_REQ    = 0x0001
	// PROTO_TEST_RESP   = 0x0002
)

var (
	ProtoTestTransportRequest  module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestTransportResponse module.ProtocolInfo = protocolInfo(0x0400)
)

type TestPeerHandler struct {
	peerHandler
	t  *testing.T
	wg *sync.WaitGroup
}

func (ph *TestPeerHandler) onPeer(p *Peer) {
	ph.t.Logf("TestPeerHandler.onPeer %v", p)
	p.setPacketCbFunc(ph.onPacket)
	if !p.incomming {
		ph.wg.Add(1)
		ph.sendPacket(NewPacket(ProtoTestTransportRequest, []byte(PayloadTestTransportRequest)), p)
		ph.t.Logf("TestPeerHandler.sendPacket ProtoTestTransportRequest %s", PayloadTestTransportRequest)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (ph *TestPeerHandler) onError(err error, p *Peer) {
	ph.t.Logf("TestPeerHandler.onError %v", err)
}

func (ph *TestPeerHandler) onPacket(pkt *Packet, p *Peer) {
	s := string(pkt.payload)
	ph.t.Logf("TestPeerHandler.onPacket %v %v", pkt, p)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case ProtoTestTransportRequest:
			ph.t.Logf("TestPeerHandler.onPacket ProtoTestTransportRequest %s", s)
			ph.sendPacket(NewPacket(ProtoTestTransportResponse, []byte(PayloadTestTransportResponse)), p)
			ph.t.Logf("TestPeerHandler.sendPacket ProtoTestTransportResponse %s", PayloadTestTransportResponse)
			ph.nextOnPeer(p)
		case ProtoTestTransportResponse:
			ph.t.Logf("TestPeerHandler.onPacket ProtoTestTransportResponse %s", s)
			ph.nextOnPeer(p)
			ph.wg.Done()
		}
	}
}

func newTestTransport(channel string, address string) (*PeerDispatcher, *Listener, *Dialer) {
	priK, pubK := crypto.GenerateKeyPair()
	pd := newPeerDispatcher(NewPeerIDFromPublicKey(pubK),
		newChannelNegotiator(),
		newAuthenticator(priK, pubK))
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
	tph := &TestPeerHandler{t: t, wg: &wg}

	pd, l, d := getTransport(testChannel)

	pd.registPeerHandler(tph)

	err := l.Listen()
	if err != nil {
		t.Fatalf("Listener.Listen fail")
	} else {
		t.Logf("Listener.Listen success")
	}

	go d.Dial(l.address)

	wg.Wait()
	time.Sleep(5 * time.Second)

	l.Close()
}
