package network

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/wallet"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

const (
	testChannel = "testchannel"
)

var (
	ProtoTestTransport         = module.ProtocolInfo(0x0000)
	ProtoTestTransportRequest  = module.ProtocolInfo(0xF300)
	ProtoTestTransportResponse = module.ProtocolInfo(0xF400)
)

type testPeerHandler struct {
	*peerHandler
	t  *testing.T
	wg *sync.WaitGroup
	ss SecureSuite
	sa SecureAeadSuite

	onPeerDelay time.Duration
}

func newTestPeerHandler(name string, t *testing.T, l log.Logger) *testPeerHandler {
	return &testPeerHandler{
		peerHandler: newPeerHandler(l.WithFields(log.Fields{LoggerFieldKeySubModule: name})),
		t:           t,
	}
}

type testTransportRequest struct {
	Message string
}

type testTransportResponse struct {
	Message string
}

func (ph *testPeerHandler) onPeer(p *Peer) {
	ph.logger.Println("onPeer", p)
	if ph.onPeerDelay > 0 {
		time.Sleep(ph.onPeerDelay)
		ph.nextOnPeer(p)
		return
	}

	if ph.ss != SecureSuiteUnknown {
		//assert secure connection
		switch c := p.conn.(type) {
		case *SecureConn:
			assert.Equal(ph.t, ph.ss, SecureSuite(SecureSuiteEcdhe))
			assert.Equal(ph.t, ph.sa, p.secureKey.sa)
			assert.Equal(ph.t, ph.sa, c.sa)
		case *tls.Conn:
			assert.Equal(ph.t, ph.ss, SecureSuite(SecureSuiteTls))
			assert.Equal(ph.t, ph.sa, p.secureKey.sa)
			var cs uint16
			switch ph.sa {
			case SecureAeadSuiteAes128Gcm:
				cs = tls.TLS_AES_128_GCM_SHA256
			case SecureAeadSuiteAes256Gcm:
				//tls.TLS_AES_256_GCM_SHA384 not supported
				cs = tls.TLS_AES_128_GCM_SHA256
			case SecureAeadSuiteChaCha20Poly1305:
				cs = tls.TLS_CHACHA20_POLY1305_SHA256
			}
			assert.Equal(ph.t, cs, c.ConnectionState().CipherSuite)
		default:
			assert.Equal(ph.t, ph.ss, SecureSuite(SecureSuiteNone))
			assert.Equal(ph.t, SecureAeadSuite(SecureAeadSuiteNone), p.secureKey.sa)
		}
	}

	if !p.In() {
		m := &testTransportRequest{Message: "Hello"}
		ph.sendMessage(ProtoTestTransport, ProtoTestTransportRequest, m, p)
		ph.logger.Println("sendProtoTestTransportRequest", m, p)
	}
}

func (ph *testPeerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.logger.Println("onPacket", pkt, p)
	switch pkt.protocol {
	case ProtoTestTransport:
		switch pkt.subProtocol {
		case ProtoTestTransportRequest:
			rm := &testTransportRequest{}
			ph.decode(pkt.payload, rm)
			ph.logger.Println("handleProtoTestTransportRequest", rm, p)

			m := &testTransportResponse{Message: "World"}
			ph.sendMessage(ProtoTestTransport, ProtoTestTransportResponse, m, p)

			p.Close("Done")
		case ProtoTestTransportResponse:
			rm := &testTransportResponse{}
			ph.decode(pkt.payload, rm)
			ph.logger.Println("handleProtoTestTransportResponse", rm, p)

			if ph.wg != nil {
				ph.wg.Done()
			}
			p.Close("Done")
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
	w, _ := wallet.NewFromPrivateKey(priK)
	return &testWallet{w}
}

func getAvailableLocalhostAddress(t *testing.T) string {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		assert.FailNow(t, err.Error(), "fail to getAvailableLocalhostAddress")
	}
	addr := ln.Addr().String()
	if err = ln.Close(); err != nil {
		assert.FailNow(t, err.Error(), "fail to close listener ", addr)
	}
	return addr
}

func sliceToString(s interface{}) string {
	r := ""
	v := reflect.ValueOf(s)
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			r += ","
		}
		r += v.Index(i).Interface().(fmt.Stringer).String()
	}
	return r
}

