package trace

import (
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

func newRandomHash(size int) []byte {
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return nil
	}
	return bs
}

func TestNewBalanceTracer(t *testing.T) {
	var err error
	bt := NewBalanceTracer(10)
	_, ok := bt.(*balanceTracer)
	assert.True(t, ok)

	txIndex := int32(0)
	txHash := newRandomHash(32)
	err = bt.OnTransactionStart(txIndex, txHash)
	assert.NoError(t, err)

	from := common.MustNewAddressFromString("hx100")
	to := common.MustNewAddressFromString("cx101")
	amount := big.NewInt(1000)
	err = bt.OnBalanceChange(module.Transfer, from, to, amount)
	assert.NoError(t, err)

	err = bt.OnEnter()
	assert.NoError(t, err)

	err = bt.OnLeave(true)
	assert.NoError(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.NoError(t, err)

	jso, ok := BalanceTracerToJSON(bt).([]interface{})
	assert.True(t, ok)
	assert.NotNil(t, jso)
	assert.Equal(t, 1, len(jso))

	txJso, ok := jso[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "0x"+hex.EncodeToString(txHash), txJso["txHash"])

	opsJso, ok := txJso["ops"].([]map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 1, len(opsJso))
	// bs, err := json.Marshal(jso)
	// assert.NoError(t, err)
	// fmt.Printf("%s\n", string(bs))
}

func TestEmpyBalanceTracer_ErrorCase(t *testing.T) {
	var err error
	txIndex := int32(0)
	txHash := newRandomHash(32)

	bt := NewBalanceTracer(10)
	_, ok := bt.(*balanceTracer)
	assert.True(t, ok)

	err = bt.OnEnter()
	assert.Error(t, err)

	err = bt.OnLeave(true)
	assert.Error(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.Error(t, err)
}

func TestEmpyBalanceTracer_NormalCase(t *testing.T) {
	var err error
	txIndex := int32(0)
	txHash := newRandomHash(32)
	treasury := common.MustNewAddressFromString("hx10")
	from := common.MustNewAddressFromString("hx11")
	to := common.MustNewAddressFromString("hx22")
	score := common.MustNewAddressFromString("cx33")

	bt := NewBalanceTracer(10)
	_, ok := bt.(*balanceTracer)
	assert.True(t, ok)

	err = bt.OnTransactionStart(txIndex, txHash)
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, to, big.NewInt(1000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Claim, treasury, from, big.NewInt(2000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Burn, from, nil, big.NewInt(3000))
	assert.NoError(t, err)

	err = bt.OnEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, score, big.NewInt(1000))
	assert.NoError(t, err)

	err = bt.OnEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Claim, treasury, score, big.NewInt(2000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Burn, score, nil, big.NewInt(3000))
	assert.NoError(t, err)

	err = bt.OnLeave(true)
	assert.NoError(t, err)

	err = bt.OnLeave(true)
	assert.NoError(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.NoError(t, err)
}

func TestOpTypeToString(t *testing.T) {
	type data struct {
		opType module.OpType
		opName string
	}

	items := []data{
		{module.Genesis, "GENESIS"},
		{module.Transfer, "TRANSFER"},
		{module.Fee, "FEE"},
		{module.Issue, "ISSUE"},
		{module.Burn, "BURN"},
		{module.Lost, "LOST"},
		{module.FSDeposit, "FS_DEPOSIT"},
		{module.FSWithdraw, "FS_WITHDRAW"},
		{module.FSFee, "FS_FEE"},
		{module.Stake, "STAKE"},
		{module.Unstake, "UNSTAKE"},
		{module.Claim, "CLAIM"},
		{module.Ghost, "GHOST"},
		{module.Reward, "REWARD"},
		{module.RegPRep, "REG_PREP"},
	}

	for _, item := range items {
		assert.Equal(t, item.opName, opTypeToString(item.opType))
	}
}
