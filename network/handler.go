package network

import (
	"log"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

//Negotiation map<channel, map<membership.name, {protocol, []subProtocol}>>
type ChannelNegotiator struct {
	peerHandler
}

func newChannelNegotiator() *ChannelNegotiator {
	return &ChannelNegotiator{}
}

//callback from PeerHandler.nextOnPeer
func (cn *ChannelNegotiator) onPeer(p *Peer) {
	log.Println("ChannelNegotiator.onPeer", p)
	if !p.incomming {
		cn.sendPacket(NewPacket(PROTO_CHAN_JOIN_REQ, []byte(p.channel)), p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (cn *ChannelNegotiator) onError(err error, p *Peer) {
	log.Println("ChannelNegotiator.onError", err, p)
	cn.peerHandler.onError(err, p)
}

//callback from Peer.receiveRoutine
func (cn *ChannelNegotiator) onPacket(pkt *Packet, p *Peer) {
	log.Println("ChannelNegotiator.onPacket", pkt)
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
	return &Authenticator{priKey: priK, pubKey: pubK}
}

//callback from PeerHandler.nextOnPeer
func (a *Authenticator) onPeer(p *Peer) {
	log.Println("Authenticator.onPeer", p)
	if !p.incomming {
		a.sendPacket(NewPacket(PROTO_AUTH_HS1, a.pubKey.SerializeCompressed()), p)
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (a *Authenticator) onError(err error, p *Peer) {
	log.Println("Authenticator.onError", err, p)
	a.peerHandler.onError(err, p)
}

//callback from Peer.receiveRoutine
func (a *Authenticator) onPacket(pkt *Packet, p *Peer) {
	log.Println("Authenticator.onPacket", pkt)
	switch pkt.protocol {
	case PROTO_CONTOL:
		switch pkt.subProtocol {
		case PROTO_AUTH_HS1:
			// req := unmarshall(pkt.payload)
			// p.pubKey := req.pubKey
			// p.authCheap := p.pubKey.decrypt(req.encrypted)
			// resp := {
			// 	encrypted: p.pubKey.encrypt({
			// 		authCheap: p.authCheap
			// 		encrypted: self.priKey.encrypt(self.authCheap),
			// 	}),
			// 	pubKey: self.pubKey,
			// }
			// p.sendPacket(NewPacket(PROTO_AUTH_HS2, marshall(resp)))

			p.pubKey, _ = crypto.ParsePublicKey(pkt.payload)
			p.id = NewPeerIDFromPublicKey(p.pubKey)
			if !p.id.Equal(pkt.src) {
				log.Println("Authenticator.onPacket PROTO_AUTH_HS1 Warnning id doesnt match[pkt:", pkt.src, ",expected:", p.id)
			}
			a.sendPacket(NewPacket(PROTO_AUTH_HS2, a.pubKey.SerializeCompressed()), p)
		case PROTO_AUTH_HS2:
			// resp := unmarshall(pkt.payload)
			// p.pubKey := resp.pubKey
			// authCheap, encrypted := self.priKey.decrypt(resp.encrypted)
			// if self.authCheap == authCheap {
			// 	 p.authCheap := p.pubKey.decrypt(encrypted)
			//   req := {
			//     encrypted: p.pubKey.encrypt(p.authCheap)
			//     channel: self.channel
			//   }
			//   p.sendPacket(NewPacket(PROTO_AUTH_HS3, marshall(req)))
			// }else{
			// 	 p.conn.Close()
			// }
			p.pubKey, _ = crypto.ParsePublicKey(pkt.payload)
			p.id = NewPeerIDFromPublicKey(p.pubKey)
			if !p.id.Equal(pkt.src) {
				log.Println("Authenticator.onPacket PROTO_AUTH_HS2 Warnning id doesnt match[pkt:", pkt.src, ",expected:", p.id)
			}
			s, _ := crypto.NewSignature(crypto.SHA3Sum256(a.pubKey.SerializeUncompressed()), a.priKey)
			sb, _ := s.SerializeRSV()
			a.sendPacket(NewPacket(PROTO_AUTH_HS3, sb), p)
		case PROTO_AUTH_HS3:
			// req := unmarshall(pkt.payload)
			// authCheap := self.priKey.decrypt(req.encrypted)
			// if self.authCheap == authCheap {
			//   resp := {result: "OK"}
			//   p.sendPacket(NewPacket(PROTO_AUTH_HS4, marshall(resp)))
			// }else{
			// 	p.conn.Close()
			// }
			s, _ := crypto.ParseSignature(pkt.payload)
			if s.Verify(crypto.SHA3Sum256(p.pubKey.SerializeUncompressed()), p.pubKey) {
				s, _ := crypto.NewSignature(crypto.SHA3Sum256(a.pubKey.SerializeUncompressed()), a.priKey)
				sb, _ := s.SerializeRSV()
				a.sendPacket(NewPacket(PROTO_AUTH_HS4, sb), p)
				a.nextOnPeer(p)
			} else {
				log.Println("Authenticator.onPacket PROTO_AUTH_HS3 Incomming PeerId Invalid signature, try close")
				err := p.conn.Close()
				if err != nil {
					log.Println("Authenticator.onPacket PROTO_AUTH_HS3 p.conn.Close()", err)
				}
			}
		case PROTO_AUTH_HS4:
			s, _ := crypto.ParseSignature(pkt.payload)
			if s.Verify(crypto.SHA3Sum256(p.pubKey.SerializeUncompressed()), p.pubKey) {
				a.nextOnPeer(p)
			} else {
				log.Println("Authenticator.onPacket PROTO_AUTH_HS4 Outgoing PeerId Invalid signature, try close")
				err := p.conn.Close()
				if err != nil {
					log.Println("Authenticator.onPacket PROTO_AUTH_HS4 p.conn.Close()", err)
				}
			}
		}
	default:
		//ignore
	}
}
