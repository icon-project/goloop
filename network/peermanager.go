package network

import (
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type peerManager struct {
	mc         messageCodec
	self       *Peer
	m          map[PeerConnectionType]*PeerSet
	transiting *PeerSet
	reject     *PeerSet
	connMtx    sync.RWMutex

	onEventCb func(evt string, p *Peer)
	onCloseCb func(p *Peer, removed bool)

	//connection limit
	cLimit    map[PeerConnectionType]int
	cLimitMtx sync.RWMutex

	l log.Logger
}

func newPeerManager(
	mc messageCodec,
	self *Peer,
	onEventCb func(evt string, p *Peer),
	onCloseCb func(p *Peer, removed bool),
	l log.Logger) *peerManager {
	pm := &peerManager{
		mc:         mc,
		self:       self,
		m:          make(map[PeerConnectionType]*PeerSet),
		transiting: NewPeerSet(),
		reject:     NewPeerSet(),
		//
		onEventCb: onEventCb,
		onCloseCb: onCloseCb,
		//
		cLimit: make(map[PeerConnectionType]int),
		//
		l: l,
	}
	for connType := p2pConnTypeNone; connType < p2pConnTypeReserved; connType++ {
		pm.m[connType] = NewPeerSet()
	}
	return pm
}

func (pm *peerManager) setConnectionLimit(connType PeerConnectionType, v int) {
	pm.cLimitMtx.Lock()
	defer pm.cLimitMtx.Unlock()

	if connType < p2pConnTypeNone || connType > p2pConnTypeOther {
		return
	}
	pm.cLimit[connType] = v
}

func (pm *peerManager) getConnectionLimit(connType PeerConnectionType) int {
	pm.cLimitMtx.RLock()
	defer pm.cLimitMtx.RUnlock()
	v, ok := pm.cLimit[connType]
	if !ok || v < 0 {
		switch connType {
		case p2pConnTypeParent:
			return DefaultParentsLimit
		case p2pConnTypeChildren:
			return DefaultChildrenLimit
		case p2pConnTypeUncle:
			return DefaultUnclesLimit
		case p2pConnTypeNephew:
			return DefaultNephewsLimit
		case p2pConnTypeOther:
			return DefaultOthersLimit
		default:
			v = -1
		}
	}
	return v
}

func (pm *peerManager) getConnectionAvailable(connType PeerConnectionType) int {
	return pm.getConnectionLimit(connType) - pm.lenPeers(connType)
}

func (pm *peerManager) _findPeer(f PeerPredicate, connTypes ...PeerConnectionType) *Peer {
	if len(connTypes) == 0 {
		for _, v := range pm.m {
			if p := v.FindOne(f); p != nil {
				return p
			}
		}
	} else {
		for _, k := range connTypes {
			if v, ok := pm.m[k]; ok {
				if p := v.FindOne(f); p != nil {
					return p
				}
			}
		}
	}
	return nil
}

func (pm *peerManager) findPeer(f PeerPredicate, connTypes ...PeerConnectionType) *Peer {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()

	return pm._findPeer(f, connTypes...)
}

func (pm *peerManager) _findPeers(f func(s *PeerSet) []*Peer, connTypes []PeerConnectionType) []*Peer {
	arr := make([]*Peer, 0)
	if len(connTypes) == 0 {
		for _, v := range pm.m {
			arr = append(arr, f(v)...)
		}
	} else {
		for _, k := range connTypes {
			if v, ok := pm.m[k]; ok {
				arr = append(arr, f(v)...)
			}
		}
	}
	return arr
}

func (pm *peerManager) findPeers(f PeerPredicate, connTypes ...PeerConnectionType) []*Peer {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()

	if f == nil {
		return pm._findPeers(func(s *PeerSet) []*Peer {
			return s.Array()
		}, connTypes)
	} else {
		return pm._findPeers(func(s *PeerSet) []*Peer {
			return s.Find(f)
		}, connTypes)
	}
}

func (pm *peerManager) onPeer(p *Peer) bool {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()

	if p.IsClosed() {
		return false
	}

	id := p.ID()
	if dp := pm._findPeer(func(p *Peer) bool {
		return p.ID().Equal(id)
	}); dp != nil {
		pm.onEventCb(p2pEventDuplicate, p)

		onCloseInLock := func(p *Peer) {
			pm.l.Debugln("onClose", p.CloseInfo(), p)
			pm._onClose(p)
		}

		//'b' is higher (ex : 'b' > 'a'), disconnect lower.outgoing
		higher := strings.Compare(pm.self.ID().String(), p.ID().String()) > 0
		diff := p.timestamp.Sub(dp.timestamp)

		if diff < DefaultDuplicatedPeerTime && dp.In() != p.In() && higher == p.In() {
			//close new which is lower outgoing
			p.setCloseCbFunc(onCloseInLock)
			p.CloseByError(ErrDuplicatedPeer)
			pm.l.Infoln("Already exists connected Peer, close new", p, diff)
			return false
		}
		//close old
		dp.setCloseCbFunc(onCloseInLock)
		dp.CloseByError(ErrDuplicatedPeer)
		pm.l.Infoln("Already exists connected Peer, close old", dp, diff)
	}
	return pm.m[p2pConnTypeNone].Add(p)
}

func (pm *peerManager) _removePeer(p *Peer) bool {
	pm.reject.Remove(p)
	pm.transiting.Remove(p)

	if v, ok := pm.m[p.ConnType()]; ok {
		return v.Remove(p)
	}
	return false
}

func (pm *peerManager) _onClose(p *Peer) {
	removed := pm._removePeer(p)
	if p.ConnType() != p2pConnTypeNone {
		pm.onEventCb(p2pEventLeave, p)
	}
	pm.onCloseCb(p, removed)
}

func (pm *peerManager) onClose(p *Peer) {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()
	pm._onClose(p)
}

func (pm *peerManager) _lenPeers(f func(s *PeerSet) int, connTypes []PeerConnectionType) int {
	n := 0
	if len(connTypes) == 0 {
		for _, v := range pm.m {
			n += f(v)
		}
	} else {
		for _, k := range connTypes {
			if v, ok := pm.m[k]; ok {
				n += f(v)
			}
		}
	}
	return n
}

func (pm *peerManager) lenPeers(connTypes ...PeerConnectionType) int {
	pm.connMtx.RLock()
	defer pm.connMtx.RUnlock()

	return pm._lenPeers(func(s *PeerSet) int {
		return s.Len()
	}, connTypes)
}

func (pm *peerManager) lenPeersByProtocol(pi module.ProtocolInfo, connTypes ...PeerConnectionType) int {
	pm.connMtx.RLock()
	defer pm.connMtx.RUnlock()

	return pm._lenPeers(func(s *PeerSet) int {
		return s.LenByProtocol(pi)
	}, connTypes)
}

func (pm *peerManager) getNetAddresses(connType PeerConnectionType) []NetAddress {
	pm.connMtx.RLock()
	defer pm.connMtx.RUnlock()
	return pm.m[connType].NetAddresses()
}

func (pm *peerManager) hasNetAddress(na NetAddress) bool {
	pm.connMtx.RLock()
	defer pm.connMtx.RUnlock()
	if pm.self.NetAddress() == na {
		return true
	}
	for _, v := range pm.m {
		if v.HasNetAddress(na) {
			return true
		}
	}
	return false
}

func (pm *peerManager) onPacket(pkt *Packet, p *Peer) bool {
	switch pkt.subProtocol {
	case p2pProtoConnReq:
		pm.handleP2PConnectionRequest(pkt, p)
	case p2pProtoConnResp:
		pm.handleP2PConnectionResponse(pkt, p)
	default:
		return false
	}
	return true
}

type P2PConnectionRequest struct {
	ConnType PeerConnectionType
}

type P2PConnectionResponse struct {
	ReqConnType PeerConnectionType
	ConnType    PeerConnectionType
}

func (pm *peerManager) sendP2PConnectionRequest(connType PeerConnectionType, p *Peer) {
	m := &P2PConnectionRequest{ConnType: connType}
	pkt := newPacket(p2pProtoControl, p2pProtoConnReq, pm.mc.encode(m), pm.self.ID())
	pkt.destPeer = p.ID()
	err := p.sendPacket(pkt)
	if err != nil {
		pm.l.Infoln("sendP2PConnectionRequest", err, p)
	} else {
		pm.l.Debugln("sendP2PConnectionRequest", m, p)
	}
}

func (pm *peerManager) handleP2PConnectionRequest(pkt *Packet, p *Peer) {
	req := &P2PConnectionRequest{}
	err := pm.mc.decode(pkt.payload, req)
	if err != nil {
		pm.l.Infoln("handleP2PConnectionRequest", err, p)
		return
	}
	pm.l.Debugln("handleP2PConnectionRequest", req, p)
	p.setRecvConnType(req.ConnType)
	rc, notAllowed, invalidReq := pm.resolveConnectionRequest(p.Role(), req.ConnType)
	if notAllowed {
		pm.l.Infoln("handleP2PConnectionRequest", "not allowed reqConnType", req.ConnType, "from", p.ID(), p.ConnType())
	} else if invalidReq {
		pm.l.Infoln("handleP2PConnectionRequest", "invalid reqConnType", req.ConnType, "from", p.ID(), p.ConnType())
	} else {
		if rc != p2pConnTypeNone && !p.EqualsAttr(AttrSupportDefaultProtocols, true) {
			rc = p2pConnTypeOther
			pm.l.Debugln("handleP2PConnectionResponse", "not support defaultProtocols", p.ID())
		}
		switch rc {
		case p2pConnTypeParent:
			if !pm.updatePeerConnectionType(p, p2pConnTypeParent) &&
				!pm.updatePeerConnectionType(p, p2pConnTypeUncle) {
				pm.l.Infoln("handleP2PConnectionRequest",
					"ignore p2pConnTypeFriend request, already has enough upstream connections", strPeerConnectionType[rc],
					"from", p.ID(), p.ConnType())
			}
		case p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeNone:
			pm.updatePeerConnectionType(p, rc)
		case p2pConnTypeChildren, p2pConnTypeNephew:
			if !pm.updatePeerConnectionType(p, rc) {
				pm.l.Infoln("handleP2PConnectionRequest", "reject by limit", strPeerConnectionType[rc],
					"from", p.ID(), p.ConnType())
			}
		}
	}
	m := &P2PConnectionResponse{ReqConnType: req.ConnType, ConnType: p.ConnType()}
	if m.ConnType == p2pConnTypeOther {
		//for legacy which is not supported p2pConnTypeOther response
		if p.EqualsAttr(AttrP2PLegacy, true) {
			switch req.ConnType {
			case p2pConnTypeParent:
				m.ConnType = p2pConnTypeChildren
			case p2pConnTypeUncle:
				m.ConnType = p2pConnTypeNephew
			}
		}
	}
	rpkt := newPacket(p2pProtoControl, p2pProtoConnResp, pm.mc.encode(m), pm.self.ID())
	rpkt.destPeer = p.ID()
	err = p.sendPacket(rpkt)
	if err != nil {
		pm.l.Infoln("handleP2PConnectionRequest", "sendP2PConnectionResponse", err, p)
	} else {
		pm.l.Debugln("handleP2PConnectionRequest", "sendP2PConnectionResponse", m, p)
	}
}

func (pm *peerManager) handleP2PConnectionResponse(pkt *Packet, p *Peer) {
	resp := &P2PConnectionResponse{}
	err := pm.mc.decode(pkt.payload, resp)
	if err != nil {
		pm.l.Infoln("handleP2PConnectionResponse", err, p)
		return
	}
	pm.l.Debugln("handleP2PConnectionResponse", resp, p)
	p.setRecvConnType(resp.ConnType)
	if resp.ReqConnType == p2pConnTypeNone {
		return
	}
	if !pm.transitPeer(p, true) {
		pm.l.Infoln("handleP2PConnectionResponse", "invalid peer", resp, p)
		return
	} else {
		if !p.EqualsAttr(AttrP2PConnectionRequest, resp.ReqConnType) {
			pm.l.Infoln("handleP2PConnectionResponse", "invalid ReqConnType", resp, p)
			return
		}
		p.RemoveAttr(AttrP2PConnectionRequest)
	}

	rc, rejectResp, invalidResp := pm.resolveConnectionResponse(p.RecvRole(), resp.ReqConnType, resp.ConnType)
	if rejectResp {
		pm.rejectPeer(p)
		pm.l.Infoln("handleP2PConnectionResponse", "rejected",
			strPeerConnectionType[resp.ReqConnType], "resp", strPeerConnectionType[resp.ConnType],
			"from", p.ID(), p.ConnType())
	} else if invalidResp {
		pm.l.Infoln("handleP2PConnectionResponse", "invalid ReqConnType", resp,
			"from", p.ID(), p.ConnType())
	} else {
		pm.l.Debugln("handleP2PConnectionResponse", "resolvedConnType", strPeerConnectionType[resp.ConnType],
			"from", p.ID(), p.ConnType())
		if rc != p2pConnTypeNone && !p.EqualsAttr(AttrSupportDefaultProtocols, true) {
			rc = p2pConnTypeOther
			pm.l.Debugln("handleP2PConnectionResponse", "not support defaultProtocols", p.ID())
		}
		switch rc {
		case p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeNone:
			pm.updatePeerConnectionType(p, rc)
		case p2pConnTypeParent:
			if !pm.updatePeerConnectionType(p, p2pConnTypeParent) {
				pm.l.Debugln("handleP2PConnectionResponse", "already p2pConnTypeParent", resp,
					"from", p.ID(), p.ConnType())
				if pm.lenPeers(p2pConnTypeUncle) < pm.getConnectionLimit(p2pConnTypeUncle) {
					pm.tryTransitPeerConnection(p, p2pConnTypeUncle)
				} else {
					p.Close("already has enough upstream connections")
				}
			}
		case p2pConnTypeUncle:
			if !pm.updatePeerConnectionType(p, p2pConnTypeUncle) {
				pm.l.Debugln("handleP2PConnectionResponse", "already p2pConnTypeUncle", resp,
					"from", p.ID(), p.ConnType())
				if pm.lenPeers(p2pConnTypeParent) < pm.getConnectionLimit(p2pConnTypeParent) {
					pm.tryTransitPeerConnection(p, p2pConnTypeParent)
				} else {
					p.Close("already has enough upstream connections")
				}
			}
		}
	}
}

func (pm *peerManager) updatePeerConnectionType(p *Peer, connType PeerConnectionType) (updated bool) {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()

	if p.IsClosed() {
		return false
	}
	from := p.ConnType()
	if connType < p2pConnTypeNone || connType > p2pConnTypeOther || connType == from {
		return
	}

	t := pm.m[connType]
	l := pm.getConnectionLimit(connType)
	if l < 0 || l > t.Len() {
		pm._removePeer(p)
		if updated = t.Add(p); !updated {
			//unexpected failure
			return
		}
		if l == t.Len() {
			pm.l.Debugln("updatePeerConnectionType", "complete", strPeerConnectionType[connType])
			if connType == p2pConnTypeParent || connType == p2pConnTypeUncle {
				pm.reject.Clear()
			}
		}
		p.setConnType(connType)
		if from == p2pConnTypeNone {
			pm.onEventCb(p2pEventJoin, p)
		}
		if connType == p2pConnTypeNone {
			pm.onEventCb(p2pEventLeave, p)
		}
	}
	return
}

func (pm *peerManager) transitPeer(p *Peer, remove bool) bool {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()
	if p.IsClosed() {
		return false
	}
	if remove {
		return pm.transiting.Remove(p)
	} else {
		if !pm.reject.Contains(p) && !pm.transiting.Contains(p) {
			return pm.transiting.Add(p)
		}
	}
	return false
}

func (pm *peerManager) rejectPeer(p *Peer) bool {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()
	if p.IsClosed() {
		return false
	}
	return pm.reject.Add(p)
}

func (pm *peerManager) clearReject() {
	pm.connMtx.Lock()
	defer pm.connMtx.Unlock()
	pm.reject.Clear()
}

func (pm *peerManager) tryTransitPeerConnection(p *Peer, connType PeerConnectionType) bool {
	switch connType {
	case p2pConnTypeNone:
		pm.updatePeerConnectionType(p, p2pConnTypeNone)
		pm.sendP2PConnectionRequest(p2pConnTypeNone, p)
		return true
	default:
		if p.EqualsAttr(AttrSupportDefaultProtocols, false) {
			return false
		}
		if pm.transitPeer(p, false) {
			p.PutAttr(AttrP2PConnectionRequest, connType)
			pm.sendP2PConnectionRequest(connType, p)
			return true
		}
	}
	return false
}

func (pm *peerManager) resolveConnectionRequest(pr PeerRoleFlag, connType PeerConnectionType) (rc PeerConnectionType, notAllowed, invalidReq bool) {
	r := pm.self.Role()
	rc = p2pConnTypeNone
	if r.Has(p2pRoleRoot) {
		switch connType {
		case p2pConnTypeFriend:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeFriend
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeOther
			} else {
				notAllowed = true
			}
		case p2pConnTypeParent:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeOther
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeChildren
			} else {
				notAllowed = true
			}
		case p2pConnTypeUncle:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeOther
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeNephew
			} else {
				notAllowed = true
			}
		case p2pConnTypeNone:
			rc = connType
		default:
			invalidReq = true
		}
	} else if r.Has(p2pRoleSeed) {
		switch connType {
		case p2pConnTypeFriend:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeParent
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeOther
			} else {
				invalidReq = true
			}
		case p2pConnTypeParent:
			if pr.Has(p2pRoleRoot) || pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeChildren
			}
		case p2pConnTypeUncle:
			if pr.Has(p2pRoleRoot) || pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeNephew
			}
		case p2pConnTypeNone:
			rc = connType
		default:
			invalidReq = true
		}
	} else {
		switch connType {
		case p2pConnTypeParent:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeNone
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeChildren
			}
		case p2pConnTypeUncle:
			if pr.Has(p2pRoleRoot) {
				rc = p2pConnTypeNone
			} else if pr.Has(p2pRoleSeed) {
				rc = p2pConnTypeNone
			} else {
				rc = p2pConnTypeNephew
			}
		case p2pConnTypeNone:
			rc = connType
		default:
			invalidReq = true
		}
	}
	return
}

