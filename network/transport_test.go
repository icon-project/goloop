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
	"github.com/icon-project/goloop/server/metric"

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

type testTransportPeerHandler struct {
	*peerHandler
	t  *testing.T
	wg *sync.WaitGroup

	expectedSecureSuite     SecureSuite
	expectedSecureAeadSuite SecureAeadSuite
}

func newTestTransportPeerHandler(name string, t *testing.T, id module.PeerID, l log.Logger) *testTransportPeerHandler {
	return &testTransportPeerHandler{
		peerHandler: newPeerHandler(id, l.WithFields(log.Fields{LoggerFieldKeySubModule: name})),
		t:           t,
	}
}

type testTransportRequest struct {
	Message string
}

type testTransportResponse struct {
	Message string
}

func (ph *testTransportPeerHandler) onPeer(p *Peer) {
	ph.logger.Println("onPeer", p)
	if ph.expectedSecureSuite != SecureSuiteUnknown {
		//assert secure connection
		switch c := p.conn.(type) {
		case *SecureConn:
			assert.Equal(ph.t, ph.expectedSecureSuite, SecureSuite(SecureSuiteEcdhe))
			assert.Equal(ph.t, ph.expectedSecureAeadSuite, p.secureKey.sa)
			assert.Equal(ph.t, ph.expectedSecureAeadSuite, c.sa)
		case *tls.Conn:
			assert.Equal(ph.t, ph.expectedSecureSuite, SecureSuite(SecureSuiteTls))
			assert.Equal(ph.t, ph.expectedSecureAeadSuite, p.secureKey.sa)
			var cs uint16
			switch ph.expectedSecureAeadSuite {
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
			assert.Equal(ph.t, ph.expectedSecureSuite, SecureSuite(SecureSuiteNone))
			assert.Equal(ph.t, SecureAeadSuite(SecureAeadSuiteNone), p.secureKey.sa)
		}
	}

	if !p.In() {
		m := &testTransportRequest{Message: "Hello"}
		ph.sendMessage(ProtoTestTransport, ProtoTestTransportRequest, m, p)
		ph.logger.Println("sendProtoTestTransportRequest", m, p)
	}
}

func (ph *testTransportPeerHandler) onPacket(pkt *Packet, p *Peer) {
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

func walletFromGeneratedPrivateKey() module.Wallet {
	priK, _ := crypto.GenerateKeyPair()
	w, _ := wallet.NewFromPrivateKey(priK)
	return w
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
	na1 := getAvailableLocalhostAddress(t)
	nt1 := NewTransport(na1, w1, l1).(*transport)
	if err := nt1.SetListenAddress(na1); err != nil {
		assert.FailNow(t, err.Error(), "Transport1.SetListenAddress fail")
	}
	assert.Equal(t, na1, nt1.GetListenAddress())

	w2 := walletFromGeneratedPrivateKey()
	l2 := log.WithFields(log.Fields{
		log.FieldKeyWallet: hex.EncodeToString(w2.Address().ID()),
	})
	l2.SetLevel(lv)
	na2 := getAvailableLocalhostAddress(t)
	nt2 := NewTransport(na2, w2, l2).(*transport)
	if err := nt2.SetListenAddress(na2); err != nil {
		assert.FailNow(t, err.Error(), "Transport1.SetListenAddress fail")
	}
	assert.Equal(t, na2, nt2.GetListenAddress())

	dph := newTestPeerHandler(nt2.PeerID(), nt2.logger)
	dph.nextOnPeerFunc = func(p *Peer) {
		time.Sleep(10 * time.Millisecond)
		dph.peerHandler.nextOnPeer(p)
	}
	nt2.pd.registerPeerHandler(dph, false)

	tph1 := newTestTransportPeerHandler("TestPeerHandler1", t, nt1.PeerID(), nt1.logger)
	tph2 := newTestTransportPeerHandler("TestPeerHandler2", t, nt2.PeerID(), nt2.logger)
	tph2.wg = &sync.WaitGroup{}

	mtr := metric.NewNetworkMetric(metric.DefaultMetricContext())
	nt1.registerPeerHandler(testChannel, tph1, mtr)
	nt2.registerPeerHandler(testChannel, tph2, mtr)

	nt1.addProtocol(testChannel, p2pProtoControl)
	nt2.addProtocol(testChannel, p2pProtoControl)

	if err := nt1.Listen(); err != nil {
		assert.FailNow(t, err.Error(), "Transport1.Start fail")
	}
	if err := nt2.Listen(); err != nil {
		assert.FailNow(t, err.Error(), "Transport2.Start fail")
	}

	//enable secureKeyLogWriter
	DefaultSecureKeyLogWriter = &testKeyLogWriter{}
	d := nt2.GetDialer(testChannel)
	for _, ss := range sss {
		for _, sa := range sas {
			t.Log("SecureSuite:", ss, "SecureAeadSuite:", sa)

			strSS := sliceToString([]SecureSuite{ss})
			assert.NoError(t, nt1.SetSecureSuites(testChannel, strSS))
			assert.Equal(t, strSS, nt1.GetSecureSuites(testChannel))
			strSA := sliceToString([]SecureAeadSuite{sa})
			assert.NoError(t, nt1.SetSecureAeads(testChannel, strSA))
			assert.Equal(t, strSA, nt1.GetSecureAeads(testChannel))

			tph1.expectedSecureSuite = ss
			tph1.expectedSecureAeadSuite = sa
			tph2.expectedSecureSuite = ss
			tph2.expectedSecureAeadSuite = sa
			tph2.wg.Add(1)

			if err := d.Dial(nt1.Address()); err != nil {
				assert.FailNow(t, err.Error(), "Transport.Dial fail")
			}

			tph2.wg.Wait()
		}
	}
	assert.NoError(t, nt1.Close(), "Transport1.Close fail")
	assert.NoError(t, nt2.Close(), "Transport2.Close fail")
}
