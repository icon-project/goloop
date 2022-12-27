package network

import (
	"crypto/elliptic"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

var (
	p2pProtoAuth                  = module.ProtocolInfo(0x0000)
	p2pProtoAuthSecureRequest     = module.ProtocolInfo(0x0100)
	p2pProtoAuthSecureResponse    = module.ProtocolInfo(0x0200)
	p2pProtoAuthSignatureRequest  = module.ProtocolInfo(0x0300)
	p2pProtoAuthSignatureResponse = module.ProtocolInfo(0x0400)

	DefaultSecureEllipticCurve = elliptic.P256()
	DefaultSecureSuites        = []SecureSuite{
		SecureSuiteNone,
		SecureSuiteTls,
		SecureSuiteEcdhe,
	}
	DefaultSecureAeadSuites = []SecureAeadSuite{
		SecureAeadSuiteChaCha20Poly1305,
		SecureAeadSuiteAes128Gcm,
		SecureAeadSuiteAes256Gcm,
	}
	DefaultSecureKeyLogWriter io.Writer
)

type Authenticator struct {
	*peerHandler
	wallet       module.Wallet
	secureSuites map[string][]SecureSuite
	secureAeads  map[string][]SecureAeadSuite
	secureKeyNum int
	secureMtx    sync.RWMutex
	mtx          sync.Mutex
}

func newAuthenticator(w module.Wallet, l log.Logger) *Authenticator {
	_, err := crypto.ParsePublicKey(w.PublicKey())
	if err != nil {
		panic(err)
	}
	a := &Authenticator{
		wallet:       w,
		secureSuites: make(map[string][]SecureSuite),
		secureAeads:  make(map[string][]SecureAeadSuite),
		secureKeyNum: 2,
		peerHandler:  newPeerHandler(l.WithFields(log.Fields{LoggerFieldKeySubModule: "authenticator"})),
	}
	return a
}

//callback from PeerHandler.nextOnPeer
func (a *Authenticator) onPeer(p *Peer) {
	a.logger.Traceln("onPeer", p)
	if !p.In() {
		a.setWaitInfo(p2pProtoAuthSecureResponse, p)
		a.sendSecureRequest(p)
	} else {
		a.setWaitInfo(p2pProtoAuthSecureRequest, p)
	}
}

//callback from Peer.receiveRoutine
func (a *Authenticator) onPacket(pkt *Packet, p *Peer) {
	switch pkt.protocol {
	case p2pProtoAuth:
		switch pkt.subProtocol {
		case p2pProtoAuthSecureRequest:
			a.handleSecureRequest(pkt, p)
		case p2pProtoAuthSecureResponse:
			a.handleSecureResponse(pkt, p)
		case p2pProtoAuthSignatureRequest:
			a.handleSignatureRequest(pkt, p)
		case p2pProtoAuthSignatureResponse:
			a.handleSignatureResponse(pkt, p)
		default:
			p.CloseByError(ErrNotRegisteredProtocol)
		}
	default:
		//ignore
	}
}

func (a *Authenticator) Signature(content []byte) []byte {
	defer a.mtx.Unlock()
	a.mtx.Lock()
	h := crypto.SHA3Sum256(content)
	sb, _ := a.wallet.Sign(h)
	return sb
}

func (a *Authenticator) VerifySignature(publicKey []byte, signature []byte, content []byte) (module.PeerID, error) {
	pubKey, err := crypto.ParsePublicKey(publicKey)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidSignature, "fail to parse public key : %s", err.Error())
	}
	s, err := crypto.ParseSignature(signature)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidSignature, "fail to parse signature : %s", err.Error())
	}
	h := crypto.SHA3Sum256(content)
	if !s.Verify(h, pubKey) {
		err = ErrInvalidSignature
	}
	return NewPeerIDFromPublicKey(pubKey), err
}

func (a *Authenticator) SetSecureSuites(channel string, ss []SecureSuite) error {
	a.secureMtx.Lock()
	defer a.secureMtx.Unlock()

	for i, s := range ss {
		for j := i + 1; j < len(ss); j++ {
			if s == ss[j] {
				return fmt.Errorf("duplicate set %s index:%d and %d", s, i, j)
			}
		}
	}
	a.secureSuites[channel] = ss
	return nil
}

