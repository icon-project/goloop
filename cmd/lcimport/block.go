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
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
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
}

type blockHeader struct {
	Height  int64
	Result  []byte
	TxRoot  []byte
	RctRoot []byte
	BlkRaw  []byte
	TxTotal *big.Int
	VltHash []byte
}

func (b *Block) Height() int64 {
	return b.height
}

func (b *Block) Result() []byte {
	return b.result
}

func (b *Block) SetResult(result []byte, validators module.ValidatorList, rcts module.ReceiptList, txTotal *big.Int) {
	b.result = result
	b.rcts = rcts
	b.txTotal = txTotal
	b.validators = validators
}

func (b *Block) Transactions() module.TransactionList {
	return b.txs
}

func (b *Block) Receipts() module.ReceiptList {
	return b.rcts
}

func (b *Block) LogBloom() module.LogsBloom {
	return b.blk.LogsBloom()
}

func (b *Block) Timestamp() int64 {
	return b.blk.Timestamp()
}

func (b *Block) ID() []byte {
	return b.blk.ID()
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
	header.RctRoot = b.rcts.Hash()
	js, err := JSONMarshalAndCompact(b.blk)
	if err != nil {
		panic(err)
	}
	header.BlkRaw = js
	header.TxTotal = b.txTotal
	if b.validators != nil {
		header.VltHash = b.validators.Hash()
	}
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