type testKeyLogWriter struct{}

func (w *testKeyLogWriter) Write(b []byte) (n int, err error) {
	log.Println("testKeyLogWriter", string(b))
	return len(b), nil
}

func Test_transport(t *testing.T) {
	lv := log.GlobalLogger().GetLevel()
	if testing.Verbose() {
		lv = log.TraceLevel
	}

	sss := []SecureSuite{
		SecureSuiteNone,
		SecureSuiteTls,
		SecureSuiteEcdhe,
	}
	sas := []SecureAeadSuite{
		SecureAeadSuiteChaCha20Poly1305,
		SecureAeadSuiteAes128Gcm,
		SecureAeadSuiteAes256Gcm,
	}
	w1 := walletFromGeneratedPrivateKey()
	l1 := log.WithFields(log.Fields{
		log.FieldKeyWallet: hex.EncodeToString(w1.Address().ID()),
	})
	l1.SetLevel(lv)
	nt1 := NewTransport(getAvailableLocalhostAddress(t), w1, l1)

	w2 := walletFromGeneratedPrivateKey()
	l2 := log.WithFields(log.Fields{
		log.FieldKeyWallet: hex.EncodeToString(w2.Address().ID()),
	})
	l2.SetLevel(lv)
	nt2 := NewTransport(getAvailableLocalhostAddress(t), w2, l2)

	dph := newTestPeerHandler("OnPeerDelayOnly", t, nt2.(*transport).logger)
	dph.onPeerDelay = 10 * time.Millisecond
	nt2.(*transport).pd.registerPeerHandler(dph, false)

	tph1 := newTestPeerHandler("TestPeerHandler1", t, nt1.(*transport).logger)
	tph2 := newTestPeerHandler("TestPeerHandler2", t, nt2.(*transport).logger)
	tph2.wg = &sync.WaitGroup{}

	nt1.(*transport).pd.registerPeerHandler(tph1, true)
	nt2.(*transport).pd.registerPeerHandler(tph2, true)

	nt1.(*transport).cn.addProtocol(testChannel, p2pProtoControl)
	nt2.(*transport).cn.addProtocol(testChannel, p2pProtoControl)

	if err := nt1.Listen(); err != nil {
		assert.FailNow(t, err.Error(), "Transport1.Start fail")
	}
	if err := nt2.Listen(); err != nil {
		assert.FailNow(t, err.Error(), "Transport2.Start fail")
	}

	DefaultSecureKeyLogWriter = &testKeyLogWriter{}
	for _, ss := range sss {
		for _, sa := range sas {
			t.Log("SecureSuite:", ss, "SecureAeadSuite:", sa)

			strSS := sliceToString([]SecureSuite{ss})
			assert.NoError(t, nt1.SetSecureSuites(testChannel, strSS))
			assert.Equal(t, strSS, nt1.GetSecureSuites(testChannel))
			strSA := sliceToString([]SecureAeadSuite{sa})
			assert.NoError(t, nt1.SetSecureAeads(testChannel, strSA))
			assert.Equal(t, strSA, nt1.GetSecureAeads(testChannel))

			tph1.ss = ss
			tph1.sa = sa
			tph2.ss = ss
			tph2.sa = sa
			tph2.wg.Add(1)

			if err := nt2.Dial(nt1.GetListenAddress(), testChannel); err != nil {
				assert.FailNow(t, err.Error(), "Transport.Dial fail")
			}

			tph2.wg.Wait()
		}
	}
	assert.NoError(t, nt1.Close(), "Transport1.Close fail")
	assert.NoError(t, nt2.Close(), "Transport2.Close fail")
}
