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
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type TxType int
const (
	TypeTransfer TxType = iota
	TypeSetStake
	TypeSetDelegation
	TypeSetBond
	TypeSetBonderList
	TypeRegisterPRep
	TypeUnregisterPRep
	TypeDisqualifyPRep
	TypeSetPRep
	TypeSetRevision
	TypeClaimIScore
	TypeSetSlashingRates
	TypeSetMinimumBond
	TypeInitCommissionRate
	TypeSetCommissionRate
	TypeRequestUnjail
	TypeHandleDoubleSignReport
	TypeSetPRepCountConfig
	TypeSetRewardFundAllocation2
)

type Transaction interface {
	Type() TxType
	From() module.Address
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
	ValidatorIndexOf(address module.Address) int
	GetStateContext() icmodule.StateContext
	TermSnapshot() *icstate.TermSnapshot
	PRepCountConfig() icstate.PRepCountConfig
	GetPReps(grade icstate.Grade) []*icstate.PRep
	GetAccountSnapshot(address module.Address) *icstate.AccountSnapshot

	GetPRepTermInJSON() map[string]interface{}
	GetMainPRepsInJSON() map[string]interface{}
	GetSubPRepsInJSON() map[string]interface{}
	GetPRepsInJSON() map[string]interface{}
	GetNetworkInfoInJSON() map[string]interface{}
	GetPRepStatsInJSON(address module.Address) map[string]interface{}

	Go(csi module.ConsensusInfo, blocks int64) error
	GoTo(csi module.ConsensusInfo, blockHeight int64) error
	GoToTermEnd(csi module.ConsensusInfo) error
	GoByBlock(csi module.ConsensusInfo, block Block) ([]Receipt, error)
	GoByTransaction(csi module.ConsensusInfo, txs ...Transaction) ([]Receipt, error)

	// Transactions

	Transfer(from, to module.Address, amount *big.Int) Transaction
	GoByTransfer(csi module.ConsensusInfo, from, to module.Address, amount *big.Int) (Receipt, error)

	SetRevision(from module.Address, revision module.Revision) Transaction
	GoBySetRevision(csi module.ConsensusInfo, from module.Address, revision module.Revision) (Receipt, error)

	GetStakeInJSON(from module.Address) map[string]interface{}
	SetStake(from module.Address, amount *big.Int) Transaction
	GoBySetStake(csi module.ConsensusInfo, from module.Address, amount *big.Int) (Receipt, error)

	QueryIScore(address module.Address) *big.Int
	ClaimIScore(from module.Address) Transaction
	GoByClaimIScore(csi module.ConsensusInfo, from module.Address) (Receipt, error)

	GetPRep(address module.Address) *icstate.PRep
	SetPRep(from module.Address, info *icstate.PRepInfo) Transaction
	GoBySetPRep(csi module.ConsensusInfo, from module.Address, info *icstate.PRepInfo) (Receipt, error)

	GetDelegationInJSON(address module.Address) map[string]interface{}
	SetDelegation(from module.Address, ds icstate.Delegations) Transaction
	GoBySetDelegation(csi module.ConsensusInfo, from module.Address, ds *icstate.Delegations) (Receipt, error)

	GetBondInJSON(address module.Address) map[string]interface{}
	SetBond(from module.Address, bonds icstate.Bonds) Transaction
	GoBySetBond(csi module.ConsensusInfo, from module.Address, bonds icstate.Bonds) (Receipt, error)

	GetBonderListInJSON(address module.Address) map[string]interface{}
	GetBonderList(address module.Address) icstate.BonderList
	SetBonderList(from module.Address, bl icstate.BonderList) Transaction
	GoBySetBonderList(csi module.ConsensusInfo, from module.Address, bl icstate.BonderList) (Receipt, error)

	RegisterPRep(from module.Address, info *icstate.PRepInfo) Transaction
	GoByRegisterPRep(csi module.ConsensusInfo, from module.Address, info *icstate.PRepInfo) (Receipt, error)
	UnregisterPRep(from module.Address) Transaction
	GoByUnregisterPRep(csi module.ConsensusInfo, from module.Address) (Receipt, error)
	DisqualifyPRep(from, address module.Address) Transaction
	GoByDisqualifyPRep(csi module.ConsensusInfo, from, address module.Address) (Receipt, error)

	// After RevisionBTP2
	//OpenBTPNetwork(networkTypeName string, name string, owner module.Address) (int64, error)
	//CloseBTPNetwork(id int64) error
	//RegisterPRepNodePublicKey(address module.Address, pubKey []byte) error
	//SetPRepNodePublicKey(pubKey []byte) error

	// After RevisionIISS4R0
	GetSlashingRate(pt icmodule.PenaltyType) (icmodule.Rate, error)
	GetSlashingRates() (map[string]interface{}, error)
	SetSlashingRates(from module.Address, rates map[string]icmodule.Rate) Transaction
	GoBySetSlashingRates(csi module.ConsensusInfo, from module.Address, rates map[string]icmodule.Rate) (Receipt, error)

	GetMinimumBond() *big.Int
	SetMinimumBond(from module.Address, bond *big.Int) Transaction
	GoBySetMinimumBond(csi module.ConsensusInfo, from module.Address, bond *big.Int) (Receipt, error)

	InitCommissionRate(from module.Address, rate, maxRate, maxChangeRate icmodule.Rate) Transaction
	GoByInitCommissionRate(csi module.ConsensusInfo, from module.Address, rate, maxRate, maxChangeRate icmodule.Rate) (Receipt, error)
	SetCommissionRate(from module.Address, rate icmodule.Rate) Transaction
	GoBySetCommissionRate(csi module.ConsensusInfo, from module.Address, rate icmodule.Rate) (Receipt, error)

	HandleDoubleSignReport(from module.Address, dsType string, dsBlockHeight int64, signer module.Address) Transaction
	GoByHandleDoubleSignReport(csi module.ConsensusInfo,
		from module.Address, dsType string, dsBlockHeight int64, signer module.Address) (Receipt, error)
	RequestUnjail(from module.Address) Transaction
	GoByRequestUnjail(csi module.ConsensusInfo, from module.Address) (Receipt, error)

	GetPRepCountConfig() (map[string]interface{}, error)
	SetPRepCountConfig(from module.Address, counts map[string]int64) Transaction
	GoBySetPRepCountConfig(csi module.ConsensusInfo, from module.Address, counts map[string]int64) (Receipt, error)

	GetRewardFundAllocation2() *icstate.RewardFund
	SetRewardFundAllocation2(from module.Address, values map[icstate.RFundKey]icmodule.Rate) Transaction
	GoBySetRewardFundAllocation2(
		csi module.ConsensusInfo, from module.Address, values map[icstate.RFundKey]icmodule.Rate) (Receipt, error)
}
