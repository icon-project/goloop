package txresult

import (
	"encoding/hex"
	"encoding/json"
	"github.com/icon-project/goloop/service/state"
	"log"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
	ugorji "github.com/ugorji/go/codec"
)

const (
	EventLogICXTransfer = "ICXTransfer(Address,Address,int)"
)

type eventLogJSON struct {
	Addr    common.Address `json:"scoreAddress"`
	Indexed []string       `json:"indexed"`
	Data    []string       `json:"data"`
}

type eventLogData struct {
	Addr    common.Address
	Indexed [][]byte
	Data    [][]byte
}

type eventLog struct {
	data eventLogData
}

func (log *eventLog) Address() module.Address {
	return &log.data.Addr
}

func (log *eventLog) Data() [][]byte {
	return log.data.Data
}

func (log *eventLog) ToJSON(v int) (*eventLogJSON, error) {
	_, pts := DecomposeEventSignature(string(log.data.Indexed[0]))
	if len(pts)+1 != len(log.data.Indexed)+len(log.data.Data) {
		return nil, errors.InvalidStateError.New("NumberOfParametersAreNotSameAsData")
	}

	eljson := new(eventLogJSON)
	eljson.Addr = log.data.Addr
	eljson.Indexed = make([]string, len(log.data.Indexed))
	eljson.Data = make([]string, len(log.data.Data))

	aidx := 0
	eljson.Indexed[0] = string(log.data.Indexed[0])
	for i, v := range log.data.Indexed[1:] {
		if s, err := EventDataBytesToStringByType(pts[aidx], v); err != nil {
			return nil, err
		} else {
			eljson.Indexed[i+1] = s
			aidx++
		}
	}
	for i, v := range log.data.Data {
		if s, err := EventDataBytesToStringByType(pts[aidx], v); err != nil {
			return nil, err
		} else {
			eljson.Data[i] = s
			aidx++
		}
	}
	return eljson, nil
}

type receiptData struct {
	Status             module.Status
	To                 common.Address
	CumulativeStepUsed common.HexInt
	StepUsed           common.HexInt
	StepPrice          common.HexInt
	LogBloom           LogBloom
	EventLogs          []*eventLog
	SCOREAddress       *common.Address
}

type receipt struct {
	data receiptData
}

func (r *receipt) SCOREAddress() module.Address {
	return r.data.SCOREAddress
}

func (r *receipt) To() module.Address {
	return &r.data.To
}

func (r *receipt) Bytes() []byte {
	bs, err := codec.MarshalToBytes(&r.data)
	if err != nil {
		log.Panicf("Fail to marshal object err=%+v", err)
	}
	return bs
}

func (r *receipt) Reset(s db.Database, k []byte) error {
	_, err := codec.UnmarshalFromBytes(k, &r.data)
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
	if err := e.Encode(&r.data); err != nil {
		log.Panicf("FailOnEncodeReceipt err=%+v", err)
	}
}

func (r *receipt) CodecDecodeSelf(d *ugorji.Decoder) {
	if err := d.Decode(&r.data); err != nil {
		log.Panicf("FailOnDecodeReceipt err=%+v", err)
	}
}

func (r *receipt) Resolve(bd merkle.Builder) error {
	return nil
}

func (r *receipt) LogBloom() module.LogBloom {
	return &r.data.LogBloom
}

func (r *receipt) EventLogIterator() module.EventLogIterator {
	return nil
}

type eventLogIterator struct {
	slice []*eventLog
	index int
}

func (it *eventLogIterator) Has() bool {
	return it.index < len(it.slice)
}

func (it *eventLogIterator) Next() error {
	if it.index >= len(it.slice) {
		return errors.InvalidStateError.New("no next item")
	}
	it.index++
	return nil
}

func (it *eventLogIterator) Get() (module.EventLog, error) {
	if !it.Has() {
		return nil, errors.InvalidStateError.New("no item")
	}
	return it.slice[it.index], nil
}

type failureReason struct {
	CodeValue    common.HexUint16 `json:"code"`
	MessageValue string           `json:"message"`
}

func (f *failureReason) Code() uint16 {
	return f.CodeValue.Value
}

func (f *failureReason) Message() string {
	return f.MessageValue
}

