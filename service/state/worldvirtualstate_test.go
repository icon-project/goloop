package state

import (
	"encoding/binary"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/icon-project/goloop/common/db"
)

func Test_NewWorldVirtualState(t *testing.T) {
	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil, nil, nil)
	v1 := big.NewInt(1000)
	v2 := big.NewInt(2000)

	lq1 := []LockRequest{
		{"", AccountWriteLock},
	}

	acid := []byte("test")

	wvs1 := NewWorldVirtualState(ws, lq1)
	go func() {
		log.Println("TX1 begin")
		time.Sleep(time.Second)
		log.Println("TX1 getAccount() before")
		as1 := wvs1.GetAccountState(acid)
		log.Println("TX1 getAccount() after")
		balance1 := as1.GetBalance()
		if balance1.Sign() != 0 {
			t.Errorf("TX1 balance isn't empty ret=%s", balance1.String())
		}
		as1.SetBalance(v1)
		log.Println("TX1 before commit")
		wvs1.Commit()
		log.Println("TX1 after commit")
	}()

	wvs2 := wvs1.GetFuture(lq1)
	go func() {
		log.Println("TX2 begin")
		time.Sleep(time.Second)
		log.Println("TX2 getAccount() before")
		as1 := wvs2.GetAccountState(acid)
		log.Println("TX2 getAccount() after")
		balance1 := as1.GetBalance()
		if balance1.Cmp(v1) != 0 {
			t.Errorf("TX2 balance isn't same exp=%s ret=%s",
				v1.String(), balance1.String())
		}
		as1.SetBalance(v2)
		log.Println("TX2 before commit")
		wvs2.Commit()
		log.Println("TX2 after commit")
	}()

	wvs2.Realize()
	wvss1 := wvs2.GetSnapshot()

	ass1 := wvss1.GetAccountSnapshot(acid)
	balance1 := ass1.GetBalance()
	if balance1.Cmp(v2) != 0 {
		t.Errorf("Resulting balance isn't same exp=%s ret=%s",
			v2.String(), balance1.String())
	}
}

func TestParallelExecution(t *testing.T) {
	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil, nil, nil)
	wvs := NewWorldVirtualState(ws, nil)

	execute := func(wvs WorldVirtualState, idx int, balance int64) WorldVirtualState {
		v1 := big.NewInt(balance)
		id := v1.Bytes()

		req := []LockRequest{{string(id), AccountWriteLock}}
		nwvs := wvs.GetFuture(req)
		go func(wvs WorldVirtualState, idx int, id []byte, v *big.Int) {
			as := wvs.GetAccountState(id)
			as.SetBalance(v)
			wvs.Commit()
		}(nwvs, idx, id, v1)
		return nwvs
	}

	count := 1000
	for idx := 1; idx <= count; idx++ {
		wvs = execute(wvs, idx, int64(idx*10))
	}
	wvs.Realize()

	wvss := wvs.GetSnapshot()
	if err := wvss.Flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
	}

	ws2 := NewWorldState(database, wvss.StateHash(), wvss.GetValidatorSnapshot(), wvss.GetExtensionSnapshot(), wvss.BTPData())
	for idx := 1; idx <= count; idx++ {
		v1 := big.NewInt(int64(idx * 10))
		id := v1.Bytes()
		ass := ws2.GetAccountSnapshot(id)
		if ass == nil {
			t.Errorf("Fail to get account idx=%d", idx)
			continue
		}
		balance := ass.GetBalance()
		if balance.Cmp(v1) != 0 {
			t.Errorf("Balance is different idx=%d exp=%s ret=%s",
				idx, v1.String(), balance.String())
		}
	}
}

func intToBytes(v uint32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	return buf[:]
}

const (
	NumberOfAccounts = 1000
)