func (pm *peerManager) resolveConnectionResponse(prr PeerRoleFlag, reqConnType, respConnType PeerConnectionType) (rc PeerConnectionType, rejectResp, invalidResp bool) {
	r := pm.self.Role()
	rc = p2pConnTypeNone
	if r.Has(p2pRoleRoot) {
		switch reqConnType {
		case p2pConnTypeFriend:
			switch respConnType {
			case p2pConnTypeFriend:
				rc = p2pConnTypeFriend
			case p2pConnTypeOther, p2pConnTypeNone:
				//in case of p2pConnTypeNone
				// for legacy which pm.others managed by discovery only,
				// legacy ignore request of p2pConnTypeFriend and response p2pConnTypeNone
				if prr.Has(p2pRoleRoot) {
					rc = p2pConnTypeFriend
				} else {
					rc = p2pConnTypeOther
				}
			case p2pConnTypeParent, p2pConnTypeUncle:
				rc = p2pConnTypeOther
			default:
				invalidResp = true
			}
		default:
			invalidResp = true
		}
	} else if r.Has(p2pRoleSeed) {
		switch reqConnType {
		case p2pConnTypeParent:
			switch respConnType {
			case p2pConnTypeChildren, p2pConnTypeOther:
				rc = p2pConnTypeParent
			default:
				rejectResp = true
			}
		case p2pConnTypeUncle:
			switch respConnType {
			case p2pConnTypeNephew, p2pConnTypeOther:
				rc = p2pConnTypeUncle
			default:
				rejectResp = true
			}
		default:
			invalidResp = true
		}
	} else {
		switch reqConnType {
		case p2pConnTypeParent:
			switch respConnType {
			case p2pConnTypeChildren:
				rc = p2pConnTypeParent
			case p2pConnTypeOther:
				rc = p2pConnTypeOther
			default:
				rejectResp = true
			}
		case p2pConnTypeUncle:
			switch respConnType {
			case p2pConnTypeNephew:
				rc = p2pConnTypeUncle
			case p2pConnTypeOther:
				rc = p2pConnTypeOther
			default:
				rejectResp = true
			}
		default:
			invalidResp = true
		}
	}
	return
}
