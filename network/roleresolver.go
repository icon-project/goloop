package network

import (
	"fmt"
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type roleResolver struct {
	self *Peer
	// trustSeeds map[DialNetAddress]NetAddress if value of map is duplicated, then old will be removed.
	trustSeeds   *NetAddressSet
	allowedRoots *PeerIDSet
	allowedSeeds *PeerIDSet
	allowedPeers *PeerIDSet

	onEventCb eventCbFunc
	pm        *peerManager
	as        *addressSyncer
	mtx       sync.Mutex

	l log.Logger
}

func newRoleResolver(
	self *Peer,
	onEventCb eventCbFunc,
	pm *peerManager,
	as *addressSyncer,
	l log.Logger) *roleResolver {
	rr := &roleResolver{
		self:         self,
		trustSeeds:   NewNetAddressSet(),
		allowedRoots: NewPeerIDSet(),
		allowedSeeds: NewPeerIDSet(),
		allowedPeers: NewPeerIDSet(),
		onEventCb:    onEventCb,
		pm:           pm,
		as:           as,
		l:            l,
	}
	rr.allowedRoots.onUpdate = rr.resolveAndApplyAll
	rr.allowedSeeds.onUpdate = rr.resolveAndApplyAll
	rr.allowedPeers.onUpdate = func(s *PeerIDSet) {
		if s.IsEmpty() {
			return
		}
		ps := rr.pm.findPeers(func(p *Peer) bool {
			return rr.isNotAllowed(p2pRoleNone, p.ID())
		})
		for _, p := range ps {
			rr.onEventCb(p2pEventNotAllowed, p)
			p.CloseByError(fmt.Errorf("onUpdate not allowed connection"))
		}
	}
	return rr
}

func (rr *roleResolver) resolveAndApplyAll(_ *PeerIDSet) {
	rr.mtx.Lock()
	defer rr.mtx.Unlock()
	rr.pm.findPeers(func(p *Peer) bool {
		rr._resolveAndApply(p)
		return false
	})
	rr._resolveAndApply(rr.self)
}

func (rr *roleResolver) _resolveAndApply(p *Peer) PeerRoleFlag {
	r := rr.resolveRole(p.RecvRole(), p.ID())
	if !p.EqualsRole(r) {
		p.setRole(r)
		rr.as.applyPeerRole(p)
	}
	return r
}

func (rr *roleResolver) resolveAndApply(p *Peer, tr PeerRoleFlag) PeerRoleFlag {
	rr.mtx.Lock()
	defer rr.mtx.Unlock()
	p.setRecvRole(tr)
	return rr._resolveAndApply(p)
}

func (rr *roleResolver) onPeer(p *Peer) bool {
	if rr.isNotAllowed(p2pRoleNone, p.ID()) {
		rr.onEventCb(p2pEventNotAllowed, p)
		p.CloseByError(fmt.Errorf("onPeer not allowed connection"))
		return false
	}
	if rr.isTrustSeed(p) {
		rr.trustSeeds.SetAndRemoveByData(p.DialNetAddress(), string(p.NetAddress()))
	}
	return true
}

func (rr *roleResolver) onClose(p *Peer) {
	if rr.isTrustSeed(p) {
		rr.trustSeeds.RemoveData(p.DialNetAddress())
	}
}

func (rr *roleResolver) setTrustSeeds(seeds []NetAddress) {
	var ss []NetAddress
	for _, s := range seeds {
		if s != rr.self.NetAddress() && s.Validate() == nil {
			ss = append(ss, s)
		}
	}
	rr.trustSeeds.ClearAndAdd(ss...)
}

func (rr *roleResolver) getTrustSeedsMap() map[NetAddress]string {
	return rr.trustSeeds.Map()
}

func (rr *roleResolver) isTrustSeed(p *Peer) bool {
	return rr.trustSeeds.Contains(p.DialNetAddress())
}

func (rr *roleResolver) resolveRole(r PeerRoleFlag, id module.PeerID) PeerRoleFlag {
	if r.Has(p2pRoleRoot) {
		if rr.isNotAllowed(p2pRoleRoot, id) {
			r.UnSetFlag(p2pRoleRoot)
		}
	} else if rr.allowedRoots.Contains(id) {
		r.SetFlag(p2pRoleRoot)
	}

	if r.Has(p2pRoleSeed) && rr.isNotAllowed(p2pRoleSeed, id) {
		r.UnSetFlag(p2pRoleSeed)
	}
	return r
}

func (rr *roleResolver) isNotAllowed(r PeerRoleFlag, id module.PeerID) bool {
	s := rr.getAllowed(r)
	return !s.IsEmpty() && !s.Contains(id)
}

func (rr *roleResolver) getAllowed(r PeerRoleFlag) *PeerIDSet {
	switch r {
	case p2pRoleRoot:
		return rr.allowedRoots
	case p2pRoleSeed:
		return rr.allowedSeeds
	case p2pRoleNone:
		return rr.allowedPeers
	default:
		return nil
	}
}

func (rr *roleResolver) updateAllowed(version int64, r PeerRoleFlag, peers ...module.PeerID) bool {
	s := rr.getAllowed(r)
	if s.version < version {
		s.version = version
		rr.l.Debugf("updateAllowed version:%v r:%v peers:%v", version, r, peers)
		return s.ClearAndAdd(peers...)
	} else {
		rr.l.Debugln("SetRole", "ignore", version, "must greater than", s.version)
	}
	return false
}
