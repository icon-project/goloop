package trace

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

var opTypeNames = []string{
	"GENESIS",
	"TRANSFER",
	"FEE",
	"ISSUE",
	"BURN",
	"LOST",
	"FS_DEPOSIT",
	"FS_WITHDRAW",
	"FS_FEE",
	"STAKE",
	"UNSTAKE",
	"CLAIM",
	"GHOST",
	"REWARD",
	"REG_PREP",
}

func opTypeToString(o module.OpType) string {
	return opTypeNames[o]
}

type operation struct {
	depth  int
	opType module.OpType
	from   module.Address
	to     module.Address
	amount *common.HexInt
}

func (o *operation) toJSON() map[string]interface{} {
	jso := map[string]interface{}{
		"opType": opTypeToString(o.opType),
		"amount": o.amount,
	}
	if o.from != nil {
		jso["from"] = o.from
	}
	if o.to != nil {
		jso["to"] = o.to
	}
	return jso
}

type callFrame struct {
	parent *callFrame
	depth  int
	ops    []*operation
}

func (c *callFrame) mergeOpsToParent() {
	parent := c.parent
	if parent == nil {
		return
	}
	parent.ops = append(parent.ops, c.ops...)
}

func (c *callFrame) toJSON() []map[string]interface{} {
	size := len(c.ops)
	if size == 0 {
		return nil
	}
	jso := make([]map[string]interface{}, size)
	for i, op := range c.ops {
		jso[i] = op.toJSON()
	}
	return jso
}

type transaction struct {
	index int32
	hash  []byte
	*callFrame
}

func (t *transaction) resetCallFrame() {
	t.callFrame = &callFrame{}
}

func (t *transaction) toJSON() map[string]interface{} {
	return map[string]interface{}{
		"txIndex": fmt.Sprintf("%#x", t.index),
		"txHash":  "0x" + hex.EncodeToString(t.hash),
		"ops":     t.callFrame.toJSON(),
	}
}

type balanceTracer struct {
	txs      []*transaction
	curFrame *callFrame
}

func (bt *balanceTracer) add(opType module.OpType, from, to module.Address, amount *big.Int) error {
	curFrame := bt.curFrame
	op := &operation{
		depth:  curFrame.depth,
		opType: opType,
		from:   from,
		to:     to,
		amount: &common.HexInt{Int: *amount},
	}
	curFrame.ops = append(curFrame.ops, op)
	return nil
}

func (bt *balanceTracer) getCurrentTx() (*transaction, error) {
	txCount := len(bt.txs)
	if txCount == 0 {
		return nil, errors.InvalidStateError.New("No transaction")
	}
	return bt.txs[txCount-1], nil
}

func (bt *balanceTracer) checkCurrentTx(curTx *transaction, txIndex int32, txHash []byte) error {
	if curTx.index != txIndex || bytes.Compare(curTx.hash, txHash) != 0 {
		return errors.InvalidStateError.Errorf(
			"Invalid txHash: curTxHash=%s hash=%s",
			hex.EncodeToString(curTx.hash), hex.EncodeToString(txHash))
	}
	return nil
}

func (bt *balanceTracer) OnTransactionStart(txIndex int32, txHash []byte) error {
	if bt.curFrame != nil {
		return errors.InvalidStateError.Errorf(
			"Invalid curFrame: txIndex=%d txHash=%s curFrame=%#v",
			txIndex, hex.EncodeToString(txHash), bt.curFrame)
	}
	frame := &callFrame{}
	tx := &transaction{index: txIndex, hash: txHash, callFrame: frame}
	bt.txs = append(bt.txs, tx)
	bt.curFrame = frame
	return nil
}

func (bt *balanceTracer) OnTransactionRerun(txIndex int32, txHash []byte) error {
	curTx, err := bt.getCurrentTx()
	if err != nil {
		return err
	}
	if err = bt.checkCurrentTx(curTx, txIndex, txHash); err != nil {
		return err
	}
	frame := &callFrame{}
	curTx.callFrame = frame
	bt.curFrame = frame
	return nil
}

func (bt *balanceTracer) OnTransactionEnd(txIndex int32, txHash []byte) error {
	curTx, err := bt.getCurrentTx()
	if err != nil {
		return err
	}
	if err = bt.checkCurrentTx(curTx, txIndex, txHash); err != nil {
		return err
	}

	depth := bt.curFrame.depth
	if depth != 0 {
		return errors.InvalidStateError.Errorf("Invalid callFrame depth: %d", depth)
	}
	bt.curFrame = nil
	return nil
}

func (bt *balanceTracer) OnEnter() error {
	if bt.curFrame == nil {
		return errors.InvalidStateError.Errorf("RosettaCallback Not Ready")
	}
	parent := bt.curFrame
	bt.curFrame = &callFrame{
		parent: parent,
		depth:  parent.depth + 1,
		ops:    make([]*operation, 0),
	}
	return nil
}

func (bt *balanceTracer) OnLeave(success bool) error {
	curFrame := bt.curFrame
	if curFrame == nil {
		return errors.InvalidStateError.New("curFrame Not Ready")
	}
	if curFrame.depth <= 0 {
		return errors.InvalidStateError.Errorf("Invalid frameDepth: %d", curFrame.depth)
	}
	if success {
		bt.curFrame.mergeOpsToParent()
	}
	bt.curFrame = bt.curFrame.parent
	return nil
}

func (bt *balanceTracer) OnBalanceChange(opType module.OpType, from, to module.Address, amount *big.Int) error {
	return bt.add(opType, from, to, amount)
}

func (bt *balanceTracer) toJSON() interface{} {
	size := len(bt.txs)
	if size == 0 {
		return nil
	}
	jso := make([]interface{}, size)
	for i, tx := range bt.txs {
		jso[i] = tx.toJSON()
	}
	return jso
}

func NewBalanceTracer(capacity int) module.BalanceTracer {
	return &balanceTracer{txs: make([]*transaction, 0, capacity)}
}

func BalanceTracerToJSON(bt module.BalanceTracer) interface{} {
	if tracer, ok := bt.(*balanceTracer); ok {
		return tracer.toJSON()
	}
	return nil
}
