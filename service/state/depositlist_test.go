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
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testDepositTerm = 100
)

type depositContext struct {
	price  *big.Int
	height int64
	period int64
	tid    []byte
	rate   *big.Int
	off    bool
}

func (d *depositContext) FeeLimit() *big.Int {
	return d.price
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
	return d.rate
}

func (d *depositContext) TransactionID() []byte {
	return d.tid
}

func (d *depositContext) FeeSharingEnabled() bool {
	return !d.off
}

func TestDepositList_WithdrawDepositV1(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		rate:   depositIssueRate,
		price:  big.NewInt(100),
		height: 10,
		period: 100,
		tid:    tid1,
	}
	amount := big.NewInt(50007)

	t.Run("duplicate add", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())

		err = dl.AddDeposit(dc, amount)
		assert.Error(t, err)
	})

	t.Run("invalid withdraw", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// invalid withdraw test
		_, _, err = dl.WithdrawDeposit(dc, tid2, nil)
		assert.Error(t, err)

		// withdraw normal
		_, _, err = dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)

		// withdraw duplicate
		_, _, err = dl.WithdrawDeposit(dc, tid1, nil)
		assert.Error(t, err)
	})

	t.Run("withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// withdraw after expiration
		dc.height += dc.period
		am, tr, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.Sign())
	})

	t.Run("withdraw before expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// withdraw before expiration without usage
		dc.height += dc.period / 2
		am, tr, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.Sign())
	})

	t.Run("paying some and withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		dc.height += 1
		steps := big.NewInt(10)
		dl.PaySteps(dc, steps)

		dc.height += dc.period
		am, tr, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.Sign())
	})

	t.Run("paying some and withdraw before expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// use some steps
		dc.height += 1
		steps := big.NewInt(10)
		dl.PaySteps(dc, steps)
		usedDeposit := new(big.Int).Mul(steps, dc.price)

		// withdrawal on not expired
		dc.height += 1

		am, tr, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, new(big.Int).Sub(amount, usedDeposit), am)
		assert.Equal(t, usedDeposit, tr)
	})

	t.Run("paying all and withdraw after expiration", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// use some steps
		dc.height += 1
		steps := new(big.Int).Mul(amount, big.NewInt(8))
		steps = steps.Div(steps, big.NewInt(100))
		steps = steps.Div(steps, dc.price)
		depositSteps, depositRemain := new(big.Int).DivMod(amount, dc.price, new(big.Int))
		steps = steps.Add(steps, depositSteps)
		paidSteps, stepsByDeposit := dl.PaySteps(dc, steps)
		assert.Equal(t, steps, paidSteps)
		assert.Equal(t, depositSteps, stepsByDeposit)

		dc.height += dc.period

		am, _, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, depositRemain.Cmp(am))
	})
}

func TestDepositList_WithdrawDeposit2(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		rate:   depositIssueRate,
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

		depositSteps := new(big.Int)

		for i := 0; i < 10; i++ {
			dc.tid = tid1
			err := dl.AddDeposit(dc, amount)
			assert.NoError(t, err)

			dc.height += 1
			if _, ds := dl.PaySteps(dc, steps); ds != nil {
				depositSteps.Add(depositSteps, ds)
			}

			dc.height += dc.period/2 - 1
			dc.tid = tid2
			if i > 0 {
				am, fee, err := dl.WithdrawDeposit(dc, tid2, nil)
				assert.NoError(t, err)
				assert.Equal(t, 0, fee.Sign())
				assert.Equal(t, 0, am.Cmp(amount))
			}
			err = dl.AddDeposit(dc, amount)
			assert.NoError(t, err)

			dc.height += 1
			if _, ds := dl.PaySteps(dc, steps); ds != nil {
				depositSteps.Add(depositSteps, ds)
			}

			dc.height += dc.period/2 - 1
			am, fee, err := dl.WithdrawDeposit(dc, tid1, nil)
			assert.NoError(t, err)
			assert.Equal(t, 0, fee.Sign())
			assert.Equal(t, 0, am.Cmp(amount))
		}

		assert.Equal(t, 0, depositSteps.Sign())
	})

	t.Run("using two deposits", func(t *testing.T) {
		dl := newDepositList()

		dc.tid = tid1
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		dc.height += 1
		dc.tid = tid2
		err = dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		steps := new(big.Int).Mul(amount, dc.DepositIssueRate())
		steps.Div(steps, bigInt100)
		steps.Div(steps, dc.price)
		steps.Mul(steps, big.NewInt(2))

		dc.height += 1
		payed, _ := dl.PaySteps(dc, steps)
		assert.Equal(t, 0, steps.Cmp(payed))
	})
}

