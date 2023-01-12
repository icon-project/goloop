package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type txInfo struct {
	id []byte
	ts int64 // unit: us
}

func intToID(v int) []byte {
	size := 32
	id := make([]byte, size)
	for i := 0; v > 0; i++ {
		id[i] = byte(v & 0xff)
		v >>= 1
	}
	return id
}

func newDummyTXs(n int, tsStart int64, tsInterval time.Duration) []*txInfo {
	txs := make([]*txInfo, n)
	interval := tsInterval.Microseconds()

	for i := 0; i < n; i++ {
		id := intToID(i)
		ts := tsStart + int64(i)*interval
		txs[i] = &txInfo{id, ts}
	}
	return txs
}

func TestTxIDCache(t *testing.T) {
	slotDuration := time.Second
	cache := NewTxIDCache(slotDuration, 100, nil)
	size := 20
	txs := newDummyTXs(size, 0, 500*time.Millisecond)

	// Initial Add
	for _, tx := range txs {
		assert.False(t, cache.Contains(tx.id, tx.ts))
		cache.Add(tx.id, tx.ts)
		assert.True(t, cache.Contains(tx.id, tx.ts))
	}
	assert.Equal(t, size, cache.Len())

	// Add duplicate transactions
	for _, tx := range txs {
		cache.Add(tx.id, tx.ts)
		assert.Equal(t, size, cache.Len())
	}

	expSize := size
	for i := 0; i <= 10; i++ {
		ts := (time.Duration(i) * time.Second).Microseconds()
		cache.RemoveOldTXsByTS(ts)
		assert.Equal(t, expSize, cache.Len())
		expSize -= 2
	}
}

func TestTxIDCache_AddWithFlush(t *testing.T) {
	slotDuration := time.Second
	slotSize := 10
	cache := NewTxIDCache(slotDuration, slotSize, nil)
	size := 12
	txs := newDummyTXs(size, 0, time.Microsecond)

	for i := 0; i < size; i++ {
		tx := txs[i]
		cache.Add(tx.id, tx.ts)
		assert.Equal(t, (i%slotSize)+1, cache.Len())
	}

	for i := 0; i < size; i++ {
		tx := txs[i]
		assert.Equal(t, i >= slotSize, cache.Contains(tx.id, tx.ts))
	}

	assert.Equal(t, 2, cache.Len())
}

func TestTxIDCache_verifyTx(t *testing.T) {
	slotDuration := time.Second
	slotSize := 10000
	cache := NewTxIDCache(slotDuration, slotSize, nil)
	txs := newDummyTXs(100, 0, time.Second)

	cache.Add(txs[0].id, -1)
	assert.Zero(t, cache.Len())
	assert.False(t, cache.Contains(txs[0].id, -1234))

	for _, tx := range txs {
		cache.Add(tx.id, tx.ts)
	}
	assert.Equal(t, len(txs), cache.Len())

	cache.RemoveOldTXsByTS((100 * time.Second).Microseconds())
	assert.Zero(t, cache.Len())

	for _, tx := range txs {
		assert.False(t, cache.Contains(tx.id, tx.ts))
		cache.Add(tx.id, tx.ts)
		assert.False(t, cache.Contains(tx.id, tx.ts))
		assert.Zero(t, cache.Len())
	}
}
