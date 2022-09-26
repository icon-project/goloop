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

func getCurrentFrameOpsLength(bt *BalanceTracer) int {
	if bt.curFrame != nil {
		return len(bt.curFrame.ops)
	}
	return 0
}

func TestNewBalanceTracer(t *testing.T) {
	var err error
	bt := NewBalanceTracer(10, nil)

	height := rand.Int63()
	txIndex := 0
	txHash := newRandomHash(32)
	err = bt.OnTransactionStart(txIndex, txHash, false)
	assert.NoError(t, err)

	from := common.MustNewAddressFromString("hx100")
	to := common.MustNewAddressFromString("cx101")
	amount := big.NewInt(1000)
	err = bt.OnBalanceChange(module.Transfer, from, to, amount)
	assert.NoError(t, err)

	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnFrameExit(true)
	assert.NoError(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.NoError(t, err)

	jso, ok := bt.ToJSON(height).([]interface{})
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
	txIndex := 0
	txHash := newRandomHash(32)

	bt := NewBalanceTracer(10, nil)

	err = bt.OnFrameEnter()
	assert.Error(t, err)

	err = bt.OnFrameExit(true)
	assert.Error(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.Error(t, err)
}

func TestEmpyBalanceTracer_NormalCase(t *testing.T) {
	var err error
	txIndex := 0
	txHash := newRandomHash(32)
	treasury := common.MustNewAddressFromString("hx10")
	from := common.MustNewAddressFromString("hx11")
	to := common.MustNewAddressFromString("hx22")
	score := common.MustNewAddressFromString("cx33")

	bt := NewBalanceTracer(10, nil)

	err = bt.OnTransactionStart(txIndex, txHash, false)
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, to, big.NewInt(1000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Claim, treasury, from, big.NewInt(2000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Burn, from, nil, big.NewInt(3000))
	assert.NoError(t, err)

	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, score, big.NewInt(1000))
	assert.NoError(t, err)

	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Claim, treasury, score, big.NewInt(2000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Burn, score, nil, big.NewInt(3000))
	assert.NoError(t, err)

	err = bt.OnFrameExit(true)
	assert.NoError(t, err)

	err = bt.OnFrameExit(true)
	assert.NoError(t, err)

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.NoError(t, err)
}

func TestEmpyBalanceTracer_OnTransactionReset(t *testing.T) {
	var err error
	txIndex := 0
	txHash := newRandomHash(32)
	treasury := common.MustNewAddressFromString("hx10")
	from := common.MustNewAddressFromString("hx11")
	to := common.MustNewAddressFromString("hx22")
	score := common.MustNewAddressFromString("cx33")

	bt := NewBalanceTracer(10, nil)

	err = bt.OnTransactionStart(txIndex, txHash, false)
	assert.NoError(t, err)

	// Frame 1
	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, to, big.NewInt(1000))
	assert.NoError(t, err)

	assert.Equal(t, 1, getCurrentFrameOpsLength(bt))

	// Enter Frame 1-1
	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Claim, treasury, from, big.NewInt(2000))
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Burn, from, nil, big.NewInt(3000))
	assert.NoError(t, err)

	assert.Equal(t, 2, getCurrentFrameOpsLength(bt))

	// Exit from Frame 1-1
	err = bt.OnFrameExit(true)
	assert.NoError(t, err)

	assert.Equal(t, 3, getCurrentFrameOpsLength(bt))

	// Enter Frame 1-2
	err = bt.OnFrameEnter()
	assert.NoError(t, err)

	err = bt.OnBalanceChange(module.Transfer, from, score, big.NewInt(1000))
	assert.NoError(t, err)

	assert.Equal(t, 1, getCurrentFrameOpsLength(bt))

	// Exit from Frame 1-2
	err = bt.OnFrameExit(false)
	assert.NoError(t, err)

	assert.Equal(t, 3, getCurrentFrameOpsLength(bt))

	// Exit from Frame 1
	err = bt.OnFrameExit(true)
	assert.NoError(t, err)

	assert.Equal(t, 3, getCurrentFrameOpsLength(bt))

	err = bt.OnTransactionReset()
	assert.NoError(t, err)

	assert.Equal(t, 0, getCurrentFrameOpsLength(bt))

	err = bt.OnTransactionEnd(txIndex, txHash)
	assert.NoError(t, err)

	assert.Equal(t, 0, getCurrentFrameOpsLength(bt))
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