func failureReasonByCode(status module.Status) *failureReason {
	return &failureReason{
		CodeValue:    common.HexUint16{Value: uint16(status)},
		MessageValue: status.String(),
	}
}

type Receipt interface {
	module.Receipt
	AddLog(addr module.Address, indexed, data [][]byte)
	SetCumulativeStepUsed(cumulativeUsed *big.Int)
	SetResult(status module.Status, used, price *big.Int, addr module.Address)
}

type receiptJSON struct {
	To                 common.Address   `json:"to"`
	CumulativeStepUsed common.HexInt    `json:"cumulativeStepUsed"`
	StepUsed           common.HexInt    `json:"stepUsed"`
	StepPrice          common.HexInt    `json:"stepPrice"`
	SCOREAddress       *common.Address  `json:"scoreAddress,omitempty"`
	Failure            *failureReason   `json:"failure,omitempty"`
	EventLogs          []*eventLogJSON  `json:"eventLogs"`
	LogBloom           LogBloom         `json:"logsBloom"`
	Status             common.HexUint16 `json:"status"`
}

func (r *receipt) ToJSON(version int) (interface{}, error) {
	switch version {
	case module.TransactionVersion2, module.TransactionVersion3:
		var rjo receiptJSON
		rjo.To = r.data.To
		rjo.CumulativeStepUsed.Set(&r.data.CumulativeStepUsed.Int)
		rjo.StepUsed.Set(&r.data.StepUsed.Int)
		rjo.StepPrice.Set(&r.data.StepPrice.Int)
		logs := make([]*eventLogJSON, len(r.data.EventLogs))
		for i, log := range r.data.EventLogs {
			if logjson, err := log.ToJSON(version); err != nil {
				return nil, err
			} else {
				logs[i] = logjson
			}
		}
		rjo.EventLogs = logs
		rjo.LogBloom.SetBytes(r.data.LogBloom.Bytes())
		if r.data.Status == module.StatusSuccess {
			rjo.Status.Value = 1
			rjo.SCOREAddress = r.data.SCOREAddress
		} else {
			rjo.Status.Value = 0
			rjo.Failure = failureReasonByCode(r.data.Status)
		}

		rjson := make(map[string]interface{})
		rjson["to"] = &rjo.To
		rjson["cumulativeStepUsed"] = &rjo.CumulativeStepUsed
		rjson["stepUsed"] = &rjo.StepUsed
		rjson["stepPrice"] = &rjo.StepPrice
		rjson["eventLogs"] = rjo.EventLogs
		rjson["logBloom"] = &rjo.LogBloom
		rjson["status"] = &rjo.Status
		if rjo.Failure != nil {
			rjson["failure"] = rjo.Failure
		}
		if rjo.SCOREAddress != nil {
			rjson["scoreAddress"] = rjo.SCOREAddress
		}
		return rjson, nil
	default:
		return nil, errors.ErrIllegalArgument
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
	data := &r.data
	if rjson.Status.Value == 1 {
		data.Status = module.StatusSuccess
		data.SCOREAddress = rjson.SCOREAddress
	} else {
		data.Status = module.Status(rjson.Failure.CodeValue.Value)
	}
	data.To = rjson.To
	data.CumulativeStepUsed.Set(&rjson.CumulativeStepUsed.Int)
	data.StepUsed.Set(&rjson.StepUsed.Int)
	data.StepPrice.Set(&rjson.StepPrice.Int)
	if len(rjson.EventLogs) > 0 {
		data.EventLogs = make([]*eventLog, len(rjson.EventLogs))
		for i, e := range rjson.EventLogs {
			if el, err := eventLogFromJSON(e); err != nil {
				return err
			} else {
				data.EventLogs[i] = el
			}
		}
	}
	data.LogBloom.SetBytes(rjson.LogBloom.Bytes())
	return nil
}

func (r *receipt) AddLog(addr module.Address, indexed, data [][]byte) {
	log := new(eventLog)
	log.data.Addr.SetBytes(addr.Bytes())
	log.data.Indexed = indexed
	log.data.Data = data

	r.data.EventLogs = append(r.data.EventLogs, log)
	r.data.LogBloom.AddLog(&log.data.Addr, log.data.Indexed)
}

func (r *receipt) SetCumulativeStepUsed(cumulativeUsed *big.Int) {
	r.data.CumulativeStepUsed.Set(cumulativeUsed)
}

func (r *receipt) SetResult(status module.Status, used, price *big.Int, addr module.Address) {
	r.data.Status = status
	if status == module.StatusSuccess && addr != nil {
		r.data.SCOREAddress = common.NewAddress(addr.Bytes())
	}
	r.data.StepUsed.Set(used)
	r.data.StepPrice.Set(price)
}

func (r *receipt) CumulativeStepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.CumulativeStepUsed.Int)
	return p
}

