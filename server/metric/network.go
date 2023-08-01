package metric

import (
	"context"
	"fmt"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	msSend     = stats.Int64("network_send", "send", stats.UnitBytes)
	msRecv     = stats.Int64("network_recv", "recv", stats.UnitBytes)
	mkDest     = NewMetricKey("dest")
	mkProtocol = NewMetricKey("protocol")
	networkMks = []tag.Key{mkDest, mkProtocol}
)

func RegisterNetwork() {
	RegisterMetricView(msSend, view.Count(), networkMks)
	RegisterMetricView(msSend, view.Sum(), networkMks)
	RegisterMetricView(msRecv, view.Count(), networkMks)
	RegisterMetricView(msRecv, view.Sum(), networkMks)
}

type NetworkMetric struct {
	ctx    context.Context
	ctxMap map[string]context.Context
	ctxMtx sync.RWMutex
}

func (m *NetworkMetric) get(key string) (context.Context, bool) {
	m.ctxMtx.RLock()
	defer m.ctxMtx.RUnlock()

	v, ok := m.ctxMap[key]
	return v, ok
}

func (m *NetworkMetric) put(key string, ctx context.Context) {
	m.ctxMtx.Lock()
	defer m.ctxMtx.Unlock()

	m.ctxMap[key] = ctx
}

func (m *NetworkMetric) getMetricContext(dest byte, ttl byte, hint byte, protocol uint16) context.Context {
	strDest := fmt.Sprintf("0x%02x%02x%02x", dest, ttl, hint)
	strProtocol := fmt.Sprintf("%#04x", protocol)
	key := strDest + strProtocol
	ctx, ok := m.get(key)
	if !ok {
		ctx = GetMetricContext(m.ctx, &mkDest, strDest)
		ctx = GetMetricContext(ctx, &mkProtocol, strProtocol)
		m.put(key, ctx)
	}
	return ctx
}

func (m *NetworkMetric) OnSend(dest byte, ttl byte, hint byte, protocol uint16, pktLen uint32) {
	ctx := m.getMetricContext(dest, ttl, hint, protocol)
	stats.Record(ctx, msSend.M(int64(pktLen)))
}

func (m *NetworkMetric) OnRecv(dest byte, ttl byte, hint byte, protocol uint16, pktLen uint32) {
	ctx := m.getMetricContext(dest, ttl, hint, protocol)
	stats.Record(ctx, msRecv.M(int64(pktLen)))
}

func NewNetworkMetric(ctx context.Context) *NetworkMetric {
	return &NetworkMetric{
		ctx: ctx,
		ctxMap: make(map[string]context.Context),
	}
}
