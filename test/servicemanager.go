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

package test

import (
	"sync"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type ServiceManager struct {
	module.ServiceManager
	dbase            db.Database
	logger           log.Logger
	plt              base.Platform
	cm               contract.ContractManager
	em               eeproxy.Manager
	chain            module.Chain
	tsc              *service.TxTimestampChecker
	mu               sync.Mutex
	emptyTXs         module.TransactionList
	nextBlockVersion int
	pool             []module.Transaction
	txWaiters        []func()
}

func NewServiceManager(
	c *Chain,
	plt base.Platform,
	cm contract.ContractManager,
	em eeproxy.Manager,
) *ServiceManager {
	dbase := c.Database()
	return &ServiceManager{
		dbase:            dbase,
		logger:           c.Logger(),
		plt:              plt,
		cm:               cm,
		em:               em,
		chain:            c,
		tsc:              service.NewTimestampChecker(),
		emptyTXs:         transaction.NewTransactionListFromSlice(dbase, nil),
		nextBlockVersion: module.BlockVersion2,
	}
}

func (sm *ServiceManager) TransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	return transaction.NewTransaction(b)
}

func (sm *ServiceManager) ProposeTransition(parent module.Transition, bi module.BlockInfo, csi module.ConsensusInfo) (module.Transition, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	txs := transaction.NewTransactionListFromSlice(sm.dbase, sm.pool)
	sm.pool = nil
	return service.NewTransition(
		parent,
		sm.emptyTXs,
		txs,
		bi,
		csi,
		true,
	), nil
}

func (sm *ServiceManager) CreateInitialTransition(
	result []byte,
	nextValidators module.ValidatorList,
) (module.Transition, error) {
	return service.NewInitTransition(
		sm.dbase,
		result,
		nextValidators,
		sm.cm,
		sm.em,
		sm.chain,
		sm.logger,
		sm.plt,
		sm.tsc,
	)
}

func (sm *ServiceManager) CreateTransition(parent module.Transition, txs module.TransactionList, bi module.BlockInfo, csi module.ConsensusInfo, validated bool) (module.Transition, error) {
	return service.NewTransition(
		parent,
		sm.emptyTXs,
		txs,
		bi,
		csi,
		validated,
	), nil
}

func (sm *ServiceManager) GetPatches(parent module.Transition, bi module.BlockInfo) module.TransactionList {
	return sm.emptyTXs
}

func (sm *ServiceManager) PatchTransition(transition module.Transition, patches module.TransactionList, bi module.BlockInfo) module.Transition {
	return transition
}

func (sm *ServiceManager) CreateSyncTransition(transition module.Transition, result []byte, vlHash []byte, noBuffer bool) module.Transition {
	panic("implement me")
}

func (sm *ServiceManager) Finalize(transition module.Transition, opt int) error {
	return service.FinalizeTransition(transition, opt, false)
}

func (sm *ServiceManager) WaitForTransaction(parent module.Transition, bi module.BlockInfo, cb func()) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.pool) > 0 {
		return false
	}
	sm.txWaiters = append(sm.txWaiters, cb)
	return true
}

func (sm *ServiceManager) GetChainID(result []byte) (int64, error) {
	return int64(sm.chain.CID()), nil
}

func (sm *ServiceManager) GetNetworkID(result []byte) (int64, error) {
	return int64(sm.chain.NID()), nil
}

func (sm *ServiceManager) GetMembers(result []byte) (module.MemberList, error) {
	return nil, nil
}

func (sm *ServiceManager) GetRoundLimit(result []byte, vl int) int64 {
	ws, err := service.NewWorldSnapshot(sm.dbase, sm.plt, result, nil)
	if err != nil {
		return 0
	}
	ass := ws.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	if as == nil {
		return 0
	}
	factor := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Int64()
	if factor == 0 {
		return 0
	}
	limit := contract.RoundLimitFactorToRound(vl, factor)
	return limit
}

func (sm *ServiceManager) GetMinimizeBlockGen(result []byte) bool {
	ws, err := service.NewWorldSnapshot(sm.dbase, sm.plt, result, nil)
	if err != nil {
		return false
	}
	ass := ws.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	if as == nil {
		return false
	}
	return scoredb.NewVarDB(as, state.VarMinimizeBlockGen).Bool()
}

func (sm *ServiceManager) GetNextBlockVersion(result []byte) int {
	if result == nil {
		return sm.plt.DefaultBlockVersionFor(sm.chain.CID())
	}
	ws, err := service.NewWorldSnapshot(sm.dbase, sm.plt, result, nil)
	if err != nil {
		return -1
	}
	ass := ws.GetAccountSnapshot(state.SystemID)
	if ass == nil {
		return sm.plt.DefaultBlockVersionFor(sm.chain.CID())
	}
	as := scoredb.NewStateStoreWith(ass)
	v := int(scoredb.NewVarDB(as, state.VarNextBlockVersion).Int64())
	if v == 0 {
		return sm.plt.DefaultBlockVersionFor(sm.chain.CID())
	}
	return v
}

func (sm *ServiceManager) getSystemByteStoreState(result []byte) (containerdb.BytesStoreState, error) {
	ws, err := service.NewWorldSnapshot(sm.dbase, sm.plt, result, nil)
	if err != nil {
		return nil, err
	}
	ass := ws.GetAccountSnapshot(state.SystemID)
	if ass == nil {
		return containerdb.EmptyBytesStoreState, nil
	}
	return scoredb.NewStateStoreWith(ass), nil
}

func (sm *ServiceManager) ImportResult(result []byte, vh []byte, src db.Database) error {
	panic("implement me")
}