func (a *Authenticator) GetSecureSuites(channel string) []SecureSuite {
	a.secureMtx.RLock()
	defer a.secureMtx.RUnlock()

	suites, ok := a.secureSuites[channel]
	if !ok || len(suites) == 0 {
		return DefaultSecureSuites
	}
	return suites
}

func (a *Authenticator) isSupportedSecureSuite(channel string, ss SecureSuite) bool {
	osss := a.secureSuites[channel]
	if len(osss) == 0 {
		osss = DefaultSecureSuites
	}
	for _, oss := range osss {
		if oss == ss {
			return true
		}
	}
	return false
}

func (a *Authenticator) resolveSecureSuite(channel string, sss []SecureSuite) SecureSuite {
	for _, ss := range sss {
		if a.isSupportedSecureSuite(channel, ss) {
			return ss
		}
	}
	return SecureSuiteUnknown
}

func (a *Authenticator) SetSecureAeads(channel string, sas []SecureAeadSuite) error {
	a.secureMtx.Lock()
	defer a.secureMtx.Unlock()

	for i, sa := range sas {
		for j := i + 1; j < len(sas); j++ {
			if sa == sas[j] {
				return fmt.Errorf("duplicate set %s index:%d and %d", sa, i, j)
			}
		}
	}
	a.secureAeads[channel] = sas
	return nil
}

func (a *Authenticator) GetSecureAeads(channel string) []SecureAeadSuite {
	a.secureMtx.RLock()
	defer a.secureMtx.RUnlock()

	aeads, ok := a.secureAeads[channel]
	if !ok || len(aeads) == 0 {
		return DefaultSecureAeadSuites
	}
	return aeads
}

func (a *Authenticator) isSupportedSecureAeadSuite(channel string, sas SecureAeadSuite) bool {
	osass := a.secureAeads[channel]
	if len(osass) == 0 {
		osass = DefaultSecureAeadSuites
	}
	for _, osas := range osass {
		if osas == sas {
			return true
		}
	}
	return false
}

func (a *Authenticator) resolveSecureAeadSuite(channel string, sass []SecureAeadSuite) SecureAeadSuite {
	for _, sas := range sass {
		if a.isSupportedSecureAeadSuite(channel, sas) {
			return sas
		}
	}
	return SecureAeadSuiteNone
}

func (a *Authenticator) applySecureConn(p *Peer, ss SecureSuite, sas SecureAeadSuite, param []byte, req bool) error {
	if !a.isSupportedSecureSuite(p.Channel(), ss) {
		return errors.Wrapf(ErrIllegalArgument, "invalid SecureSuite %d", ss)
	}
	//When SecureSuite is SecureSuiteNone, fix SecureAeadSuite as SecureAeadSuiteNone
	if ss == SecureSuiteNone {
		sas = SecureAeadSuiteNone
	} else if !a.isSupportedSecureAeadSuite(p.Channel(), sas) {
		return errors.Wrapf(ErrIllegalArgument, "invalid SecureAeadSuite %d", ss)
	}
	if err := p.secureKey.setup(sas, param, p.In(), a.secureKeyNum); err != nil {
		return errors.Wrapf(err, "fail to secureKey.setup")
	}
	switch ss {
	case SecureSuiteEcdhe:
		if secureConn, err := NewSecureConn(p.conn, sas, p.secureKey); err != nil {
			return err
		} else {
			p.ResetConn(secureConn)
		}
	case SecureSuiteTls:
		if config, err := p.secureKey.tlsConfig(); err != nil {
			return err
		} else {
			var tlsConn *tls.Conn
			if req {
				tlsConn = tls.Server(p.conn, config)
			} else {
				tlsConn = tls.Client(p.conn, config)
				if err = tlsConn.Handshake(); err != nil {
					return err
				}
			}
			p.ResetConn(tlsConn)
		}
	default:
		//SecureSuiteNone:
		//Nothing to do
	}
	return nil
}

