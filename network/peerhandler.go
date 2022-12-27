package network

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	AttrWaitSubProtocolInfo = "waitSubProtocolInfo"
)

type PeerHandler interface {
	onPeer(p *Peer)
	onPacket(pkt *Packet, p *Peer)
	onClose(p *Peer)
	setNext(ph PeerHandler)
}

type peerHandler struct {
	next PeerHandler
	self module.PeerID
	//log
	logger log.Logger
}

func newPeerHandler(id module.PeerID, l log.Logger) *peerHandler {
	return &peerHandler{self: id, logger: l}
}

func (ph *peerHandler) onPeer(p *Peer) {
	ph.logger.Traceln("onPeer", p)
	ph.nextOnPeer(p)
}

func (ph *peerHandler) onPacket(pkt *Packet, p *Peer) {
	ph.logger.Traceln("onPacket", p, pkt)
}

func (ph *peerHandler) nextOnPeer(p *Peer) {
	ph.clearWaitInfo(p)
	if ph.next != nil {
		p.setPacketCbFunc(ph.next.onPacket)
		p.setCloseCbFunc(ph.next.onClose)
		ph.next.onPeer(p)
	}
}

func (ph *peerHandler) onClose(p *Peer) {
	ph.logger.Traceln("onClose", p.CloseInfo(), p)
}

func (ph *peerHandler) setNext(next PeerHandler) {
	ph.next = next
}

func (ph *peerHandler) sendMessage(pi module.ProtocolInfo, spi module.ProtocolInfo, m interface{}, p *Peer) {
	pkt := newPacket(pi, spi, ph.encode(m), ph.self)
	err := p.sendDirect(pkt)
	if err != nil {
		ph.logger.Infoln("sendMessage", err)
		p.CloseByError(err)
	} else {
		ph.logger.Traceln("sendMessage", m, p)
	}
}

func (ph *peerHandler) encode(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.MP.NewEncoderBytes(&b)
	if err := enc.Encode(v); err != nil {
		log.Panicf("fail to encode object v=%+v err=%+v", v, err)
	}
	return b
}

func (ph *peerHandler) decodePeerPacket(p *Peer, buf interface{}, pkt *Packet) bool {
	if err := ph.decode(pkt.payload, buf); err != nil {
		p.CloseByError(err)
		return false
	}
	return true
}

func (ph *peerHandler) decode(b []byte, v interface{}) error {
	if remain, err := codec.MP.UnmarshalFromBytes(b, v); err == nil {
		if len(remain) > 0 {
			return errors.Errorf("ExtraBytes(size=%d)", len(remain))
		}
		return nil
	} else {
		return err
	}
}

type waitInfo struct {
	pi         module.ProtocolInfo
	processing bool
}

func (ph *peerHandler) setWaitInfo(pi module.ProtocolInfo, p *Peer) {
	p.PutAttr(AttrWaitSubProtocolInfo, &waitInfo{pi, false})
}

func (ph *peerHandler) clearWaitInfo(p *Peer) {
	p.RemoveAttr(AttrWaitSubProtocolInfo)
}

func (ph *peerHandler) checkWaitInfo(pkt *Packet, p *Peer) bool {
	v, ok := p.GetAndHandleAttr(AttrWaitSubProtocolInfo, func(v interface{}, exists bool) bool {
		if exists {
			if wi, ok := v.(*waitInfo); ok {
				if !wi.processing && wi.pi.Uint16() == pkt.subProtocol.Uint16() {
					wi.processing = true
				} else {
					return false
				}
			}
		}
		return true
	})
	if !ok {
		err := errors.Wrapf(ErrInvalidMessageSequence, "expected:%s received:%s", v.(*waitInfo).pi, pkt.subProtocol)
		p.CloseByError(err)
		return false
	}
	return true
}
