package network

import (
	"sync"

	"github.com/icon-project/goloop/module"

	"github.com/icon-project/goloop/common/crypto"
)

//Negotiation map<channel, map<membership.name, {protocol, []subProtocol}>>
type ChannelNegotiator struct {
	*peerHandler
	netAddress NetAddress
}

func newChannelNegotiator(netAddress NetAddress) *ChannelNegotiator {
	cn := &ChannelNegotiator{
		netAddress:  netAddress,
		peerHandler: newPeerHandler(newLogger("ChannelNegotiator", "")),
	}
	return cn
}

//callback from PeerHandler.nextOnPeer
func (cn *ChannelNegotiator) onPeer(p *Peer) {
	cn.log.Println("onPeer", p)
	if !p.incomming {
		cn.sendJoinRequest(p)

	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (cn *ChannelNegotiator) onError(err error, p *Peer) {
	cn.log.Println("onError", err, p)
	cn.peerHandler.onError(err, p)
}

//callback from Peer.receiveRoutine
func (cn *ChannelNegotiator) onPacket(pkt *Packet, p *Peer) {
	cn.log.Println("onPacket", pkt, p)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case PROTO_CHAN_JOIN_REQ:
			cn.handleJoinRequest(pkt, p)
		case PROTO_CHAN_JOIN_RESP:
			cn.handleJoinResponse(pkt, p)
		}
	}
}

type JoinRequest struct {
	Channel string
	Addr    NetAddress
}

type JoinResponse struct {
	Channel string
	Addr    NetAddress
}

func (cn *ChannelNegotiator) sendJoinRequest(p *Peer) {
	m := &JoinRequest{Channel: p.channel, Addr: cn.netAddress}
	cn.sendPacket(PROTO_CHAN_JOIN_REQ, m, p)
	cn.log.Println("sendJoinRequest", m, p)
}

func (cn *ChannelNegotiator) handleJoinRequest(pkt *Packet, p *Peer) {
	rm := &JoinRequest{}
	cn.decode(pkt.payload, rm)
	cn.log.Println("handleJoinRequest", rm, p)
	p.channel = rm.Channel
	p.netAddress = rm.Addr

	m := &JoinResponse{Channel: p.channel, Addr: cn.netAddress}
	cn.sendPacket(PROTO_CHAN_JOIN_RESP, m, p)

	cn.nextOnPeer(p)
}

func (cn *ChannelNegotiator) handleJoinResponse(pkt *Packet, p *Peer) {
	rm := &JoinResponse{}
	cn.decode(pkt.payload, rm)
	cn.log.Println("handleJoinResponse", rm, p)
	p.channel = rm.Channel
	p.netAddress = rm.Addr

	cn.nextOnPeer(p)
}

type Authenticator struct {
	*peerHandler
	wallet module.Wallet
	pubKey *crypto.PublicKey
	mtx    sync.Mutex
}

func newAuthenticator(w module.Wallet) *Authenticator {
	pubK, err := crypto.ParsePublicKey(w.PublicKey())
	if err != nil {
		panic(err)
	}
	a := &Authenticator{
		wallet:      w,
		pubKey:      pubK,
		peerHandler: newPeerHandler(newLogger("Authenticator", "")),
	}
	return a
}

