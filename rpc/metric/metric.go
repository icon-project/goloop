package metric

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"sync"
	"time"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/icon-project/goloop/module"
)

//metric common tag key
var (
	MetricKeyHostname = NewMetricKey("hostname")
	MetricKeyChannel  = NewMetricKey("channel")
	mKeys             = []tag.Key{MetricKeyHostname, MetricKeyChannel}
	MetricTagHostname tag.Mutator
	mTags             = make(map[*tag.Key]map[string]tag.Mutator)
	mtMtx             sync.Mutex
)

func NewMetricKey(k string) tag.Key {
	key, err := tag.NewKey(k)
	if err != nil {
		log.Fatalf("Fail tag.NewKey %s %+v", k, err)
	}

	mTags[&key] = make(map[string]tag.Mutator)
	return key
}

var aggTypeName = map[view.AggType]string{
	view.AggTypeNone:         "",
	view.AggTypeCount:        "_cnt",
	view.AggTypeSum:          "_sum",
	view.AggTypeDistribution: "_dist",
	view.AggTypeLastValue:    "",
}

func NewMetricView(m stats.Measure, a *view.Aggregation, tks []tag.Key) *view.View {
	return &view.View{
		Name:        m.Name() + aggTypeName[a.Type],
		Description: m.Description() + " Aggregated " + a.Type.String(),
		Measure:     m,
		Aggregation: a,
		TagKeys:     append(mKeys, tks...),
	}
}

func GetMetricTag(mk *tag.Key, v string) tag.Mutator {
	defer mtMtx.Unlock()
	mtMtx.Lock()

	m, ok := mTags[mk]
	if !ok {
		m = make(map[string]tag.Mutator)
		mTags[mk] = m
	}

	mt, ok := m[v]
	if !ok {
		mt = tag.Upsert(*mk, v)
		m[v] = mt
	}
	return mt
}

func NewMetricContext(channel string, mts ...tag.Mutator) context.Context {
	mtChannel := GetMetricTag(&MetricKeyChannel, channel)
	ms := append([]tag.Mutator{MetricTagHostname, mtChannel}, mts...)
	ctx, err := tag.New(context.Background(), ms...)
	if err != nil {
		log.Fatalf("Fail tag.New %+v", err)
	}
	return ctx
}

func Initialize(w module.Wallet) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = hex.EncodeToString(w.Address().ID()[:4])
	}
	MetricTagHostname = tag.Insert(MetricKeyHostname, nodeName)
}

func PromethusExporter() *prometheus.Exporter {
	// prometheus
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "goloop",
	})

	if err != nil {
		log.Printf("Failed to create Prometheus exporter: %+v", err)
	}

	view.RegisterExporter(pe)
	// Set reporting period to report data at every second.
	view.SetReportingPeriod(1000 * time.Millisecond)

	RegisterConsensus()
	RegisterNetwork()
	return pe
}
