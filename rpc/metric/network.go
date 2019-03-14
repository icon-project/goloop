package metric

import (
	"fmt"
	"log"

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
	err := view.Register(
		NewMetricView(msSend, view.Count(), networkMks),
		NewMetricView(msSend, view.Sum(), networkMks),
		NewMetricView(msRecv, view.Count(), networkMks),
		NewMetricView(msRecv, view.Sum(), networkMks),
	)
	if err != nil {
		log.Fatalf("Fail RegisterMetric view.Register %+v", err)
	}
}

func recordNetwork(channel string, dest byte, ttl byte, hint byte, protocol uint16, m stats.Measurement) {
	strDest := fmt.Sprintf("0x%02x%02x%02x", dest, ttl, hint)
	mtDest := GetMetricTag(&mkDest, strDest)

	strProtocol := fmt.Sprintf("%#04x", protocol)
	mtProtocol := GetMetricTag(&mkProtocol, strProtocol)

	ctx := NewMetricContext(channel, mtDest, mtProtocol)
	stats.Record(ctx, m)
}

func RecordOnSend(channel string, dest byte, ttl byte, hint byte, protocol uint16, pktLen uint32) {
	recordNetwork(channel, dest, ttl, hint, protocol, msSend.M(int64(pktLen)))
}

func RecordOnRecv(channel string, dest byte, ttl byte, hint byte, protocol uint16, pktLen uint32) {
	recordNetwork(channel, dest, ttl, hint, protocol, msRecv.M(int64(pktLen)))
}
