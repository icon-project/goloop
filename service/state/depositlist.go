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
	"math/big"

	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type DepositContext interface {
	StepPrice() *big.Int
	BlockHeight() int64
	DepositTerm() int64
	DepositIssueRate() *big.Int
	TransactionID() []byte
}

type PayContext interface {
	FeeSharingEnabled() bool
	StepPrice() *big.Int
	FeeLimit() *big.Int
	BlockHeight() int64
}

type depositList []*deposit

func (dl depositList) Has() bool {
	return len(dl) > 0
}

func (dl depositList) Equal(di2 depositList) bool {
	if len(dl) != len(di2) {
		return false
	}
	for idx, dp := range dl {
		if !dp.Equal(di2[idx]) {
			return false
		}
	}
	return true
}

func (dl depositList) Clone() depositList {
	if dl == nil {
		return nil
	}
	deposits := make([]*deposit, len(dl))
	for i, d := range dl {
		deposits[i] = d.Clone()
	}
	return deposits
}

func (dl *depositList) AddDeposit(dc DepositContext, value *big.Int) error {
	term := dc.DepositTerm()
	tid := dc.TransactionID()
	if term == 0 {
		tid = []byte{}
	}

	for _, dp := range *dl {
		if dp.IsIdentifiedBy(tid) {
			return dp.Add(dc.DepositIssueRate(), dc.StepPrice(), value)
		}
	}

	var dp depositImpl
	var err error
	if term == 0 {
		dp, err = newDepositV2(dc, value)
	} else {
		dp, err = newDepositV1(dc, value)
	}
	if err != nil {
		return err
	}
	*dl = append(*dl, &deposit{dp})
	return nil
}

// WithdrawDeposit withdraw the deposit specified by id.
// It returns amount of deposit, fee to be charged and error object
func (dl *depositList) WithdrawDeposit(dc DepositContext, id []byte, value *big.Int) (*big.Int, *big.Int, error) {
	deposits := *dl
	if value != nil && value.Sign() < 0 {
		return nil, nil, scoreresult.InvalidRequestError.Errorf("InvalidAmount(value=%d)", value)
	}
	for idx, dp := range deposits {
		if dp.IsIdentifiedBy(id) {
			amount, penalty, removal, err :=
				dp.Withdraw(dc.BlockHeight(), dc.StepPrice(), value)
			if err != nil {
				return nil, nil, err
			}
			if removal {
				copy(deposits[idx:], deposits[idx+1:])
				deposits = deposits[0 : len(deposits)-1]
				if len(deposits) > 0 {
					*dl = deposits
				} else {
					*dl = nil
				}
			}
			return amount, penalty, nil
		}
	}
	return nil, nil, scoreresult.InvalidRequestError.New("DepositNotFound")
}

func (dl depositList) getAvailableDeposit(bh int64) *big.Int {
	deposit := new(big.Int)
	for _, dp := range dl {
		deposit.Add(deposit, dp.GetAvailableDeposit(bh))
	}
	return deposit
}

// PaySteps returns consumes virtual steps and also deposits.
// It returns paid steps
func (dl *depositList) PaySteps(pc PayContext, steps *big.Int) (*big.Int, *big.Int) {
	bh := pc.BlockHeight()
	price := pc.StepPrice()

	// Unable to pay with non positive price or empty deposit list.
	if price.Sign() <= 0 || !dl.Has() {
		return nil, nil
	}

	// pay steps with issued virtual steps
	remains := steps
	for idx, _ := range *dl {
		dp := (*dl)[idx]
		remains = dp.ConsumeSteps(bh, remains)
		if remains.Sign() == 0 {
			return steps, nil
		}
	}

	// calculate fee to charge
	deposit := dl.getAvailableDeposit(bh)
	payableSteps := new(big.Int).Div(deposit, price)
	var stepsByDeposit *big.Int
	var paidSteps *big.Int
	if payableSteps.Cmp(remains) < 0 {
		stepsByDeposit = payableSteps
		paidSteps = new(big.Int).Sub(steps, remains)
		paidSteps = paidSteps.Add(paidSteps, stepsByDeposit)
	} else {
		stepsByDeposit = remains
		paidSteps = steps
	}
	fee := new(big.Int).Mul(stepsByDeposit, price)

	// pay fee with deposits
	for idx, _ := range *dl {
		dp := (*dl)[idx]
		fee = dp.ConsumeDepositLv1(bh, fee)
		if fee.Sign() == 0 {
			break
		}
	}
	if fee.Sign() != 0 {
		for idx, _ := range *dl {
			dp := (*dl)[idx]
			fee = dp.ConsumeDepositLv2(bh, fee)
			if fee.Sign() == 0 {
				break
			}
		}
	}

	return paidSteps, stepsByDeposit
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

	for idx, p := range dl {
		deposits[idx] = p.ToJSON(v)
		availSteps.Add(availSteps, p.GetAvailableSteps(bh))
		availDeps.Add(availDeps, p.GetUsableDeposit(bh))
	}
	jso["deposits"] = deposits
	jso["availableVirtualStep"] = intconv.FormatBigInt(availSteps)
	jso["availableDeposit"] = intconv.FormatBigInt(availDeps)

	return jso, nil
}

func (dl depositList) CanPay(pc PayContext) bool {
	height := pc.BlockHeight()
	limit := pc.FeeLimit()
	for _, d := range dl {
		if d.CanPay(height, limit) {
			return true
		}
	}
	return false
}

func newDepositList() depositList {
	return nil
}
