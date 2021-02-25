package txresult

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestReceiptList(t *testing.T) {
	for _, rev := range []module.Revision{0, module.UseMPTOnEvents} {
		t.Run(fmt.Sprintf("Revision:%#x", rev), func(t *testing.T) {
			testReceiptListByRev(t, rev)
		})
	}
}

func testReceiptListByRev(t *testing.T, rev module.Revision) {
	mdb := db.NewMapDB()
	rslice := make([]Receipt, 0)

	var used, price big.Int

	addr1 := common.MustNewAddressFromString("hx8888888888888888888888888888888888888888")
	for i := 0; i < 5; i++ {
		r1 := NewReceipt(mdb, rev, addr1)
		used.SetInt64(int64(i * 100))
		price.SetInt64(int64(i * 10))
		r1.SetResult(module.StatusOutOfBalance, &used, &price, nil)
		rslice = append(rslice, r1)
	}
	addr2 := common.MustNewAddressFromString("cx0003737589788888888888888888888888888888")
	for i := 0; i < 5; i++ {
		r2 := NewReceipt(mdb, rev, addr2)
		used.SetInt64(int64(i * 100))
		price.SetInt64(int64(i * 10))
		r2.SetResult(module.StatusOutOfBalance, &used, &price, nil)
		rslice = append(rslice, r2)
	}

	rl1 := NewReceiptListFromSlice(mdb, rslice)
	hash := rl1.Hash()
	rl1.Flush()

	log.Printf("Hash for receipts : <%x>", hash)

	rl2 := NewReceiptListFromHash(mdb, hash)
	idx := 0
	for itr := rl2.Iterator(); itr.Has(); itr.Next() {
		r, err := itr.Get()
		if err != nil {
			t.Errorf("Fail on Get Receipt from ReceiptList from Hash err=%+v", err)
		}
		if !bytes.Equal(rslice[idx].Bytes(), r.Bytes()) {
			t.Errorf("Fail on comparing bytes for Receipt[%d]", idx)
		}
		if err := rslice[idx].Check(r); err != nil {
			t.Errorf("Fail on Check Receipt[%d] err=%+v", idx, err)
		}
		idx++
	}
}
