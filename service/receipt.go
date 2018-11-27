package service

import (
	"bytes"
	"encoding/json"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
	"log"
	"math/big"
	"reflect"
)

var FailureUnknown = &failureReason{
	CodeValue:    common.HexInt32{Value: int32(0)},
	MessageValue: "Unknown",
}

var FailureNotPayable = &failureReason{
	CodeValue:    common.HexInt32{Value: int32(0x7d64)},
	MessageValue: "This is not payable",
}

var FailureOutOfBalance = &failureReason{
	CodeValue:    common.HexInt32{Value: int32(0x7f58)},
	MessageValue: "Out of balance",
}

var FailureOutOfStepForInput = &failureReason{
	CodeValue:    common.HexInt32{Value: int32(0x7d64)},
	MessageValue: "Out of step: input",
}

var FailureOutOfStep = &failureReason{
	CodeValue:    common.HexInt32{Value: int32(0x7d64)},
	MessageValue: "Out of step",
}

type eventLog struct {
	Addr    common.Address `json:"scoreAddress"`
	Indexed []string       `json:"indexed"`
	Data    []string       `json:"data"`
}

type receipt struct {
	to                 common.Address
	cumulativeStepUsed big.Int
	stepUsed           big.Int
	stepPrice          big.Int
	logBloom           logBloom
	eventLogs          []eventLog
	success            bool
	result             interface{}
}

func (r *receipt) To() module.Address {
	return &r.to
}

func (r *receipt) Bytes() []byte {
	bs, err := codec.MarshalToBytes(r)
	if err != nil {
		log.Panicf("Fail to marshal object err=%+v", err)
	}
	return bs
}

func (r *receipt) Reset(s db.Database, k []byte) error {
	_, err := codec.UnmarshalFromBytes(k, r)
	return err
}

func (r *receipt) Flush() error {
	return nil
}

func (r *receipt) Equal(o trie.Object) bool {
	if rct, ok := o.(*receipt); ok {
		if rct == r {
			return true
		}
	}
	return false
}

func (r *receipt) CodecEncodeSelf(e *ugorji.Encoder) {
	e.Encode(r.to)
	e.Encode(r.success)
	if r.success {
		if bs, ok := r.result.([]byte); ok {
			e.Encode(bs)
		} else {
			e.Encode([]byte{})
		}
	} else {
		if reason, ok := r.result.(*failureReason); ok {
			e.Encode(reason)
		} else {
			e.Encode((*failureReason)(nil))
		}
	}
	e.Encode(r.cumulativeStepUsed.Bytes())
	e.Encode(r.stepUsed.Bytes())
	e.Encode(r.stepPrice.Bytes())
	if r.eventLogs == nil {
		e.Encode([]eventLog{})
	} else {
		e.Encode(r.eventLogs)
	}
	e.Encode(r.logBloom)
}

func (r *receipt) CodecDecodeSelf(d *ugorji.Decoder) {
	d.Decode(&r.to)
	d.Decode(&r.success)
	if r.success {
		var result []byte
		d.Decode(&result)
		r.result = result
	} else {
		failure := new(failureReason)
		d.Decode(failure)
		r.result = failure
	}
	var bs []byte
	d.Decode(&bs)
	r.cumulativeStepUsed.SetBytes(bs)
	d.Decode(&bs)
	r.stepUsed.SetBytes(bs)
	d.Decode(&bs)
	r.stepPrice.SetBytes(bs)
	d.Decode(&r.eventLogs)
	d.Decode(&r.logBloom)
}

type failureReason struct {
	CodeValue    common.HexInt32 `json:"code"`
	MessageValue string          `json:"message"`
}

func (f *failureReason) Code() int32 {
	return f.CodeValue.Value
}

func (f *failureReason) Message() string {
	return f.MessageValue
}

func NewReason(code int32, message string) module.Reason {
	return &failureReason{
		CodeValue:    common.HexInt32{Value: code},
		MessageValue: message,
	}
}

type receiptJSON struct {
	To                 common.Address  `json:"to"`
	CumulativeStepUsed common.HexUint  `json:"cumulativeStepUsed"`
	StepUsed           common.HexUint  `json:"stepUsed"`
	StepPrice          common.HexUint  `json:"stepPrice"`
	ScoreAddress       *common.Address `json:"scoreAddress,omitempty"`
	Failure            *failureReason  `json:"failure,omitempty"`
	EventLogs          []eventLog      `json:"eventLogs"`
	LogBloom           logBloom        `json:"logsBloom"`
	Status             common.HexInt16 `json:"status"`
}

func (r *receipt) ToJSON(version int) (interface{}, error) {
	switch version {
	case module.TransactionVersion2, module.TransactionVersion3:
		var rjo receiptJSON
		rjo.To = r.to
		rjo.CumulativeStepUsed.Set(&r.cumulativeStepUsed)
		rjo.StepUsed.Set(&r.stepUsed)
		rjo.StepPrice.Set(&r.stepPrice)
		if r.eventLogs != nil {
			rjo.EventLogs = r.eventLogs
		} else {
			rjo.EventLogs = []eventLog{}
		}
		rjo.LogBloom.SetBytes(r.logBloom.Bytes())
		if r.success {
			rjo.Status.Value = 1
			if bs, ok := r.result.([]byte); ok {
				if len(bs) == common.AddressBytes {
					rjo.ScoreAddress = common.NewAddress(bs)
				}
			}
		} else {
			rjo.Status.Value = 0
			if failure, ok := r.result.(*failureReason); ok {
				rjo.Failure = failure
			} else {
				rjo.Failure = FailureUnknown
			}
		}

		rjson := make(map[string]interface{})
		rjson["to"] = &rjo.To
		rjson["cumulativeStepUsed"] = &rjo.CumulativeStepUsed
		rjson["stepUsed"] = &rjo.StepUsed
		rjson["stepPrice"] = &rjo.StepPrice
		rjson["eventLogs"] = rjo.EventLogs
		rjson["logBloom"] = rjo.LogBloom
		rjson["status"] = &rjo.Status
		if rjo.Failure != nil {
			rjson["failure"] = rjo.Failure
		}
		if rjo.ScoreAddress != nil {
			rjson["scoreAddress"] = rjo.ScoreAddress
		}
		return rjson, nil
	default:
		return nil, common.ErrIllegalArgument
	}
}

