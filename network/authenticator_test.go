package network

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

const (
	testSecureSuite     = SecureSuiteEcdhe + 1
	testSecureAeadSuite = SecureAeadSuiteAes256Gcm + 1
)

func Test_Authenticator(t *testing.T) {
	w := walletFromGeneratedPrivateKey()
	a := newAuthenticator(w, testLogger())
	assert.Equal(t, DefaultSecureSuites, a.GetSecureSuites(testChannel))
	assert.False(t, a.isSupportedSecureSuite(testChannel, testSecureSuite))
	assert.Equal(t, DefaultSecureAeadSuites, a.GetSecureAeads(testChannel))
	assert.False(t, a.isSupportedSecureAeadSuite(testChannel, testSecureAeadSuite))

	p := w.PublicKey()
	b := []byte("test")
	s := a.Signature(b)
	pid, err := a.VerifySignature(p, s, b)
	assert.NoError(t, err)
	assert.Equal(t, NewPeerIDFromAddress(w.Address()), pid)

	//fail to parse public key
	_, err = a.VerifySignature(b, s, b)
	assert.Error(t, err)
	//fail to parse signature
	_, err = a.VerifySignature(p, s[:0], b)
	assert.Error(t, err)
	//fail to verify signature
	_, err = a.VerifySignature(p, a.Signature(b[:0]), b)
	assert.Error(t, err)

	sss := []SecureSuite{
		SecureSuiteNone,
		SecureSuiteTls,
		SecureSuiteEcdhe,
		testSecureSuite,
	}
	err = a.SetSecureSuites(testChannel, sss)
	assert.NoError(t, err)
	assert.Equal(t, sss, a.GetSecureSuites(testChannel))

	for _, ss := range a.GetSecureSuites(testChannel) {
		assert.True(t, a.isSupportedSecureSuite(testChannel, ss))
	}
	assert.Equal(t, sss[0], a.resolveSecureSuite(testChannel, sss))
	assert.Equal(t, sss[1], a.resolveSecureSuite(testChannel, sss[1:]))
	assert.Equal(t, SecureSuite(SecureSuiteUnknown), a.resolveSecureSuite(testChannel, sss[:0]))

	//duplicated
	isss := []SecureSuite{SecureSuiteNone, SecureSuiteNone}
	err = a.SetSecureSuites(testChannel, isss)
	assert.Error(t, err)

	sas := []SecureAeadSuite{
		SecureAeadSuiteChaCha20Poly1305,
		SecureAeadSuiteAes128Gcm,
		SecureAeadSuiteAes256Gcm,
		testSecureAeadSuite,
	}
	err = a.SetSecureAeads(testChannel, sas)
	assert.NoError(t, err)
	assert.Equal(t, sas, a.GetSecureAeads(testChannel))

	for _, sa := range a.GetSecureAeads(testChannel) {
		assert.True(t, a.isSupportedSecureAeadSuite(testChannel, sa))
	}
	assert.Equal(t, sas[0], a.resolveSecureAeadSuite(testChannel, sas))
	assert.Equal(t, sas[1], a.resolveSecureAeadSuite(testChannel, sas[1:]))
	assert.Equal(t, SecureAeadSuite(SecureAeadSuiteNone), a.resolveSecureAeadSuite(testChannel, sas[:0]))

	//duplicated
	isas := []SecureAeadSuite{SecureAeadSuiteNone, SecureAeadSuiteNone}
	err = a.SetSecureAeads(testChannel, isas)
	assert.Error(t, err)
}

