package metric

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	msHeight     = stats.Int64("consensus_height", "height", stats.UnitDimensionless)
	msRound      = stats.Int64("consensus_round", "round", stats.UnitDimensionless)
	msHeightD    = stats.Int64("consensus_height_duration", "block_duration", stats.UnitMilliseconds)
	msRoundD     = stats.Int64("consensus_round_duration", "block_duration", stats.UnitMilliseconds)
	consensusMks = []tag.Key{}
)

func RegisterConsensus() {
	RegisterMetricView(msHeight, view.LastValue(), consensusMks)
	RegisterMetricView(msRound, view.LastValue(), consensusMks)
	RegisterMetricView(msHeightD, view.LastValue(), consensusMks)
	RegisterMetricView(msRoundD, view.LastValue(), consensusMks)
}

type ConsensusMetric struct {
	ctx context.Context
	heightTs time.Time
	roundTs time.Time
}

func (m *ConsensusMetric) OnHeight(height int64) {
	now := time.Now()
	d := now.Sub(m.heightTs)
	m.heightTs = now
	stats.Record(m.ctx, msHeight.M(height), msHeightD.M(int64(d/time.Millisecond)))
}

func (m *ConsensusMetric) OnRound(round int32) {
	now := time.Now()
	d := now.Sub(m.roundTs)
	m.roundTs = now
	stats.Record(m.ctx, msRound.M(int64(round)), msRoundD.M(int64(d/time.Millisecond)))
}

func NewConsensusMetric(ctx context.Context) *ConsensusMetric {
	return &ConsensusMetric{
		ctx : ctx,
	}
}
