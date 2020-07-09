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
	m["friends"] = peerSetToMapArray(mgr.p2p.friends, informal)
	m["parent"] = peerToMap(mgr.p2p.getParent(), informal)
	m["children"] = peerSetToMapArray(mgr.p2p.children, informal)
	m["uncles"] = peerSetToMapArray(mgr.p2p.uncles, informal)
	m["nephews"] = peerSetToMapArray(mgr.p2p.nephews, informal)
	m["orphanages"] = peerSetToMapArray(mgr.p2p.orphanages, informal)
	if informal {
		m["pre"] = peerSetToMapArray(mgr.p2p.pre, informal)
		m["reject"] = peerSetToMapArray(mgr.p2p.reject, informal)
	}
	return m
}

func inspectProtocol(mgr *manager) map[string]interface{} {
	m := make(map[string]interface{})
	for _, ph := range mgr.protocolHandlers {
		m[ph.name] = protocolHandlerToMap(ph)
	}
	return m
}

func peerSetToMapArray(s *PeerSet, informal bool) []map[string]interface{} {
	rarr := make([]map[string]interface{}, s.Len())
	for i, v := range s.Array() {
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
		m["id"] = p.id.String()
		m["addr"] = string(p.netAddress)
		m["in"] = p.incomming
		m["role"] = p.role
		if informal {
			m["channel"] = p.channel
			m["conn"] = p.connType
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
		m["priority"] = ph.priority

		parr := make([]int, 0)
		for _, p := range ph.subProtocols {
			parr = append(parr, int(p.Uint16()))
		}
		sort.Ints(parr)
		sarr := make([]string, len(parr))
		for i, p := range parr {
			sarr[i] = fmt.Sprintf("%#04x", p)
		}
		m["subProtocols"] = strings.Join(sarr, ",")

		m["receiveQueue"] = ph.receiveQueue.Available()
		m["eventQueue"] = ph.eventQueue.Available()
		m["sendQueue"] = ph.m.p2p.sendQueue.Available(int(ph.protocol.ID()))
	}
	return m
}
