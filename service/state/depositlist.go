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
	"reflect"

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
	return reflect.DeepEqual([]*deposit(dl), []*deposit(di2))
}

func (dl depositList) Clone() depositList {
	if dl == nil {
		return nil
	}
	deposits := make([]*deposit, len(dl))
	copy(deposits, dl)
	return depositList(deposits)
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
	return nil, nil, scoreresult.InvalidParameterError.New("DepositNotFound")
}

func (dl depositList) getAvailableDeposit(bh int64) *big.Int {
	deposit := new(big.Int)
	for _, dp := range dl {
		deposit.Add(deposit, dp.GetAvailableDeposit(bh))
	}
	return deposit
}

// PaySteps returns consumes virtual steps and also deposits.
// It returns payed steps
func (dl *depositList) PaySteps(dc DepositContext, steps *big.Int) *big.Int {
	bh := dc.BlockHeight()
	price := dc.StepPrice()

	// Unable to pay with non positive price or empty deposit list.
	if price.Sign() <= 0 || !dl.Has() {
		return nil
	}

	// pay steps with issued virtual steps
	remains := steps
	for idx, _ := range *dl {
		dp := (*dl)[idx]
		remains = dp.ConsumeSteps(bh, remains)
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
		dp := (*dl)[idx]
		fee = dp.ConsumeDepositLv1(bh, fee)
		if fee.BitLen() == 0 {
			break
		}
	}
	if fee.BitLen() != 0 {
		for idx, _ := range *dl {
			dp := (*dl)[idx]
			fee = dp.ConsumeDepositLv2(bh, fee)
			if fee.BitLen() == 0 {
				return steps
			}
		}
	}

	return new(big.Int).Sub(steps, remains)
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
