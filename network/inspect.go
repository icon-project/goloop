package network

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/icon-project/goloop/module"
)

func Inspect(c module.Chain, informal bool) map[string]interface{} {
	var mgr *manager
	if nm := c.NetworkManager(); nm == nil {
		return nil
	} else {
		mgr = nm.(*manager)
	}
	m := make(map[string]interface{})
	m["p2p"] = inspectP2P(mgr, informal)
	if informal {
		m["protocol"] = inspectProtocol(mgr)
	}
	return m
}

func inspectP2P(mgr *manager, informal bool) map[string]interface{} {
	m := make(map[string]interface{})
	m["self"] = peerToMap(mgr.p2p.self, informal)
	m["seeds"] = mgr.p2p.seeds.Map()
	m["roots"] = mgr.p2p.roots.Map()
	m["friends"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeFriend), informal)
	var parent *Peer
	parents := mgr.p2p.findPeers(nil, p2pConnTypeParent)
	if len(parents) > 0 {
		parent = parents[0]
	}
	m["parent"] = peerToMap(parent, informal)
	m["children"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeChildren), informal)
	m["uncles"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeUncle), informal)
	m["nephews"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeNephew), informal)
	m["others"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeOther), informal)
	m["orphanages"] = peerArrayToMapArray(mgr.p2p.findPeers(nil, p2pConnTypeNone), informal)
	if informal {
		m["transiting"] = peerSetToMapArray(mgr.p2p.transiting, informal)
		m["reject"] = peerSetToMapArray(mgr.p2p.reject, informal)
	}
	m["trustSeeds"] = mgr.p2p.rr.getTrustSeedsMap()
	return m
}

func inspectProtocol(mgr *manager) map[string]interface{} {
	m := make(map[string]interface{})
	for _, ph := range mgr.protocolHandlers {
		m[ph.getName()] = protocolHandlerToMap(ph)
	}
	return m
}

func peerSetToMapArray(s *PeerSet, informal bool) []map[string]interface{} {
	return peerArrayToMapArray(s.Array(), informal)
}

func peerArrayToMapArray(arr []*Peer, informal bool) []map[string]interface{} {
	rarr := make([]map[string]interface{}, len(arr))
	for i, v := range arr {
		rarr[i] = peerToMap(v, informal)
	}
	sort.Slice(rarr, func(i int, j int) bool {
		return rarr[i]["addr"].(string) < rarr[j]["addr"].(string)
	})
	return rarr
}

func peerToMap(p *Peer, informal bool) map[string]interface{} {
	m := make(map[string]interface{})
	if p != nil {
		m["id"] = p.ID().String()
		m["addr"] = string(p.NetAddress())
		m["in"] = p.In()
		m["role"] = p.Role()
		if informal {
			m["channel"] = p.Channel()
			m["conn"] = p.ConnType()
			m["rrole"] = p.RecvRole()
			m["rconn"] = p.RecvConnType()
			m["rtt"] = p.rtt.String()
			if p.q != nil {
				sq := make([]string, DefaultSendQueueMaxPriority)
				for i := 0; i < DefaultSendQueueMaxPriority; i++ {
					sq[i] = strconv.Itoa(p.q.Available(i))
				}
				m["sendQueue"] = strings.Join(sq, ",")
			}
		}
	}
	return m
}
func protocolHandlerToMap(ph *protocolHandler) map[string]interface{} {
	m := make(map[string]interface{})
	if ph != nil {
		m["protocol"] = fmt.Sprintf("%#04x,", ph.protocol.Uint16())
		m["priority"] = ph.getPriority()

		l := make([]int, 0)
		spis := ph.getSubProtocols()
		for _, spi := range spis {
			l = append(l, int(spi.Uint16()))
		}
		sort.Ints(l)
		sarr := make([]string, len(spis))
		for i, v := range l {
			sarr[i] = fmt.Sprintf("%#04x", v)
		}
		m["subProtocols"] = strings.Join(sarr, ",")

		m["receiveQueue"] = ph.receiveQueue.Available()
		m["eventQueue"] = ph.eventQueue.Available()
		m["sendQueue"] = ph.m.p2p.sendQueue.Available(int(ph.protocol.ID()))
	}
	return m
}