func (r *receipt) MarshalJSON() ([]byte, error) {
	obj, err := r.ToJSON(module.TransactionVersion3)
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

func (r *receipt) UnmarshalJSON(bs []byte) error {
	var rjson receiptJSON
	if err := json.Unmarshal(bs, &rjson); err != nil {
		return err
	}
	r.to = rjson.To
	r.cumulativeStepUsed.Set(&rjson.CumulativeStepUsed.Int)
	r.stepUsed.Set(&rjson.StepUsed.Int)
	r.stepPrice.Set(&rjson.StepPrice.Int)
	r.logBloom.SetBytes(rjson.LogBloom.Bytes())
	r.eventLogs = rjson.EventLogs
	r.success = rjson.Status.Value == 1
	if r.success {
		if rjson.ScoreAddress != nil {
			r.result = rjson.ScoreAddress.Bytes()
		} else {
			r.result = []byte{}
		}
	} else {
		if rjson.Failure == nil {
			log.Printf("FailureIsEmpty\nJSON:%s", bs)
			return errors.New("FailureIsEmpty")
		}
		r.result = rjson.Failure
	}
	return nil
}

func (r *receipt) AddLog(addr module.Address, indexed, data []string) {
	var log eventLog
	log.Addr.SetBytes(addr.Bytes())
	log.Indexed = make([]string, len(indexed))
	copy(log.Indexed, indexed)
	log.Data = make([]string, len(data))
	copy(log.Data, data)

	r.eventLogs = append(r.eventLogs, log)

	r.logBloom.AddEvent(&log)
}

func (r *receipt) SetCumulativeStepUsed(cumulativeUsed *big.Int) {
	r.cumulativeStepUsed.Set(cumulativeUsed)
}

func (r *receipt) SetResult(success bool, result interface{}, used, price *big.Int) {
	r.success = success
	if success {
		if result == nil {
			r.result = []byte{}
		} else {
			r.result = result.([]byte)
		}
	} else {
		r.result = result.(*failureReason)
	}
	r.stepUsed.Set(used)
	r.stepPrice.Set(price)
}

func (r *receipt) CumulativeStepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.cumulativeStepUsed)
	return p
}

func (r *receipt) StepPrice() *big.Int {
	p := new(big.Int)
	p.Set(&r.stepPrice)
	return p
}

func (r *receipt) StepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.stepUsed)
	return p
}

func (r *receipt) Success() bool {
	return r.success
}

func (r *receipt) Result() []byte {
	if r.success {
		if bs, ok := r.result.([]byte); ok {
			return bs
		}
	}
	return nil
}

func (r *receipt) Check(r2 module.Receipt) error {
	rct2, ok := r2.(*receipt)
	if !ok {
		return errors.New("IncompatibleReceipt")
	}
	if rct2.success != r.success {
		return errors.New("DifferentStatus")
	}
	if rct2.stepUsed.Cmp(&r.stepUsed) != 0 {
		return errors.New("DifferentStepUsed")
	}
	if rct2.stepPrice.Cmp(&r.stepPrice) != 0 {
		return errors.New("DifferentStepPrice")
	}
	if rct2.cumulativeStepUsed.Cmp(&r.cumulativeStepUsed) != 0 {
		return errors.New("DifferentCumulativeStepUsed")
	}
	if r.success {
		if r.result != rct2.result {
			if r.result == nil || rct2.result == nil {
				return errors.New("DifferentResultValueWitNull")
			}
			if !bytes.Equal(r.result.([]byte), rct2.result.([]byte)) {
				return errors.New("DifferentResultValue")
			}
		}
		if len(r.eventLogs) != len(rct2.eventLogs) {
			return errors.New("EventLogHasDifferentLength")
		}
		for i, e := range r.eventLogs {
			e2 := &rct2.eventLogs[i]
			if !e2.Addr.Equal(&e.Addr) {
				return errors.Errorf("Event(%d)NotMatchOnAddress", i)
			}
			if !reflect.DeepEqual(e2.Indexed, e.Indexed) {
				return errors.Errorf("Event(%d)IndexedNotMatch", i)
			}
			if !reflect.DeepEqual(e2.Data, e.Data) {
				return errors.Errorf("Event(%d)IndexedNotMatch", i)
			}
		}
	} else {
		f1 := r.result.(*failureReason)
		f2 := r.result.(*failureReason)
		if f1.CodeValue.Value != f2.CodeValue.Value {
			return errors.New("DifferentFailureCode")
		}
	}
	return nil
}

func (r *receipt) Reason() module.Reason {
	if r.success {
		return nil
	} else {
		if f, ok := r.result.(module.Reason); ok {
			return f
		}
		return nil
	}
}

func NewReceiptFromJSON(bs []byte, version int) (Receipt, error) {
	r := new(receipt)
	if err := json.Unmarshal(bs, r); err != nil {
		return nil, err
	}
	return r, nil
}

func NewReceipt(to module.Address) Receipt {
	r := new(receipt)
	r.to.SetBytes(to.Bytes())
	return r
}
