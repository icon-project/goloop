package metric

import (
	"context"
	"strconv"

	"github.com/labstack/gommon/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	TxTypePatch  = "patch"
	TxTypeNormal = "normal"
)

var (
	msAddTx    = stats.Int64("txpool_add", "add transaction", stats.UnitBytes)
	msRemoveTx = stats.Int64("txpool_remove", "remove transaction", stats.UnitBytes)
	msDropTx   = stats.Int64("txpool_drop", "drop transaction", stats.UnitBytes)
	mkTxType   = NewMetricKey("tx_type")
	txPoolMks  = []tag.Key{mkTxType}
)

func RegisterTxPool() {
	err := view.Register(
		NewMetricView(msAddTx, view.Count(), txPoolMks),
		NewMetricView(msAddTx, view.Sum(), txPoolMks),
		NewMetricView(msRemoveTx, view.Count(), txPoolMks),
		NewMetricView(msRemoveTx, view.Sum(), txPoolMks),
		NewMetricView(msDropTx, view.Count(), txPoolMks),
		NewMetricView(msDropTx, view.Sum(), txPoolMks),
	)
	if err != nil {
		log.Fatalf("Fail RegisterMetric view.Register %+v", err)
	}
}

func RecordOnAddTx(c context.Context, bytes int) {
	stats.Record(c, msAddTx.M(int64(bytes)))
}

func RecordOnRemoveTx(c context.Context, n int) {
	stats.Record(c, msRemoveTx.M(int64(n)))
}

func RecordOnDropTx(c context.Context, n int) {
	stats.Record(c, msDropTx.M(int64(n)))
}

func NewTxPoolContext(nid int, t string) context.Context {
	mtTxType := GetMetricTag(&mkTxType, t)
	return NewMetricContext(strconv.Itoa(nid), mtTxType)
}