func Test_Authenticator_Request(t *testing.T) {
	w := walletFromGeneratedPrivateKey()
	a := newAuthenticator(w, testLogger())

	sk := newSecureKey(DefaultSecureEllipticCurve, DefaultSecureKeyLogWriter)
	givenSecureRequest := &SecureRequest{
		Channel:          testChannel,
		SecureSuites:     []SecureSuite{SecureSuiteNone},
		SecureAeadSuites: []SecureAeadSuite{SecureAeadSuiteNone},
		SecureParam:      sk.marshalPublicKey(),
	}
	expectSecureResponse := &SecureResponse{
		Channel:         testChannel,
		SecureSuite:     SecureSuiteNone,
		SecureAeadSuite: SecureAeadSuiteNone,
		SecureError:     SecureErrorNone,
	}
	scens := []struct {
		givenSecureRequest      *SecureRequest
		expectSecureResponse    *SecureResponse
		givenSignatureRequest   *SignatureRequest
		expectSignatureResponse *SignatureResponse
	}{
		{
			givenSecureRequest: &SecureRequest{
				Channel:          testChannel,
				SecureSuites:     []SecureSuite{SecureSuiteUnknown},
				SecureAeadSuites: []SecureAeadSuite{SecureAeadSuiteChaCha20Poly1305},
				SecureParam:      sk.marshalPublicKey(),
			},
			expectSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteUnknown,
				SecureAeadSuite: SecureAeadSuiteNone,
				SecureError:     SecureErrorInvalid,
			},
		},
		{
			givenSecureRequest: &SecureRequest{
				Channel:          testChannel,
				SecureSuites:     []SecureSuite{SecureSuiteTls},
				SecureAeadSuites: []SecureAeadSuite{SecureAeadSuiteNone},
				SecureParam:      sk.marshalPublicKey(),
			},
			expectSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteTls,
				SecureAeadSuite: SecureAeadSuiteNone,
				SecureError:     SecureErrorInvalid,
			},
		},
		{
			givenSecureRequest: &SecureRequest{
				Channel:          testChannel,
				SecureSuites:     []SecureSuite{SecureSuiteTls},
				SecureAeadSuites: []SecureAeadSuite{SecureAeadSuiteChaCha20Poly1305},
				SecureParam:      []byte{0x00},
			},
			expectSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteTls,
				SecureAeadSuite: SecureAeadSuiteChaCha20Poly1305,
				SecureError:     SecureErrorNone,
			},
		},
		{
			givenSignatureRequest: &SignatureRequest{
				PublicKey: []byte{0x00},
				Signature: []byte{0x00},
				Rtt:       0,
			},
			expectSignatureResponse: &SignatureResponse{
				Error: "fail to parse public key : malformed public key: invalid length: 1",
			},
		},
		{
			givenSignatureRequest: &SignatureRequest{
				PublicKey: w.PublicKey(),
				Signature: []byte{0x00},
				Rtt:       0,
			},
			expectSignatureResponse: &SignatureResponse{
				Error: "fail to parse signature : wrong raw signature format",
			},
		},
		{
			givenSignatureRequest: &SignatureRequest{
				PublicKey: w.PublicKey(),
				Signature: a.Signature([]byte{0x00}),
				Rtt:       0,
			},
			expectSignatureResponse: &SignatureResponse{
				Error: "InvalidSignatureError",
			},
		},
	}
	for _, scen := range scens {
		p, conn := newPeerWithFakeConn(true)
		a.onPeer(p)
		t.Log(p)

		var secureRequest *SecureRequest
		if scen.givenSecureRequest != nil {
			secureRequest = scen.givenSecureRequest
		} else {
			secureRequest = givenSecureRequest
		}
		pkt := newPacket(p2pProtoAuth, p2pProtoAuthSecureRequest,
			codec.MP.MustMarshalToBytes(secureRequest), nil)
		a.handleSecureRequest(pkt, p)

		pkt = conn.Packet()
		assert.NotNil(t, pkt)
		actualSecureResponse := &SecureResponse{}
		if err := a.decode(pkt.payload, actualSecureResponse); err != nil {
			assert.FailNow(t, err.Error())
		}
		assert.Equal(t, testChannel, p.Channel())
		var secureResponse *SecureResponse
		if scen.expectSecureResponse != nil {
			secureResponse = scen.expectSecureResponse
		} else {
			secureResponse = expectSecureResponse
		}
		secureResponse.SecureParam = p.secureKey.marshalPublicKey()
		assert.Equal(t, *secureResponse, *actualSecureResponse)

		if scen.givenSignatureRequest != nil {
			pkt = newPacket(
				p2pProtoAuth,
				p2pProtoAuthSignatureRequest,
				codec.MP.MustMarshalToBytes(scen.givenSignatureRequest),
				nil)
			a.handleSignatureRequest(pkt, p)
			pkt = conn.Packet()
			assert.NotNil(t, pkt)
			actualSignatureResponse := &SignatureResponse{}
			if err := a.decode(pkt.payload, actualSignatureResponse); err != nil {
				assert.FailNow(t, err.Error())
			}
			assert.Equal(t, testChannel, p.Channel())
			assert.Equal(t, *scen.expectSignatureResponse, *actualSignatureResponse)
		}
		assert.True(t, p.IsClosed())
	}
}

