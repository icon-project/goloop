package metric

import (
	"log"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	msHeight        = stats.Int64("consensus_height", "height", stats.UnitDimensionless)
	msRound         = stats.Int64("consensus_round", "round", stats.UnitDimensionless)
	msHeightD       = stats.Int64("consensus_height_duration", "block_duration", stats.UnitMilliseconds)
	msRoundD        = stats.Int64("consensus_round_duration", "block_duration", stats.UnitMilliseconds)
	mkProposer      = NewMetricKey("proposer")
	consensusMks    = []tag.Key{}
	mtProposerTrue  = tag.Upsert(mkProposer, "true")
	mtProposerFalse = tag.Upsert(mkProposer, "false")
	nsHeight        int64
	nsRound         int64
)

func RegisterConsensus() {
	err := view.Register(
		NewMetricView(msHeight, view.LastValue(), consensusMks),
		NewMetricView(msRound, view.LastValue(), consensusMks),
		NewMetricView(msHeightD, view.LastValue(), consensusMks),
		NewMetricView(msRoundD, view.LastValue(), consensusMks),
	)
	if err != nil {
		log.Fatalf("Fail RegisterMetric view.Register %+v", err)
	}
}

func recordConsensus(channel string, isProposer bool, ms ...stats.Measurement) {
	//mtProposer := mtProposerFalse
	//if isProposer {
	//	mtProposer = mtProposerTrue
	//}
	ctx := NewMetricContext(channel)
	stats.Record(ctx, ms...)
}

func RecordOnHeight(channel string, isProposer bool, height int64) {
	n := time.Now().UnixNano()
	d := (n - nsHeight) / 1000000
	nsHeight = n
	recordConsensus(channel, isProposer, msHeight.M(height), msHeightD.M(d))
}

func RecordOnRound(channel string, isProposer bool, round int32) {
	n := time.Now().UnixNano()
	d := (n - nsRound) / 1000000
	nsRound = n
	recordConsensus(channel, isProposer, msRound.M(int64(round)), msRoundD.M(d))
}
