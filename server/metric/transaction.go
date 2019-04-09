package metric

import (
	"context"
	"strconv"
	"sync"
	"time"

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
	msAddTx         = stats.Int64("txpool_add", "Add Transaction", stats.UnitBytes)
	msRemoveTx      = stats.Int64("txpool_remove", "Remove Transaction", stats.UnitBytes)
	msDropTx        = stats.Int64("txpool_drop", "Drop Transaction", stats.UnitBytes)
	msFinLatency    = stats.Int64("txlatency_finalize", "Finalize Transaction Latency", stats.UnitMilliseconds)
	msCommitLatency = stats.Int64("txlatency_commit", "Commit Transaction Latency", stats.UnitMilliseconds)
	mkTxType        = NewMetricKey("tx_type")
	txPoolMks       = []tag.Key{mkTxType}
)

func RegisterTransaction() {
	err := view.Register(
		NewMetricView(msAddTx, view.Count(), txPoolMks),
		NewMetricView(msAddTx, view.Sum(), txPoolMks),
		NewMetricView(msRemoveTx, view.Count(), txPoolMks),
		NewMetricView(msRemoveTx, view.Sum(), txPoolMks),
		NewMetricView(msDropTx, view.Count(), txPoolMks),
		NewMetricView(msDropTx, view.Sum(), txPoolMks),
		NewMetricView(msFinLatency, view.LastValue(), txPoolMks),
		NewMetricView(msCommitLatency, view.LastValue(), txPoolMks),
	)
	if err != nil {
		log.Fatalf("Fail RegisterMetric view.Register %+v", err)
	}
}

type commitRecord struct {
	timestamp time.Time
	duration  time.Duration
}

type TxMetric struct {
	lock    sync.Mutex
	context context.Context
	commits map[string]*commitRecord
}

func (c *TxMetric) OnAddTx(n int) {
	stats.Record(c.context, msAddTx.M(int64(n)))
}

func (c *TxMetric) OnRemoveTx(n int) {
	stats.Record(c.context, msRemoveTx.M(int64(n)))
}

func (c *TxMetric) OnDropTx(n int) {
	stats.Record(c.context, msDropTx.M(int64(n)))
}

func (c *TxMetric) OnFinalize(hash []byte, ts time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()

	sHash := string(hash)
	commit, ok := c.commits[sHash]
	if !ok {
		return
	}
	delete(c.commits, sHash)
	d := ts.Sub(commit.timestamp) + commit.duration
	stats.Record(c.context, msFinLatency.M(int64(d/time.Millisecond)))
}

func (c *TxMetric) OnCommit(hash []byte, ts time.Time, d time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.commits[string(hash)] = &commitRecord{
		timestamp: ts,
		duration:  d,
	}
	stats.Record(c.context, msCommitLatency.M(int64(d/time.Millisecond)))
}

func NewTransactionMetric(nid int, t string) *TxMetric {
	mtTxType := GetMetricTag(&mkTxType, t)
	return &TxMetric{
		context: NewMetricContext(strconv.Itoa(nid), mtTxType),
		commits: make(map[string]*commitRecord),
	}
}
