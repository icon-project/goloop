package network

import (
	"fmt"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

var (
	p2pProtoChanJoinReq  = module.ProtocolInfo(0x0500)
	p2pProtoChanJoinResp = module.ProtocolInfo(0x0600)
)

//Negotiation map<channel, map<protocolHandler.name, {protocol, []subProtocol}>>
type ChannelNegotiator struct {
	*peerHandler
	netAddress NetAddress
}

func newChannelNegotiator(netAddress NetAddress, l log.Logger) *ChannelNegotiator {
	cn := &ChannelNegotiator{
		netAddress:  netAddress,
		peerHandler: newPeerHandler(l.WithFields(log.Fields{LoggerFieldKeySubModule: "negotiator"})),
	}
	return cn
}

func (cn *ChannelNegotiator) onPeer(p *Peer) {
	cn.logger.Traceln("onPeer", p)
	if !p.In() {
		cn.setWaitInfo(p2pProtoChanJoinResp, p)
		cn.sendJoinRequest(p)
	} else {
		cn.setWaitInfo(p2pProtoChanJoinReq, p)
	}
}

func (cn *ChannelNegotiator) onError(err error, p *Peer, pkt *Packet) {
	cn.logger.Infoln("onError", err, p, pkt)
	cn.peerHandler.onError(err, p, pkt)
}

func (cn *ChannelNegotiator) onPacket(pkt *Packet, p *Peer) {
	switch pkt.protocol {
	case p2pProtoControl:
		switch pkt.subProtocol {
		case p2pProtoChanJoinReq:
			cn.handleJoinRequest(pkt, p)
		case p2pProtoChanJoinResp:
			cn.handleJoinResponse(pkt, p)
		default:
			p.CloseByError(ErrNotRegisteredProtocol)
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
	m := &JoinRequest{Channel: p.Channel(), Addr: cn.netAddress}
	cn.sendMessage(p2pProtoChanJoinReq, m, p)
	cn.logger.Traceln("sendJoinRequest", m, p)
}

func (cn *ChannelNegotiator) handleJoinRequest(pkt *Packet, p *Peer) {
	if !cn.checkWaitInfo(pkt, p) {
		return
	}

	rm := &JoinRequest{}
	if !cn.decodePeerPacket(p, rm, pkt) {
		return
	}
	cn.logger.Traceln("handleJoinRequest", rm, p)
	if p.Channel() != rm.Channel {
		err := fmt.Errorf("handleJoinRequest error[%v]", "invalid channel")
		cn.logger.Infoln("handleJoinRequest", p.ConnString(), "ChannelNegotiatorError", err)
		p.CloseByError(err)
		return
	}

	p.setNetAddress(rm.Addr)

	m := &JoinResponse{Channel: p.Channel(), Addr: cn.netAddress}
	cn.sendMessage(p2pProtoChanJoinResp, m, p)

	cn.nextOnPeer(p)
}

func (cn *ChannelNegotiator) handleJoinResponse(pkt *Packet, p *Peer) {
	if !cn.checkWaitInfo(pkt, p) {
		return
	}

	rm := &JoinResponse{}
	if !cn.decodePeerPacket(p, rm, pkt) {
		return
	}
	cn.logger.Traceln("handleJoinResponse", rm, p)
	if p.Channel() != rm.Channel {
		err := fmt.Errorf("handleJoinResponse error[%v]", "invalid channel")
		cn.logger.Infoln("handleJoinResponse", p.ConnString(), "ChannelNegotiatorError", err)
		p.CloseByError(err)
		return
	}
	p.setNetAddress(rm.Addr)

	cn.nextOnPeer(p)
}
