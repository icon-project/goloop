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
	tsc := NewTimestampChecker()
	tim, _ := NewTXIDManager(dbase, tsc, nil)
	pool := NewTransactionPool(module.TransactionGroupNormal, 5000, tim, &mockMonitor{}, log.New())

	addr := common.MustNewAddressFromString("hx1111111111111111111111111111111111111111")
	tx1 := newMockTransaction([]byte("tx1"), addr, 1)
	tx1.NID = 1

	if err := pool.Add(tx1, true); err != nil {
		t.Error("Fail to add transaction with valid network ID")
	}
}
