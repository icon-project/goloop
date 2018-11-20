package service

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
	ugorji "github.com/ugorji/go/codec"
	"math/big"
)

type eventLog struct {
	Addr    common.Address `json:"scoreAddress"`
	Indexed []string       `json:"indexed"`
	Data    []string       `json:"data"`
}

type receipt struct {
	cumulativeStepUsed big.Int
	stepUsed           big.Int
	stepPrice          big.Int
	logBloom           LogBloom
	eventLogs          []eventLog
	success            bool
}

func (r *receipt) CodecEncodeSelf(e *ugorji.Encoder) {
	e.Encode(r.success)
	e.Encode(r.cumulativeStepUsed.Bytes())
	e.Encode(r.stepUsed.Bytes())
	e.Encode(r.stepPrice.Bytes())
	e.Encode(r.eventLogs)
	e.Encode(r.logBloom)
}

func (r *receipt) CodecDecodeSelf(d *ugorji.Decoder) {
	d.Decode(&r.success)
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

func (r *receipt) Bytes() ([]byte, error) {
	return codec.MarshalToBytes(r)
}

type receiptJSON struct {
	CumulativeStepUsed common.HexInt   `json:"cumulativeStepUsed"`
	StepUsed           common.HexInt   `json:"stepUsed"`
	StepPrice          common.HexInt   `json:"stepPrice"`
	ScoreAddress       common.Address  `json:"scoreAddress"`
	EventLogs          []eventLog      `json:"eventLogs"`
	LogBloom           common.HexBytes `json:"logsBloom"`
	Status             common.HexInt16 `json:"status"`
}

func (r *receipt) ToJSON(version int) (interface{}, error) {
	var rjson receiptJSON
	rjson.CumulativeStepUsed.Set(&r.cumulativeStepUsed)
	rjson.StepUsed.Set(&r.stepUsed)
	rjson.StepPrice.Set(&r.stepPrice)
	rjson.EventLogs = r.eventLogs
	rjson.LogBloom = r.logBloom.Bytes()
	if r.success {
		rjson.Status.Value = 1
	}
	return &rjson, nil
}

func (r *receipt) AddLog(addr module.Address, indexed, data []string) {
	var log eventLog
	log.Addr.SetBytes(addr.Bytes())
	log.Indexed = make([]string, len(indexed))
	copy(log.Indexed, indexed)
	log.Data = make([]string, len(data))
	copy(log.Data, data)

	r.eventLogs = append(r.eventLogs, log)

	r.logBloom.AddLog(addr.String())
	r.logBloom.AddLog(log.Indexed...)
}

func (r *receipt) SetCumulativeStepUsed(cumulativeUsed *big.Int) {
	r.cumulativeStepUsed.Set(cumulativeUsed)
}

func (r *receipt) SetResult(success bool, used, price *big.Int) {
	r.stepUsed.Set(used)
	r.stepPrice.Set(price)
	r.success = success
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