func Test_Authenticator_Response(t *testing.T) {
	w := walletFromGeneratedPrivateKey()
	a := newAuthenticator(w, testLogger())

	sk := newSecureKey(DefaultSecureEllipticCurve, DefaultSecureKeyLogWriter)
	expectSecureRequest := &SecureRequest{
		Channel:          testChannel,
		SecureSuites:     DefaultSecureSuites,
		SecureAeadSuites: DefaultSecureAeadSuites,
		//SecureParam:      p.secureKey.marshalPublicKey(),
	}
	givenSecureResponse := &SecureResponse{
		Channel:         testChannel,
		SecureSuite:     SecureSuiteNone,
		SecureAeadSuite: SecureAeadSuiteNone,
		SecureParam:     sk.marshalPublicKey(),
		SecureError:     SecureErrorNone,
	}
	scens := []struct {
		givenSecureResponse    *SecureResponse
		givenSignatureResponse *SignatureResponse
	}{
		{
			givenSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteUnknown,
				SecureAeadSuite: SecureAeadSuiteNone,
				SecureParam:     sk.marshalPublicKey(),
				SecureError:     SecureErrorInvalid,
			},
		},
		{
			givenSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteUnknown,
				SecureAeadSuite: SecureAeadSuiteChaCha20Poly1305,
				SecureParam:     sk.marshalPublicKey(),
				SecureError:     SecureErrorNone,
			},
		},
		{
			givenSecureResponse: &SecureResponse{
				Channel:         testChannel,
				SecureSuite:     SecureSuiteTls,
				SecureAeadSuite: SecureAeadSuiteNone,
				SecureParam:     sk.marshalPublicKey(),
				SecureError:     SecureErrorNone,
			},
		},
		{
			givenSignatureResponse: &SignatureResponse{
				Error: "error",
			},
		},
		{
			givenSignatureResponse: &SignatureResponse{
				PublicKey: []byte{0x00},
				Signature: a.Signature(sk.extra),
				Rtt:       0,
			},
		},
		{
			givenSignatureResponse: &SignatureResponse{
				PublicKey: w.PublicKey(),
				Signature: []byte{0x00},
				Rtt:       0,
			},
		},
		{
			givenSignatureResponse: &SignatureResponse{
				PublicKey: w.PublicKey(),
				Signature: a.Signature(sk.extra),
				Rtt:       0,
			},
		},
	}
	for _, scen := range scens {
		p, conn := newPeerWithFakeConn(false)
		//setChannel in PeerDispatcher.onConnect
		p.setChannel(testChannel)
		a.onPeer(p)
		t.Log(p)
		pkt := conn.Packet()
		assert.NotNil(t, pkt)
		if p.IsClosed() {
			assert.FailNow(t, "closed")
		}
		actualSecureRequest := &SecureRequest{}
		if err := a.decode(pkt.payload, actualSecureRequest); err != nil {
			assert.FailNow(t, err.Error())
		}
		expectSecureRequest.SecureParam = p.secureKey.marshalPublicKey()
		assert.Equal(t, *expectSecureRequest, *actualSecureRequest)

		var secureResponse *SecureResponse
		if scen.givenSecureResponse != nil {
			secureResponse = scen.givenSecureResponse
		} else {
			secureResponse = givenSecureResponse
		}
		pkt = newPacket(p2pProtoAuth, p2pProtoAuthSecureResponse,
			codec.MP.MustMarshalToBytes(secureResponse), nil)
		a.handleSecureResponse(pkt, p)

		if scen.givenSignatureResponse != nil {
			pkt = conn.Packet()
			assert.NotNil(t, pkt)
			if p.IsClosed() {
				assert.FailNow(t, "closed")
			}
			actualSignatureRequest := &SignatureRequest{}
			if err := a.decode(pkt.payload, actualSignatureRequest); err != nil {
				assert.FailNow(t, err.Error())
			}
			assert.Equal(t, w.PublicKey(), actualSignatureRequest.PublicKey)
			assert.Equal(t, a.Signature(p.secureKey.extra), actualSignatureRequest.Signature)
			last, _ := p.rtt.Value()
			assert.Equal(t, last, actualSignatureRequest.Rtt)

			pkt = newPacket(p2pProtoAuth, p2pProtoAuthSignatureResponse,
				codec.MP.MustMarshalToBytes(scen.givenSignatureResponse), nil)
			a.handleSignatureResponse(pkt, p)
		}
		assert.True(t, p.IsClosed())
	}
}

