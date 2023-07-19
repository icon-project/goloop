/*
 * Copyright 2022 ICON Foundation
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

package contract

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type payAccountState struct {
	state.AccountState
	store   map[string][]byte
	deposit *big.Int
}

func (as *payAccountState) ensureStore() {
	if as.store == nil {
		as.store = make(map[string][]byte)
	}
}

func (as *payAccountState) GetValue(k []byte) ([]byte, error) {
	as.ensureStore()
	return as.store[string(k)], nil
}

func (as *payAccountState) SetValue(k, v []byte) ([]byte, error) {
	as.ensureStore()
	ks := string(k)
	old := as.store[ks]
	as.store[ks] = v
	return old, nil
}

func (as *payAccountState) DeleteValue(k []byte) ([]byte, error) {
	as.ensureStore()
	ks := string(k)
	old := as.store[ks]
	delete(as.store, ks)
	return old, nil
}

func (as *payAccountState) PaySteps(dc state.PayContext, steps *big.Int) (*big.Int, *big.Int, error) {
	if as.deposit != nil {
		var toPay *big.Int
		if as.deposit.Cmp(steps) <= 0 {
			toPay = as.deposit
			as.deposit = nil
		} else {
			toPay = steps
			as.deposit = new(big.Int).Sub(as.deposit, steps)
		}
		return toPay, toPay, nil
	}
	return new(big.Int), nil, nil
}

func newTestAccount() *payAccountState {
	return &payAccountState{
		store: make(map[string][]byte),
	}
}

type payCallContext struct {
	CallContext
	accounts  map[string]*payAccountState
	stepPrice *big.Int
}

func (cc *payCallContext) GetAccountState(id []byte) state.AccountState {
	ids := string(id)
	if as, ok := cc.accounts[ids]; ok {
		return as
	} else {
		as = newTestAccount()
		cc.accounts[ids] = as
		return as
	}
}

func (cc *payCallContext) StepPrice() *big.Int {
	return cc.stepPrice
}

type feePayment struct {
	Payer module.Address
	Paid  *big.Int
	Fee   *big.Int
}

type payReceipt struct {
	txresult.Receipt
	payments []*feePayment
}

func (r *payReceipt) AddPayment(addr module.Address, steps *big.Int, feeSteps *big.Int) {
	r.payments = append(r.payments, &feePayment{
		addr, steps, feeSteps,
	})
}

func TestFeePayerInfo_PaySteps(t *testing.T) {
	contract1 := common.MustNewAddressFromString("cx12")
	contract1IDStr := string(contract1.ID())
	contract2 := common.MustNewAddressFromString("cx34")
	contract2IDStr := string(contract2.ID())

	t.Run("simple_pay_all", func(t *testing.T) {
		var base FeePayerInfo

		cc := &payCallContext{
			accounts: map[string]*payAccountState{
				contract1IDStr: {
					deposit: big.NewInt(1000),
				},
			},
			stepPrice: big.NewInt(10),
		}

		err := base.SetFeeProportion(contract1, 100)
		assert.NoError(t, err)

		steps := big.NewInt(700)
		paid, err := base.PaySteps(cc, steps)
		assert.NoError(t, err)
		assert.Equal(t, steps, paid)
		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.True(t, hasLog)
		expectedPayments := []*feePayment {
			{ contract1, paid, paid },
		}
		assert.Equal(t, expectedPayments, rct.payments)
	})

	t.Run("no_set_in_middle", func(t *testing.T) {
		// Scenario
		//  fee payer info      | own   | frame | prop
		//  base                | 1001  | 1576  | c2 100%
		//      \-> call1       | 11    | 575   |
		//          \-> call2   | 564   | 564   | c1 100%
		var base FeePayerInfo
		var call1 FeePayerInfo
		var call2 FeePayerInfo
		var err error

		err = call2.SetFeeProportion(contract1, 100)
		assert.NoError(t, err)
		call1.Apply(call2, big.NewInt(564))

		err = base.SetFeeProportion(contract2, 100)
		assert.NoError(t, err)
		base.Apply(call1, big.NewInt(1576))

		cc := &payCallContext{
			accounts: map[string]*payAccountState{
				contract1IDStr: {
					deposit: big.NewInt(565),
				},
				contract2IDStr: {
					deposit: big.NewInt(1013),
				},
			},
			stepPrice: big.NewInt(10),
		}
		steps, err := base.PaySteps(cc, big.NewInt(1576))
		assert.NoError(t, err)
		assert.True(t, big.NewInt(1576).Cmp(steps) == 0)

		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.True(t, hasLog)

		expectedPayments := []*feePayment{
			{contract1, big.NewInt(564), big.NewInt(564)},
			{contract2, big.NewInt(1012), big.NewInt(1012)},
		}
		sort.SliceStable(rct.payments, func(i, j int) bool {
			return bytes.Compare(rct.payments[i].Payer.Bytes(), rct.payments[j].Payer.Bytes()) < 0
		})
		assert.Equal(t, expectedPayments, rct.payments)
	})
	t.Run("no_deposit_test", func(t *testing.T) {
		// Scenario
		//  fee payer info          | own   | frame | prop      |
		//  base                    | 1098  | 1419  |           |
		//      \-> call1           | 121   | 321   |           |
		//          \-> call2       | 101   | 200   | c1 50%    |
		//              \-> call2   | 99    | 99    | c2 100%   |
		// c2 has no deposit
		// c1 has 200 deposit
		var base FeePayerInfo
		var call1 FeePayerInfo
		var call2 FeePayerInfo
		var call3 FeePayerInfo

		var err error

		err = call3.SetFeeProportion(contract2, 100)
		assert.NoError(t, err)
		call2.Apply(call3, big.NewInt(99))

		err = call2.SetFeeProportion(contract1, 50)
		assert.NoError(t, err)
		call1.Apply(call2, big.NewInt(200))

		base.Apply(call1, big.NewInt(321))
		err = base.SetFeeProportion(state.SystemAddress, 50)
		assert.NoError(t, err)

		cc := &payCallContext{
			accounts: map[string]*payAccountState{
				contract1IDStr: {
					deposit: big.NewInt(201),
				},
				contract2IDStr: {
					deposit: big.NewInt(0),
				},
			},
			stepPrice: big.NewInt(10),
		}
		steps, err := base.PaySteps(cc, big.NewInt(1419))
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(759), steps)

		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.True(t, hasLog)

		expectedPayments := []*feePayment{
			{state.SystemAddress, big.NewInt(659), nil},
			{contract1, big.NewInt(100), big.NewInt(100)},
		}
		sort.SliceStable(rct.payments, func(i, j int) bool {
			return bytes.Compare(rct.payments[i].Payer.Bytes(), rct.payments[j].Payer.Bytes()) < 0
		})
		assert.Equal(t, expectedPayments, rct.payments)
	})

	t.Run("complex", func(t *testing.T) {
		// Scenario
		//  fee payer info          | own   | frame | prop      |
		//  base                    | 1098  | 2767  | c2 70%    |
		//      \-> call1           | 653   | 1669  | c1 50%    |
		//          \-> call2_1     | 784   | 784   | s  50%    |
		//          \-> call2_2     | 232   | 232   | c2 100%   |
		//
		//  c1  : (653+392)//2 = 522
		//  c2  : 232+1134 = 1366
		//  s   : 392
		//  u   : 487
		var base FeePayerInfo
		var call1 FeePayerInfo
		var call2_1 FeePayerInfo
		var call2_2 FeePayerInfo

		var err error

		err = call2_1.SetFeeProportion(state.SystemAddress, 50)
		assert.NoError(t, err)
		call1.Apply(call2_1, big.NewInt(784))

		err = call2_2.SetFeeProportion(contract2, 100)
		assert.NoError(t, err)
		call1.Apply(call2_2, big.NewInt(232))

		err = call1.SetFeeProportion(contract1, 50)
		assert.NoError(t, err)
		base.Apply(call1, big.NewInt(1669))

		err = base.SetFeeProportion(contract2, 70)
		assert.NoError(t, err)

		cc := &payCallContext{
			accounts: map[string]*payAccountState{
				contract1IDStr: {
					deposit: big.NewInt(523),
				},
				contract2IDStr: {
					deposit: big.NewInt(1367),
				},
			},
			stepPrice: big.NewInt(10),
		}
		steps, err := base.PaySteps(cc, big.NewInt(2767))
		assert.NoError(t, err)
		assert.True(t, big.NewInt(2280).Cmp(steps) == 0)

		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.True(t, hasLog)

		expectedPayments := []*feePayment{
			{state.SystemAddress, big.NewInt(392), nil},
			{contract1, big.NewInt(522), big.NewInt(522)},
			{contract2, big.NewInt(1366), big.NewInt(1366)},
		}
		sort.SliceStable(rct.payments, func(i, j int) bool {
			return bytes.Compare(rct.payments[i].Payer.Bytes(), rct.payments[j].Payer.Bytes()) < 0
		})
		assert.Equal(t, expectedPayments, rct.payments)
	})
}

func TestFeePayerInfo_GetLogs(t *testing.T) {
	contract1 := common.MustNewAddressFromString("cx12")
	contract1IDStr := string(contract1.ID())
	t.Run("no_entry_get_logs", func(t *testing.T) {
		var base FeePayerInfo
		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.False(t, hasLog)
	});
	t.Run("entry_no_pay_get_logs", func(t *testing.T) {
		var base FeePayerInfo
		err := base.SetFeeProportion(contract1, 100)
		assert.NoError(t, err)

		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.False(t, hasLog)
	});
	t.Run("entry_pay_get_logs", func(t *testing.T) {
		var base FeePayerInfo
		err := base.SetFeeProportion(contract1, 100)
		assert.NoError(t, err)

		cc := &payCallContext{
			accounts: map[string]*payAccountState{
				contract1IDStr: {
					deposit: big.NewInt(100),
				},
			},
			stepPrice: big.NewInt(10),
		}
		expSteps := big.NewInt(100)
		steps := big.NewInt(1000)
		paid, err := base.PaySteps(cc, steps)
		assert.NoError(t, err)
		assert.Equal(t, expSteps, paid)

		rct := new(payReceipt)
		hasLog := base.GetLogs(rct)
		assert.True(t, hasLog)
		expectedPayments := []*feePayment{
			{contract1, expSteps, expSteps},
		}
		assert.Equal(t, expectedPayments, rct.payments)
	});
}