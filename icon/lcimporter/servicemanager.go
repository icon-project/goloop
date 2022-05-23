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
	"fmt"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	VarNextBlockHeight = "nextBlockHeight"
	VarLastBlockHeight = "lastBlockHeight"
	VarCurrentMerkle   = "currentMerkleHeader"
)

const logServiceManager = false

type BlockV1ProofStorage interface {
	GetBlockV1Proof() (*hexary.MerkleHeader, *blockv0.BlockVoteList, error)
	SetBlockV1Proof(root []byte, size int64, votes *blockv0.BlockVoteList) error
}

type ServiceManager struct {
	ch  module.Chain
	ex  *Executor
	log log.Logger
	db  db.Database
	cb  ImportCallback
	ps  BlockV1ProofStorage

	lock sync.Mutex
	next int64
	last int64

	initialValidators module.ValidatorList
	emptyTransactions module.TransactionList
	emptyReceipts     module.ReceiptList
	defaultReceipt    txresult.Receipt
}

type ImportCallback interface {
	OnResult(err error)
}

func (sm *ServiceManager) ProposeTransition(parent module.Transition, bi module.BlockInfo, csi module.ConsensusInfo) (ntr module.Transition, ret error) {
	defer func() {
		_ = sm.handleError(ret)
	}()
	pt := parent.(*transition)
	from := pt.getNextHeight()
	if logServiceManager {
		sm.log.Warnf("SM.ProposeTransactions(from=%d)", from)
	}
	bts, err := sm.ex.ProposeTransactions(from)
	if err != nil {
		if errors.Is(err, ErrAfterLastBlock) {
			last := pt.getLastHeight()
			if last == 0 {
				// we finish our migration
				mh, err := sm.ex.GetMerkleHeader(from)
				if err != nil {
					sm.log.Errorf("SM.ProposeTransactions: FAIL on GetMerkleHeader(%d)", from)
					return nil, err
				}
				bts = []*BlockTransaction{
					&BlockTransaction{
						Height: mh.Leaves,
						Result: mh.RootHash,
					},
				}
			} else {
				bts = []*BlockTransaction{}
			}
		} else {
			return nil, err
		}
	}
	txs := make([]module.Transaction, 0, len(bts))
	for _, bt := range bts {
		txs = append(txs, transaction.Wrap(bt))
	}
	txl := transaction.NewTransactionListFromSlice(sm.db, txs)
	return createTransition(pt, bi, txl, true), nil
}

func (sm *ServiceManager) CreateInitialTransition(result []byte, nextValidators module.ValidatorList) (module.Transition, error) {
	tr := createInitialTransition(sm.db, result, nextValidators, sm, sm.ex)
	return tr, nil
}

func (sm *ServiceManager) CreateTransition(parent module.Transition, txs module.TransactionList, bi module.BlockInfo, csi module.ConsensusInfo, validated bool) (module.Transition, error) {
	pt := parent.(*transition)
	tr := createTransition(pt, bi, txs, false)
	return tr, nil
}

func (sm *ServiceManager) GetPatches(parent module.Transition, bi module.BlockInfo) module.TransactionList {
	return nil
}

func (sm *ServiceManager) PatchTransition(tr module.Transition, patches module.TransactionList, bi module.BlockInfo) module.Transition {
	return tr
}

func (sm *ServiceManager) CreateSyncTransition(tr module.Transition, result []byte, vlHash []byte) module.Transition {
	return createSyncTransition(tr.(*transition))
}

func (sm *ServiceManager) Finalize(tr module.Transition, opt int) error {
	t := tr.(*transition)
	if (opt & module.FinalizeNormalTransaction) != 0 {
		if err := sm.handleError(t.finalizeTransactions()); err != nil {
			return err
		}
	}
	if (opt & module.FinalizeResult) != 0 {
		if err := sm.handleError(t.finalizeResult()); err != nil {
			return err
		}
		sm.setState(t.getNextHeight(), t.getLastHeight())
	}
	return nil
}

func (sm *ServiceManager) WaitForTransaction(tr module.Transition, bi module.BlockInfo, cb func()) bool {
	if !sm.Finished() {
		go cb()
	}
	return true
}

func (sm *ServiceManager) Start() {
	sm.ex.Start()
}

