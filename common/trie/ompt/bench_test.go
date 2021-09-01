package ompt

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/db"
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
