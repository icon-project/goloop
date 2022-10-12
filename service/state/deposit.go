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

package state

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	DepositVersion1 int = iota
	DepositVersion2
)

type depositImpl interface {
	GetAvailableSteps(height int64) *big.Int
	GetAvailableDeposit(height int64) *big.Int
	GetUsableDeposit(height int64) *big.Int
	Version() int
	RLPDecodeFields(d codec.Decoder) error
	RLPEncodeFields(e codec.Encoder) error
	IsIdentifiedBy(id []byte) bool
	ToJSON(version module.JSONVersion) interface{}
	CanPay(height int64, feeLimit *big.Int) bool
	Add(rate, price, value *big.Int) error
	Withdraw(height int64, price, value *big.Int) (*big.Int, *big.Int, bool, error)
	ConsumeSteps(height int64, steps *big.Int) *big.Int
	ConsumeDepositLv1(height int64, amount *big.Int) *big.Int
	ConsumeDepositLv2(height int64, amount *big.Int) *big.Int
	Clone() depositImpl
	Equal(depositImpl) bool
}

type deposit struct {
	depositImpl
}

func (dp *deposit) Clone() *deposit {
	return &deposit{dp.depositImpl.Clone()}
}

func (dp *deposit) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.Encode(dp.depositImpl.Version()); err != nil {
		return err
	}
	return dp.depositImpl.RLPEncodeFields(e2)
}

func (dp *deposit) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var version int
	if err := d2.Decode(&version); err != nil {
		return err
	}
	switch version {
	case DepositVersion1:
		dp.depositImpl = new(depositV1)
	case DepositVersion2:
		dp.depositImpl = new(depositV2)
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidDepositVersion(version=%d)", version)
	}
	return dp.depositImpl.RLPDecodeFields(d2)
}

func (dp *deposit) Equal(dp2 *deposit) bool {
	return dp.depositImpl.Equal(dp2.depositImpl)
}

type depositV1 struct {
	ID            []byte
	DepositAmount *big.Int
	DepositRemain *big.Int
	ExpireHeight  int64
	StepIssued    *big.Int
	StepRemain    *big.Int
	isExhausted   bool
}

func (d *depositV1) Version() int {
	return DepositVersion1
}

func (d *depositV1) RLPDecodeFields(dec codec.Decoder) error {
	if _, err := dec.DecodeMulti(
		&d.ID,
		&d.DepositAmount,
		&d.DepositRemain,
		&d.ExpireHeight,
		&d.StepIssued,
		&d.StepRemain,
	); err != nil {
		return err
	}
	min := minDeposit(d.DepositAmount)
	d.isExhausted = d.DepositRemain.Cmp(min) <= 0
	return nil
}

func (d *depositV1) RLPEncodeFields(e codec.Encoder) error {
	return e.EncodeMulti(
		d.ID,
		d.DepositAmount,
		d.DepositRemain,
		d.ExpireHeight,
		d.StepIssued,
		d.StepRemain,
	)
}

var bigInt10 = big.NewInt(10)
var bigInt100 = big.NewInt(100)

func minDeposit(amount *big.Int) *big.Int {
	return new(big.Int).Div(amount, bigInt10)
}

func calcVirtualSteps(amount, rate, price *big.Int) *big.Int {
	if price.Sign() <= 0 {
		return new(big.Int)
	}
	issue := new(big.Int).Mul(amount, rate)
	issue.Div(issue, bigInt100)
	issue.Div(issue, price)
	return issue
}

func (d *depositV1) expireAt(height int64) bool {
	return height >= d.ExpireHeight
}

func (d *depositV1) GetAvailableSteps(height int64) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	return d.StepRemain
}

// ConsumeSteps consume virtual steps issuing if it's required.
// It returns remaining steps to pay
func (d *depositV1) ConsumeSteps(height int64, steps *big.Int) *big.Int {
	if d.expireAt(height) {
		return steps
	}

	if d.StepRemain.Sign() == 0 {
		return steps
	}
	if d.StepRemain.Cmp(steps) < 0 {
		steps = new(big.Int).Sub(steps, d.StepRemain)
		d.StepRemain = new(big.Int)
		return steps
	} else {
		d.StepRemain = new(big.Int).Sub(d.StepRemain, steps)
		return new(big.Int)
	}
}