//callback from PeerHandler.nextOnPeer
func (a *Authenticator) onPeer(p *Peer) {
	a.log.Println("onPeer", p)
	if !p.incomming {
		a.sendPublicKeyRequest(p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (a *Authenticator) onError(err error, p *Peer) {
	a.log.Println("onError", err, p)
	a.peerHandler.onError(err, p)
}

//callback from Peer.receiveRoutine
func (a *Authenticator) onPacket(pkt *Packet, p *Peer) {
	a.log.Println("onPacket", pkt, p)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case PROTO_AUTH_KEY_REQ:
			a.handlePublicKeyRequest(pkt, p)
		case PROTO_AUTH_KEY_RESP:
			a.handlePublicKeyResponse(pkt, p)
		case PROTO_AUTH_SIGN_REQ:
			a.handleSignatureRequest(pkt, p)
		case PROTO_AUTH_SIGN_RESP:
			a.handleSignatureResponse(pkt, p)
		}
	default:
		//ignore
	}
}

func (a *Authenticator) Signature() []byte {
	defer a.mtx.Unlock()
	a.mtx.Lock()
	pb := a.pubKey.SerializeUncompressed()
	h := crypto.SHA3Sum256(pb)
	sb, _ := a.wallet.Sign(h)
	return sb
}

func (a *Authenticator) VerifySignature(s *crypto.Signature, p *Peer) bool {
	pb := p.pubKey.SerializeUncompressed()
	h := crypto.SHA3Sum256(pb)
	return s.Verify(h, p.pubKey)
}

type PublicKeyRequest struct {
	PublicKey []byte
}
type PublicKeyResponse struct {
	PublicKey []byte
}
type SignatureRequest struct {
	Signature []byte
}
type SignatureResponse struct {
	Signature []byte
}

func (a *Authenticator) sendPublicKeyRequest(p *Peer) {
	m := &PublicKeyRequest{PublicKey: a.pubKey.SerializeCompressed()}
	a.sendPacket(PROTO_AUTH_KEY_REQ, m, p)
	a.log.Println("sendPublicKeyRequest", m, p)
}

func (a *Authenticator) handlePublicKeyRequest(pkt *Packet, p *Peer) {
	rm := &PublicKeyRequest{}
	a.decode(pkt.payload, rm)
	a.log.Println("handlePublicKeyRequest", rm, p)
	p.pubKey, _ = crypto.ParsePublicKey(rm.PublicKey)
	p.id = NewPeerIDFromPublicKey(p.pubKey)
	if !p.id.Equal(pkt.src) {
		a.log.Println("handlePublicKeyRequest Warnning id doesnt match pkt:", pkt.src, ",expected:", p.id)
	}

	m := &PublicKeyResponse{PublicKey: a.pubKey.SerializeCompressed()}
	a.sendPacket(PROTO_AUTH_KEY_RESP, m, p)
}

func (a *Authenticator) handlePublicKeyResponse(pkt *Packet, p *Peer) {
	rm := &PublicKeyResponse{}
	a.decode(pkt.payload, rm)
	a.log.Println("handlePublicKeyResponse", rm, p)
	p.pubKey, _ = crypto.ParsePublicKey(rm.PublicKey)
	p.id = NewPeerIDFromPublicKey(p.pubKey)
	if !p.id.Equal(pkt.src) {
		a.log.Println("handlePublicKeyResponse Warnning id doesnt match pkt:", pkt.src, ",expected:", p.id)
	}

	m := &SignatureRequest{Signature: a.Signature()}
	a.sendPacket(PROTO_AUTH_SIGN_REQ, m, p)
}

func (a *Authenticator) handleSignatureRequest(pkt *Packet, p *Peer) {
	rm := &SignatureRequest{}
	a.decode(pkt.payload, rm)
	a.log.Println("handleSignatureRequest", rm, p)
	s, _ := crypto.ParseSignature(rm.Signature)
	if a.VerifySignature(s, p) {
		m := &SignatureResponse{Signature: a.Signature()}
		a.sendPacket(PROTO_AUTH_SIGN_RESP, m, p)

		a.nextOnPeer(p)
	} else {
		a.log.Println("handleSignatureRequest Incomming PeerId Invalid signature, try close")
		err := p.conn.Close()
		if err != nil {
			a.log.Println("handleSignatureRequest p.conn.Close()", err)
		}
	}
}

func (a *Authenticator) handleSignatureResponse(pkt *Packet, p *Peer) {
	rm := &SignatureResponse{}
	a.decode(pkt.payload, rm)
	a.log.Println("handleSignatureResponse", rm, p)
	s, _ := crypto.ParseSignature(rm.Signature)
	if a.VerifySignature(s, p) {
		a.nextOnPeer(p)
	} else {
		a.log.Println("handleSignatureResponse Outgoing PeerId Invalid signature, try close")
		err := p.conn.Close()
		if err != nil {
			a.log.Println("handleSignatureResponse p.conn.Close()", err)
		}
	}
}
