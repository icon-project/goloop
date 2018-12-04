package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"math/big"
	"testing"
)

func TestReceipt_JSON(t *testing.T) {
	addr := common.NewAddressFromString("cx0000000000000000000000000000000000000001")
	r := NewReceipt(addr)
	r.SetResult(module.StatusSuccess, big.NewInt(100), big.NewInt(1000), nil)
	r.SetCumulativeStepUsed(big.NewInt(100))
	jso, err := r.ToJSON(module.TransactionVersion3)
	if err != nil {
		t.Errorf("Fail on ToJSON err=%+v", err)
	}
	jb, err := json.MarshalIndent(jso, "", "    ")

	fmt.Printf("JSON: %s\n", jb)

	r2, err := NewReceiptFromJSON(jb, module.TransactionVersion3)
	if err != nil {
		t.Errorf("Fail on Making Receipt from JSON err=%+v", err)
		return
	}
	if !bytes.Equal(r.Bytes(), r2.Bytes()) {
		t.Errorf("Different bytes from Unmarshaled Receipt")
	}
}