type SecureRequest struct {
	Channel          string
	SecureSuites     []SecureSuite
	SecureAeadSuites []SecureAeadSuite
	SecureParam      []byte
}
type SecureResponse struct {
	Channel         string
	SecureSuite     SecureSuite
	SecureAeadSuite SecureAeadSuite
	SecureParam     []byte
	SecureError     SecureError
}
type SignatureRequest struct {
	PublicKey []byte
	Signature []byte
	Rtt       time.Duration
}
type SignatureResponse struct {
	PublicKey []byte
	Signature []byte
	Rtt       time.Duration
	Error     string
}

func (a *Authenticator) sendSecureRequest(p *Peer) {
	p.secureKey = newSecureKey(DefaultSecureEllipticCurve, DefaultSecureKeyLogWriter)
	sms := a.secureSuites[p.Channel()]
	if len(sms) == 0 {
		sms = DefaultSecureSuites
	}
	sas := a.secureAeads[p.Channel()]
	if len(sas) == 0 {
		sas = DefaultSecureAeadSuites
	}
	m := &SecureRequest{
		Channel:          p.Channel(),
		SecureSuites:     sms,
		SecureAeadSuites: sas,
		SecureParam:      p.secureKey.marshalPublicKey(),
	}

	p.rtt.Start()
	a.sendMessage(p2pProtoAuth, p2pProtoAuthSecureRequest, m, p)
	a.logger.Traceln("sendSecureRequest", m, p)
}

func (a *Authenticator) handleSecureRequest(pkt *Packet, p *Peer) {
	if !a.checkWaitInfo(pkt, p) {
		return
	}

	rm := &SecureRequest{}
	if !a.decodePeerPacket(p, rm, pkt) {
		return
	}
	a.logger.Traceln("handleSecureRequest", rm, p)
	p.setChannel(rm.Channel)
	m := &SecureResponse{
		Channel:         p.Channel(),
		SecureSuite:     a.resolveSecureSuite(p.Channel(), rm.SecureSuites),
		SecureAeadSuite: SecureAeadSuiteNone,
		SecureError:     SecureErrorNone,
	}

	a.logger.Traceln("handleSecureRequest", p.ConnString(), "SecureSuite", m.SecureSuite)
	if m.SecureSuite == SecureSuiteUnknown {
		m.SecureError = SecureErrorInvalid
	} else if m.SecureSuite != SecureSuiteNone {
		m.SecureAeadSuite = a.resolveSecureAeadSuite(p.Channel(), rm.SecureAeadSuites)
		a.logger.Traceln("handleSecureRequest", p.ConnString(), "SecureAeadSuite", m.SecureAeadSuite)
		if m.SecureAeadSuite == SecureAeadSuiteNone {
			m.SecureError = SecureErrorInvalid
		}
	} else {
		//in case of m.SecureSuite is SecureSuiteNone for legacy Authenticator which is not supported SecureAeadSuiteNone
		m.SecureAeadSuite = a.resolveSecureAeadSuite(p.Channel(), rm.SecureAeadSuites)
		a.logger.Traceln("handleSecureRequest", p.ConnString(), "SecureAeadSuite", m.SecureAeadSuite)
	}

	p.secureKey = newSecureKey(DefaultSecureEllipticCurve, DefaultSecureKeyLogWriter)
	m.SecureParam = p.secureKey.marshalPublicKey()

	if m.SecureError != SecureErrorNone {
		a.sendMessage(p2pProtoAuth, p2pProtoAuthSecureResponse, m, p)
		err := fmt.Errorf("handleSecureRequest error[%v]", m.SecureError)
		a.logger.Infoln("handleSecureRequest", p.ConnString(), "SecureError", err)
		p.CloseByError(err)
		return
	}

	p.rtt.Start()
	a.setWaitInfo(p2pProtoAuthSignatureRequest, p)
	a.sendMessage(p2pProtoAuth, p2pProtoAuthSecureResponse, m, p)

	if err := a.applySecureConn(p, m.SecureSuite, m.SecureAeadSuite, rm.SecureParam, true); err != nil {
		a.logger.Infoln("handleSecureRequest", p.ConnString(), "failed SecureConn", err)
		p.CloseByError(err)
		return
	}
}

