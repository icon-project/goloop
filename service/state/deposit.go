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
	"fmt"
	"math/big"
	"reflect"

	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type deposit struct {
	ID             []byte
	DepositAmount  *big.Int
	DepositRemains *big.Int
	CreatedHeight  int64
	ExpireHeight   int64
	StepIssued     *big.Int
	StepRemains    *big.Int
	NextHeight     int64
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

func (d *deposit) isInNewTerm(height int64) bool {
	return d.NextHeight != 0 && height >= d.NextHeight
}

func (d *deposit) expireAt(height int64) bool {
	return d.ExpireHeight > 0 && height >= d.ExpireHeight
}

func (d *deposit) isExhausted() bool {
	return d.NextHeight == 0
}

func (d *deposit) updateSteps(height, duration int64, rate, price *big.Int) {
	if d.isInNewTerm(height) {
		issue := calcVirtualSteps(d.DepositRemains, rate, price)
		d.StepIssued = issue
		d.StepRemains = issue

		if duration != 0 {
			heightDiff := duration + height - d.NextHeight
			d.NextHeight += heightDiff - (heightDiff % duration)
		} else {
			d.NextHeight = 0
		}
	}
}

func (d *deposit) getAvailableSteps(height int64, rate, price *big.Int) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	if d.isInNewTerm(height) {
		return calcVirtualSteps(d.DepositRemains, rate, price)
	} else {
		return d.StepRemains
	}
}

// ConsumeSteps consume virtual steps issuing if it's required.
// It returns remaining steps to pay
func (d *deposit) ConsumeSteps(height, duration int64, rate, price, steps *big.Int) *big.Int {
	if d.expireAt(height) {
		return steps
	}

	d.updateSteps(height, duration, rate, price)
	if d.StepRemains.BitLen() == 0 {
		return steps
	}
	if d.StepRemains.Cmp(steps) < 0 {
		steps = new(big.Int).Sub(steps, d.StepRemains)
		d.StepRemains = new(big.Int)
		return steps
	} else {
		d.StepRemains = new(big.Int).Sub(d.StepRemains, steps)
		return new(big.Int)
	}
}

func (d *deposit) getAvailableDeposit(height int64) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	return d.DepositRemains
}

func (d *deposit) getUsableDeposit(height int64) *big.Int {
	if d.expireAt(height) {
		return new(big.Int)
	}
	min := minDeposit(d.DepositAmount)
	if d.DepositRemains.Cmp(min) <= 0 {
		return new(big.Int)
	} else {
		return new(big.Int).Sub(d.DepositRemains, min)
	}
}

// ConsumeDepositLv1 pay fee with deposit.
// It returns remaining fee to consume.
func (d *deposit) ConsumeDepositLv1(height int64, amount *big.Int) *big.Int {
	if d.expireAt(height) || d.isExhausted() {
		return amount
	}

	min := minDeposit(d.DepositAmount)
	if d.DepositRemains.Cmp(min) <= 0 {
		return amount
	}
	payable := new(big.Int).Sub(d.DepositRemains, min)
	if payable.Cmp(amount) < 0 {
		d.DepositRemains = new(big.Int).Sub(d.DepositRemains, payable)
		return new(big.Int).Sub(amount, payable)
	} else {
		d.DepositRemains = new(big.Int).Sub(d.DepositRemains, amount)
		return new(big.Int)
	}
}

func (d *deposit) ConsumeDepositLv2(height int64, amount *big.Int) *big.Int {
	if d.expireAt(height) {
		return amount
	}
	d.NextHeight = 0
	if d.DepositRemains.Cmp(amount) < 0 {
		payable := d.DepositRemains
		d.DepositRemains = new(big.Int)
		return new(big.Int).Sub(amount, payable)
	} else {
		d.DepositRemains = new(big.Int).Sub(d.DepositRemains, amount)
		return new(big.Int)
	}
}

func (d *deposit) CanPay(height int64) bool {
	if d.expireAt(height) || d.isExhausted() {
		return false
	}
	return true
}

func (d *deposit) ToJSON(height int64, rate, price *big.Int, v module.JSONVersion) interface{} {
	jso := make(map[string]interface{})
	jso["id"] = "0x" + hex.EncodeToString(d.ID)
	jso["depositAmount"] = intconv.FormatBigInt(d.DepositAmount)
	depositUsed := new(big.Int).Sub(d.DepositAmount, d.DepositRemains)
	jso["depositUsed"] = intconv.FormatBigInt(depositUsed)
	jso["created"] = intconv.FormatInt(d.CreatedHeight)
	jso["expires"] = intconv.FormatInt(d.ExpireHeight)
	if d.isInNewTerm(height) && !d.expireAt(height) {
		issued := intconv.FormatBigInt(d.getAvailableSteps(height, rate, price))
		jso["virtualStepIssued"] = issued
		jso["virtualStepUsed"] = "0x0"
	} else {
		jso["virtualStepIssued"] = intconv.FormatBigInt(d.StepIssued)
		stepUsed := new(big.Int).Sub(d.StepIssued, d.StepRemains)
		jso["virtualStepUsed"] = intconv.FormatBigInt(stepUsed)
	}
	return jso
}

type DepositContext interface {
	StepPrice() *big.Int
	BlockHeight() int64
	DepositTerm() int64
	DepositIssueRate() *big.Int
	TransactionID() []byte
}

type depositList []deposit

func (dl depositList) Has() bool {
	return len(dl) > 0
}

func (dl depositList) Equal(di2 depositList) bool {
	return reflect.DeepEqual([]deposit(dl), []deposit(di2))
}

