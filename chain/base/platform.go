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
	"math/big"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type Platform interface {
	NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error)
	NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot
	NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot
	OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot, logger log.Logger)
	ToRevision(value int) module.Revision
	NewBaseTransaction(wc state.WorldContext) (module.Transaction, error)
	OnValidateTransactions(wc state.WorldContext, patches, txs module.TransactionList) error
	OnExecutionBegin(wc state.WorldContext, logger log.Logger) error
	OnExecutionEnd(wc state.WorldContext, er ExecutionResult, logger log.Logger) error
	OnTransactionEnd(wc state.WorldContext, logger log.Logger, rct txresult.Receipt) error
	DefaultBlockVersionFor(cid int) int
	NewBlockHandlers(c Chain) []BlockHandler
	NewConsensus(c Chain, walDir string) (module.Consensus, error)
	CommitVoteSetDecoder() module.CommitVoteSetDecoder
	Term()
	NewSeedState(wc state.WorldSnapshot) (module.SeedState, error)
}

type ExecutionResult interface {
	PatchReceipts() module.ReceiptList
	NormalReceipts() module.ReceiptList
	TotalFee() *big.Int
	VirtualFee() *big.Int
}
