/*
 * Copyright 2021 ICON Foundation
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

package main

import (
	"bytes"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type Block struct {
	height     int64
	result     []byte
	txs        module.TransactionList
	rcts       module.ReceiptList
	blk        blockv0.Block
	txTotal    *big.Int
	validators module.ValidatorList
	oldRcts    module.ReceiptList
}

type blockHeader struct {
	Height  int64
	Result  []byte
	TxRoot  []byte
	RctRoot []byte
	BlkRaw  []byte
	TxTotal *big.Int
	VltHash []byte
	OldRct  []byte
}

func (b *Block) Height() int64 {
	return b.height
}

func (b *Block) Result() []byte {
	return b.result
}

func (b *Block) NextValidators() module.ValidatorList {
	return b.validators
}

func (b *Block) SetResult(result []byte, validators module.ValidatorList, rcts module.ReceiptList, txTotal *big.Int) {
	b.result = result
	b.rcts = rcts
	b.txTotal = txTotal
	b.validators = validators
}

func (b *Block) CheckResult(logger log.Logger, result []byte, validators module.ValidatorList, rcts module.ReceiptList, txTotal *big.Int) error {
	if len(b.result) == 0 {
		return errors.New("NoStoredResult")
	}
	if !bytes.Equal(b.result, result) {
		if rv, err :=ParseResult(b.result); err == nil && rv != nil {
			js, _ := JSONMarshalIndent(rv)
			logger.Errorf("Expected   : %s", js)
		}

		if rv, err :=ParseResult(result); err == nil && rv != nil {
			js, _ := JSONMarshalIndent(rv)
			logger.Errorf("Calculated : %s", js)
		}
		return errors.Errorf("DifferentResult(stored=%#x,real=%#x)", b.result, result)
	}
	if exp, real := b.validators.Hash(), validators.Hash(); !bytes.Equal(exp, real) {
		return errors.Errorf("DifferentValidators(stored=%#x,real=%#x)", exp, real)
	}
	if txTotal.Cmp(b.txTotal) != 0 {
		return errors.Errorf("DifferentTxCount(stored=%d,real=%d)", b.txTotal, txTotal)
	}
	return nil
}

func (b *Block) Transactions() module.TransactionList {
	return b.txs
}

func (b *Block) Receipts() module.ReceiptList {
	return b.rcts
}

func (b *Block) OldReceipts() module.ReceiptList {
	return b.oldRcts
}

func (b *Block) LogBloom() module.LogsBloom {
	return b.blk.LogsBloom()
}

func (b *Block) Timestamp() int64 {
	if b == nil || b.blk == nil {
		return 0
	}
	return b.blk.Timestamp()
}

func (b *Block) ID() []byte {
	return b.blk.ID()
}

func (b *Block) Flush() error {
	if len(b.result) == 0 {
		if err := b.txs.Flush(); err != nil {
			return err
		}
	}
	if b.rcts == nil || !bytes.Equal(b.rcts.Hash(), b.oldRcts.Hash()) {
		return b.oldRcts.Flush()
	}
	return nil
}

func (b *Block) Reset(database db.Database, bs []byte) error {
	var header blockHeader
	if _, err := codec.BC.UnmarshalFromBytes(bs, &header); err != nil {
		return err
	}
	b.result = header.Result
	b.height = header.Height
	b.txs = transaction.NewTransactionListFromHash(database, header.TxRoot)
	b.rcts = txresult.NewReceiptListFromHash(database, header.RctRoot)
	store := db.GetFlag(database, FlagExecutor).(blockv0.Store)
	if blk, err := blockv0.ParseBlock(header.BlkRaw, store); err != nil {
		return err
	} else {
		b.blk = blk
	}
	b.txTotal = header.TxTotal
	if len(header.VltHash) > 0 {
		if vlt, err := state.ValidatorSnapshotFromHash(database, header.VltHash); err != nil {
			return err
		} else {
			b.validators = vlt
		}
	} else {
		b.validators, _ = state.ValidatorSnapshotFromHash(database, nil)
	}
	if len(header.OldRct) > 0 {
		b.oldRcts = txresult.NewReceiptListFromHash(database, header.OldRct)
	} else {
		b.oldRcts = b.rcts
	}
	return nil
}

func (b *Block) NewWorldSnapshot(database db.Database, plt service.Platform) (state.WorldSnapshot, error) {
	return service.NewWorldSnapshot(database, plt, b.result, b.validators)
}

func (b *Block) Bytes() []byte {
	var header blockHeader
	header.Result = b.result
	header.Height = b.height
	header.TxRoot = b.txs.Hash()
	if b.rcts != nil {
		header.RctRoot = b.rcts.Hash()
	}
	js, err := JSONMarshalAndCompact(b.blk)
	if err != nil {
		panic(err)
	}
	header.BlkRaw = js
	header.TxTotal = b.txTotal
	if b.validators != nil {
		header.VltHash = b.validators.Hash()
	}
	header.OldRct = b.oldRcts.Hash()
	bs, _ := codec.BC.MarshalToBytes(&header)
	return bs
}

func (b *Block) Original() blockv0.Block {
	if b == nil {
		return nil
	}
	return b.blk
}

func (b *Block) TxTotal() *big.Int {
	if b == nil {
		return new(big.Int)
	}
	return b.txTotal
}

func (b *Block) TxCount() *big.Int {
	if b == nil || b.blk == nil {
		return new(big.Int)
	} else {
		return big.NewInt(int64(len(b.blk.NormalTransactions())))
	}
}
