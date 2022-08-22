package iiss

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/ictest"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

func newWorldContext() state.WorldContext {
	dbase := db.NewMapDB()
	plt := ictest.NewPlatform()
	ws := state.NewWorldState(dbase, nil, nil, nil)
	return state.NewWorldContext(ws, nil, nil, plt)
}

func TestWorldContextImpl_GetBalance(t *testing.T) {
	address := common.MustNewAddressFromString("hx1")
	wc := newWorldContext()
	iwc := NewWorldContext(wc, nil)

	initBalance := icutils.ToLoop(100)
	as := wc.GetAccountState(address.ID())
	as.SetBalance(initBalance)
	assert.Zero(t, as.GetBalance().Cmp(initBalance))
	assert.Zero(t, iwc.GetBalance(address).Cmp(initBalance))
}

func TestWorldContextImpl_Deposit(t *testing.T) {
	var err error
	address := common.MustNewAddressFromString("hx1")
	wc := newWorldContext()
	iwc := NewWorldContext(wc, nil)

	balance := iwc.GetBalance(address)
	assert.NotNil(t, balance)
	assert.Zero(t, balance.Int64())

	var sum int64
	for i := int64(0); i < 10; i++ {
		amount := big.NewInt(i)
		err = iwc.Deposit(address, amount, module.Genesis)
		assert.NoError(t, err)

		sum += i
		balance = iwc.GetBalance(address)
		assert.Equal(t, sum, balance.Int64())
	}

	err = iwc.Deposit(address, big.NewInt(-100), module.Genesis)
	assert.Error(t, err)

	err = iwc.Deposit(address, big.NewInt(0), module.Genesis)
	assert.NoError(t, err)
	assert.Equal(t, sum, balance.Int64())
}

func TestWorldContextImpl_Withdraw(t *testing.T) {
	var err error
	address := common.MustNewAddressFromString("hx1")
	wc := newWorldContext()
	iwc := NewWorldContext(wc, nil)

	balance := iwc.GetBalance(address)
	assert.NotNil(t, balance)
	assert.Zero(t, balance.Int64())

	expectedBalance := int64(50)
	err = iwc.Deposit(address, big.NewInt(expectedBalance), module.Genesis)
	assert.NoError(t, err)

	// Subtract 100 from 50
	err = iwc.Withdraw(address, big.NewInt(100), module.Stake)
	assert.Error(t, err)

	for i := 0; i < 5; i++ {
		err = iwc.Withdraw(address, big.NewInt(10), module.Stake)
		assert.NoError(t, err)

		expectedBalance -= 10
		balance = iwc.GetBalance(address)
		assert.Equal(t, expectedBalance, balance.Int64())
	}
	assert.Zero(t, balance.Sign())

	// Negative amount is not allowed
	err = iwc.Withdraw(address, big.NewInt(-100), module.Stake)
	assert.Error(t, err)

	// Subtract 100 from 0
	err = iwc.Withdraw(address, big.NewInt(100), module.Stake)
	assert.Error(t, err)
}

func TestWorldContextImpl_Transfer(t *testing.T) {
	var err error
	from := common.MustNewAddressFromString("hx1")
	to := common.MustNewAddressFromString("hx2")
	wc := newWorldContext()
	iwc := NewWorldContext(wc, nil)

	initBalance := int64(100)
	err = iwc.Deposit(from, big.NewInt(initBalance), module.Genesis)
	assert.NoError(t, err)
	err = iwc.Deposit(to, big.NewInt(initBalance), module.Genesis)
	assert.NoError(t, err)

	// transfer 30 from "from" to "to"
	// from: 100 - 30 = 70
	// to: 100 + 30 = 130
	err = iwc.Transfer(from, to, big.NewInt(30), module.Transfer)
	assert.NoError(t, err)
	assert.Zero(t, big.NewInt(70).Cmp(iwc.GetBalance(from)))
	assert.Zero(t, big.NewInt(130).Cmp(iwc.GetBalance(to)))
}

func TestWorldContextImpl_TotalSupply(t *testing.T) {
	var err error
	wc := newWorldContext()
	iwc := NewWorldContext(wc, nil)

	ts := iwc.GetTotalSupply()
	assert.NotNil(t, ts)
	assert.Zero(t, ts.Sign())

	sum := new(big.Int)
	amount := icutils.ToLoop(100)
	for i := 0; i < 10; i++ {
		ts, err = iwc.AddTotalSupply(amount)
		assert.NoError(t, err)
		sum.Add(sum, amount)
		assert.Zero(t, ts.Cmp(sum))
	}
	assert.Zero(t, ts.Cmp(iwc.GetTotalSupply()))
}

func TestWorldContextImpl_SetScoreOwner_SanityCheck(t *testing.T) {
	var err error
	from := common.MustNewAddressFromString("hx1")
	score := common.MustNewAddressFromString("cx1")
	owner := common.MustNewAddressFromString("hx2")

	wc := NewWorldContext(newWorldContext(), nil)

	// Case: from is nil
	err = wc.SetScoreOwner(nil, score, owner)
	assert.Equal(t, scoreresult.InvalidParameterError, errors.CodeOf(err))

	// Case: score is nil
	err = wc.SetScoreOwner(from, nil, owner)
	assert.Equal(t, scoreresult.InvalidParameterError, errors.CodeOf(err))

	// Case: score is not a contract address
	err = wc.SetScoreOwner(from, common.MustNewAddressFromString("hx3"), owner)
	assert.Equal(t, icmodule.IllegalArgumentError, errors.CodeOf(err))

	newOwners := []module.Address{
		nil,
		common.MustNewAddressFromString("cx1"),
		common.MustNewAddressFromString("cx2"),
		common.MustNewAddressFromString("hx2"),
	}
	errCodes := []errors.Code{
		scoreresult.InvalidParameterError,
		icmodule.IllegalArgumentError,
		icmodule.IllegalArgumentError,
		icmodule.IllegalArgumentError,
	}
	for i := range newOwners {
		err = wc.SetScoreOwner(from, score, newOwners[i])
		assert.Equal(t, errCodes[i], errors.CodeOf(err))
	}
}
