package network

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	testChannel          = "testchannel"
	testTransportAddress = "127.0.0.1:8081"
)

var (
	ProtoTestTransportRequest  module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestTransportResponse module.ProtocolInfo = protocolInfo(0x0400)
)

type testPeerHandler struct {
	*peerHandler
	t  *testing.T
	wg *sync.WaitGroup
}

func newTestPeerHandler(name string, t *testing.T, wg *sync.WaitGroup) *testPeerHandler {
	return &testPeerHandler{newPeerHandler(newLogger(name, "")), t, wg}
}

type testTransportRequest struct {
	Message string
}

type testTransportResponse struct {
	Message string
}

func (ph *testPeerHandler) onPeer(p *Peer) {
	ph.log.Println("onPeer", p)
	p.setPacketCbFunc(ph.onPacket)
	if !p.incomming {
		ph.wg.Add(1)
		m := &testTransportRequest{Message: "Hello"}
		ph.sendPacket(ProtoTestTransportRequest, m, p)
		ph.log.Println("sendProtoTestTransportRequest", m, p)
	}
}

func (ph *testPeerHandler) onError(err error, p *Peer, pkt *Packet) {
	ph.log.Println("onError", err, p, pkt)
	ph.peerHandler.onError(err, p, pkt)
	assert.Fail(ph.t, "TestPeerHandler.onError", err, p, pkt)
}

func (ph *testPeerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.log.Println("onPacket", pkt, p)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case ProtoTestTransportRequest:
			rm := &testTransportRequest{}
			ph.decode(pkt.payload, rm)
			ph.log.Println("handleProtoTestTransportRequest", rm, p)

			m := &testTransportResponse{Message: "World"}
			ph.sendPacket(ProtoTestTransportResponse, m, p)

			ph.nextOnPeer(p)
		case ProtoTestTransportResponse:
			rm := &testTransportResponse{}
			ph.decode(pkt.payload, rm)
			ph.log.Println("handleProtoTestTransportResponse", rm, p)

			ph.nextOnPeer(p)
			ph.wg.Done()
		}
	}
}

func generatePrivateKey() *crypto.PrivateKey {
	priK, _ := crypto.GenerateKeyPair()
	return priK
}

func walletFromGeneratedPrivateKey() module.Wallet {
	priK, _ := crypto.GenerateKeyPair()
	w, _ := common.WalletFromPrivateKey(priK)
	return w
}

func Test_transport(t *testing.T) {
	var wg sync.WaitGroup

	nt1 := GetTransport()
	nt2 := NewTransport(testTransportAddress, walletFromGeneratedPrivateKey())

	tph1 := newTestPeerHandler("TestPeerHandler1", t, &wg)
	tph2 := newTestPeerHandler("TestPeerHandler2", t, &wg)
	nt1.(*transport).pd.registPeerHandler(tph1)
	nt2.(*transport).pd.registPeerHandler(tph2)

	assert.Nil(t, nt1.Listen(), "Transport.Start fail")
	assert.Nil(t, nt2.Listen(), "Transport.Start fail")

	assert.Nil(t, nt2.Dial(nt1.Address(), ""), "Transport.Dial fail")

	wg.Wait()
	time.Sleep(1 * time.Second)
}
