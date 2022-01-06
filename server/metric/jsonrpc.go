package metric

import (
	"context"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	mkMethod    = NewMetricKey("method")
	msFailure   = stats.Int64("jsonrpc_failure", "jsonrpc failures", stats.UnitDimensionless)
	failureMks  = []tag.Key{mkMethod}
	msRetrieve  = stats.Int64("jsonrpc_retrieve", "jsonrpc retrieve methods", stats.UnitDimensionless)
	retrieveMks = []tag.Key{mkMethod}
	msSendTx    = stats.Int64("jsonrpc_send_transaction", "jsonrpc icx_sendTransaction method", stats.UnitDimensionless)
	msExecutes  = map[string]*stats.Int64Measure{
		"icx_call":           stats.Int64("jsonrpc_call", "jsonrpc icx_call method", stats.UnitDimensionless),
		"debug_getTrace":     stats.Int64("jsonrpc_get_trace", "jsonrpc debug_getTrace method", stats.UnitDimensionless),
		"debug_estimateStep": stats.Int64("jsonrpc_estimate_step", "jsonrpc debug_estimateStep method", stats.UnitDimensionless),
	}
	msWaits = map[string]*stats.Int64Measure{
		"icx_sendTransactionAndWait": stats.Int64("jsonrpc_send_and_wait", "jsonrpc icx_sendTransactionAndWait method", stats.UnitDimensionless),
		"icx_waitTransactionResult":  stats.Int64("jsonrpc_wait_result", "jsonrpc icx_waitTransactionResult method", stats.UnitDimensionless),
	}
	emptyMks = []tag.Key{}

	msRespTimes = make(map[stats.Measure]*stats.Int64Measure)

	jms    = make([]*JsonrpcMetric, 0)
	jmsMtx sync.RWMutex
)

const (
	DefaultJsonrpcDurationsSize         = 20000
	DefaultJsonrpcDurationsUpdateExpire = 10 * time.Second
)

func _registerMetricViewWithResponseTimeMeasure(ms stats.Measure, tks []tag.Key) {
	msRespTime, ok := msRespTimes[ms]
	if !ok {
		msRespTime = stats.Int64(ms.Name()+"_response_time", ms.Description()+" response time", "ns")
		msRespTimes[ms] = msRespTime
	}
	RegisterMetricView(ms, view.Count(), tks)
	RegisterMetricView(msRespTime, view.LastValue(), tks)
}

func RegisterJsonrpc() {
	_registerMetricViewWithResponseTimeMeasure(msFailure, failureMks)
	_registerMetricViewWithResponseTimeMeasure(msRetrieve, retrieveMks)
	_registerMetricViewWithResponseTimeMeasure(msSendTx, emptyMks)
	for _, ms := range msExecutes {
		_registerMetricViewWithResponseTimeMeasure(ms, emptyMks)
	}
	for _, ms := range msWaits {
		_registerMetricViewWithResponseTimeMeasure(ms, emptyMks)
	}

	RegisterBeforeExportFunc(func() {
		jmsMtx.RLock()
		defer jmsMtx.RUnlock()
		for _, jm := range jms {
			jm.Remove(DefaultJsonrpcDurationsUpdateExpire)
		}
	})
}

type JsonrpcMeasure struct {
	ms         *stats.Int64Measure
	msRespTime *stats.Int64Measure
	d          *Durations
	m          map[string]context.Context
	mtx        sync.RWMutex
}

func (m *JsonrpcMeasure) EnsureContext(ctx context.Context) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	k := tag.FromContext(ctx).String()
	if _, ok := m.m[k]; !ok {
		m.m[k] = ctx
	}
}

func (m *JsonrpcMeasure) RemoveAndRecord(ctx context.Context, ts time.Time, expire time.Duration) {
	m.EnsureContext(ctx)
	d := m.d.Add(ts)
	stats.Record(ctx, m.ms.M(int64(d.d/time.Millisecond)))
	m.d.Remove(expire)
	//stats.Record(ctx, m.msRespTime.M(int64(m.d.Avg()/time.Millisecond)))
	stats.Record(ctx, m.msRespTime.M(int64(m.d.Avg())))
}

func (m *JsonrpcMeasure) Remove(expire time.Duration) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	m.d.Remove(expire)
	for _, ctx := range m.m {
		stats.Record(ctx, m.msRespTime.M(int64(m.d.Avg())))
	}
}

