package ompt

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/trie/cache"
)

var definedValue = []byte("abcdefghijklmnopqrstuvwxyz")

func makeKeyValue(i int) (key, value []byte) {
	idx := []byte(fmt.Sprint(i))
	hash := sha3.Sum256(idx)
	key = make([]byte, hashSize)
	copy(key, hash[:])
	value = make([]byte, len(definedValue)+len(idx))
	copy(value, definedValue)
	copy(value[len(definedValue):], idx)
	return
}

func BenchmarkTrendNoCache(b *testing.B) { benchmarkTrend(0, 0, b) }
func BenchmarkTrend5(b *testing.B)       { benchmarkTrend(5, 0, b) }
func BenchmarkTrend6(b *testing.B)       { benchmarkTrend(6, 0, b) }
func BenchmarkTrend51(b *testing.B)      { benchmarkTrend(5, 1, b) }

func benchmarkTrend(depth, fdepth int, b *testing.B) {
	b.StopTimer()
	dbType := "goleveldb"
	dbPath := ".db"

	d, err := db.Open(dbPath, dbType, b.Name()+fmt.Sprint(b.N))
	if err != nil {
		b.Errorf("Fail to open DB err=%+v", err)
		b.FailNow()
	}

	mpt := NewMPTForBytes(d, nil)
	if depth > 0 || fdepth > 0 {
		mpt.cache = cache.NewNodeCache(depth, fdepth, ".cache."+b.Name())
	}

	blockUnit := 1000
	txUnit := 10

	var ms runtime.MemStats
	begin := time.Now()
	for i := 1; i <= b.N; i++ {
		// tx start
		if i%txUnit == 1 {
			mpt.GetSnapshot()
		}

		key, value := makeKeyValue(i)
		mpt.Set(key, value)

		// on block end
		if i%blockUnit == 0 {
			ss := mpt.GetSnapshot()
			mpt.ClearCache()
			ss.Flush()
			end := time.Now()
			dur := end.Sub(begin)
			ops := time.Duration(blockUnit) * time.Second / dur
			runtime.ReadMemStats(&ms)
			fmt.Printf("%d,%d,%d,%d\n",
				i, ops, ms.HeapInuse, ms.HeapIdle,
			)

			begin = end
		}
	}
}

func BenchmarkMPTTraverseDuringFlush(b *testing.B) {
	const COUNT = 100000
	b.StopTimer()
	dbase, err := db.Open(b.TempDir(), "rocksdb", b.Name())
	if err != nil {
		b.Errorf("fail to open db err=%+v", err)
		b.FailNow()
	}

	makeKeyValue := func(i int) (k, v []byte) {
		k, _ = rlpEncode(intconv.Int64ToBytes(int64(i)));
		hash := sha3.Sum256(k)
		v = make([]byte, hashSize)
		copy(v, hash[:])
		return
	}

	state := NewMPTForBytes(dbase, nil)
	keys := make([][]byte, COUNT)
	values := make([][]byte, COUNT)
	for i:= 0 ; i<COUNT ; i++ {
		k, v := makeKeyValue(i)
		_, err := state.Set(k, v)
		assert.NoError(b, err)
		keys[i] = k;
		values[i] = v;
	}
	ss := state.GetSnapshot()


	ch := make(chan error, 1)
	go func() {
		ch <- ss.Flush()
	}()

	b.StartTimer()

	for r := 0 ; r < b.N ; r++ {
		idx := 0
		for itr := ss.Iterator() ; itr.Has() ; itr.Next() {
			value, key, err := itr.Get()
			assert.NoError(b, err)
			assert.Equal(b, keys[idx], key)
			assert.Equal(b, values[idx], value)
			idx += 1
		}
	}

	b.StopTimer()

	assert.NoError(b, <-ch)


	ss = NewMPTForBytes(dbase, ss.Hash())
	for idx:= 0 ; idx<COUNT ; idx++ {
		value, err := ss.Get(keys[idx])
		assert.NoError(b, err)
		assert.Equal(b, values[idx], value)
	}
}

func BenchmarkMPTConcurrentReadDuringFlush(b *testing.B) {
	const COUNT = 100000
	b.StopTimer()
	dbase, err := db.Open(b.TempDir(), "rocksdb", b.Name())
	if err != nil {
		b.Errorf("fail to open db err=%+v", err)
		b.FailNow()
	}

	state := NewMPTForBytes(dbase, nil)
	keys := make([][]byte, COUNT)
	values := make([][]byte, COUNT)
	for i:= 0 ; i<COUNT ; i++ {
		k, v := makeKeyValue(i)
		_, err := state.Set(k, v)
		assert.NoError(b, err)
		keys[i] = k;
		values[i] = v;
	}
	ss := state.GetSnapshot()

	b.StartTimer()

	ch := make(chan error, 1)
	go func() {
		ch <- ss.Flush()
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := rand.Intn(COUNT)
			value, err := ss.Get(keys[idx])
			assert.NoError(b, err)
			assert.Equal(b, values[idx], value)
		}
	})

	b.StopTimer()

	assert.NoError(b, <-ch)

	ss = NewMPTForBytes(dbase, ss.Hash())
	for idx:= 0 ; idx<COUNT ; idx++ {
		value, err := ss.Get(keys[idx])
		assert.NoError(b, err)
		assert.Equal(b, values[idx], value)
	}
}

func BenchmarkMPTConcurrentRead(b *testing.B) {
	const COUNT = 100000
	b.StopTimer()
	dbase, err := db.Open(b.TempDir(), "rocksdb", b.Name())
	if err != nil {
		b.Errorf("fail to open db err=%+v", err)
		b.FailNow()
	}
	state := NewMPTForBytes(dbase, nil)
	keys := make([][]byte, COUNT)
	values := make([][]byte, COUNT)
	for i:= 0 ; i<COUNT ; i++ {
		k, v := makeKeyValue(i)
		_, err := state.Set(k, v)
		assert.NoError(b, err)
		keys[i] = k;
		values[i] = v;
	}
	ss := state.GetSnapshot()

	err = ss.Flush()
	assert.NoError(b, err)
	ss.ClearCache()

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := rand.Intn(COUNT)
			value, err := ss.Get(keys[idx])
			assert.NoError(b, err)
			assert.Equal(b, values[idx], value)
		}
	})
}