func (sm *ServiceManager) Term() {
	sm.ex.Term()
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

func (sm *ServiceManager) SendTransaction(result []byte, height int64, tx interface{}) ([]byte, error) {
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
	// It doesn't store NID and CID, so return configuration value.
	return int64(sm.ch.NID()), nil
}

func (sm *ServiceManager) GetChainID(result []byte) (int64, error) {
	// It doesn't store NID and CID, so return configuration value.
	return int64(sm.ch.CID()), nil
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
	return true
}

func (sm *ServiceManager) GetNextBlockVersion(result []byte) int {
	return module.BlockVersion2
}

func (sm *ServiceManager) HasTransaction(id []byte) bool {
	return false
}

func (sm *ServiceManager) SendTransactionAndWait(result []byte, height int64, tx interface{}) ([]byte, <-chan interface{}, error) {
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

func (sm *ServiceManager) AddSyncRequest(id db.BucketID, key []byte) error {
	return errors.ErrInvalidState
}

func (sm *ServiceManager) BTPSectionFromResult(result []byte) (module.BTPSection, error) {
	return btp.ZeroBTPSection, nil
}

func (sm *ServiceManager) BTPDigestFromResult(result []byte) (module.BTPDigest, error) {
	//TODO implement me
	panic("implement me")
}

func (sm *ServiceManager) BTPNetworkTypeFromResult(result []byte, ntid int64) (module.BTPNetworkType, error) {
	//TODO implement me
	panic("implement me")
}

func (sm *ServiceManager) BTPNetworkFromResult(result []byte, nid int64) (module.BTPNetwork, error) {
	//TODO implement me
	panic("implement me")
}

func (sm *ServiceManager) NextProofContextMapFromResult(result []byte) (module.BTPProofContextMap, error) {
	return btp.ZeroProofContextMap, nil
}

func newValidatorListFromSlice(dbase db.Database, addrs []*common.Address) (module.ValidatorList, error) {
	vls := make([]module.Validator, len(addrs))
	for i, addr := range addrs {
		if validator, err := state.ValidatorFromAddress(addr); err != nil {
			return nil, err
		} else {
			vls[i] = validator
		}
	}
	if vl, err := state.ValidatorSnapshotFromSlice(dbase, vls); err != nil {
		return nil, err
	} else {
		return vl, nil
	}
}

func (sm *ServiceManager) getInitialValidators() module.ValidatorList {
	return sm.initialValidators
}

func (sm *ServiceManager) Finished() bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	return sm.finishedInLock()
}

func (sm *ServiceManager) finishedInLock() bool {
	return sm.last > 0 && sm.next >= sm.last+2
}

func (sm *ServiceManager) setState(next, last int64) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.next = next
	sm.last = last
}

func (sm *ServiceManager) handleError(err error) error {
	if err != nil {
		go sm.cb.OnResult(err)
	}
	return err
}

func (sm *ServiceManager) GetImportedBlocks() int64 {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	return sm.next
}

func (sm *ServiceManager) GetStatus() string {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if sm.finishedInLock() {
		return fmt.Sprintf("%d finished", sm.next)
	} else {
		return fmt.Sprintf("%d running", sm.next)
	}
}

func NewServiceManagerWithExecutor(chain module.Chain, ex *Executor, ps BlockV1ProofStorage, vs []*common.Address, cb ImportCallback) (*ServiceManager, error) {
	logger := chain.Logger()
	dbase := chain.Database()
	zero := new(big.Int)
	rct := txresult.NewReceipt(dbase, module.LatestRevision, state.SystemAddress)
	rct.SetResult(module.StatusSuccess, zero, zero, nil)

	vl, err := newValidatorListFromSlice(dbase, vs)
	if err != nil {
		return nil, err
	}
	return &ServiceManager{
		ch:  chain,
		ex:  ex,
		log: logger,
		db:  dbase,
		cb:  cb,
		ps:  ps,

		initialValidators: vl,
		emptyTransactions: transaction.NewTransactionListFromHash(dbase, nil),
		emptyReceipts:     txresult.NewReceiptListFromHash(dbase, nil),
		defaultReceipt:    rct,
	}, nil
}

func NewServiceManager(chain module.Chain, rdb db.Database, cfg *Config, cb ImportCallback) (*ServiceManager, error) {
	ex, err := NewExecutor(chain, rdb, cfg)
	if err != nil {
		return nil, err
	}
	return NewServiceManagerWithExecutor(chain, ex, cfg.Platform.(BlockV1ProofStorage), cfg.Validators, cb)
}
