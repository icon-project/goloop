package network

import (
	"fmt"
	"time"

	"github.com/icon-project/goloop/common/log"
)

type queryHandler struct {
	mc   messageCodec
	self *Peer
	pm   *peerManager
	rr   *roleResolver
	as   *addressSyncer
	rh   *rttHandler
	l    log.Logger
}

func newQueryHandler(
	mc messageCodec,
	self *Peer,
	pm *peerManager,
	rr *roleResolver,
	sa *addressSyncer,
	rh *rttHandler,
	l log.Logger) *queryHandler {
	return &queryHandler{
		mc:   mc,
		self: self,
		pm:   pm,
		rr:   rr,
		as:   sa,
		rh:   rh,
		l:    l.WithFields(log.Fields{LoggerFieldKeySubModule: "query"}),
	}
}

func (h *queryHandler) onPacket(pkt *Packet, p *Peer) bool {
	switch pkt.subProtocol {
	case p2pProtoQueryReq:
		h.handleQuery(pkt, p)
	case p2pProtoQueryResp:
		h.handleQueryResult(pkt, p)
	case p2pProtoRttReq:
		h.handleRttRequest(pkt, p)
	case p2pProtoRttResp:
		h.handleRttResponse(pkt, p)
	default:
		return false
	}
	return true
}

type QueryMessage struct {
	Role PeerRoleFlag
}

type QueryResultMessage struct {
	Role     PeerRoleFlag
	Seeds    []NetAddress
	Roots    []NetAddress
	Children []NetAddress
	Nephews  []NetAddress
	Message  string
}

type RttMessage struct {
	Last    time.Duration
	Average time.Duration
}

func (h *queryHandler) sendQuery(p *Peer) {
	m := &QueryMessage{Role: h.self.Role()}
	pkt := newPacket(p2pProtoControl, p2pProtoQueryReq, h.mc.encode(m), h.self.ID())
	pkt.destPeer = p.ID()
	err := p.sendPacket(pkt)
	if err != nil {
		h.l.Infoln("sendQuery", err, p)
	} else {
		h.rh.startRtt(p)
		h.l.Traceln("sendQuery", m, p)
	}
}

func (h *queryHandler) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessage{}
	err := h.mc.decode(pkt.payload, qm)
	if err != nil {
		h.l.Infoln("handleQuery", err, p)
		return
	}
	h.l.Traceln("handleQuery", qm, p)

	r := h.self.Role()
	m := &QueryResultMessage{
		Role:     r,
		Children: h.pm.getNetAddresses(p2pConnTypeChildren),
		Nephews:  h.pm.getNetAddresses(p2pConnTypeNephew),
	}
	rr := h.rr.resolveRole(qm.Role, p.ID(), true)
	if rr != qm.Role {
		m.Message = fmt.Sprintf("not equal resolved role %d, expected %d", rr, qm.Role)
		h.l.Infoln("handleQuery", m.Message, p)
	}
	p.setRecvRole(qm.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		h.as.applyPeerRole(p)
	}
	if rr.Has(p2pRoleSeed) || rr.Has(p2pRoleRoot) {
		m.Roots = h.as.getNetAddresses(p2pRoleRoot)
		m.Seeds = h.as.getNetAddresses(p2pRoleSeed)
	} else {
		if r.Has(p2pRoleRoot) {
			h.l.Infoln("handleQuery", "not allowed connection", p)
			p.Close("handleQuery not allowed connection")
			return
		}
		m.Seeds = h.as.getUniqueNetAddresses(p2pRoleSeed)
	}

	//prevent propagation of addresses via normal nodes
	if r == p2pRoleNone {
		m.Roots = m.Roots[:0]
		m.Seeds = m.Seeds[:0]
	}

	if len(m.Roots) > DefaultQueryElementLength {
		m.Roots = m.Roots[:DefaultQueryElementLength]
	}
	if len(m.Seeds) > DefaultQueryElementLength {
		m.Seeds = m.Seeds[:DefaultQueryElementLength]
	}

	rpkt := newPacket(p2pProtoControl, p2pProtoQueryResp, h.mc.encode(m), h.self.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		h.l.Infoln("handleQuery", "sendQueryResult", err, p)
	} else {
		h.rh.startRtt(p)
		h.l.Traceln("handleQuery", "sendQueryResult", m, p)
	}
}