func TestDepositList_WithdrawDepositV2(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		rate:   depositIssueRate,
		price:  big.NewInt(100),
		height: 10,
		period: 0,
		tid:    tid1,
	}
	amount := big.NewInt(50000)

	t.Run("add deposit multiple and withdraw", func(t *testing.T) {
		dl := newDepositList()
		dc.tid = tid1

		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())

		err = dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())

		dc.height += 1
		dc.tid = tid2

		err = dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())

		exp := new(big.Int).Mul(amount, big.NewInt(3))
		assert.Equal(t, 0, dl.getAvailableDeposit(0).Cmp(exp))

		exp1 := exp
		dl1 := dl.Clone()

		exp = new(big.Int).Sub(exp, amount)
		am, tr, err := dl.WithdrawDeposit(dc, nil, amount)
		assert.NoError(t, err)
		assert.Equal(t, 0, tr.Sign())
		assert.Equal(t, 0, amount.Cmp(am))

		assert.Equal(t, 0, dl1.getAvailableDeposit(0).Cmp(exp1))

		am, tr, err = dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, exp, am)
		assert.Equal(t, 0, tr.Sign())
		assert.Equal(t, 0, dl.getAvailableDeposit(0).Sign())

		assert.Equal(t, 0, dl1.getAvailableDeposit(0).Cmp(exp1))
	})

	t.Run("invalid withdraw", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// invalid withdraw test
		_, _, err = dl.WithdrawDeposit(dc, tid2, nil)
		assert.Error(t, err)

		// withdraw normal
		_, _, err = dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)

		// withdraw duplicate
		_, _, err = dl.WithdrawDeposit(dc, nil, nil)
		assert.Error(t, err)
	})

	t.Run("withdraw", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		// withdraw without usage
		dc.height += 1
		am, tr, err := dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, amount, am)
		assert.Equal(t, 0, tr.Sign())
	})

	t.Run("paying some and withdraw", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)

		dc.height += 1
		steps := big.NewInt(10)
		dl.PaySteps(dc, steps)

		used := new(big.Int).Mul(steps, dc.price)
		remains := new(big.Int).Sub(amount, used)

		dc.height += 1
		am, tr, err := dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, remains, am)
		assert.Equal(t, 0, tr.Sign())
	})

	t.Run("paying all and withdraw", func(t *testing.T) {
		amount2 := big.NewInt(50010)
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount2)
		assert.NoError(t, err)

		assert.True(t, dl.CanPay(dc))

		// try to pay more steps, but it's limited to the deposit
		dc.height += 1
		steps, remains := new(big.Int).DivMod(amount2, dc.price, new(big.Int))
		payed, depositSteps := dl.PaySteps(dc, new(big.Int).Add(big.NewInt(3), steps))
		assert.Equal(t, steps, payed)
		assert.Equal(t, payed, depositSteps)

		dc.height += 1

		assert.True(t, dl.Has())
		assert.False(t, dl.CanPay(dc))

		am, _, err := dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, remains.Cmp(am))

		assert.False(t, dl.Has())
		assert.False(t, dl.CanPay(dc))
	})

	t.Run("withdraw some and all", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())
		assert.True(t, dl.CanPay(dc))

		dc.height += 1
		value := big.NewInt(200)

		am, tr, err := dl.WithdrawDeposit(dc, nil, value)
		assert.NoError(t, err)
		assert.Equal(t, 0, value.Cmp(am))
		assert.Equal(t, 0, tr.Sign())
		assert.True(t, dl.Has())
		assert.True(t, dl.CanPay(dc))

		dc.height += 1
		am, tr, err = dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, 0, tr.Sign())
		total := new(big.Int).Add(am, value)
		assert.Equal(t, amount, total)
		assert.False(t, dl.Has())
		assert.False(t, dl.CanPay(dc))
	})

	t.Run("withdraw exact all", func(t *testing.T) {
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount)
		assert.NoError(t, err)
		assert.True(t, dl.Has())
		assert.True(t, dl.CanPay(dc))

		dc.height += 1
		am, tr, err := dl.WithdrawDeposit(dc, nil, amount)
		assert.NoError(t, err)
		assert.Equal(t, 0, amount.Cmp(am))
		assert.Equal(t, 0, tr.Sign())
		assert.True(t, dl.Has())
		assert.False(t, dl.CanPay(dc))
	})
}

func TestDepositList_WithdrawDepositV1ToV2(t *testing.T) {
	tid1 := []byte{0x00}
	tid2 := []byte{0x01}
	dc := &depositContext{
		rate:   depositIssueRate,
		price:  big.NewInt(100),
		height: 10,
		period: testDepositTerm,
		tid:    tid1,
	}
	amount1 := big.NewInt(50000)
	amount2 := big.NewInt(30000)

	t.Run("add v1 v2 withdraw v1 v2", func(t *testing.T) {
		dc.period = testDepositTerm
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount1)
		assert.NoError(t, err)

		dc.height += 1
		dc.period = 0
		dc.tid = tid2

		err = dl.AddDeposit(dc, amount2)
		assert.NoError(t, err)

		dc.height += 1
		am, tr, err := dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, am, amount1)
		assert.Equal(t, 0, tr.Sign())
		assert.True(t, dl.Has())
		assert.True(t, dl.CanPay(dc))

		dc.height += 1
		am, tr, err = dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, am, amount2)
		assert.Equal(t, 0, tr.Sign())
		assert.False(t, dl.Has())
		assert.False(t, dl.CanPay(dc))
	})

	t.Run("add v1 v2 withdraw v2 v1", func(t *testing.T) {
		dc.period = testDepositTerm
		dc.tid = tid1
		dl := newDepositList()
		err := dl.AddDeposit(dc, amount1)
		assert.NoError(t, err)

		dc.height += 1
		dc.period = 0
		dc.tid = tid2

		err = dl.AddDeposit(dc, amount2)
		assert.NoError(t, err)

		dc.height += 1
		am, tr, err := dl.WithdrawDeposit(dc, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, am, amount2)
		assert.Equal(t, 0, tr.Sign())
		assert.True(t, dl.Has())
		assert.True(t, dl.CanPay(dc))

		dc.height += 1
		am, tr, err = dl.WithdrawDeposit(dc, tid1, nil)
		assert.NoError(t, err)
		assert.Equal(t, am, amount1)
		assert.Equal(t, 0, tr.Sign())
		assert.False(t, dl.Has())
		assert.False(t, dl.CanPay(dc))
	})
}
