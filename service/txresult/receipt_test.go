package txresult

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

var receiptRevisions = []module.Revision{0, module.UseMPTOnEvents}

func TestReceipt_JSON(t *testing.T) {
	for _, rev := range receiptRevisions {
		t.Run(fmt.Sprint("Revision", rev), func(t *testing.T) {
			testReceiptJSONByRev(t, rev)
		})
	}
}

func testReceiptJSONByRev(t *testing.T, rev module.Revision) {
	database := db.NewMapDB()
	addr := common.MustNewAddressFromString("cx0000000000000000000000000000000000000001")
	r := NewReceipt(database, rev, addr)
	r.SetResult(module.StatusSuccess, big.NewInt(100), big.NewInt(1000), nil)
	r.SetCumulativeStepUsed(big.NewInt(100))
	jso, err := r.ToJSON(module.JSONVersionLast)
	if err != nil {
		t.Errorf("Fail on ToJSON err=%+v", err)
	}
	jb, err := json.MarshalIndent(jso, "", "    ")

	fmt.Printf("JSON: %s\n", jb)

	r2, err := NewReceiptFromJSON(database, rev, jb)
	if err != nil {
		t.Errorf("Fail on Making Receipt from JSON err=%+v", err)
		return
	}
	if !bytes.Equal(r.Bytes(), r2.Bytes()) {
		t.Errorf("Different bytes from Unmarshaled Receipt")
	}

	t.Logf("Encoded: % X", r.Bytes())

	r3 := new(receipt)
	err = r3.Reset(db.NewMapDB(), r.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, r3.Bytes(), r2.Bytes())
}

func Test_EventLog_BytesEncoding(t *testing.T) {
	var ev eventLog

	ev.eventLogData.Addr.SetTypeAndID(false, []byte{0x02})
	ev.eventLogData.Indexed = [][]byte{
		[]byte("Test(int)"),
		[]byte{0x01},
	}
	ev.eventLogData.Data = nil

	evj := ev.ToJSON(module.JSONVersion3)
	evs, err := json.Marshal(evj)
	t.Logf("JSON:%s", evs)

	bs, err := codec.MarshalToBytes(&ev)
	assert.NoError(t, err)

	log.Printf("ENCODED:% x", bs)

	var ev2 eventLog
	_, err = codec.UnmarshalFromBytes(bs, &ev2)
	assert.NoError(t, err)

	evj = ev2.ToJSON(module.JSONVersion3)
	assert.NoError(t, err)
	evs2, err := json.Marshal(evj)
	t.Logf("JSON:%s", evs2)

	assert.Equal(t, evs, evs2)
}

func TestReceipt_DisableLogBloom(t *testing.T) {
	dbase := db.NewMapDB()
	to := common.MustNewAddressFromString("hx9834234")
	score := common.MustNewAddressFromString("cx1234")
	for _, rev := range []module.Revision{module.NoRevision, module.LatestRevision} {
		t.Run(fmt.Sprint("Rev", rev), func(t *testing.T) {
			rct := NewReceipt(dbase, rev, to)
			rct.AddLog(score, [][]byte{[]byte("TestEvent(int)"), []byte{0x02}}, [][]byte{})
			rct.DisableLogsBloom()
			rct.SetResult(module.StatusSuccess, new(big.Int), new(big.Int), nil)

			assert.Equal(t, []byte{}, rct.LogsBloom().Bytes())

			err := rct.Flush()
			assert.NoError(t, err)

			// json marshalling test
			jso, err := rct.ToJSON(module.JSONVersionLast)
			assert.NoError(t, err)
			jb, err := json.Marshal(jso)
			assert.NoError(t, err)

			fmt.Println("JSON:", string(jb))

			rct2, err := NewReceiptFromJSON(dbase, rev, jb)
			assert.NoError(t, err)
			err = rct.Check(rct2)
			assert.NoError(t, err)

			// binary marshalling test
			bs := codec.BC.MustMarshalToBytes(rct)

			fmt.Printf("BYTES:%#x\n", bs)

			rct3 := new(receipt)
			err = rct3.Reset(dbase, bs)
			assert.NoError(t, err)
			err = rct.Check(rct2)
			assert.NoError(t, err)
		})
	}
}

