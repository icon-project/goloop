package network

import (
	"sync"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/crypto"
)

const (
	testListenAddress = "127.0.0.1:8081"
	testChannel       = "testchannel"
	PROTO_TEST_REQ    = 0x0001
	PROTO_TEST_RESP   = 0x0002
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
		ph.sendPacket(NewPacket(0x0001, []byte("hello")), p)
	}
}

func (ph *TestPeerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.t.Logf("TestPeerHandler.onPacket %v", pkt)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case PROTO_TEST_REQ:
			ph.sendPacket(NewPacket(PROTO_TEST_RESP, pkt.payload), p)
			ph.nextOnPeer(p)
		case PROTO_TEST_RESP:
			ph.nextOnPeer(p)
			ph.wg.Done()
		}
	}
}

func newTestTransport(channel string, address string) (*PeerDispatcher, *Listener, *Dialer) {
	priK, pubK := crypto.GenerateKeyPair()
	pd := newPeerDispatcher(NewPeerIdFromPublicKey(pubK),
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
	tpd, tl, _ := newTestTransport(testChannel, testListenAddress)

	pd.registPeerHandler(tph)
	tpd.registPeerHandler(tph)

	err := l.Listen()
	if err != nil {
		t.Fatalf("Listener.Listen fail")
	} else {
		t.Logf("Listener.Listen success")
	}

	err = tl.Listen()
	if err != nil {
		t.Fatalf("TestListener.Listen fail")
	} else {
		t.Logf("TestListener.Listen success")
	}

	go d.Dial(l.address)

	wg.Wait()
	time.Sleep(5 * time.Second)

	l.Close()
	tl.Close()
}