func wvexecuteTransfer(ws WorldState, id1, id2 []byte, value *big.Int) {
	ws.GetSnapshot()
	as1 := ws.GetAccountState(id1)
	as2 := ws.GetAccountState(id2)
	balance1 := as1.GetBalance()
	balance2 := as2.GetBalance()
	if balance1.Cmp(value) >= 0 {
		as1.SetBalance(new(big.Int).Sub(balance1, value))
		as2.SetBalance(new(big.Int).Add(balance2, value))
	}
}

func executeTransferInVirtual(wvs WorldVirtualState, id1, id2 []byte, value *big.Int) WorldVirtualState {
	reqs := []LockRequest{
		{string(id1), AccountWriteLock},
		{string(id2), AccountWriteLock},
	}
	wvs = wvs.GetFuture(reqs)
	go func(wvs WorldVirtualState, id1, id2 []byte, value *big.Int) {
		wvexecuteTransfer(wvs, id1, id2, value)
		wvs.Commit()
	}(wvs, id1, id2, value)
	return wvs
}

func TestIndependentTrasferInSequential(t *testing.T) {
	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil, nil, nil)
	startBalance := big.NewInt(1000)
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		as := ws.GetAccountState(intToBytes(i))
		as.SetBalance(startBalance)
	}
	ws.GetSnapshot()

	transfer := big.NewInt(10)
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		wvexecuteTransfer(ws, intToBytes(i), intToBytes(i+1), transfer)
	}
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		wvexecuteTransfer(ws, intToBytes(i+1), intToBytes(i), transfer)
	}

	wss := ws.GetSnapshot()
	log.Printf("Resuling Hash:[%x]", wss.StateHash())
}

func TestIndependentTrasferInPanrallel(t *testing.T) {
	database := db.NewMapDB()
	ws := NewWorldState(database, nil, nil, nil, nil)
	startBalance := big.NewInt(1000)
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		as := ws.GetAccountState(intToBytes(i))
		as.SetBalance(startBalance)
	}
	ws.GetSnapshot()

	wvs := NewWorldVirtualState(ws, nil)

	transfer := big.NewInt(10)
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		wvs = executeTransferInVirtual(wvs, intToBytes(i), intToBytes(i+1), transfer)
	}
	for i := uint32(0); i < NumberOfAccounts; i += 2 {
		wvs = executeTransferInVirtual(wvs, intToBytes(i+1), intToBytes(i), transfer)
	}
	wvs.Realize()
	wvss := wvs.GetSnapshot()
	log.Printf("Resuling Hash:[%x]", wvss.StateHash())
}

func TestSequentialExecutionChainedAccount(t *testing.T) {
	database := db.NewMapDB()

	ws := NewWorldState(database, nil, nil, nil, nil)
	as := ws.GetAccountState(intToBytes(0))
	as.SetBalance(big.NewInt(100))

	wvs := NewWorldVirtualState(ws, nil)

	execute := func(wvs WorldVirtualState, idx int, balance int64) WorldVirtualState {
		v1 := big.NewInt(balance)
		id1 := intToBytes(uint32(idx))
		id2 := intToBytes(uint32(idx + 1))

		req := []LockRequest{
			{string(id1), AccountWriteLock},
			{string(id2), AccountWriteLock},
		}
		nwvs := wvs.GetFuture(req)
		go func(wvs WorldVirtualState, idx int, id1, id2 []byte, v *big.Int) {
			as1 := wvs.GetAccountState(id1)
			as2 := wvs.GetAccountState(id2)
			balance1 := as1.GetBalance()
			balance2 := as2.GetBalance()
			as1.SetBalance(new(big.Int).Sub(balance1, v))
			as2.SetBalance(new(big.Int).Add(balance2, v))

			wvs.Commit()
		}(nwvs, idx, id1, id2, v1)
		return nwvs
	}

	count := 1000
	for idx := 0; idx < count; idx++ {
		wvs = execute(wvs, idx, 100)
	}
	wvs.Realize()

	wvss := wvs.GetSnapshot()
	if err := wvss.Flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
	}

	v1 := big.NewInt(0)
	v2 := big.NewInt(100)
	ws2 := NewWorldState(database, wvss.StateHash(), wvss.GetValidatorSnapshot(), wvss.GetExtensionSnapshot(), wvss.BTPData())
	for idx := 1; idx < count; idx++ {
		ass := ws2.GetAccountSnapshot(intToBytes(uint32(idx)))
		if ass == nil {
			t.Errorf("Fail to get account idx=%d", idx)
			continue
		}
		balance := ass.GetBalance()
		if balance.Cmp(v1) != 0 {
			t.Errorf("Balance is different idx=%d exp=%s ret=%s",
				idx, v1.String(), balance.String())
		}
	}

	ass := ws2.GetAccountSnapshot(intToBytes(uint32(count)))
	balance2 := ass.GetBalance()
	if balance2.Cmp(v2) != 0 {
		t.Errorf("Final balance is different exp=%s ret=%s",
			v2.String(), balance2.String())
	}
}

