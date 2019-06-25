package network

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/icon-project/goloop/module"
)

func Inspect(c module.Chain) map[string]interface{} {
	mgr := c.NetworkManager().(*manager)
	m := make(map[string]interface{})
	m["p2p"] = inspectP2P(mgr)
	m["protocol"] = inspectProtocol(mgr)
	return m
}

func inspectP2P(mgr *manager) map[string]interface{} {
	m := make(map[string]interface{})
	m["self"] = peerToMap(mgr.p2p.self)
	m["seeds"] = mgr.p2p.seeds.Map()
	m["roots"] = mgr.p2p.roots.Map()
	m["friends"] = peerToMapArray(mgr.p2p.friends)
	m["parent"] = peerToMap(mgr.p2p.getParent())
	m["children"] = peerToMapArray(mgr.p2p.children)
	m["uncles"] = peerToMapArray(mgr.p2p.uncles)
	m["nephews"] = peerToMapArray(mgr.p2p.nephews)
	m["orphanages"] = peerToMapArray(mgr.p2p.orphanages)
	m["pre"] = peerToMapArray(mgr.p2p.pre)
	m["reject"] = peerToMapArray(mgr.p2p.reject)
	return m
}

func inspectProtocol(mgr *manager) map[string]interface{} {
	m := make(map[string]interface{})
	for _, ph := range mgr.protocolHandlers {
		m[ph.name] = protocolHandlerToMap(ph)
	}
	return m
}

func peerToMapArray(s *PeerSet) []map[string]interface{} {
	rarr := make([]map[string]interface{}, s.Len())
	for i, v := range s.Array() {
		rarr[i] = peerToMap(v)
	}
	sort.Slice(rarr, func(i int, j int) bool{
		return rarr[i]["addr"].(string) < rarr[j]["addr"].(string)
	})
	return rarr
}
func peerToMap(p *Peer) map[string]interface{} {
	m := make(map[string]interface{})
	if p != nil {
		m["id"] = p.id.String()
		m["addr"] = string(p.netAddress)
		m["in"] = p.incomming
		m["channel"] = p.channel
		m["role"] = p.role
		m["conn"] = p.connType
		m["rtt"] = p.rtt.String()
		if p.q != nil {
			sq := make([]string,DefaultSendQueueMaxPriority)
			for i:=0;i<DefaultSendQueueMaxPriority;i++{
				sq[i] = strconv.Itoa(p.q.Available(i))
			}
			m["sendQueue"] = strings.Join(sq,",")
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