func (sm *ServiceManager) GenesisTransactionFromBytes(b []byte, blockVersion int) (module.Transaction, error) {
	return transaction.NewGenesisTransaction(b)
}

func (sm *ServiceManager) TransactionListFromHash(hash []byte) module.TransactionList {
	return transaction.NewTransactionListFromHash(sm.dbase, hash)
}

func (sm *ServiceManager) ReceiptListFromResult(result []byte, g module.TransactionGroup) (module.ReceiptList, error) {
	panic("implement me")
}

func (sm *ServiceManager) SendTransaction(result []byte, height int64, tx interface{}) ([]byte, error) {
	t, err := transaction.NewTransactionFromJSON(([]byte)(tx.(string)))
	if err != nil {
		return nil, err
	}

	locker := common.Lock(&sm.mu)
	defer locker.Unlock()

	sm.pool = append(sm.pool, t)
	txWaiters := sm.txWaiters
	sm.txWaiters = nil

	locker.Unlock()

	for _, cb := range txWaiters {
		cb()
	}
	return t.ID(), nil
}

func (sm *ServiceManager) ValidatorListFromHash(hash []byte) module.ValidatorList {
	vl, err := state.ValidatorSnapshotFromHash(sm.dbase, hash)
	if err != nil {
		return nil
	}
	return vl
}

func (sm *ServiceManager) TransactionListFromSlice(txs []module.Transaction, version int) module.TransactionList {
	switch version {
	case module.BlockVersion0:
		return transaction.NewTransactionListV1FromSlice(txs)
	case module.BlockVersion1, module.BlockVersion2:
		return transaction.NewTransactionListFromSlice(sm.chain.Database(), txs)
	default:
		return nil
	}
}

func (sm *ServiceManager) SendTransactionAndWait(result []byte, height int64, tx interface{}) ([]byte, <-chan interface{}, error) {
	panic("implement me")
}

func (sm *ServiceManager) WaitTransactionResult(id []byte) (<-chan interface{}, error) {
	return nil, service.ErrCommittedTransaction
}

type transitionResult struct {
	StateHash         []byte
	PatchReceiptHash  []byte
	NormalReceiptHash []byte
	ExtensionData     []byte
	BTPData           []byte
}

func newTransitionResultFromBytes(bs []byte) (*transitionResult, error) {
	tresult := new(transitionResult)
	if len(bs) > 0 {
		if _, err := codec.UnmarshalFromBytes(bs, tresult); err != nil {
			return nil, err
		}
	}
	return tresult, nil
}

func (sm *ServiceManager) ExportResult(result []byte, vh []byte, dst db.Database) error {
	r, err := newTransitionResultFromBytes(result)
	if err != nil {
		return err
	}
	e := merkle.NewCopyContext(sm.dbase, dst)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.NormalReceiptHash)
	txresult.NewReceiptListWithBuilder(e.Builder(), r.PatchReceiptHash)
	ess := sm.plt.NewExtensionWithBuilder(e.Builder(), r.ExtensionData)
	state.NewWorldSnapshotWithBuilder(e.Builder(), r.StateHash, vh, ess, r.BTPData)
	return e.Run()
}

func (sm *ServiceManager) BTPDigestFromResult(result []byte) (module.BTPDigest, error) {
	dh, err := service.BTPDigestHashFromResult(result)
	if err != nil {
		return nil, err
	}
	bk, err := sm.dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	digestBytes, err := bk.Get(dh)
	if err != nil {
		return nil, err
	}
	digest, err := btp.NewDigestFromBytes(digestBytes)
	if err != nil {
		return nil, err
	}
	return digest, nil
}

func (sm *ServiceManager) BTPSectionFromResult(result []byte) (module.BTPSection, error) {
	digest, err := sm.BTPDigestFromResult(result)
	if err != nil {
		return nil, err
	}
	store, err := sm.getSystemByteStoreState(result)
	if err != nil {
		return nil, err
	}
	btpContext := state.NewBTPContext(nil, store)
	return btp.NewSection(digest, btpContext, sm.dbase)
}

func (sm *ServiceManager) BTPNetworkFromResult(result []byte, nid int64) (module.BTPNetwork, error) {
	sbss, err := sm.getSystemByteStoreState(result)
	if err != nil {
		return nil, err
	}
	btpContext := state.NewBTPContext(nil, sbss)
	nw, err := btpContext.GetNetwork(nid)
	if err != nil {
		return nil, err
	}
	return nw, nil
}

func (sm *ServiceManager) BTPNetworkTypeFromResult(result []byte, ntid int64) (module.BTPNetworkType, error) {
	sbss, err := sm.getSystemByteStoreState(result)
	if err != nil {
		return nil, err
	}
	btpContext := state.NewBTPContext(nil, sbss)
	nt, err := btpContext.GetNetworkType(ntid)
	if err != nil {
		return nil, err
	}
	return nt, nil
}

func (sm *ServiceManager) BTPNetworkTypeIDsFromResult(result []byte) ([]int64, error) {
	sbss, err := sm.getSystemByteStoreState(result)
	if err != nil {
		return nil, err
	}
	btpContext := state.NewBTPContext(nil, sbss)
	ntids, err := btpContext.GetNetworkTypeIDs()
	if err != nil {
		return nil, err
	}
	return ntids, nil
}

func (sm *ServiceManager) NextProofContextMapFromResult(result []byte) (module.BTPProofContextMap, error) {
	sbss, err := sm.getSystemByteStoreState(result)
	if err != nil {
		return nil, err
	}
	btpContext := state.NewBTPContext(nil, sbss)
	return btp.NewProofContextsMap(btpContext)
}