func TestSequentialExecutionDistributeWithRollbacks(t *testing.T) {
	database := db.NewMapDB()

	ws := NewWorldState(database, nil, nil, nil, nil)
	as := ws.GetAccountState(intToBytes(0))
	as.SetBalance(big.NewInt(1000))

	wvs := NewWorldVirtualState(ws, nil)

	execute := func(wvs WorldVirtualState, idx int, from, to uint32, balance int64) WorldVirtualState {
		v1 := big.NewInt(balance)
		id1 := intToBytes(from)
		id2 := intToBytes(to)

		req := []LockRequest{
			{string(id1), AccountWriteLock},
			{string(id2), AccountWriteLock},
		}
		nwvs := wvs.GetFuture(req)
		go func(wvs WorldVirtualState, idx int, id1, id2 []byte, v *big.Int) {
			wvss := wvs.GetSnapshot()

			as1 := wvs.GetAccountState(id1)
			as2 := wvs.GetAccountState(id2)
			balance1 := as1.GetBalance()
			balance2 := as2.GetBalance()
			as1.SetBalance(new(big.Int).Sub(balance1, v))
			as2.SetBalance(new(big.Int).Add(balance2, v))

			if (idx % 2) == 1 {
				if err := wvs.Reset(wvss); err != nil {
					t.Errorf("Fail to reset snapshot err=%+v", err)
				}
			}
			wvs.Commit()
		}(nwvs, idx, id1, id2, v1)
		return nwvs
	}

	count := 1000
	for idx := 0; idx < count; idx++ {
		wvs = execute(wvs, idx, 0, uint32(idx+1), 1)
	}
	wvs.Realize()

	wvss := wvs.GetSnapshot()
	if err := wvss.Flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
	}

	v1 := big.NewInt(1)
	v2 := big.NewInt(0)
	remain := big.NewInt(int64(count / 2))
	ws2 := NewWorldState(database, wvss.StateHash(), wvss.GetValidatorSnapshot(), wvss.GetExtensionSnapshot(), wvss.BTPData())
	for idx := 1; idx <= count; idx++ {
		ass := ws2.GetAccountSnapshot(intToBytes(uint32(idx)))
		if ass == nil {
			t.Errorf("Fail to get account idx=%d", idx)
			continue
		}
		balance := ass.GetBalance()

		exp := v1
		if (idx % 2) == 0 {
			exp = v2
		}
		if balance.Cmp(exp) != 0 {
			t.Errorf("Balance is different idx=%d exp=%s ret=%s",
				idx, exp.String(), balance.String())
		}
	}

	ass := ws2.GetAccountSnapshot(intToBytes(uint32(0)))
	balance2 := ass.GetBalance()
	if balance2.Cmp(remain) != 0 {
		t.Errorf("Final balance is different exp=%s ret=%s",
			v2.String(), balance2.String())
	}
}
