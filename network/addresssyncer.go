package network

import "github.com/icon-project/goloop/common/log"

type addressSyncer struct {
	m  map[PeerRoleFlag]*NetAddressSet
	d  *Dialer
	pm *peerManager
	l  log.Logger
}

func newAddressSyncer(d *Dialer, pm *peerManager, l log.Logger) *addressSyncer {
	return &addressSyncer{
		m: map[PeerRoleFlag]*NetAddressSet{
			p2pRoleRoot: NewNetAddressSet(),
			p2pRoleSeed: NewNetAddressSet(),
		},
		d:  d,
		pm: pm,
		l:  l,
	}
}

var (
	applyPeerRoleLogs = map[PeerRoleFlag]string{
		p2pRoleSeed: "addSeed",
		p2pRoleRoot: "addRoot",
	}
)

func (as *addressSyncer) applyPeerRole(p *Peer) {
	r := p.Role()
	na := p.NetAddress()
	id := p.ID().String()
	for tr, s := range as.m {
		if r.Has(tr) {
			opLog := applyPeerRoleLogs[tr]
			c, o := s.SetAndRemoveByData(na, id)
			if o != "" {
				as.l.Debugln("applyPeerRole", opLog, "updated NetAddress old:", o, ", now:", na, ",peerID:", id)
			}
			if c != "" {
				as.l.Infoln("applyPeerRole", opLog, "conflict NetAddress", na, "removed:", c, ",now:", id)
			}
		} else {
			s.Remove(na)
		}
	}
}

func (as *addressSyncer) getNetAddresses(r PeerRoleFlag) []NetAddress {
	return as.m[r].Array()
}

func (as *addressSyncer) getUniqueNetAddresses(r PeerRoleFlag) []NetAddress {
	l := as.m[r].Array()
	var ret []NetAddress
	for _, v := range l {
		contains := false
		for k, s := range as.m {
			if r == k {
				continue
			}
			if s.Contains(v) {
				contains = true
				break
			}
		}
		if !contains {
			ret = append(ret, v)
		}
	}
	return ret
}

func (as *addressSyncer) filterForMerge(r PeerRoleFlag, l []NetAddress) []NetAddress {
	s := as.m[r]
	var ret []NetAddress
	for _, na := range l {
		if d, ok := s.Data(na); !ok || len(d) == 0 {
			ret = append(ret, na)
		}
	}
	return ret
}

func (as *addressSyncer) mergeNetAddresses(r PeerRoleFlag, l []NetAddress) {
	switch r {
	case p2pRoleRoot, p2pRoleSeed:
		s := as.m[r]
		s.Merge(as.filterForMerge(p2pRoleSeed, l)...)
	}
}

func (as *addressSyncer) removeData(p *Peer) {
	r := p.Role()
	na := p.NetAddress()
	for tr, s := range as.m {
		if r.Has(tr) {
			s.RemoveData(na)
		}
	}
}

func (as *addressSyncer) _dial(na NetAddress) error {
	if err := as.d.Dial(string(na)); err != nil {
		if err == ErrAlreadyDialing {
			as.l.Infoln("Dial ignore", na, err)
			return nil
		}
		as.l.Infoln("Dial fail", na, err)
		return err
	}
	return nil
}

func (as *addressSyncer) dial(r PeerRoleFlag) int {
	s := as.m[r]
	n := 0
	arr := s.Array()
	for _, na := range arr {
		if !as.pm.hasNetAddress(na) {
			as.l.Debugf("dial to role:%v %v", r, na)
			if err := as._dial(na); err != nil {
				s.Remove(na)
			} else {
				n++
			}
		}
	}
	return n
}

func (as *addressSyncer) contains(r PeerRoleFlag, na NetAddress) bool {
	return as.m[r].Contains(na)
}