func TestReceipt_Fee(t *testing.T) {
	database := db.NewMapDB()
	eoa1 := common.MustNewAddressFromString("hx9834234")
	contract1 := common.MustNewAddressFromString("cx0000000000000000000000000000000000000001")
	contract2 := common.MustNewAddressFromString("cx0000000000000000000000000000000000000002")
	contract3 := common.MustNewAddressFromString("cx0000000000000000000000000000000000000003")

	t.Run("paid all by eoa", func(t *testing.T) {
		r := NewReceipt(database, module.LatestRevision, contract1)
		stepUsed := big.NewInt(1000)
		stepPrice := big.NewInt(85)
		r.SetResult(module.StatusSuccess, stepUsed, stepPrice, nil)

		fee := new(big.Int).Mul(stepUsed, stepPrice)
		assert.Equal(t, fee, r.Fee())
		assert.Equal(t, fee, r.FeeByEOA())
	})

	t.Run("paid by eoa and contract", func(t *testing.T) {
		r := NewReceipt(database, module.LatestRevision, contract1)
		stepPrice := big.NewInt(85)

		stepByVirtual := big.NewInt(100)
		stepByDeposit := big.NewInt(120)
		stepByContract := new(big.Int).Add(stepByVirtual, stepByDeposit)
		stepByEOA := big.NewInt(780)
		stepUsed := new(big.Int).Add(stepByContract, stepByEOA)

		r.AddPayment(contract1, stepByContract, stepByDeposit)
		r.AddPayment(eoa1, stepByEOA, stepByEOA)
		r.SetResult(module.StatusSuccess, stepUsed, stepPrice, nil)

		stepsForFee := new(big.Int).Add(stepByDeposit, stepByEOA)
		fee := new(big.Int).Mul(stepsForFee, stepPrice)
		feeByEOA := new(big.Int).Mul(stepByEOA, stepPrice)
		assert.Equal(t, fee, r.Fee())
		assert.Equal(t, feeByEOA, r.FeeByEOA())
	})

	t.Run("paid by virtual step only", func(t *testing.T) {
		r := NewReceipt(database, module.LatestRevision, contract1)
		stepPrice := big.NewInt(85)

		stepByVirtual := big.NewInt(100)
		stepByContract := stepByVirtual
		stepByEOA := big.NewInt(0)
		stepUsed := new(big.Int).Add(stepByContract, stepByEOA)

		r.AddPayment(contract1, stepByVirtual, nil)
		r.SetResult(module.StatusSuccess, stepUsed, stepPrice, nil)

		fee := new(big.Int)
		feeByEOA := new(big.Int).Mul(stepByEOA, stepPrice)
		assert.Equal(t, fee, r.Fee())
		assert.Equal(t, feeByEOA, r.FeeByEOA())
	})

	t.Run("paid by eoa and multiple contracts", func(t *testing.T) {
		r := NewReceipt(database, module.LatestRevision, contract1)
		stepPrice := big.NewInt(85)

		stepByVirtual := big.NewInt(100)
		stepByDeposit := big.NewInt(120)
		stepByContract := new(big.Int).Add(stepByVirtual, stepByDeposit)

		stepByVirtual2 := big.NewInt(0)
		stepByDeposit2 := big.NewInt(200)
		stepByContract2 := new(big.Int).Add(stepByVirtual2, stepByDeposit2)

		stepByVirtual3 := big.NewInt(10)
		stepByContract3 := stepByVirtual3

		stepByEOA := big.NewInt(780)

		stepUsed := new(big.Int).Add(stepByContract, stepByContract2)
		stepUsed.Add(stepUsed, stepByContract3)
		stepUsed.Add(stepUsed, stepByEOA)

		r.AddPayment(contract1, stepByContract, stepByDeposit)
		r.AddPayment(contract2, stepByContract2, stepByDeposit2)
		r.AddPayment(contract3, stepByContract3, nil)
		r.AddPayment(eoa1, stepByEOA, stepByEOA)
		r.SetResult(module.StatusSuccess, stepUsed, stepPrice, nil)

		stepForFee := new(big.Int).Add(stepByDeposit, stepByEOA)
		stepForFee.Add(stepForFee, stepByDeposit2)
		fee := new(big.Int).Mul(stepForFee, stepPrice)

		feeByEOA := new(big.Int).Mul(stepByEOA, stepPrice)
		assert.Equal(t, fee, r.Fee())
		assert.Equal(t, feeByEOA, r.FeeByEOA())
	})
}

func TestDecomposeEventSignature(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 []string
	}{
		{"Nothing", args{""}, "", nil},
		{"NoParenthesis", args{"EventName"}, "", nil},
		{"NoParam", args{"EventXY()"}, "EventXY", []string{}},
		{"1Param", args{"EventXYZ(Address)"}, "EventXYZ", []string{"Address"}},
		{"3Param", args{"EventSIB(str,int,bytes)"}, "EventSIB", []string{"str", "int", "bytes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := DecomposeEventSignature(tt.args.s)
			assert.Equalf(t, tt.want, got, "DecomposeEventSignature(%v)", tt.args.s)
			assert.Equalf(t, tt.want1, got1, "DecomposeEventSignature(%v)", tt.args.s)
		})
	}
}
