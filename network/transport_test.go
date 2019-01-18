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
	testTransportAddress1 = "127.0.0.1:8080"
	testTransportAddress2 = "127.0.0.1:8081"
)

var (
	ProtoTestTransportRequest  = protocolInfo(0xF300)
	ProtoTestTransportResponse = protocolInfo(0xF400)
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
		m := &testTransportRequest{Message: "Hello"}
		ph.sendMessage(ProtoTestTransportRequest, m, p)
		ph.log.Println("sendProtoTestTransportRequest", m, p)
	}
}

func (ph *testPeerHandler) onError(err error, p *Peer, pkt *Packet) {
	ph.log.Println("onError", err, p, pkt)
	ph.peerHandler.onError(err, p, pkt)
	assert.Fail(ph.t, "TestPeerHandler.onError", err.Error(), p, pkt)
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
			ph.sendMessage(ProtoTestTransportResponse, m, p)

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

//Using mutex for prevent panic d.nx != 0
////crypto/sha256/sha256.go:253 (*digest).checkSum
////crypto/sha256/sha256.go:229 (*digest).Sum
////github.com/icon-project/goloop/vendor/github.com/haltingstate/secp256k1-go/secp256_rand.go:23 SumSHA256
////github.com/icon-project/goloop/vendor/github.com/haltingstate/secp256k1-go/secp256_rand.go:50 (*EntropyPool).Mix256
////github.com/icon-project/goloop/vendor/github.com/haltingstate/secp256k1-go/secp256_rand.go:71 (*EntropyPool).Mix
////github.com/icon-project/goloop/vendor/github.com/haltingstate/secp256k1-go/secp256_rand.go:133 RandByte
var walletMutex sync.Mutex

type testWallet struct {
	module.Wallet
}

func (w *testWallet) Sign(data []byte) ([]byte, error) {
	defer walletMutex.Unlock()
	walletMutex.Lock()
	return w.Wallet.Sign(data)
}

func walletFromGeneratedPrivateKey() module.Wallet {
	priK, _ := crypto.GenerateKeyPair()
	w, _ := common.NewWalletFromPrivateKey(priK)
	return &testWallet{w}
}

func Test_transport(t *testing.T) {
	var wg sync.WaitGroup

	ExcludeLoggers = []string{}

	nt1 := NewTransport(testTransportAddress1, walletFromGeneratedPrivateKey())
	nt2 := NewTransport(testTransportAddress2, walletFromGeneratedPrivateKey())

	wg.Add(1)
	tph1 := newTestPeerHandler("TestPeerHandler1", t, &wg)
	tph2 := newTestPeerHandler("TestPeerHandler2", t, &wg)

	nt1.(*transport).pd.registPeerHandler(tph1, false)
	nt2.(*transport).pd.registPeerHandler(tph2, false)

	assert.Nil(t, nt1.Listen(), "Transport.Start fail")
	assert.Nil(t, nt2.Listen(), "Transport.Start fail")

	assert.Nil(t, nt2.Dial(nt1.Address(), ""), "Transport.Dial fail")

	wg.Wait()
	time.Sleep(1 * time.Second)
}
