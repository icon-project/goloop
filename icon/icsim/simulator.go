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

package icsim

import (
	"math/big"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type TxType int
const (
	TypeSetStake TxType = iota
	TypeSetDelegation
	TypeSetBond
	TypeSetBonderList
	TypeRegisterPRep
	TypeUnregisterPRep
	TypeDisqualifyPRep
	TypeSetPRep
	TypeSetRevision
	TypeClaimIScore
)

type Transaction interface {
	Type() TxType
	Args() []interface{}
}

type Block interface {
	Txs() []Transaction
	AddTransaction(tx Transaction)
}

type block struct {
	txs []Transaction
}

func (b *block) Txs() []Transaction {
	return b.txs
}

func (b *block) AddTransaction(tx Transaction) {
	b.txs = append(b.txs, tx)
}

func NewBlock() Block {
	return &block{
		txs: make([]Transaction, 0),
	}
}

type Simulator interface {
	Database() db.Database
	BlockHeight() int64
	Revision() module.Revision
	GetBalance(from module.Address) *big.Int
	TotalBond() *big.Int
	TotalStake() *big.Int
	TotalSupply() *big.Int
	ValidatorList() []module.Validator

	GetPRepTerm() map[string]interface{}
	GetMainPReps() map[string]interface{}
	GetSubPReps() map[string]interface{}
	GetPReps() map[string]interface{}
	GetNetworkInfo() map[string]interface{}
	TermSnapshot() *icstate.TermSnapshot

	Go(csi module.ConsensusInfo, blocks int64) error
	GoTo(csi module.ConsensusInfo, blockHeight int64) error
	GoToTermEnd(csi module.ConsensusInfo) error
	GoByBlock(csi module.ConsensusInfo, block Block) ([]Receipt, error)
	GoByTransaction(csi module.ConsensusInfo, txs ...Transaction) ([]Receipt, error)
	SetRevision(revision module.Revision) Transaction

	GetStake(from module.Address) map[string]interface{}
	SetStake(from module.Address, amount *big.Int) Transaction

	QueryIScore(address module.Address) *big.Int
	ClaimIScore(from module.Address) Transaction

	GetPRepStats(address module.Address) map[string]interface{}
	GetPRep(address module.Address) *icstate.PRep
	SetPRep(from module.Address, info *icstate.PRepInfo) Transaction

	GetDelegation(address module.Address) map[string]interface{}
	SetDelegation(from module.Address, ds icstate.Delegations) Transaction

	GetBond(address module.Address) map[string]interface{}
	SetBond(from module.Address, bonds icstate.Bonds) Transaction

	GetBonderList(address module.Address) map[string]interface{}
	SetBonderList(from module.Address, bl icstate.BonderList) Transaction

	RegisterPRep(from module.Address, info *icstate.PRepInfo) Transaction
	UnregisterPRep(from module.Address) Transaction
	DisqualifyPRep(from module.Address, address module.Address) Transaction
}
