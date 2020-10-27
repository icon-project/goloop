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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_depositList_Clone(t *testing.T) {
	tests := []struct {
		name string
		di   depositList
	}{
		{"Nil", nil},
		{"One", []deposit{
			{
				ID:             []byte{0x00, 0x12, 0x45},
				DepositAmount:  big.NewInt(10),
				DepositRemains: big.NewInt(10),
				CreatedHeight:  10,
				ExpireHeight:   1000,
				NextHeight:     10,
				StepIssued:     big.NewInt(123),
				StepRemains:    big.NewInt(456),
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.di.Clone(); !reflect.DeepEqual(got, tt.di) {
				t.Errorf("Clone() = %v, want %v", got, tt.di)
			}
		})
	}
}

type depositContext struct {
	price  *big.Int
	height int64
	period int64
	tid    []byte
}

func (d *depositContext) StepPrice() *big.Int {
	return d.price
}

func (d *depositContext) BlockHeight() int64 {
	return d.height
}

func (d *depositContext) DepositTerm() int64 {
	return d.period
}

var depositIssueRate = big.NewInt(8)

func (d *depositContext) DepositIssueRate() *big.Int {
	return depositIssueRate
}

func (d *depositContext) TransactionID() []byte {
	return d.tid
}

func TestDepositList_WithdrawDeposit1(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		price:  big.NewInt(100),
		height: 10,
		period: 100,
		tid:    tid1,
	}
	amount := big.NewInt(50007)

	t.Run("invalid withdraw", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// invalid withdraw test
		_, _, err = dl.WithdrawDeposit(dc, tid2)
		assert.Error(t, err)

		// withdraw normal
		_, _, err = dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)

		// withdraw duplicate
		_, _, err = dl.WithdrawDeposit(dc, tid1)
		assert.Error(t, err)
	})

	t.Run("withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// withdraw after expiration
		dc.height += dc.period
		am, tr, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.BitLen())
	})

	t.Run("withdraw before expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// withdraw before expiration without usage
		dc.height += dc.period / 2
		am, tr, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.BitLen())
	})

	t.Run("paying some and withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		dc.height += 1
		steps := big.NewInt(10)
		dl.PaySteps(dc, steps)

		dc.height += dc.period
		am, tr, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.BitLen())
	})

	t.Run("paying some and withdraw before expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// use some steps
		dc.height += 1
		steps := big.NewInt(10)
		dl.PaySteps(dc, steps)
		usedDeposit := new(big.Int).Mul(steps, dc.price)

		// withdrawal on not expired
		dc.height += 1

		am, tr, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, new(big.Int).Sub(amount, usedDeposit), am)
		assert.Equal(t, usedDeposit, tr)
	})

	t.Run("paying all and withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// use some steps
		dc.height += 1
		steps := new(big.Int).Mul(amount, big.NewInt(8))
		steps = steps.Div(steps, big.NewInt(100))
		steps = steps.Div(steps, dc.price)
		depositSteps, depositRemain := new(big.Int).DivMod(amount, dc.price, new(big.Int))
		steps = steps.Add(steps, depositSteps)
		payed := dl.PaySteps(dc, steps)
		assert.Equal(t, steps, payed)

		dc.height += dc.period

		am, _, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, 0, depositRemain.Cmp(am))
	})

	t.Run("paying all and withdraw before expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		// use some steps
		dc.height += 1
		steps := new(big.Int).Mul(amount, big.NewInt(8))
		steps = steps.Div(steps, big.NewInt(100))
		steps = steps.Div(steps, dc.price)
		depositSteps, _ := new(big.Int).DivMod(amount, dc.price, new(big.Int))
		steps = steps.Add(steps, depositSteps)
		payed := dl.PaySteps(dc, steps)
		assert.Equal(t, steps, payed)

		dc.height += 1
		am, _, err := dl.WithdrawDeposit(dc, tid1)
		assert.NoError(t, err)
		assert.Equal(t, 0, am.BitLen())
	})
}

func TestDepositList_WithdrawDeposit2(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		price:  big.NewInt(100),
		height: 10,
		period: 100,
		tid:    tid1,
	}
	amount := big.NewInt(50000)

	t.Run("continue pay with two deposit", func(t *testing.T) {
		dl := newDepositList()

		steps := new(big.Int).Mul(amount, dc.DepositIssueRate())
		steps = steps.Div(steps, bigInt100)
		steps = steps.Div(steps, dc.price)

		for i := 0; i < 10; i++ {
			dc.tid = tid1
			err := dl.AddDeposit(dc, amount, 1)
			assert.NoError(t, err)

			dc.height += 1
			dl.PaySteps(dc, steps)

			dc.height += dc.period/2 - 1
			dc.tid = tid2
			if i > 0 {
				am, fee, err := dl.WithdrawDeposit(dc, tid2)
				assert.NoError(t, err)
				assert.Equal(t, 0, fee.BitLen())
				assert.Equal(t, 0, am.Cmp(amount))
			}
			err = dl.AddDeposit(dc, amount, 1)
			assert.NoError(t, err)

			dc.height += 1
			dl.PaySteps(dc, steps)

			dc.height += dc.period/2 - 1
			am, fee, err := dl.WithdrawDeposit(dc, tid1)
			assert.NoError(t, err)
			assert.Equal(t, 0, fee.BitLen())
			assert.Equal(t, 0, am.Cmp(amount))
		}
	})

	t.Run("using two deposits", func(t *testing.T) {
		dl := newDepositList()

		dc.tid = tid1
		err := dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		dc.height += 1
		dc.tid = tid2
		err = dl.AddDeposit(dc, amount, 1)
		assert.NoError(t, err)

		steps := new(big.Int).Mul(amount, dc.DepositIssueRate())
		steps.Div(steps, bigInt100)
		steps.Div(steps, dc.price)
		steps.Mul(steps, big.NewInt(2))

		dc.height += 1
		payed := dl.PaySteps(dc, steps)
		assert.Equal(t, 0, steps.Cmp(payed))
	})
}
