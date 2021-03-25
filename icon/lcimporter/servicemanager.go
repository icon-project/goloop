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

package lcimporter

import (
	"math/big"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ServiceManager struct {
	log log.Logger
	db  db.Database
}

func (sm *ServiceManager) ProposeTransition(parent module.Transition, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	panic("implement me")
}

func (sm *ServiceManager) CreateInitialTransition(result []byte, nextValidators module.ValidatorList) (module.Transition, error) {

	panic("implement me")
}

func (sm *ServiceManager) CreateTransition(parent module.Transition, txs module.TransactionList, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetPatches(parent module.Transition, bi module.BlockInfo) module.TransactionList {
	return nil
}

func (sm *ServiceManager) PatchTransition(tr module.Transition, patches module.TransactionList, bi module.BlockInfo) module.Transition {
	return tr
}

func (sm *ServiceManager) CreateSyncTransition(tr module.Transition, result []byte, vlHash []byte) module.Transition {
	return CreateSyncTransition(tr.(*transition))
}

func (sm *ServiceManager) Finalize(tr module.Transition, opt int) error {
	panic("implement me")
}

func (sm *ServiceManager) WaitForTransaction(parent module.Transition, bi module.BlockInfo, cb func()) bool {
	// it should not be called. anyway, it returns true always.
	return true
}

func (sm *ServiceManager) Start() {
	panic("implement me")
}

func (sm *ServiceManager) Term() {
	panic("implement me")
}

func (sm *ServiceManager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	panic("implement me")
}

func (sm *ServiceManager) GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	panic("implement me")
}

func (sm *ServiceManager) TransactionListFromHash(hash []byte) module.TransactionList {
	panic("implement me")
}

func (sm *ServiceManager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	panic("implement me")
}

func (sm *ServiceManager) ReceiptListFromResult(result []byte, g module.TransactionGroup) (module.ReceiptList, error) {
	panic("implement me")
}

func (sm *ServiceManager) SendTransaction(tx interface{}) ([]byte, error) {
	panic("implement me")
}

func (sm *ServiceManager) SendPatch(patch module.Patch) error {
	panic("implement me")
}

func (sm *ServiceManager) Call(result []byte, vl module.ValidatorList, js []byte, bi module.BlockInfo) (interface{}, error) {
	panic("implement me")
}

func (sm *ServiceManager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	panic("implement me")
}

func (sm *ServiceManager) GetBalance(result []byte, addr module.Address) (*big.Int, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetTotalSupply(result []byte) (*big.Int, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetNetworkID(result []byte) (int64, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetChainID(result []byte) (int64, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetAPIInfo(result []byte, addr module.Address) (module.APIInfo, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetMembers(result []byte) (module.MemberList, error) {
	panic("implement me")
}

func (sm *ServiceManager) GetRoundLimit(result []byte, vl int) int64 {
	panic("implement me")
}

func (sm *ServiceManager) GetMinimizeBlockGen(result []byte) bool {
	panic("implement me")
}

func (sm *ServiceManager) HasTransaction(id []byte) bool {
	panic("implement me")
}

func (sm *ServiceManager) SendTransactionAndWait(tx interface{}) ([]byte, <-chan interface{}, error) {
	panic("implement me")
}

func (sm *ServiceManager) WaitTransactionResult(id []byte) (<-chan interface{}, error) {
	panic("implement me")
}

func (sm *ServiceManager) ExportResult(result []byte, vh []byte, dst db.Database) error {
	panic("implement me")
}

func (sm *ServiceManager) ImportResult(result []byte, vh []byte, src db.Database) error {
	panic("implement me")
}

func (sm *ServiceManager) ExecuteTransaction(result []byte, vh []byte, js []byte, bi module.BlockInfo) (module.Receipt, error) {
	panic("implement me")
}