func Test_Authenticator_Packet(t *testing.T) {
	w := walletFromGeneratedPrivateKey()
	a := newAuthenticator(w, testLogger())

	sk := newSecureKey(DefaultSecureEllipticCurve, DefaultSecureKeyLogWriter)
	secureRequest := newPacket(
		p2pProtoAuth,
		p2pProtoAuthSecureRequest,
		codec.MP.MustMarshalToBytes(&SecureRequest{
			Channel:          testChannel,
			SecureSuites:     []SecureSuite{SecureSuiteNone},
			SecureAeadSuites: []SecureAeadSuite{SecureAeadSuiteNone},
			SecureParam:      sk.marshalPublicKey(),
		}),
		nil)
	secureResponse := newPacket(
		p2pProtoAuth,
		p2pProtoAuthSecureResponse,
		codec.MP.MustMarshalToBytes(&SecureResponse{
			Channel:         testChannel,
			SecureSuite:     SecureSuiteNone,
			SecureAeadSuite: SecureAeadSuiteNone,
			SecureError:     SecureErrorNone,
			SecureParam:     sk.marshalPublicKey(),
		}),
		nil)

	args := []struct {
		in          bool
		givenPacket *Packet
		wait        module.ProtocolInfo
		invalidWait module.ProtocolInfo
	}{
		{
			in:          true,
			wait:        p2pProtoAuthSecureRequest,
			invalidWait: p2pProtoAuthSecureResponse,
		},
		{
			in:          false,
			wait:        p2pProtoAuthSecureResponse,
			invalidWait: p2pProtoAuthSecureRequest,
		},
		{
			in:          true,
			givenPacket: secureRequest,
			wait:        p2pProtoAuthSignatureRequest,
			invalidWait: p2pProtoAuthSignatureResponse,
		},
		{
			in:          false,
			givenPacket: secureResponse, //sendReq, waitResp, sendReq, waitResp, invalidWait
			wait:        p2pProtoAuthSignatureResponse,
			invalidWait: p2pProtoAuthSignatureRequest,
		},
	}
	for _, arg := range args {
		for _, pi := range []module.ProtocolInfo{arg.wait, arg.invalidWait} {
			p, _ := newPeerWithFakeConn(arg.in)
			a.onPeer(p)

			if arg.givenPacket != nil {
				a.onPacket(arg.givenPacket, p)
			}
			pkt := newPacket(p2pProtoAuth, pi, []byte{0x00}, nil)
			a.onPacket(pkt, p)
			assert.True(t, p.IsClosed())
		}
	}

	p, _ := newPeerWithFakeConn(true)
	a.onPeer(p)
	pkt := newPacket(p2pProtoAuth, module.ProtocolInfo(0xFFFF), []byte{0x00}, nil)
	a.onPacket(pkt, p)
	assert.True(t, p.HasCloseError(ErrNotRegisteredProtocol))
}
