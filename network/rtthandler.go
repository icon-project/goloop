package network

import (
	"time"

	"github.com/icon-project/goloop/common/log"
)

type rttHandler struct {
	l log.Logger
}

func newRTTHandler(l log.Logger) *rttHandler {
	return &rttHandler{l: l}
}

func (h *rttHandler) startRtt(p *Peer) {
	p.rtt.StartWithAfterFunc(DefaultRttLogTimeout, func() {
		h.l.Warnln("RTT Timeout", DefaultRttLogTimeout, p)
	})
}

func (h *rttHandler) stopRtt(p *Peer) time.Duration {
	rttLast := p.rtt.Stop()
	if rttLast >= DefaultRttLogThreshold {
		h.l.Warnln("RTT Threshold", DefaultRttLogThreshold, p)
	}
	return rttLast
}

func (h *rttHandler) checkAccuracy(p *Peer, v time.Duration) {
	last, _ := p.rtt.Value()
	df := v - last
	if df > DefaultRttAccuracy {
		h.l.Debugln("checkAccuracy", df, "DefaultRttAccuracy", DefaultRttAccuracy, p)
	}
}
