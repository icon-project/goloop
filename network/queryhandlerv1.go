package network

import (
	"encoding/hex"
	"fmt"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

var (
	p2pProtoControlV1   = p2pProtoControl + 1
	p2pProtoQueryReqV1  = p2pProtoQueryReq + 1
	p2pProtoQueryRespV1 = p2pProtoQueryResp + 1
	queryItemToConnType = map[QueryItemID]PeerConnectionType{
		QueryItemChildren: p2pConnTypeChildren,
		QueryItemNephews:  p2pConnTypeNephew,
		QueryItemOther:    p2pConnTypeOther,
	}
	connTypeToQueryItems = map[PeerConnectionType][]QueryItemID{
		p2pConnTypeFriend: {QueryItemChildren, QueryItemNephews, QueryItemOther},
		p2pConnTypeParent: {QueryItemChildren, QueryItemNephews},
		p2pConnTypeUncle:  {QueryItemChildren, QueryItemNephews},
	}
)

type QueryItem struct {
	ID      QueryItemID
	Version int64
}

func (qi QueryItem) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "QueryItem{ID:%v,Version:%v}", qi.ID, qi.Version)
	case 's':
		fmt.Fprintf(f, "{ID:%v,Version:%v}", qi.ID, qi.Version)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type QueryResultItem struct {
	ID     QueryItemID
	Error  errors.Code
	Result []byte
}

func (qri QueryResultItem) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "QueryResultItem{ID:%v,Error:%v,Result:%v}", qri.ID, qri.Error, hex.EncodeToString(qri.Result))
	case 's':
		fmt.Fprintf(f, "{ID:%v,Error:%v,Result:%v}", qri.ID, qri.Error, hex.EncodeToString(qri.Result))
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type QueryErrorResult struct {
	Message string
}

func (qer QueryErrorResult) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "QueryErrorResult{Message:%v}", qer.Message)
	case 's':
		fmt.Fprintf(f, "{Message:%v}", qer.Message)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type QueryMessageV1 struct {
	Role  RoleSync
	RTT   RttMessage
	Items []QueryItem
}

func (qm QueryMessageV1) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "QueryMessageV1{Role:%v,RTT:%v,Items:%v}", qm.Role, qm.RTT, qm.Items)
	case 's':
		fmt.Fprintf(f, "{Role:%s,RTT:%s,Items:%s}", qm.Role, qm.RTT, qm.Items)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type QueryResultMessageV1 struct {
	Role        RoleSync
	RTT         RttMessage
	ResultItems []QueryResultItem
}

func (qrm QueryResultMessageV1) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "QueryResultMessageV1{Role:%v,RTT:%v,ResultItems:%v}", qrm.Role, qrm.RTT, qrm.ResultItems)
	case 's':
		fmt.Fprintf(f, "{Role:%s,RTT:%s,ResultItems:%s}", qrm.Role, qrm.RTT, qrm.ResultItems)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type RoleSync struct {
	Role  PeerRoleFlag
	Proof []byte
}

func (rs RoleSync) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "RoleSync{Role:%v,Proof:%v}", rs.Role, hex.EncodeToString(rs.Proof))
	case 's':
		fmt.Fprintf(f, "{Role:%v,Proof:%v}", rs.Role, hex.EncodeToString(rs.Proof))
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

type QueryItemID int

const (
	QueryItemRoots QueryItemID = iota
	QueryItemSeeds
	QueryItemChildren
	QueryItemNephews
	QueryItemOther
)

type queryHandlerV1 struct {
	mc   messageCodec
	self *Peer
	pm   *peerManager
	rr   *roleResolver
	as   *addressSyncer
	rh   *rttHandler
	sm   *SeedManager
	l    log.Logger
}

func newQueryHandlerV1(
	mc messageCodec,
	self *Peer,
	pm *peerManager,
	rr *roleResolver,
	as *addressSyncer,
	rh *rttHandler,
	sm *SeedManager,
	l log.Logger) *queryHandlerV1 {
	return &queryHandlerV1{
		mc:   mc,
		self: self,
		pm:   pm,
		rr:   rr,
		as:   as,
		rh:   rh,
		sm:   sm,
		l:    l.WithFields(log.Fields{LoggerFieldKeySubModule: "query"}),
	}
}

func (h *queryHandlerV1) onPacket(pkt *Packet, p *Peer) bool {
	switch pkt.subProtocol {
	case p2pProtoQueryReqV1:
		h.handleQuery(pkt, p)
	case p2pProtoQueryRespV1:
		h.handleQueryResult(pkt, p)
	default:
		return false
	}
	return true
}

