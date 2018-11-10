package service

import (
	"encoding/binary"
	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"log"
	"math/big"
	"testing"
	"time"
)

func Test_newWorldVirtualState(t *testing.T) {
	database := db.NewMapDB()
	ws := NewWorldState(database, nil)
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
		if balance1.BitLen() != 0 {
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
	ws := NewWorldState(database, nil)
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

	ws2 := NewWorldState(database, wvss.StateHash())
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

type LoopChainDB struct {
	blockbk, scorebk db.Bucket
}

func (lc *LoopChainDB) getBlockByHeight(height int) ([]byte, error) {
	prefix := "block_height_key"
	key := make([]byte, len(prefix)+12)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix)+4:], uint64(height))
	bid, err := lc.blockbk.Get(key)
	if err != nil || bid == nil {
		return bid, err
	}

	return lc.blockbk.Get(bid)
}

func Test_LoopChainTransactionExecution(t *testing.T) {
	var (
		directory = "data/testnet"
		blockname = "block"
		scorename = "score"
	)

	blockdb, err := db.NewGoLevelDB(blockname, directory)
	if err != nil {
		log.Panicf("Fail to open database err=%+v", err)
	}

	blockbk, err := blockdb.GetBucket("")
	if err != nil {
		log.Panicf("Fail to get bucket err=%+v", err)
	}

	scoredb, err := db.NewGoLevelDB(scorename, directory)
	if err != nil {
		log.Panicf("Fail to open database err=%+v", err)
	}

	scorebk, err := scoredb.GetBucket("")
	if err != nil {
		log.Panicf("Fail to get bucket err=%+v", err)
	}

	lc := &LoopChainDB{blockbk, scorebk}

	c2db, err := db.NewGoLevelDB("goloop", directory)
	if err != nil {
		log.Panicf("Fail to make database err=%+v", err)
	}

	ws := NewWorldState(c2db, nil)
	for i := 1; i < 20000; i++ {
		blkJSON, err := lc.getBlockByHeight(i)
		if err != nil {
			log.Println("Fail to get block err=%v", err)
			continue
		}
		if blkJSON == nil {
			log.Println("Fail to get block (not exist)")
			continue
		}
		blk, err := block.NewBlockV1(blkJSON)
		if err != nil {
			log.Printf("Fail to convert block err=%+v", blkJSON)
			continue
		}
		wvs := NewWorldVirtualState(ws, nil)
		txList := blk.NormalTransactions()
		for itr := txList.Iterator(); itr.Has(); itr.Next() {
			tx, _, _ := itr.Get()
			reqs := []LockRequest{
				{string(tx.From().Bytes()), AccountWriteLock},
				{string(tx.To().Bytes()), AccountWriteLock},
			}
			wvs = wvs.GetFuture(reqs)

			go func(ws WorldVirtualState, tx module.Transaction) {
				from := ws.GetAccountState(tx.From().Bytes())
				to := ws.GetAccountState(tx.To().Bytes())
				value := tx.Value()
				fromBalance := from.GetBalance()
				if fromBalance.Cmp(value) >= 0 {
					fromBalance.Sub(fromBalance, value)
					toBalance := to.GetBalance()
					toBalance.Add(toBalance, value)

					from.SetBalance(fromBalance)
					to.SetBalance(toBalance)
				}
				ws.Commit()
			}(wvs, tx)
		}

		wvs.Realize()
		wvss := wvs.GetSnapshot()
		wvss.Flush()
	}
}