func (dl depositList) Clone() depositList {
	if dl == nil {
		return nil
	}
	deposits := make([]deposit, len(dl))
	copy(deposits, dl)
	return depositList(deposits)
}

func (dl *depositList) AddDeposit(dc DepositContext, value *big.Int, period int64) error {
	for _, dp := range *dl {
		if bytes.Equal(dp.ID, dc.TransactionID()) {
			return scoreresult.UnknownFailureError.New("DuplicateDeposit")
		}
	}
	issue := calcVirtualSteps(value, dc.DepositIssueRate(), dc.StepPrice())
	d := deposit{
		ID:             dc.TransactionID(),
		CreatedHeight:  dc.BlockHeight(),
		ExpireHeight:   dc.BlockHeight() + dc.DepositTerm()*period,
		NextHeight:     dc.BlockHeight() + dc.DepositTerm(),
		DepositAmount:  value,
		DepositRemains: value,
		StepIssued:     issue,
		StepRemains:    issue,
	}
	*dl = append(*dl, d)
	return nil
}

// WithdrawDeposit withdraw the deposit specified by id.
// It returns amount of deposit, fee to be charged and error object
func (dl *depositList) WithdrawDeposit(dc DepositContext, id []byte) (*big.Int, *big.Int, error) {
	deposits := *dl
	for idx, dp := range deposits {
		if bytes.Equal(dp.ID, id) {
			// remove item from the list
			copy(deposits[idx:], deposits[idx+1:])
			deposits = deposits[0 : len(deposits)-1]
			if len(deposits) > 0 {
				*dl = deposits
			} else {
				*dl = nil
			}

			// refund
			amount := dp.DepositRemains
			penalty := new(big.Int)
			bh := dc.BlockHeight()
			if !(dp.expireAt(bh) || dp.isInNewTerm(bh)) {
				penalty.Sub(dp.StepIssued, dp.StepRemains)
				penalty.Mul(penalty, dc.StepPrice())
				if penalty.Cmp(amount) <= 0 {
					amount = new(big.Int).Sub(amount, penalty)
				} else {
					penalty = amount
					amount = new(big.Int)
				}
			}
			return amount, penalty, nil
		} else {
			fmt.Printf("SKIP %#x", dp.ID)
		}
	}
	return nil, nil, scoreresult.InvalidParameterError.New("DepositNotFound")
}

func (dl depositList) getAvailableDeposit(bh int64) *big.Int {
	deposit := new(big.Int)
	for _, dp := range dl {
		deposit.Add(deposit, dp.getAvailableDeposit(bh))
	}
	return deposit
}

// PaySteps returns consumes virtual steps and also deposits.
// It returns payed steps
func (dl *depositList) PaySteps(dc DepositContext, steps *big.Int) *big.Int {
	bh := dc.BlockHeight()
	period := dc.DepositTerm()
	rate := dc.DepositIssueRate()
	price := dc.StepPrice()

	// Unable to pay with non positive price or empty deposit list.
	if price.Sign() <= 0 || !dl.Has() {
		return nil
	}

	// pay steps with issued virtual steps
	remains := steps
	for idx, _ := range *dl {
		dp := &(*dl)[idx]
		remains = dp.ConsumeSteps(bh, period, rate, price, remains)
		if remains.BitLen() == 0 {
			return steps
		}
	}

	// calculate fee to charge
	deposit := dl.getAvailableDeposit(bh)
	payableSteps := new(big.Int).Div(deposit, price)
	var fee *big.Int
	if payableSteps.Cmp(remains) < 0 {
		fee = new(big.Int).Mul(payableSteps, price)
		remains = new(big.Int).Sub(remains, payableSteps)
	} else {
		fee = new(big.Int).Mul(remains, price)
		remains = new(big.Int)
	}

	// charge fee
	for idx, _ := range *dl {
		dp := &(*dl)[idx]
		fee = dp.ConsumeDepositLv1(bh, fee)
		if fee.BitLen() == 0 {
			break
		}
	}
	if fee.BitLen() != 0 {
		for idx, _ := range *dl {
			dp := &(*dl)[idx]
			fee = dp.ConsumeDepositLv2(bh, fee)
			if fee.BitLen() == 0 {
				return steps
			}
		}
	}

	return new(big.Int).Sub(steps, remains)
}

func (dl depositList) CanPay(height int64) bool {
	if len(dl) > 0 {
		for _, d := range dl {
			if d.CanPay(height) {
				return true
			}
		}
		return false
	}
	return true
}

func (dl depositList) ToJSON(dc DepositContext, v module.JSONVersion) (map[string]interface{}, error) {
	if len(dl) == 0 {
		return nil, nil
	}
	jso := make(map[string]interface{})
	deposits := make([]interface{}, len(dl))

	availSteps := new(big.Int)
	availDeps := new(big.Int)

	bh := dc.BlockHeight()
	rate := dc.DepositIssueRate()
	price := dc.StepPrice()

	for idx, p := range dl {
		deposits[idx] = p.ToJSON(bh, rate, price, v)
		availSteps.Add(availSteps, p.getAvailableSteps(bh, rate, price))
		availDeps.Add(availDeps, p.getUsableDeposit(bh))
	}
	jso["deposits"] = deposits
	jso["availableVirtualStep"] = intconv.FormatBigInt(availSteps)
	jso["availableDeposit"] = intconv.FormatBigInt(availDeps)

	return jso, nil
}

func newDepositList() depositList {
	return nil
}