func (h *queryHandlerV1) sendQuery(p *Peer) {
	r := h.self.Role()
	last, avg := p.rtt.Value()
	qm := &QueryMessageV1{
		Role: RoleSync{
			Role: r,
		},
		RTT: RttMessage{
			Last:    last,
			Average: avg,
		},
		Items: []QueryItem{
			{ID: QueryItemSeeds},
		},
	}
	id := h.self.ID()
	sv := h.sm.getSV(id)
	if sv != nil && !p.EqualsAttr(AttrSRHeight, sv.Message.Height()) {
		p.PutAttr(AttrSRHeight, sv.Message.Height())
		svb, err := sv.MarshalBinary()
		if err != nil {
			return
		}
		qm.Role.Proof = svb
	}
	if r.Has(p2pRoleSeed) || r.Has(p2pRoleRoot) {
		qm.Items = append(qm.Items, QueryItem{ID: QueryItemRoots})
	}
	connsItemIDs := connTypeToQueryItems[p.ConnType()]
	for _, qid := range connsItemIDs {
		qm.Items = append(qm.Items, QueryItem{
			ID:      qid,
			Version: p.Conns(queryItemToConnType[qid]).Version(),
		})
	}

	pkt := newPacket(p2pProtoControlV1, p2pProtoQueryReqV1, h.mc.encode(qm), id)
	pkt.destPeer = p.ID()
	if err := p.sendPacket(pkt); err != nil {
		h.l.Infoln("sendQuery", err, p)
	} else {
		h.rh.startRtt(p)
		h.l.Traceln("sendQuery", qm, p)
	}
}

func (h *queryHandlerV1) handleQuery(pkt *Packet, p *Peer) {
	qm := &QueryMessageV1{}
	err := h.mc.decode(pkt.payload, qm)
	if err != nil {
		h.l.Infoln("handleQuery", err, p)
		return
	}
	h.l.Traceln("handleQuery", qm, p)

	if qm.Role.Role.Has(p2pRoleSeed) && len(qm.Role.Proof) > 0 {
		if err = h.sm.handleSV(qm.Role.Proof, p.ID()); err != nil {
			h.l.Infoln("handleQuery", "invalid role proof", err, p)
		}
	}
	rr := h.rr.resolveRole(qm.Role.Role, p.ID(), true)
	if rr != qm.Role.Role {
		errMsg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, qm.Role)
		h.l.Infoln("handleQuery", errMsg, p)
	}
	p.setRecvRole(qm.Role.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		h.as.applyPeerRole(p)
	}
	isPeerRootOrSeed := rr.Has(p2pRoleSeed) || rr.Has(p2pRoleRoot)
	r := h.self.Role()
	if r.Has(p2pRoleRoot) && !isPeerRootOrSeed {
		h.l.Infoln("handleQuery", "not allowed connection", p)
		p.Close("handleQuery not allowed connection")
		return
	}
	h.rh.checkAccuracy(p, qm.RTT.Last)
	rttLast, rttAvg := p.rtt.Value()

	qrm := &QueryResultMessageV1{
		Role: RoleSync{
			Role: r,
		},
		RTT: RttMessage{
			Last:    rttLast,
			Average: rttAvg,
		},
	}
	id := h.self.ID()
	sv := h.sm.getSV(id)
	if sv != nil && !p.EqualsAttr(AttrSRHeight, sv.Message.Height()) {
		p.PutAttr(AttrSRHeight, sv.Message.Height())
		svb, err := sv.MarshalBinary()
		if err != nil {
			return
		}
		qrm.Role.Proof = svb
	}
	for _, qi := range qm.Items {
		qri := QueryResultItem{
			ID:    qi.ID,
			Error: errors.Success,
		}
		var result interface{}
		if result, err = h.queryResult(qi, isPeerRootOrSeed, r); err != nil {
			qri.Error = errors.CodeOf(err)
			result = QueryErrorResult{
				Message: err.Error(),
			}
		}
		qri.Result = h.mc.encode(result)
		qrm.ResultItems = append(qrm.ResultItems, qri)
	}

	rpkt := newPacket(p2pProtoControlV1, p2pProtoQueryRespV1, h.mc.encode(qrm), id)
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		h.l.Infoln("handleQuery", "sendQueryResult", err, p)
	} else {
		h.l.Traceln("handleQuery", "sendQueryResult", qrm, p)
	}
}

