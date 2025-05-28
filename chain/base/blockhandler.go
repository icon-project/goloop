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

package base

import (
	"context"
	"io"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type BlockHandlerContext interface {
	GetBlockByHeight(height int64) (module.Block, error)
}

type BlockVersionSpec interface {
	FinalizeHeader(dbase db.Database) error
	// GetVoters returns the voters for the block. Note that this is different
	// from the voted, which is a subset of the voters.
	GetVoters(ctx BlockHandlerContext) (module.ValidatorList, error)
	// VerifyTimestamp verifies timestamp of the block.
	VerifyTimestamp(prev module.BlockData, prevVoters module.ValidatorList) error
}

type Block interface {
	BlockVersionSpec
	module.Block
}

type BlockData interface {
	BlockVersionSpec
	module.BlockData
}

type BlockHandler interface {
	Version() int
	// NewBlock returns a new block. It is used for propose or genesis.
	NewBlock(
		height int64, ts int64, proposer module.Address, prev module.Block,
		logsBloom module.LogsBloom, result []byte,
		patchTransactions module.TransactionList,
		normalTransactions module.TransactionList,
		nextValidators module.ValidatorList, votes module.CommitVoteSet,
		bs module.BTPSection,
	) Block
	NewBlockFromHeaderReader(r io.Reader) (Block, error)
	NewBlockDataFromReader(io.Reader) (BlockData, error)
	GetBlock(id []byte) (Block, error)
}

type Chain interface {
	MaxBlockTxBytes() int
	Database() db.Database
	CommitVoteSetDecoder() module.CommitVoteSetDecoder
	ServiceManager() module.ServiceManager
	MetricContext() context.Context
	CID() int
	NID() int
	Logger() log.Logger
	NetworkManager() module.NetworkManager
	BlockManager() module.BlockManager
	Regulator() module.Regulator
	Wallet() module.Wallet
	WalletFor(dsa string) module.BaseWallet
}

const blockSizeMargin = 10 * 1024

func MaxBlockSize(c Chain) int {
	return c.MaxBlockTxBytes() + blockSizeMargin
}
