package service

import (
	"testing"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
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
	bk, _ := dbase.GetBucket(db.TransactionLocatorByHash)
	pool := NewTransactionPool(module.TransactionGroupNormal, 5000, bk, &mockMonitor{}, log.New())

	addr := common.NewAddressFromString("hx1111111111111111111111111111111111111111")
	tx1 := newMockTransaction([]byte("tx1"), addr, 1)
	tx1.NID = 1

	if err := pool.Add(tx1, true); err != nil {
		t.Error("Fail to add transaction with valid network ID")
	}
}
