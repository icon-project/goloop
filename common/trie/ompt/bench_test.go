package ompt

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/db"
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

func BenchmarkTrend(b *testing.B) {
	b.StopTimer()
	dbType := "goleveldb"
	dbPath := ".db"

	d, err := db.Open(dbPath, dbType, b.Name()+string(b.N))
	if err != nil {
		b.Errorf("Fail to open DB err=%+v", err)
		b.FailNow()
	}

	mpt := NewMPTForBytes(d, nil)
	mpt.cache = NewNodeCache(5)

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
			ss.ClearCache()
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
