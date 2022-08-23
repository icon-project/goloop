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
	mkMethod  = NewMetricKey("method")
	msFailure = &measure{
		ms:    stats.Int64("jsonrpc_failure", "jsonrpc failures", "ns"),
		msAvg: stats.Int64("jsonrpc_failure_avg", "moving average of jsonrpc failures", "ns"),
		mks:   []tag.Key{mkMethod},
	}
	msRetrieve = &measure{
		ms:    stats.Int64("jsonrpc_retrieve", "jsonrpc retrieve methods", "ns"),
		msAvg: stats.Int64("jsonrpc_retrieve_avg", "moving average of jsonrpc retrieve methods", "ns"),
		mks:   []tag.Key{mkMethod},
	}
	emptyMks = []tag.Key{}
	msMap    = map[string]*measure{
		"icx_getLastBlock":     msRetrieve,
		"icx_getBlockByHeight": msRetrieve,
		"icx_getBlockByHash":   msRetrieve,
		"icx_call": {
			stats.Int64("jsonrpc_call", "jsonrpc icx_call method", "ns"),
			stats.Int64("jsonrpc_call_avg", "moving average of jsonrpc icx_call method", "ns"),
			emptyMks,
		},
		"icx_getBalance":           msRetrieve,
		"icx_getScoreApi":          msRetrieve,
		"icx_getTotalSupply":       msRetrieve,
		"icx_getTransactionResult": msRetrieve,
		"icx_getTransactionByHash": msRetrieve,
		"icx_sendTransaction": {
			stats.Int64("jsonrpc_send_transaction", "jsonrpc icx_sendTransaction method", "ns"),
			stats.Int64("jsonrpc_send_transaction_avg", "moving average of jsonrpc icx_sendTransaction methods", "ns"),
			emptyMks,
		},
		"icx_sendTransactionAndWait": {
			stats.Int64("jsonrpc_send_transaction_and_wait", "jsonrpc icx_sendTransactionAndWait method", "ns"),
			stats.Int64("jsonrpc_send_transaction_and_wait_avg", "moving average of jsonrpc icx_sendTransactionAndWait method", "ns"),
			emptyMks,
		},
		"icx_waitTransactionResult": {
			stats.Int64("jsonrpc_wait_transaction_result", "jsonrpc icx_waitTransactionResult method", "ns"),
			stats.Int64("jsonrpc_wait_transaction_result_avg", "moving average of jsonrpc icx_waitTransactionResult method", "ns"),
			emptyMks,
		},
		"icx_getDataByHash":          msRetrieve,
		"icx_getBlockHeaderByHeight": msRetrieve,
		"icx_getVotesByHeight":       msRetrieve,
		"icx_getProofForResult":      msRetrieve,
		"icx_getProofForEvents":      msRetrieve,
		"icx_getScoreStatus":         msRetrieve,
		"btp_getNetworkInfo":         msRetrieve,
		"btp_getNetworkTypeInfo":     msRetrieve,
		"btp_getMessages":            msRetrieve,
		"btp_getHeader":              msRetrieve,
		"btp_getProof":               msRetrieve,
		"btp_getSourceInformation":   msRetrieve,
		"debug_getTrace": {
			stats.Int64("jsonrpc_get_trace", "jsonrpc debug_getTrace method", "ns"),
			stats.Int64("jsonrpc_get_trace_avg", "moving average of jsonrpc debug_getTrace method", "ns"),
			emptyMks,
		},
		"debug_estimateStep": {
			stats.Int64("jsonrpc_estimate_step", "jsonrpc debug_estimateStep method", "ns"),
			stats.Int64("jsonrpc_estimate_step_avg", "moving average of jsonrpc debug_estimateStep method", "ns"),
			emptyMks,
		},
	}
	jms    = make([]*JsonrpcMetric, 0)
	jmsMtx sync.RWMutex
)

type measure struct {
	ms    *stats.Int64Measure
	msAvg *stats.Int64Measure
	mks   []tag.Key
}

const (
	DefaultJsonrpcDurationsSize   = 20000
	DefaultJsonrpcDurationsExpire = 10 * time.Second
)

func RegisterJsonrpc() {
	RegisterMetricView(msFailure.ms, view.Count(), msFailure.mks)
	RegisterMetricView(msFailure.msAvg, view.LastValue(), emptyMks)
	RegisterMetricView(msRetrieve.ms, view.Count(), msRetrieve.mks)
	RegisterMetricView(msRetrieve.msAvg, view.LastValue(), emptyMks)
	for _, v := range msMap {
		if v != msRetrieve {
			RegisterMetricView(v.ms, view.Count(), v.mks)
			RegisterMetricView(v.msAvg, view.LastValue(), emptyMks)
		}
	}

	RegisterBeforeExportFunc(func() {
		jmsMtx.RLock()
		defer jmsMtx.RUnlock()
		for _, jm := range jms {
			jm.Remove()
		}
	})
}

type JsonrpcMeasure struct {
	*measure
	d   *Durations
	ctx context.Context
	mtx sync.RWMutex
}

func (m *JsonrpcMeasure) RemoveAndRecord(ctx context.Context, ts time.Time, expire time.Duration) {
	d := m.d.Add(ts)
	stats.Record(ctx, m.ms.M(int64(d.d)))
	m.d.Remove(expire)
	stats.Record(ctx, m.msAvg.M(int64(m.d.Avg())))
}

func (m *JsonrpcMeasure) Remove(expire time.Duration) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	m.d.Remove(expire)
	stats.Record(m.ctx, m.msAvg.M(int64(m.d.Avg())))
}

type JsonrpcMetric struct {
	expire        time.Duration
	durationsSize int
	useDefault    bool
	m             map[string]*JsonrpcMeasure
	mtx           sync.RWMutex
}

func (m *JsonrpcMetric) EnsureMeasure(ctx context.Context, ms *measure) *JsonrpcMeasure {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	key := ms.ms.Name()
	if chainId, ok := tag.FromContext(ctx).Value(MetricKeyChain); ok {
		key += chainId
	}
	jm, ok := m.m[key]
	if !ok {
		jm = &JsonrpcMeasure{
			measure: ms,
			d:       NewDurations(m.durationsSize),
			ctx:     ctx,
		}
		m.m[key] = jm
	}
	return jm
}

func (m *JsonrpcMetric) Remove() {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	for _, v := range m.m {
		v.Remove(m.expire)
	}
}

func (m *JsonrpcMetric) OnHandle(ctx context.Context, method string, ts time.Time, err error) {
	var ms *measure
	if err == nil {
		ok := false
		if ms, ok = msMap[method]; !ok {
			if m.useDefault {
				ms = msRetrieve
			} else {
				panic("not exists measure " + method)
			}
		}

		if ms == msRetrieve {
			ctx = GetMetricContext(ctx, &mkMethod, method)
		}
	} else {
		ms = msFailure
		ctx = GetMetricContext(ctx, &mkMethod, method)
	}

	jm := m.EnsureMeasure(ctx, ms)
	jm.RemoveAndRecord(ctx, ts, m.expire)
}

func NewJsonrpcMetric(expire time.Duration, durationsSize int, useDefault bool) *JsonrpcMetric {
	jmsMtx.Lock()
	defer jmsMtx.Unlock()
	jm := &JsonrpcMetric{
		expire:        expire,
		durationsSize: durationsSize,
		useDefault:    useDefault,
		m:             make(map[string]*JsonrpcMeasure),
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

	if ds.count == 0 || ds.sum == 0 {
		return 0
	} else {
		return ds.sum / time.Duration(ds.count)
	}
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