func (r *receipt) StepPrice() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.StepPrice.Int)
	return p
}

func (r *receipt) StepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.StepUsed.Int)
	return p
}

func (r *receipt) Status() module.Status {
	return r.data.Status
}

func (r *receipt) Check(r2 module.Receipt) error {
	rct2, ok := r2.(*receipt)
	if !ok {
		return state.IllegalTypeError.New("IncompatibleReceipt")
	}
	if !reflect.DeepEqual(&r.data, &rct2.data) {
		return errors.InvalidStateError.New("DataIsn'tEqual")
	}
	return nil
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
	r.data.To.SetBytes(to.Bytes())
	return r
}

func DecomposeEventSignature(s string) (string, []string) {
	reg := regexp.MustCompile(`^(\w+)\(((?:\w+)(?:,(?:\w+))*)\)$`)
	if reg == nil {
		return "", nil
	}
	matches := reg.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], strings.Split(matches[2], ",")
}

func EventDataBytesToStringByType(t string, v []byte) (string, error) {
	switch t {
	case "Address":
		var addr common.Address
		if err := addr.SetBytes(v); err != nil {
			return "", err
		}
		return addr.String(), nil
	case "int":
		var ivalue common.HexInt
		ivalue.SetBytes(v)
		return ivalue.String(), nil
	case "str":
		return string(v), nil
	case "bytes":
		return "0x" + hex.EncodeToString(v), nil
	case "bool":
		var ivalue common.HexInt
		ivalue.SetBytes(v)
		if ivalue.Sign() == 0 {
			return "0x0", nil
		} else {
			return "0x1", nil
		}
	default:
		return "", errors.Errorf("UnknownType(%s)For(<% x>)", t, v)
	}
}

func EventDataStringToBytesByType(t string, v string) ([]byte, error) {
	switch t {
	case "Address":
		var addr common.Address
		if err := addr.SetString(v); err != nil {
			return nil, err
		}
		return addr.Bytes(), nil
	case "int":
		var ivalue common.HexInt
		ivalue.SetString(v, 0)
		return ivalue.Bytes(), nil
	case "str":
		return []byte(v), nil
	case "bytes":
		if len(v) < 3 {
			return []byte{}, nil
		}
		if v[0:2] != "0x" {
			return nil, errors.Errorf("IllegalFormatForBytes(%s)", v)
		}
		return hex.DecodeString(v[2:])
	case "bool":
		if v == "0x1" {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	default:
		return nil, errors.Errorf("UnknownType(%s)For(%s)", t, v)
	}
}

func eventLogFromJSON(e *eventLogJSON) (*eventLog, error) {
	el := new(eventLog)
	el.data.Addr = e.Addr
	el.data.Indexed = make([][]byte, len(e.Indexed))
	el.data.Data = make([][]byte, len(e.Data))
	_, pts := DecomposeEventSignature(e.Indexed[0])

	if len(pts)+1 != len(e.Indexed)+len(e.Data) {
		return nil, errors.InvalidStateError.New("InvalidSignatureCount")
	}

	el.data.Indexed[0] = []byte(e.Indexed[0])

	aidx := 0
	for i, is := range e.Indexed[1:] {
		if bs, err := EventDataStringToBytesByType(pts[aidx], is); err != nil {
			return nil, err
		} else {
			el.data.Indexed[i+1] = bs
			aidx++
		}
	}

	for i, is := range e.Data {
		if bs, err := EventDataStringToBytesByType(pts[aidx], is); err != nil {
			return nil, err
		} else {
			el.data.Data[i] = bs
			aidx++
		}
	}

	return el, nil
}
