package network

import (
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

	l log.Logger
}

func newRoleResolver(self *Peer, l log.Logger) *roleResolver {
	return &roleResolver{
		self:         self,
		trustSeeds:   NewNetAddressSet(),
		allowedRoots: NewPeerIDSet(),
		allowedSeeds: NewPeerIDSet(),
		allowedPeers: NewPeerIDSet(),
		l:            l,
	}
}

func (rr *roleResolver) onPeer(p *Peer) {
	if rr.isTrustSeed(p) {
		rr.trustSeeds.SetAndRemoveByData(p.DialNetAddress(), string(p.NetAddress()))
	}
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

func (rr *roleResolver) resolveRole(r PeerRoleFlag, id module.PeerID, onlyUnSet bool) PeerRoleFlag {
	if onlyUnSet {
		if r.Has(p2pRoleRoot) && !rr.allowedRoots.IsEmpty() && !rr.allowedRoots.Contains(id) {
			r.UnSetFlag(p2pRoleRoot)
		}
		if r.Has(p2pRoleSeed) && !rr.allowedSeeds.IsEmpty() && !rr.allowedSeeds.Contains(id) {
			r.UnSetFlag(p2pRoleSeed)
		}
	} else {
		if rr.allowedRoots.Contains(id) {
			r.SetFlag(p2pRoleRoot)
		} else if r.Has(p2pRoleRoot) && !rr.allowedRoots.IsEmpty() {
			r.UnSetFlag(p2pRoleRoot)
		}
		if rr.allowedSeeds.Contains(id) {
			r.SetFlag(p2pRoleSeed)
		} else if r.Has(p2pRoleSeed) && !rr.allowedSeeds.IsEmpty() {
			r.UnSetFlag(p2pRoleSeed)
		}
	}
	return r
}

func (rr *roleResolver) isAllowed(r PeerRoleFlag, id module.PeerID) bool {
	s := rr.getAllowed(r)
	return s.IsEmpty() || s.Contains(id)
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
		return s.ClearAndAdd(peers...)
	} else {
		rr.l.Debugln("SetRole", "ignore", version, "must greater than", s.version)
	}
	return false
}
