package metric

import (
	"context"
	"encoding/hex"
	"os"
	"strconv"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

//metric common tag key
var (
	MetricKeyHostname = NewMetricKey("hostname")
	MetricKeyChain    = NewMetricKey("channel")
	mKeys             = []tag.Key{MetricKeyHostname, MetricKeyChain}
	mTags             = make(map[*tag.Key]map[string]tag.Mutator)
	mViews            = make(map[string]*view.View)
	mViewMtx          sync.RWMutex
	mTagMtx           sync.Mutex

	rootMetricCtx    = GetMetricContext(context.Background(), &MetricKeyHostname, _resolveHostname(nil))
	defaultMetricCtx = GetMetricContext(rootMetricCtx, &MetricKeyChain, "UNKNOWN")
	chainMetricCtxs  = make(map[string]context.Context)
	chainMetricMtx   sync.RWMutex

	beforeExportFuncs    = make([]func(), 0)
	beforeExportFuncsMtx sync.RWMutex

	mtOnce sync.Once
)

func NewMetricKey(k string) tag.Key {
	key, err := tag.NewKey(k)
	if err != nil {
		log.Panicf("Fail tag.NewKey %s %+v", k, err)
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
	if err := view.Register(v); err != nil {
		log.Panicf("Fail RegisterMetricView view.Register %+v", err)
	}
	mViews[v.Name] = v
	return v
}

func RegisterBeforeExportFunc(f func()) {
	beforeExportFuncsMtx.Lock()
	defer beforeExportFuncsMtx.Unlock()
	beforeExportFuncs = append(beforeExportFuncs, f)
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

	ctx, err := tag.New(p, mt)
	if err != nil {
		log.Panicf("Fail tag.New %+v", err)
	}
	return ctx
}

func RemoveMetricContext(mk *tag.Key, v string) {
	defer mTagMtx.Unlock()
	mTagMtx.Lock()

	if m, ok := mTags[mk]; ok {
		if _, ok := m[v]; ok {
			delete(m, v)
		}
	}
}

func DefaultMetricContext() context.Context {
	return defaultMetricCtx
}

func GetMetricContextByCID(cid int) context.Context {
	chainMetricMtx.Lock()
	defer chainMetricMtx.Unlock()

	chainID := strconv.FormatInt(int64(cid), 16)
	ctx, ok := chainMetricCtxs[chainID]
	if !ok {
		ctx = GetMetricContext(rootMetricCtx, &MetricKeyChain, chainID)
		chainMetricCtxs[chainID] = ctx
	}
	return ctx
}

func RemoveMetricContextByCID(cid int) {
	chainMetricMtx.Lock()
	defer chainMetricMtx.Unlock()

	chainID := strconv.FormatInt(int64(cid), 16)
	if _, ok := chainMetricCtxs[chainID]; ok {
		delete(chainMetricCtxs, chainID)
		RemoveMetricContext(&MetricKeyChain, chainID)
	}
}

func GetCIDFromMetricContext(ctx context.Context) (int, bool) {
	if chainID, ok := tag.FromContext(ctx).Value(MetricKeyChain); ok {
		if cid, err := strconv.ParseInt(chainID, 16, 32); err == nil {
			return int(cid), true
		}
	}
	return 0, false
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
		log.Println("Initialize rootMetricCtx")
		rootMetricCtx = GetMetricContext(context.Background(), &MetricKeyHostname, _resolveHostname(w))
		defaultMetricCtx = GetMetricContext(rootMetricCtx, &MetricKeyChain, "UNKNOWN")
	})
}

func PrometheusExporter() *prometheus.Exporter {
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
	RegisterJsonrpc()
	return pe
}

func BeforeExport() {
	beforeExportFuncsMtx.RLock()
	defer beforeExportFuncsMtx.RUnlock()

	for _, f := range beforeExportFuncs {
		f()
	}
}

func ParseMetricData(r *view.Row, prev interface{}, cnt int) interface{} {
	switch data := r.Data.(type) {
	case *view.CountData:
		if prev != nil {
			return prev.(int64) + data.Value
		}
		return data.Value
	case *view.DistributionData:
		//TODO aggregation DistributionData
		return data
	case *view.SumData:
		if prev != nil {
			return prev.(float64) + data.Value
		}
		return data.Value
	case *view.LastValueData:
		if prev != nil {
			return (prev.(float64)*float64(cnt-1) + data.Value) / float64(cnt)
		}
		return data.Value
	}
	return nil
}

func Inspect(c module.Chain, informal bool) map[string]interface{} {
	mViewMtx.RLock()
	defer mViewMtx.RUnlock()

	BeforeExport()

	chainID, ok := tag.FromContext(c.MetricContext()).Value(MetricKeyChain)
	if !ok {
		return nil
	}
	m := make(map[string]interface{})
	for k, v := range mViews {
		i := 1
		m[v.Name] = nil
		rows, _ := view.RetrieveData(k)
		for _, r := range rows {
		tagLoop:
			for _, t := range r.Tags {
				if t.Key.Name() == MetricKeyChain.Name() && t.Value == chainID {
					m[v.Name] = ParseMetricData(r, m[v.Name], i)
					i++
					break tagLoop
				}
			}
		}
	}

	return m
}

func ResetMetricViews() {
	mViewMtx.RLock()
	defer mViewMtx.RUnlock()
	vs := make([]*view.View, 0)
	for _, v := range mViews {
		vs = append(vs, v)
	}
	view.Unregister(vs...)

	if err := view.Register(vs...); err != nil {
		log.Panicf("Fail ResetMetricViews view.Register %+v", err)
	}
}
