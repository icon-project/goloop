package metric

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"strconv"
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
	MetricKeyChain    = NewMetricKey("channel")
	mKeys             = []tag.Key{MetricKeyHostname, MetricKeyChain}
	RootMetricCtx     = context.Background()
	mTags             = make(map[*tag.Key]map[string]tag.Mutator)
	mCtxs             = make(map[tag.Mutator]context.Context)
	mViews            = make(map[string]*view.View)
	mViewMtx          sync.RWMutex
	mTagMtx           sync.Mutex
	mtOnce            sync.Once
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

func RegisterMetricView(m stats.Measure, a *view.Aggregation, tks []tag.Key) *view.View {
	defer mViewMtx.Unlock()
	mViewMtx.Lock()

	v := &view.View{
		Name:        m.Name() + aggTypeName[a.Type],
		Description: m.Description() + " Aggregated " + a.Type.String(),
		Measure:     m,
		Aggregation: a,
		TagKeys:     append(mKeys, tks...),
	}
	err := view.Register(v)
	if err != nil {
		log.Fatalf("Fail RegisterMetricView view.Register %+v", err)
	}
	mViews[v.Name] = v
	return v
}

func GetMetricContext(p context.Context, mk *tag.Key, v string) context.Context {
	defer mTagMtx.Unlock()
	mTagMtx.Lock()

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

	ctx, ok := mCtxs[mt]
	if !ok {
		tCtx, err := tag.New(p, mt)
		if err != nil {
			log.Fatalf("Fail tag.New %+v", err)
		}
		mCtxs[mt] = tCtx
		ctx = tCtx
	}
	return ctx
}

func DefaultMetricContext() context.Context {
	return GetMetricContext(RootMetricCtx, &MetricKeyChain, "UNKNOWN")
}

func GetMetricContextByNID(NID int) context.Context {
	chainID := strconv.FormatInt(int64(NID), 16)
	return GetMetricContext(RootMetricCtx, &MetricKeyChain, chainID)
}

func _resolveHostname(w module.Wallet) string {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		if w == nil {
			nodeName, _ = os.Hostname()
		} else {
			nodeName = hex.EncodeToString(w.Address().ID()[:4])
		}
	}
	return nodeName
}

func Initialize(w module.Wallet) {
	mtOnce.Do(func() {
		log.Println("Initialize RootMetricCtx")
		RootMetricCtx = GetMetricContext(context.Background(), &MetricKeyHostname, _resolveHostname(w))
	})
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
	RegisterTransaction()
	return pe
}

func ParseMetricData(r *view.Row) interface{} {
	switch data := r.Data.(type) {
	case *view.CountData:
		return data.Value
	case *view.DistributionData:
		return data
	case *view.SumData:
		return data.Value
	case *view.LastValueData:
		return data.Value
	}
	return nil
}

func Inspect(c module.Chain) map[string]interface{} {
	mViewMtx.RLock()
	defer mViewMtx.RUnlock()

	//c.MetricContext()
	chainID, ok := tag.FromContext(c.MetricContext()).Value(MetricKeyChain)
	if !ok {
		return nil
	}
	m := make(map[string]interface{})
	for k, v := range mViews {
		m[v.Name] = nil
		rows, _ := view.RetrieveData(k)
		for _, r := range rows {
		LoopTag:
			for _, t := range r.Tags {
				if t.Key.Name() == MetricKeyChain.Name() && t.Value == chainID {
					m[v.Name] = ParseMetricData(r)
					break LoopTag
				}
			}
		}
	}
	return m
}