func (h *queryHandlerV1) queryResult(qi QueryItem, isPeerRootOrSeed bool, r PeerRoleFlag) (interface{}, error) {
	switch qi.ID {
	case QueryItemRoots:
		if !isPeerRootOrSeed || r == p2pRoleNone {
			return nil, errors.Errorf("not allowed QueryItemID:%v", qi.ID)
		}
		roots := h.as.getNetAddresses(p2pRoleRoot)
		if len(roots) > DefaultQueryElementLength {
			roots = roots[:DefaultQueryElementLength]
		}
		return roots, nil
	case QueryItemSeeds:
		var seeds []NetAddress
		if isPeerRootOrSeed {
			seeds = h.as.getNetAddresses(p2pRoleSeed)
		} else {
			seeds = h.as.getUniqueNetAddresses(p2pRoleSeed)
		}
		if len(seeds) > DefaultQueryElementLength {
			seeds = seeds[:DefaultQueryElementLength]
		}
		return seeds, nil
	case QueryItemChildren, QueryItemNephews, QueryItemOther:
		return h.pm.peerAddressSetDiff(queryItemToConnType[qi.ID], qi.Version)
	default:
		return nil, errors.Errorf("not supported QueryItemID:%v", qi.ID)
	}
}

func (h *queryHandlerV1) handleQueryResult(pkt *Packet, p *Peer) {
	qrm := &QueryResultMessageV1{}
	err := h.mc.decode(pkt.payload, qrm)
	if err != nil {
		h.l.Infoln("handleQueryResult", err, p)
		return
	}
	h.l.Traceln("handleQueryResult", qrm, p)
	h.rh.stopRtt(p)
	h.rh.checkAccuracy(p, qrm.RTT.Last)
	if qrm.Role.Role.Has(p2pRoleSeed) && len(qrm.Role.Proof) > 0 {
		if err = h.sm.handleSV(qrm.Role.Proof, p.ID()); err != nil {
			h.l.Infoln("handleQueryResult", "invalid role proof", err, p)
		}
	}
	rr := h.rr.resolveRole(qrm.Role.Role, p.ID(), true)
	if rr != qrm.Role.Role {
		msg := fmt.Sprintf("not equal resolved role %d, expected %d", rr, qrm.Role.Role)
		h.l.Infoln("handleQueryResult", msg, p)
	}
	p.setRecvRole(qrm.Role.Role)
	if !p.EqualsRole(rr) {
		p.setRole(rr)
		h.as.applyPeerRole(p)
	}
	isPeerRootOrSeed := rr.Has(p2pRoleSeed) || rr.Has(p2pRoleRoot)
	if !isPeerRootOrSeed && !h.rr.isTrustSeed(p) {
		h.l.Infoln("handleQueryResult", "invalid query, not allowed connection", p)
		p.CloseByError(fmt.Errorf("handleQueryResult invalid query, resolved role %d", rr))
		return
	}

	r := h.self.Role()
	for _, qri := range qrm.ResultItems {
		if qri.Error != errors.Success {
			qer := QueryErrorResult{}
			if err = h.mc.decode(qri.Result, &qer); err != nil {
				h.l.Infoln("handleQueryResult", "fail to decode QueryErrorResult", qri.ID, err, p)
			} else {
				h.l.Infoln("handleQueryResult", "QueryErrorResult", qri.ID, qer, p)
			}
			continue
		}
		switch qri.ID {
		case QueryItemRoots:
			if r.Has(p2pRoleSeed) || r.Has(p2pRoleRoot) {
				var roots []NetAddress
				if err = h.mc.decode(qri.Result, &roots); err != nil {
					h.l.Infoln("handleQueryResult", "fail to decode []NetAddress", qri.ID, err, p)
				}
				if len(roots) > DefaultQueryElementLength {
					h.l.Infoln("handleQueryResult", "invalid Roots Length:", len(roots), p)
					roots = roots[:DefaultQueryElementLength]
				}
				h.as.mergeNetAddresses(p2pRoleRoot, roots)
			}
		case QueryItemSeeds:
			var seeds []NetAddress
			if err = h.mc.decode(qri.Result, &seeds); err != nil {
				h.l.Infoln("handleQueryResult", "fail to decode []NetAddress", qri.ID, err, p)
			}
			if len(seeds) > DefaultQueryElementLength {
				h.l.Infoln("handleQueryResult", "invalid Seeds Length:", len(seeds), p)
				seeds = seeds[:DefaultQueryElementLength]
			}
			h.as.mergeNetAddresses(p2pRoleSeed, seeds)
		case QueryItemChildren, QueryItemNephews, QueryItemOther:
			dr := DiffResult[PeerAddress]{}
			if err = h.mc.decode(qri.Result, &dr); err != nil {
				h.l.Infoln("handleQueryResult", "fail to decode []NetAddress", qri.ID, err, p)
			}
			p.UpdateConns(queryItemToConnType[qri.ID], dr)
		default:
			h.l.Infoln("handleQueryResult", "not supported QueryItemID", qri.ID, p)
		}
	}
}
