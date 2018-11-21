package network

import (
	"sync"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

//Negotiation map<channel, map<membership.name, {protocol, []subProtocol}>>
type ChannelNegotiator struct {
	peerHandler
}

func newChannelNegotiator() *ChannelNegotiator {
	cn := &ChannelNegotiator{peerHandler{log: &logger{"ChannelNegotiator", ""}}}
	return cn
}

//callback from PeerHandler.nextOnPeer
func (cn *ChannelNegotiator) onPeer(p *Peer) {
	cn.log.Println("onPeer", p)
	if !p.incomming {
		cn.sendPacket(NewPacket(PROTO_CHAN_JOIN_REQ, []byte(p.channel)), p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (cn *ChannelNegotiator) onError(err error, p *Peer) {
	cn.log.Println("onError", err, p)
	cn.peerHandler.onError(err, p)
}

//callback from Peer.receiveRoutine
func (cn *ChannelNegotiator) onPacket(pkt *Packet, p *Peer) {
	cn.log.Println("onPacket", pkt)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case PROTO_CHAN_JOIN_REQ:
			p.channel = string(pkt.payload)
			cn.sendPacket(NewPacket(PROTO_CHAN_JOIN_RESP, []byte(p.channel)), p)
			cn.nextOnPeer(p)
		case PROTO_CHAN_JOIN_RESP:
			cn.nextOnPeer(p)
		}
	}
}

type Authenticator struct {
	peerHandler
	peers  map[module.PeerID]*Peer
	seq    int
	priKey *crypto.PrivateKey
	pubKey *crypto.PublicKey
	mtx    sync.Mutex
}

type AuthRequest struct {
	PubKey    []byte
	Encrypted []byte
}

type AuthKeyRequest struct {
	PubKey []byte
	Cheap  string
}

func newAuthenticator(priK *crypto.PrivateKey, pubK *crypto.PublicKey) *Authenticator {
	a := &Authenticator{priKey: priK, pubKey: pubK, peerHandler: peerHandler{log: &logger{"Authenticator", ""}}}
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
	a.log.Println("onPacket", pkt)
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
	s, _ := crypto.NewSignature(h, a.priKey)
	sb, _ := s.SerializeRSV()
	return sb
}

func (a *Authenticator) VerifySignature(s *crypto.Signature, p *Peer) bool {
	pb := p.pubKey.SerializeUncompressed()
	h := crypto.SHA3Sum256(pb)
	return s.Verify(h, p.pubKey)
}

func (a *Authenticator) sendPublicKeyRequest(p *Peer) {
	pkt := NewPacket(PROTO_AUTH_KEY_REQ, a.pubKey.SerializeCompressed())
	a.sendPacket(pkt, p)
	a.log.Println("sendPublicKeyRequest", pkt)
}

func (a *Authenticator) handlePublicKeyRequest(pkt *Packet, p *Peer) {
	p.pubKey, _ = crypto.ParsePublicKey(pkt.payload)
	p.id = NewPeerIDFromPublicKey(p.pubKey)
	a.log.Println("handlePublicKeyRequest", p.pubKey, p.id)
	if !p.id.Equal(pkt.src) {
		a.log.Println("handlePublicKeyRequest Warnning id doesnt match pkt:", pkt.src, ",expected:", p.id)
	}
	a.sendPacket(NewPacket(PROTO_AUTH_KEY_RESP, a.pubKey.SerializeCompressed()), p)
}

func (a *Authenticator) handlePublicKeyResponse(pkt *Packet, p *Peer) {
	p.pubKey, _ = crypto.ParsePublicKey(pkt.payload)
	p.id = NewPeerIDFromPublicKey(p.pubKey)
	a.log.Println("handlePublicKeyResponse", p.pubKey, p.id)
	if !p.id.Equal(pkt.src) {
		a.log.Println("handlePublicKeyResponse Warnning id doesnt match pkt:", pkt.src, ",expected:", p.id)
	}

	rpkt := NewPacket(PROTO_AUTH_SIGN_REQ, a.Signature())
	a.sendPacket(rpkt, p)
}

func (a *Authenticator) handleSignatureRequest(pkt *Packet, p *Peer) {
	s, _ := crypto.ParseSignature(pkt.payload)
	a.log.Println("handleSignatureRequest", s.String(), p.id)
	if a.VerifySignature(s, p) {
		rpkt := NewPacket(PROTO_AUTH_SIGN_RESP, a.Signature())
		a.sendPacket(rpkt, p)

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
	s, _ := crypto.ParseSignature(pkt.payload)
	a.log.Println("handleSignatureResponse", s.String(), p.id)
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