func (d *depositV1) GetAvailableDeposit(height int64) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	return d.DepositRemain
}

func (d *depositV1) GetUsableDeposit(height int64) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	min := minDeposit(d.DepositAmount)
	if d.DepositRemain.Cmp(min) <= 0 {
		return new(big.Int)
	} else {
		return new(big.Int).Sub(d.DepositRemain, min)
	}
}

// ConsumeDepositLv1 pay fee with deposit.
// It returns remaining fee to consume.
func (d *depositV1) ConsumeDepositLv1(height int64, amount *big.Int) *big.Int {
	if d.expireAt(height) || d.isExhausted {
		return amount
	}

	min := minDeposit(d.DepositAmount)
	if d.DepositRemain.Cmp(min) <= 0 {
		return amount
	}
	payable := new(big.Int).Sub(d.DepositRemain, min)
	if payable.Cmp(amount) <= 0 {
		d.DepositRemain = new(big.Int).Sub(d.DepositRemain, payable)
		d.isExhausted = true
		return new(big.Int).Sub(amount, payable)
	} else {
		d.DepositRemain = new(big.Int).Sub(d.DepositRemain, amount)
		return new(big.Int)
	}
}

func (d *depositV1) ConsumeDepositLv2(height int64, amount *big.Int) *big.Int {
	if d.expireAt(height) {
		return amount
	}

	if d.DepositRemain.Cmp(amount) < 0 {
		payable := d.DepositRemain
		d.DepositRemain = new(big.Int)
		return new(big.Int).Sub(amount, payable)
	} else {
		d.DepositRemain = new(big.Int).Sub(d.DepositRemain, amount)
		return new(big.Int)
	}
}

func (d *depositV1) CanPay(height int64, feeLimit *big.Int) bool {
	if d.expireAt(height) || d.isExhausted {
		return false
	}
	return true
}

func (d *depositV1) ToJSON(v module.JSONVersion) interface{} {
	jso := make(map[string]interface{})
	jso["id"] = "0x" + hex.EncodeToString(d.ID)
	jso["depositAmount"] = intconv.FormatBigInt(d.DepositAmount)
	depositUsed := new(big.Int).Sub(d.DepositAmount, d.DepositRemain)
	jso["depositUsed"] = intconv.FormatBigInt(depositUsed)
	jso["expires"] = intconv.FormatInt(d.ExpireHeight)
	jso["virtualStepIssued"] = intconv.FormatBigInt(d.StepIssued)
	stepUsed := new(big.Int).Sub(d.StepIssued, d.StepRemain)
	jso["virtualStepUsed"] = intconv.FormatBigInt(stepUsed)
	return jso
}

func (d *depositV1) IsIdentifiedBy(id []byte) bool {
	return bytes.Equal(d.ID, id)
}

func (d *depositV1) Add(rate, price, value *big.Int) error {
	return scoreresult.UnknownFailureError.New("DuplicateDeposit")
}

func (d *depositV1) Withdraw(height int64, price, value *big.Int) (*big.Int, *big.Int, bool, error) {
	if value != nil {
		return nil, nil, false,
			scoreresult.InvalidParameterError.New("PartialWithdrawIsDenied")
	}
	amount := d.DepositRemain
	penalty := new(big.Int)
	if !d.expireAt(height) {
		penalty.Sub(d.StepIssued, d.StepRemain)
		penalty.Mul(penalty, price)
		if penalty.Cmp(amount) <= 0 {
			amount = new(big.Int).Sub(amount, penalty)
		} else {
			penalty = amount
			amount = new(big.Int)
		}
	}
	return amount, penalty, true, nil
}

func (d *depositV1) Clone() depositImpl {
	d2 := new(depositV1)
	*d2 = *d
	return d2
}

