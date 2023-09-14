package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/txlocator"
	"github.com/icon-project/goloop/module"
)

type mockMonitor struct {
}

func (m *mockMonitor) OnDropTx(n int, user bool) {
	// do nothing
}

func (m *mockMonitor) OnAddTx(n int, user bool) {
	// do nothing
}

func (m *mockMonitor) OnRemoveTx(n int, user bool) {
	// do nothing
}

func (m *mockMonitor) OnCommit(id []byte, ts time.Time, d time.Duration) {
	// do nothing
}

func TestTransactionPool_Add(t *testing.T) {
	dbase := db.NewMapDB()
	tsc := NewTimestampChecker()
	logger := log.New()
	lm, err := txlocator.NewManager(dbase, logger)
	assert.NoError(t, err)
	tim, _ := NewTXIDManager(lm, tsc, nil)
	pool := NewTransactionPool(module.TransactionGroupNormal, 5000, tim, &mockMonitor{}, logger)

	addr := common.MustNewAddressFromString("hx1111111111111111111111111111111111111111")
	tx1 := newMockTransaction([]byte("tx1"), addr, 1)
	tx1.NID = 1

	if err := pool.Add(tx1, true); err != nil {
		t.Error("Fail to add transaction with valid network ID")
	}
}
