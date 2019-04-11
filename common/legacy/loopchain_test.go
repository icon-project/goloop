// +build ignore

package legacy

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoopChainDB_GetReceiptByTransaction(t *testing.T) {
	dbase, err := OpenDatabase("./data/testnet/block", "")
	if err != nil {
		t.Errorf("Fail to open database err=%+v", err)
		return
	}

	for i := 1; i < 10; i++ {
		blk, err := dbase.GetBlockByHeight(i)
		if err != nil {
			t.Errorf("Fail to get block err=%+v", err)
			return
		}
		txl := blk.NormalTransactions()
		for i := txl.Iterator(); i.Has(); i.Next() {
			tx, _, err := i.Get()
			if err != nil {
				t.Errorf("Fail to get transaction err=%+v", err)
				continue
			}
			r, err := dbase.GetReceiptByTransaction(tx.ID())
			if err != nil {
				t.Errorf("Fail to get receipt for tx=%x err=%+v", tx.ID(), err)
				continue
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "    ")
			enc.Encode(r)
			os.Stdout.Sync()
		}
	}
}