func (d *depositV1) Equal(d2 depositImpl) bool {
	if d2p, ok := d2.(*depositV1); !ok {
		return false
	} else {
		if d == d2p {
			return true
		}
		return bytes.Equal(d.ID, d2p.ID) &&
			d.DepositAmount.Cmp(d2p.DepositAmount) == 0 &&
			d.DepositRemain.Cmp(d2p.DepositRemain) == 0 &&
			d.ExpireHeight == d2p.ExpireHeight &&
			d.StepIssued.Cmp(d2p.StepIssued) == 0 &&
			d.StepRemain.Cmp(d2p.StepRemain) == 0
	}
}

func newDepositV1(dc DepositContext, value *big.Int) (*depositV1, error) {
	issue := calcVirtualSteps(value, dc.DepositIssueRate(), dc.StepPrice())
	return &depositV1{
		ID:            dc.TransactionID(),
		ExpireHeight:  dc.BlockHeight() + dc.DepositTerm(),
		DepositAmount: value,
		DepositRemain: value,
		StepIssued:    issue,
		StepRemain:    issue,
	}, nil
}

type depositV2 struct {
	DepositRemain *big.Int
}

func (d *depositV2) ToJSON(version module.JSONVersion) interface{} {
	return map[string]interface{}{
		"depositRemain": intconv.FormatBigInt(d.DepositRemain),
	}
}

func (d *depositV2) CanPay(height int64, feeLimit *big.Int) bool {
	return d.DepositRemain.Cmp(feeLimit) >= 0
}

func (d *depositV2) GetAvailableSteps(height int64) *big.Int {
	return new(big.Int)
}

func (d *depositV2) GetAvailableDeposit(height int64) *big.Int {
	return d.DepositRemain
}

func (d *depositV2) GetUsableDeposit(height int64) *big.Int {
	return d.DepositRemain
}

func (d *depositV2) RLPDecodeFields(dec codec.Decoder) error {
	return dec.Decode(&d.DepositRemain)
}

func (d *depositV2) RLPEncodeFields(e codec.Encoder) error {
	return e.Encode(d.DepositRemain)
}

func (d *depositV2) IsIdentifiedBy(id []byte) bool {
	return len(id) == 0
}

func (d *depositV2) Add(rate, price, value *big.Int) error {
	d.DepositRemain = new(big.Int).Add(d.DepositRemain, value)
	return nil
}

func (d *depositV2) Withdraw(height int64, price, value *big.Int) (*big.Int, *big.Int, bool, error) {
	amount := value
	if amount == nil {
		amount = d.DepositRemain
	}
	if cmp := d.DepositRemain.Cmp(amount); cmp > 0 {
		d.DepositRemain = new(big.Int).Sub(d.DepositRemain, amount)
		return amount, new(big.Int), false, nil
	} else if cmp == 0 {
		if value != nil {
			d.DepositRemain = new(big.Int)
		}
		return amount, new(big.Int), value == nil, nil
	} else {
		return nil, nil, false, scoreresult.OutOfBalanceError.New("NotEnoughBalance")
	}
}

func (d *depositV2) ConsumeSteps(height int64, steps *big.Int) *big.Int {
	return steps
}

func (d *depositV2) ConsumeDepositLv1(height int64, amount *big.Int) *big.Int {
	if d.DepositRemain.Cmp(amount) <= 0 {
		remains := new(big.Int).Sub(amount, d.DepositRemain)
		d.DepositRemain = new(big.Int)
		return remains
	} else {
		d.DepositRemain = new(big.Int).Sub(d.DepositRemain, amount)
		return new(big.Int)
	}
}

func (d *depositV2) ConsumeDepositLv2(height int64, amount *big.Int) *big.Int {
	return amount
}

func (d *depositV2) Version() int {
	return DepositVersion2
}

func (d *depositV2) Clone() depositImpl {
	d2 := new(depositV2)
	*d2 = *d
	return d2
}

func (d *depositV2) Equal(d2 depositImpl) bool {
	if d2p, ok := d2.(*depositV2); !ok {
		return false
	} else {
		return d.DepositRemain.Cmp(d2p.DepositRemain) == 0
	}
}

func newDepositV2(dc DepositContext, value *big.Int) (*depositV2, error) {
	return &depositV2{
		DepositRemain: value,
	}, nil
}