type JsonrpcMetric struct {
	expire time.Duration
	//each ms.Name + chain
	m   map[string]*JsonrpcMeasure
	mtx sync.RWMutex
}

func (m *JsonrpcMetric) EnsureMeasure(ctx context.Context, ms *stats.Int64Measure) *JsonrpcMeasure {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	key := ms.Name()
	chainID, ok := tag.FromContext(ctx).Value(MetricKeyChain)
	if ok {
		key += chainID
	}

	var jm *JsonrpcMeasure
	jm, ok = m.m[key]
	if !ok {
		jm = &JsonrpcMeasure{
			ms:         ms,
			msRespTime: msRespTimes[ms],
			d:          NewDurations(DefaultJsonrpcDurationsSize),
			m:          make(map[string]context.Context),
		}
		m.m[key] = jm
	}
	return jm
}

func (m *JsonrpcMetric) Remove(expire time.Duration) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	for _, v := range m.m {
		v.Remove(expire)
	}
}

func (m *JsonrpcMetric) OnHandle(ctx context.Context, method string, ts time.Time, err error) {
	var ms *stats.Int64Measure
	var ok bool
	if err != nil {
		ms, ok = msFailure, true
		ctx = GetMetricContext(ctx, &mkMethod, method)
	}
	ms, ok = msExecutes[method]
	if !ok {
		ms, ok = msWaits[method]
	}
	if !ok && method == "icx_sendTransaction" {
		ms, ok = msSendTx, true
	}
	if !ok {
		ms, ok = msRetrieve, true
		ctx = GetMetricContext(ctx, &mkMethod, method)
	}
	jm := m.EnsureMeasure(ctx, ms)
	jm.RemoveAndRecord(ctx, ts, m.expire)
}

func NewJsonrpcMetric() *JsonrpcMetric {
	jmsMtx.Lock()
	defer jmsMtx.Unlock()
	jm := &JsonrpcMetric{
		expire: DefaultJsonrpcDurationsUpdateExpire,
		m:      make(map[string]*JsonrpcMeasure),
	}
	jms = append(jms, jm)
	return jm
}

type Durations struct {
	buf   []*Duration
	head  int
	next  int
	size  int
	count int
	sum   time.Duration
	avg   time.Duration
	mtx   sync.RWMutex
}

type Duration struct {
	t time.Time
	d time.Duration
}

func (ds *Durations) Add(ts time.Time) *Duration {
	now := time.Now()
	d := &Duration{t: now, d: now.Sub(ts)}

	ds.mtx.Lock()
	defer ds.mtx.Unlock()

	next := ds.next + 1
	if next == ds.size {
		next = 0
	}
	if next == ds.head {
		ds._pop()
	}
	ds.buf[ds.next] = d
	ds.count += 1
	ds.sum += d.d
	ds.next = next
	ds._avg()
	return d
}

func (ds *Durations) _pop() {
	d := ds.buf[ds.head]
	ds.sum -= d.d
	ds.count -= 1
	ds.buf[ds.head] = nil
	ds.head++
	if ds.head >= ds.size {
		ds.head = 0
	}
}

func (ds *Durations) _avg() {
	if ds.count == 0 || ds.sum == 0 {
		ds.avg = 0
	} else {
		ds.avg = ds.sum / time.Duration(ds.count)
	}
}

func (ds *Durations) Remove(expire time.Duration) {
	ds.mtx.Lock()
	defer ds.mtx.Unlock()
	now := time.Now()
	for {
		d := ds.buf[ds.head]
		if d == nil || ds.head == ds.next || now.Sub(d.t) < expire {
			break
		}
		ds._pop()
	}
	ds._avg()
}

func (ds *Durations) Size() int {
	return ds.size - 1
}

func (ds *Durations) Count() int {
	ds.mtx.RLock()
	defer ds.mtx.RUnlock()
	return ds.count
}

func (ds *Durations) Sum() int64 {
	ds.mtx.RLock()
	defer ds.mtx.RUnlock()
	return int64(ds.sum)
}

func (ds *Durations) Avg() time.Duration {
	ds.mtx.RLock()
	defer ds.mtx.RUnlock()
	return ds.avg
}

func NewDurations(size int) *Durations {
	if size < 1 {
		panic("size must be positive number")
	}
	size += 1
	ds := &Durations{
		buf:  make([]*Duration, size),
		size: size,
	}
	return ds
}
