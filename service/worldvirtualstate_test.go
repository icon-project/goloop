package service

import (
	"github.com/icon-project/goloop/common/db"
	"log"
	"math/big"
	"testing"
	"time"
)

func Test_newWorldVirtualState(t *testing.T) {
	database := db.NewMapDB()
	ws := newWorldState(database, nil)
	v1 := big.NewInt(1000)
	v2 := big.NewInt(2000)

	lq1 := []lockRequest{
		{"", accountWriteLock},
	}

	acid := []byte("test")

	wvs1 := newWorldVirtualState(ws, lq1)
	go func() {
		log.Println("TX1 begin")
		time.Sleep(time.Second)
		log.Println("TX1 getAccount() before")
		as1 := wvs1.getAccountState(acid)
		log.Println("TX1 getAccount() after")
		balance1 := as1.getBalance()
		if balance1.BitLen() != 0 {
			t.Errorf("TX1 balance isn't empty ret=%s", balance1.String())
		}
		as1.setBalance(v1)
		log.Println("TX1 before commit")
		wvs1.commit()
		log.Println("TX1 after commit")
	}()

	wvs2 := wvs1.getFuture(lq1)
	go func() {
		log.Println("TX2 begin")
		time.Sleep(time.Second)
		log.Println("TX2 getAccount() before")
		as1 := wvs2.getAccountState(acid)
		log.Println("TX2 getAccount() after")
		balance1 := as1.getBalance()
		if balance1.Cmp(v1) != 0 {
			t.Errorf("TX2 balance isn't same exp=%s ret=%s",
				v1.String(), balance1.String())
		}
		as1.setBalance(v2)
		log.Println("TX2 before commit")
		wvs2.commit()
		log.Println("TX2 after commit")
	}()

	wvs2.realize()
	wvss1 := wvs2.getSnapshot()

	ass1 := wvss1.getAccountSnapshot(acid)
	balance1 := ass1.getBalance()
	if balance1.Cmp(v2) != 0 {
		t.Errorf("Resulting balance isn't same exp=%s ret=%s",
			v2.String(), balance1.String())
	}
}

func TestParallelExecution(t *testing.T) {
	database := db.NewMapDB()
	ws := newWorldState(database, nil)
	wvs := newWorldVirtualState(ws, nil)

	execute := func(wvs *worldVirtualState, idx int, balance int64) *worldVirtualState {
		v1 := big.NewInt(balance)
		id := v1.Bytes()

		req := []lockRequest{{string(id), accountWriteLock}}
		nwvs := wvs.getFuture(req)
		go func(wvs *worldVirtualState, idx int, id []byte, v *big.Int) {
			log.Printf("TX[%d] BEGIN\n", idx)
			as := wvs.getAccountState(id)
			as.setBalance(v)
			wvs.commit()
			log.Printf("TX[%d] END\n", idx)
		}(nwvs, idx, id, v1)
		return nwvs
	}

	count := 5
	for idx := 1; idx <= count; idx++ {
		wvs = execute(wvs, idx, int64(idx*10))
	}
	log.Println("Main realize before")
	wvs.realize()
	log.Printf("Main realize after commited=%p", wvs.committed)
	wvss := wvs.getSnapshot()
	log.Printf("Hash:%x", wvss.stateHash())
	if err := wvss.flush(); err != nil {
		t.Errorf("Fail to flush err=%+v", err)
	}

	ws2 := newWorldState(database, wvss.stateHash())
	for idx := 1; idx <= count; idx++ {
		v1 := big.NewInt(int64(idx * 10))
		id := v1.Bytes()
		ass := ws2.getAccountSnapshot(id)
		if ass == nil {
			t.Errorf("Fail to get account idx=%d", idx)
			continue
		}
		balance := ass.getBalance()
		if balance.Cmp(v1) != 0 {
			t.Errorf("Balance is different idx=%d exp=%s ret=%s",
				idx, v1.String(), balance.String())
		}
	}
}
