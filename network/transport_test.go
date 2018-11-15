package network

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	testListenAddress = "127.0.0.1:8081"
	testChannel       = "testchannel"
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
	ph.t.Logf("TestPeerHandler.onPeer in:%v", p.incomming)
	p.setPacketCbFunc(ph.onPacket)
	if !p.incomming {
		ph.wg.Add(1)
		ph.sendPacket(NewPacket(ProtoTestTransportRequest, []byte("hello")), p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (ph *TestPeerHandler) onError(err error, p *Peer) {
	log.Println("TestPeerHandler.onError", err)
}

func (ph *TestPeerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.t.Logf("TestPeerHandler.onPacket %v", pkt)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case ProtoTestTransportRequest:
			ph.sendPacket(NewPacket(ProtoTestTransportResponse, pkt.payload), p)
			ph.nextOnPeer(p)
		case ProtoTestTransportResponse:
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