func (a *Authenticator) handleSecureResponse(pkt *Packet, p *Peer) {
	if !a.checkWaitInfo(pkt, p) {
		return
	}

	rm := &SecureResponse{}
	if !a.decodePeerPacket(p, rm, pkt) {
		return
	}
	a.logger.Traceln("handleSecureResponse", rm, p)
	rttLast := p.rtt.Stop()

	if rm.SecureError != SecureErrorNone {
		err := fmt.Errorf("handleSecureResponse error[%v]", rm.SecureError)
		a.logger.Infoln("handleSecureResponse", p.ConnString(), "SecureError", err)
		p.CloseByError(err)
		return
	}

	if err := a.applySecureConn(p, rm.SecureSuite, rm.SecureAeadSuite, rm.SecureParam, false); err != nil {
		a.logger.Infoln("handleSecureResponse", p.ConnString(), "failed SecureConn", err)
		p.CloseByError(err)
		return
	}

	m := &SignatureRequest{
		PublicKey: a.wallet.PublicKey(),
		Signature: a.Signature(p.secureKey.extra),
		Rtt:       rttLast,
	}
	a.setWaitInfo(p2pProtoAuthSignatureResponse, p)
	a.sendMessage(p2pProtoAuth, p2pProtoAuthSignatureRequest, m, p)
}

func (a *Authenticator) handleSignatureRequest(pkt *Packet, p *Peer) {
	if !a.checkWaitInfo(pkt, p) {
		return
	}

	rm := &SignatureRequest{}
	if !a.decodePeerPacket(p, rm, pkt) {
		return
	}
	a.logger.Traceln("handleSignatureRequest", rm, p)

	rttLast := p.rtt.Stop()
	df := rm.Rtt - rttLast
	if df > DefaultRttAccuracy {
		a.logger.Debugln("handleSignatureRequest", df, "DefaultRttAccuracy", DefaultRttAccuracy)
	}

	m := &SignatureResponse{
		PublicKey: a.wallet.PublicKey(),
		Signature: a.Signature(p.secureKey.extra),
		Rtt:       rttLast,
	}

	id, err := a.VerifySignature(rm.PublicKey, rm.Signature, p.secureKey.extra)
	if err != nil {
		m = &SignatureResponse{Error: err.Error()}
	} else if id.Equal(a.self) {
		m = &SignatureResponse{Error: "selfAddress"}
	}
	p.setID(id)
	a.sendMessage(p2pProtoAuth, p2pProtoAuthSignatureResponse, m, p)

	if m.Error != "" {
		err := fmt.Errorf("handleSignatureRequest error[%v]", m.Error)
		a.logger.Infoln("handleSignatureRequest", p.ConnString(), "Error", err)
		p.CloseByError(err)
		return
	}
	a.nextOnPeer(p)
}

func (a *Authenticator) handleSignatureResponse(pkt *Packet, p *Peer) {
	if !a.checkWaitInfo(pkt, p) {
		return
	}

	rm := &SignatureResponse{}
	if !a.decodePeerPacket(p, rm, pkt) {
		return
	}
	a.logger.Traceln("handleSignatureResponse", rm, p)

	rttLast, _ := p.rtt.Value()
	df := rm.Rtt - rttLast
	if df > DefaultRttAccuracy {
		a.logger.Debugln("handleSignatureResponse", df, "DefaultRttAccuracy", DefaultRttAccuracy)
	}

	if rm.Error != "" {
		err := fmt.Errorf("handleSignatureResponse error[%v]", rm.Error)
		a.logger.Infoln("handleSignatureResponse", p.ConnString(), "Error", err)
		p.CloseByError(err)
		return
	}

	id, err := a.VerifySignature(rm.PublicKey, rm.Signature, p.secureKey.extra)
	if err != nil {
		err := fmt.Errorf("handleSignatureResponse error[%v]", err)
		a.logger.Infoln("handleSignatureResponse", p.ConnString(), "Error", err)
		p.CloseByError(err)
		return
	}
	p.setID(id)
	if !p.ID().Equal(pkt.src) {
		a.logger.Infoln("handleSignatureResponse", "id doesnt match pkt:", pkt.src, ",expected:", p.ID())
	}
	a.nextOnPeer(p)
}
