package service

import (
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const testReceiptNum = 10

var resultReceipt map[int][]byte

func iterReceiptList(receiptList module.ReceiptList) {
	m := make(map[int]bool)
	for iter := receiptList.Iterator(); iter.Has(); iter.Next() {
		if r, err := iter.Get(); err == nil {
			if rImple, ok := r.(*receipt); ok {
				if rImple.cumulativeStepUsed.Cmp(&rImple.stepPrice) != 0 {
					panic("FAILED")
				} else if rImple.stepPrice.Cmp(&rImple.stepUsed) != 0 {
					panic("Failed 2")
				}
				m[int(rImple.cumulativeStepUsed.Int64())] = true
			} else {
				log.Panicf("failed\n")
			}
		} else {
			log.Panicf("err is not nil\n")
		}
	}

	for i := 0; i < testReceiptNum; i++ {
		if m[i] == false {
			log.Panicf("%d is not received\n", i)
		}
	}
}

func loopGetReceiptList(receiptList module.ReceiptList) {
	for i := 0; i < testReceiptNum; i++ {
		if r, err := receiptList.Get(i); err == nil {
			if bytes.Compare(r.Bytes(), resultReceipt[i]) != 0 {
				panic("FAILED")
			}
		}
	}
}

func TestReceiptList(t *testing.T) {
	resultReceipt = make(map[int][]byte)
	receiptSlice := make([]*receipt, testReceiptNum)
	for i := 0; i < testReceiptNum; i++ {
		receiptSlice[i] =
			&receipt{cumulativeStepUsed: *big.NewInt(int64(i)),
				stepUsed:  *big.NewInt(int64(i)),
				stepPrice: *big.NewInt(int64(i)),
				eventLogs: nil,
				success:   i%2 == 0,
			}
	}
	mdb := db.NewMapDB()

	interfaceReceipts := make([]Receipt, testReceiptNum)
	for i, v := range receiptSlice {
		interfaceReceipts[i] = v
		resultReceipt[i] = v.Bytes()
	}
	receiptListFromSlice := NewReceiptListFromSlice(mdb, interfaceReceipts)
	iterReceiptList(receiptListFromSlice)
	loopGetReceiptList(receiptListFromSlice)
	hash := receiptListFromSlice.Hash()
	receiptListFromSlice.Flush()

	receiptListFromHash := NewReceiptListFromHash(mdb, hash)
	loopGetReceiptList(receiptListFromHash)
	iterReceiptList(receiptListFromHash)
}
