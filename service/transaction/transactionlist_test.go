package transaction

import (
	"bytes"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestTransactionList_Iterator(t *testing.T) {
	txjsons := []string{
		"{\"from\": \"hx54f7853dc6481b670caf69c5a27c7c8fe5be8269\", \"to\": \"hx49a23bd156932485471f582897bf1bec5f875751\", \"value\": \"0x56bc75e2d63100000\", \"fee\": \"0x2386f26fc10000\", \"nonce\": \"0x1\", \"tx_hash\": \"375540830d475a73b704cf8dee9fa9eba2798f9d2af1fa55a85482e48daefd3b\", \"signature\": \"bjarKeF3izGy469dpSciP3TT9caBQVYgHdaNgjY+8wJTOVSFm4o/ODXycFOdXUJcIwqvcE9If8x6Zmgt//XmkQE=\", \"method\": \"icx_sendTransaction\"}",
	}
	txslice := make([]module.Transaction, len(txjsons))

	for i, txjson := range txjsons {
		if tx, err := NewTransactionFromJSON([]byte(txjson)); err != nil {
			t.Errorf("Fail to make TX from JSON err=%+v", err)
		} else {
			txslice[i] = tx
		}
	}

	mdb := db.NewMapDB()
	tl := NewTransactionListFromSlice(mdb, txslice)
	tl.Flush()

	idx := 0
	tl2 := NewTransactionListFromHash(mdb, tl.Hash())
	for itr := tl2.Iterator(); itr.Has(); itr.Next() {
		tx, _, err := itr.Get()
		if err != nil {
			t.Errorf("Fail to get transaction[%d] err=%+v", idx, err)
			return
		}
		if !bytes.Equal(tx.ID(), txslice[idx].ID()) {
			t.Errorf("Different ID() of transaction[%d]", idx)
		}
		if !bytes.Equal(tx.Bytes(), txslice[idx].Bytes()) {
			t.Errorf("Different Bytes() of transaction[%d]", idx)
		}
		idx++
	}
}