func (h *queryHandler) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessage{}
	err := h.mc.decode(pkt.payload, qrm)
	if err != nil {
		h.l.Infoln("handleQueryResult", err, p)
		return
	}
	h.rh.stopRtt(p)
	if len(qrm.Roots) > DefaultQueryElementLength {
		h.l.Infoln("handleQueryResult", "invalid Roots Length:", len(qrm.Roots), p)
		qrm.Roots = qrm.Roots[:DefaultQueryElementLength]
	}
	if len(qrm.Seeds) > DefaultQueryElementLength {
		h.l.Infoln("handleQueryResult", "invalid Seeds Length:", len(qrm.Seeds), p)
		qrm.Seeds = qrm.Seeds[:DefaultQueryElementLength]
	}
	if len(qrm.Children) > DefaultQueryElementLength {
		h.l.Infoln("handleQueryResult", "invalid Children Length:", len(qrm.Children), p)
		qrm.Children = qrm.Children[:DefaultQueryElementLength]
	}
	if len(qrm.Nephews) > DefaultQueryElementLength {
		h.l.Infoln("handleQueryResult", "invalid Nephews Length:", len(qrm.Nephews), p)
		qrm.Nephews = qrm.Nephews[:DefaultQueryElementLength]
	}
	h.l.Traceln("handleQueryResult", qrm, p)

	p.children.ClearAndAdd(qrm.Children...)
	p.nephews.ClearAndAdd(qrm.Nephews...)

	rr := h.rr.resolveRole(qrm.Role, p.ID(), true)
	if rr != qrm.Role {
		msg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, qrm.Role)
		h.l.Infoln("handleQueryResult", msg, p)
	}
	p.setRecvRole(qrm.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		h.as.applyPeerRole(p)
	}
	if !rr.Has(p2pRoleSeed) && !rr.Has(p2pRoleRoot) {
		if !h.rr.isTrustSeed(p) {
			h.l.Infoln("handleQueryResult", "invalid query, not allowed connection", p)
			p.CloseByError(fmt.Errorf("handleQueryResult invalid query, resolved role %d", rr))
			return
		}
	}

	r := h.self.Role()
	if r.Has(p2pRoleSeed) || r.Has(p2pRoleRoot) {
		h.as.mergeNetAddresses(p2pRoleRoot, qrm.Roots)
	}
	h.as.mergeNetAddresses(p2pRoleSeed, qrm.Seeds)

	last, avg := p.rtt.Value()
	m := &RttMessage{Last: last, Average: avg}
	rpkt := newPacket(p2pProtoControl, p2pProtoRttReq, h.mc.encode(m), h.self.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		h.l.Infoln("handleQueryResult", "sendRttRequest", err, p)
	} else {
		h.l.Traceln("handleQueryResult", "sendRttRequest", m, p)
	}
}

func (h *queryHandler) handleRttRequest(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := h.mc.decode(pkt.payload, rm)
	if err != nil {
		h.l.Infoln("handleRttRequest", err, p)
		return
	}
	h.l.Traceln("handleRttRequest", rm, p)
	h.rh.stopRtt(p)
	h.rh.checkAccuracy(p, rm.Last)
	last, avg := p.rtt.Value()
	m := &RttMessage{Last: last, Average: avg}
	rpkt := newPacket(p2pProtoControl, p2pProtoRttResp, h.mc.encode(m), h.self.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		h.l.Infoln("handleRttRequest", "sendRttResponse", err, p)
	} else {
		h.l.Traceln("handleRttRequest", "sendRttResponse", m, p)
	}
}

func (h *queryHandler) handleRttResponse(pkt *Packet, p *Peer) {
	rm := &RttMessage{}
	err := h.mc.decode(pkt.payload, rm)
	if err != nil {
		h.l.Infoln("handleRttResponse", err, p)
		return
	}
	h.l.Traceln("handleRttResponse", rm, p)
	h.rh.checkAccuracy(p, rm.Last)
}
