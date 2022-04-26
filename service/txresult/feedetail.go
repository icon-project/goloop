/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package txresult

import (
	"bytes"
	"encoding/json"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type feePayment struct {
	Payer  common.Address
	Amount common.HexInt
}

type feeDetail []*feePayment

func (d *feeDetail) AddPayment(address module.Address, steps *big.Int) bool {
	if steps.Sign() == 0 {
		return false
	}
	for _, p := range *d {
		if p.Payer.Equal(address) {
			p.Amount.Add(&p.Amount.Int, steps)
			return true
		}
	}
	p := new(feePayment)
	p.Payer.Set(address)
	p.Amount.Set(steps)
	*d = append(*d, p)
	return true
}

func (d feeDetail) Has() bool {
	return len(d) > 0
}

func (d feeDetail) ToJSON(v module.JSONVersion) (interface{}, error) {
	jso := make(map[string]string)
	for _, p := range d {
		jso[p.Payer.String()] = p.Amount.String()
	}
	return jso, nil
}

func (d *feeDetail) UnmarshalJSON(s []byte) error {
	var jso map[string]string
	if err := json.Unmarshal(s, &jso); err != nil {
		return err
	}
	for addr, value := range jso {
		p := new(feePayment)
		if err := p.Payer.SetStringStrict(addr); err != nil {
			return errors.CriticalFormatError.Wrapf(err,
				"InvalidAddress(str=%q)", addr)
		}
		if _, ok := p.Amount.SetString(value, 0); !ok {
			return errors.CriticalFormatError.Errorf(
				"InvalidIntValue(str=%q)", addr)
		}
		*d = append(*d, p)
	}
	d.Normalize()
	return nil
}

func (d feeDetail) Normalize() {
	dl := []*feePayment(d)
	sort.Slice(dl, func(i, j int) bool {
		return bytes.Compare(dl[i].Payer.Bytes(), dl[j].Payer.Bytes()) < 0
	})
}

func (d *feeDetail) RLPEncodeSelf(e codec.Encoder) error {
	dl := []*feePayment(*d)
	return e.Encode(dl)
}

func (d *feeDetail) RLPDecodeSelf(e codec.Decoder) error {
	var dl []*feePayment
	if err := e.Decode(&dl); err != nil {
		return err
	}
	*d = dl
	return nil
}

func (d feeDetail) Iterator() module.FeePaymentIterator {
	return &feeIterator{
		feeDetail: d,
		index:     0,
	}
}

func (d feeDetail) GetStepsPaidByEOA() *big.Int {
	steps := new(big.Int)
	for _, v := range d {
		if !v.Payer.IsContract() {
			steps.Add(steps, v.Amount.Value())
		}
	}
	return steps
}

type feePaymentItem struct {
	*feePayment
}

func (i feePaymentItem) Payer() module.Address {
	return &i.feePayment.Payer
}

func (i feePaymentItem) Amount() *big.Int {
	return &i.feePayment.Amount.Int
}

type feeIterator struct {
	feeDetail feeDetail
	index     int
}

func (itr *feeIterator) Has() bool {
	return itr.index < len(itr.feeDetail)
}

func (itr *feeIterator) Next() error {
	if itr.index < len(itr.feeDetail) {
		itr.index = itr.index + 1
		return nil
	} else {
		return common.ErrInvalidState
	}
}

func (itr *feeIterator) Get() (module.FeePayment, error) {
	if itr.index < len(itr.feeDetail) {
		return feePaymentItem{itr.feeDetail[itr.index]}, nil
	} else {
		return nil, common.ErrInvalidState
	}
}
