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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	VarNextBlockHeight = "nextBlockHeight"
)

type ServiceManager struct {
	cfg *Config
	ex  *Executor
	log log.Logger
	db  db.Database

	emptyTransactions module.TransactionList
	emptyReceipts     module.ReceiptList
}

func (sm *ServiceManager) ProposeTransition(parent module.Transition, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	pt := parent.(*transition)
	bts, err := sm.ex.ProposeTransactions()
	if err != nil {
		return nil, err
	}
	txs := make([]module.Transaction, 0, len(bts))
	for _, bt := range bts {
		txs = append(txs, bt)
	}
	txl := transaction.NewTransactionListFromSlice(sm.db, txs)
	return CreateTransition(pt, bi, txl, true), nil
}

func (sm *ServiceManager) CreateInitialTransition(result []byte, nextValidators module.ValidatorList) (module.Transition, error) {
	tr := CreateInitialTransition(sm.db, result, nextValidators, sm, sm.ex)
	return tr, nil
}

func (sm *ServiceManager) CreateTransition(parent module.Transition, txs module.TransactionList, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	pt := parent.(*transition)
	tr := CreateTransition(pt, bi, txs, false)
	return tr, nil
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
	t := tr.(*transition)
	if (opt & module.FinalizeNormalTransaction) != 0 {
		if err := t.finalizeTransactions(); err != nil {
			return err
		}
	}
	if (opt & module.FinalizeResult) != 0 {
		if err := t.finalizeResult(); err != nil {
			return err
		}
	}
	return nil
}

func (sm *ServiceManager) WaitForTransaction(parent module.Transition, bi module.BlockInfo, cb func()) bool {
	// it should not be called. anyway, it returns false always because it will not call cb.
	return false
}

func (sm *ServiceManager) Start() {
	// TODO need to start executor
}

func (sm *ServiceManager) Term() {
	// TODO Stop executor
}

func (sm *ServiceManager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := transaction.NewTransaction(b)
	if err != nil {
		sm.log.Warnf("sm.TransactionFromBytes() fails with err=%+v", err)
	}
	return tx, nil
}

func (sm *ServiceManager) GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	tx, err := transaction.NewGenesisTransaction(b)
	if err != nil {
		sm.log.Warnf("sm.GenesisTransactionFromBytes() fails with err=%+v", err)
	}
	return tx, nil
}

func (sm *ServiceManager) TransactionListFromHash(hash []byte) module.TransactionList {
	return transaction.NewTransactionListFromHash(sm.db, hash)
}

func (sm *ServiceManager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	switch version {
	case module.BlockVersion0:
		return transaction.NewTransactionListV1FromSlice(txs)
	case module.BlockVersion1, module.BlockVersion2:
		return transaction.NewTransactionListFromSlice(sm.db, txs)
	default:
		return nil
	}
}

func (sm *ServiceManager) ReceiptListFromResult(result []byte, g module.TransactionGroup) (module.ReceiptList, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) SendTransaction(tx interface{}) ([]byte, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) SendPatch(patch module.Patch) error {
	return errors.ErrInvalidState
}

func (sm *ServiceManager) Call(result []byte, vl module.ValidatorList, js []byte, bi module.BlockInfo) (interface{}, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	if vs, err := state.ValidatorSnapshotFromHash(sm.db, hash); err != nil {
		panic(err)
	} else {
		return vs
	}
}

func (sm *ServiceManager) GetBalance(result []byte, addr module.Address) (*big.Int, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) GetTotalSupply(result []byte) (*big.Int, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) GetNetworkID(result []byte) (int64, error) {
	return 1, nil
}

func (sm *ServiceManager) GetChainID(result []byte) (int64, error) {
	return 1, nil
}

func (sm *ServiceManager) GetAPIInfo(result []byte, addr module.Address) (module.APIInfo, error) {
	return nil, common.ErrInvalidState
}

func (sm *ServiceManager) GetMembers(result []byte) (module.MemberList, error) {
	return nil, nil
}

func (sm *ServiceManager) GetRoundLimit(result []byte, vl int) int64 {
	return 0
}

func (sm *ServiceManager) GetMinimizeBlockGen(result []byte) bool {
	return false
}

func (sm *ServiceManager) HasTransaction(id []byte) bool {
	return false
}

func (sm *ServiceManager) SendTransactionAndWait(tx interface{}) ([]byte, <-chan interface{}, error) {
	return nil, nil, errors.ErrInvalidState
}

func (sm *ServiceManager) WaitTransactionResult(id []byte) (<-chan interface{}, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) ExportResult(result []byte, vh []byte, dst db.Database) error {
	return errors.ErrInvalidState
}

func (sm *ServiceManager) ImportResult(result []byte, vh []byte, src db.Database) error {
	return errors.ErrInvalidState
}

func (sm *ServiceManager) ExecuteTransaction(result []byte, vh []byte, js []byte, bi module.BlockInfo) (module.Receipt, error) {
	return nil, errors.ErrInvalidState
}

func (sm *ServiceManager) getValidators() (module.ValidatorList, error) {
	vls := make([]module.Validator, len(sm.cfg.Validators))
	for i, addr := range sm.cfg.Validators {
		if validator, err := state.ValidatorFromAddress(addr); err != nil {
			return nil, err
		} else {
			vls[i] = validator
		}
	}
	if vl, err := state.ValidatorSnapshotFromSlice(sm.db, vls); err != nil {
		return nil, err
	} else {
		return vl, nil
	}
}

func NewServiceManager(chain module.Chain, config *Config) (*ServiceManager, error) {
	dbase := chain.Database()
	logger := chain.Logger()

	ex := NewExecutor(dbase, logger)
	return &ServiceManager{
		db:  dbase,
		log: logger,
		ex:  ex,

		emptyTransactions: transaction.NewTransactionListFromHash(dbase, nil),
		emptyReceipts:     txresult.NewReceiptListFromHash(dbase, nil),
	}, nil
}
