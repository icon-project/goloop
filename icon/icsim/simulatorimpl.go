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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

func newWorldState(wss state.WorldSnapshot, readonly bool) state.WorldState {
	if readonly {
		return state.NewReadOnlyWorldState(wss)
	}
	dbase := wss.Database()
	stateHash := wss.StateHash()
	vss := wss.GetValidatorSnapshot()
	ess := wss.GetExtensionSnapshot()
	return state.NewWorldState(dbase, stateHash, vss, ess, nil)
}

type transactionImpl struct {
	txType TxType
	args   []interface{}
}

func (t *transactionImpl) Type() TxType {
	return t.txType
}

func (t *transactionImpl) Args() []interface{} {
	return t.args
}

func NewTransaction(txType TxType, args []interface{}) Transaction {
	return &transactionImpl{txType, args}
}

// ==========================================================

type emptyConsensusInfoMaker struct{}

var emptyConsensusInfo = common.NewConsensusInfo(nil, nil, nil)

func (maker *emptyConsensusInfoMaker) Run(
	wss state.WorldSnapshot, blockHeight int64, revision module.Revision) module.ConsensusInfo {
	return nil
}

// ==========================================================

type simulatorImpl struct {
	config *config
	plt    platform
	logger log.Logger

	blockHeight int64
	revision    module.Revision
	stepPrice   *big.Int
	wss         state.WorldSnapshot
	revHandlers map[int]func(wc state.WorldState) error
}

func (sim *simulatorImpl) init(validators []module.Validator, balances map[string]*big.Int) error {
	dbase := db.NewMapDB()
	sim.initRevHandler()

	vss, err := state.ValidatorSnapshotFromSlice(dbase, validators)
	if err != nil {
		return err
	}
	ws := state.NewWorldState(dbase, nil, vss, nil, nil)
	totalSupply := new(big.Int)

	// Initialize balances
	for k, amount := range balances {
		as := ws.GetAccountState([]byte(k)[1:])
		as.SetBalance(amount)
		totalSupply.Add(totalSupply, amount)
	}

	// Initialize totalSupply
	as := ws.GetAccountState(state.SystemID)
	tsVarDB := scoredb.NewVarDB(as, state.VarTotalSupply)
	if err = tsVarDB.Set(totalSupply); err != nil {
		return err
	}
	if err = sim.handleRevisionChange(ws, 0, sim.revision.Value()); err != nil {
		return err
	}

	// Save initial values to database
	wss := ws.GetSnapshot()
	if err = wss.Flush(); err != nil {
		return err
	}

	sim.onFinalize(wss)
	sim.wss = wss
	return nil
}

func (sim *simulatorImpl) initRevHandler() {
	sim.revHandlers = map[int]func(wc state.WorldState) error{
		icmodule.Revision5:  sim.handleRev5,
		icmodule.Revision6:  sim.handleRev6,
		icmodule.Revision9:  sim.handleRev9,
		icmodule.Revision10: sim.handleRev10,
		icmodule.Revision14: sim.handleRev14,
		icmodule.Revision15: sim.handleRev15,
		icmodule.Revision17: sim.handleRev17,
	}
}

func (sim *simulatorImpl) getExtensionState(readonly bool) *iiss.ExtensionStateImpl {
	ws := newWorldState(sim.wss, readonly)
	return ws.GetExtensionState().(*iiss.ExtensionStateImpl)
}

func (sim *simulatorImpl) Database() db.Database {
	return sim.wss.Database()
}

func (sim *simulatorImpl) BlockHeight() int64 {
	return sim.blockHeight
}

func (sim *simulatorImpl) Revision() module.Revision {
	return sim.revision
}

func (sim *simulatorImpl) TotalBond() *big.Int {
	es := sim.getExtensionState(true)
	return es.State.GetTotalBond()
}

func (sim *simulatorImpl) TotalStake() *big.Int {
	es := sim.getExtensionState(true)
	return es.State.GetTotalStake()
}

func (sim *simulatorImpl) TotalSupply() *big.Int {
	ws := state.NewReadOnlyWorldState(sim.wss)
	as := ws.GetAccountState(state.SystemID)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	if ts := tsVar.BigInt(); ts != nil {
		return ts
	}
	return icmodule.BigIntZero
}

func (sim *simulatorImpl) GetBalance(address module.Address) *big.Int {
	ws := state.NewReadOnlyWorldState(sim.wss)
	as := ws.GetAccountState(address.ID())
	return as.GetBalance()
}

func (sim *simulatorImpl) SetRevision(revision module.Revision) Transaction {
	return NewTransaction(TypeSetRevision, []interface{}{revision})
}

func (sim *simulatorImpl) setRevision(wc icmodule.WorldContext, tx Transaction) error {
	args := tx.Args()
	revision := args[0].(module.Revision)
	curRev := sim.revision.Value()
	newRev := revision.Value()

	if newRev > icmodule.MaxRevision {
		return errors.Errorf(
			"IllegalArgument(max=%d,new=%d)",
			icmodule.MaxRevision, newRev,
		)
	}
	if newRev < curRev {
		return errors.Errorf("IllegalArgument(current=%d,new=%d)", curRev, newRev)
	}

	ws := wc.(state.WorldState)
	err := sim.handleRevisionChange(ws, curRev, newRev)
	sim.revision = revision
	return err
}

func (sim *simulatorImpl) GetStake(from module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ia := es.State.GetAccountSnapshot(from)
	if ia == nil {
		ia = icstate.GetEmptyAccountSnapshot()
	}
	return ia.GetStakeInJSON(sim.BlockHeight())
}

func (sim *simulatorImpl) SetStake(from module.Address, amount *big.Int) Transaction {
	return NewTransaction(TypeSetStake, []interface{}{from, amount})
}

func (sim *simulatorImpl) setStake(es *iiss.ExtensionStateImpl, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	amount := args[1].(*big.Int)
	return es.State.GetAccountState(from).SetStake(amount)
}

// Go generates as many blocks as the number passed as a parameter "blocks"
func (sim *simulatorImpl) Go(csi module.ConsensusInfo, blocks int64) error {
	if blocks == 0 {
		return nil
	}
	if blocks < 0 {
		return errors.Errorf("Invalid blocks: %d", blocks)
	}

	var err error
	wss := sim.wss
	blockHeight := sim.blockHeight

	for i := int64(0); i < blocks; i++ {
		blockHeight++

		ws := newWorldState(wss, false)
		wc := NewWorldContext(ws, blockHeight, sim.Revision(), csi, sim.stepPrice)

		if err = sim.onExecutionBegin(wc); err != nil {
			return err
		}
		if err = sim.onBaseTx(wc); err != nil {
			return err
		}
		if err = sim.onExecutionEnd(wc); err != nil {
			return err
		}

		wss = ws.GetSnapshot()
		if err = wss.Flush(); err != nil {
			return err
		}

		sim.onFinalize(wss)
	}

	sim.wss = wss
	sim.blockHeight = blockHeight
	return nil
}

func (sim *simulatorImpl) onExecutionBegin(wc WorldContext) error {
	return sim.plt.OnExecutionBegin(wc, sim.logger)
}

func (sim *simulatorImpl) onBaseTx(wc WorldContext) error {
	cc := NewCallContext(wc, state.SystemAddress)
	es := wc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.HandleConsensusInfo(cc)
}

func (sim *simulatorImpl) onExecutionEnd(wc WorldContext) error {
	return sim.plt.OnExecutionEnd(wc, sim.logger)
}

func (sim *simulatorImpl) onFinalize(wss state.WorldSnapshot) {
	sim.plt.OnExtensionSnapshotFinalization(wss.GetExtensionSnapshot(), sim.logger)
}

func (sim *simulatorImpl) GoTo(csi module.ConsensusInfo, blockHeight int64) error {
	blocks := blockHeight - sim.blockHeight
	if blocks <= 0 {
		return errors.Errorf("Invalid blockHeight: cur=%d <= new=%d", sim.blockHeight, blockHeight)
	}
	return sim.Go(csi, blocks)
}

func (sim *simulatorImpl) GoToTermEnd(csi module.ConsensusInfo) error {
	es := sim.getExtensionState(true)
	tss := es.State.GetTermSnapshot()

	blocks := tss.GetEndHeight() - sim.blockHeight
	return sim.Go(csi, blocks)
}

func (sim *simulatorImpl) GoByBlock(csi module.ConsensusInfo, block Block) ([]Receipt, error) {
	var err error
	wss := sim.wss
	ws := newWorldState(wss, false)
	ws.ClearCache()

	blockHeight := sim.blockHeight + 1
	receipts := make([]Receipt, len(block.Txs()))

	for i, tx := range block.Txs() {
		wss = ws.GetSnapshot()
		err = sim.executeTx(csi, ws, tx)
		receipts[i] = NewReceipt(blockHeight, err)

		if err != nil {
			if err = ws.Reset(wss); err != nil {
				return nil, err
			}
		}
	}

	wss = ws.GetSnapshot()
	if err = wss.Flush(); err != nil {
		return nil, err
	}

	sim.wss = wss
	sim.blockHeight = blockHeight
	return receipts, nil
}

// GoByTransaction generates a block including given transactions and executes it
func (sim *simulatorImpl) GoByTransaction(csi module.ConsensusInfo, txs ...Transaction) ([]Receipt, error) {
	blk := NewBlock()
	for _, tx := range txs {
		blk.AddTransaction(tx)
	}
	return sim.GoByBlock(csi, blk)
}

func (sim *simulatorImpl) executeTx(csi module.ConsensusInfo, ws state.WorldState, tx Transaction) error {
	var err error
	revision := sim.revision
	wc := NewWorldContext(ws, sim.blockHeight+1, revision, csi, sim.stepPrice)
	es := wc.GetExtensionState().(*iiss.ExtensionStateImpl)

	switch tx.Type() {
	case TypeSetStake:
		err = sim.setStake(es, tx)
	case TypeSetDelegation:
		err = sim.setDelegation(es, wc, tx)
	case TypeSetBond:
		err = sim.setBond(es, wc, tx)
	case TypeSetBonderList:
		err = sim.setBonderList(es, tx)
	case TypeRegisterPRep:
		err = sim.registerPRep(es, wc, tx)
	case TypeUnregisterPRep:
		err = sim.unregisterPRep(es, wc, tx)
	case TypeDisqualifyPRep:
		err = sim.disqualifyPRep(es, wc, tx)
	case TypeSetPRep:
		err = sim.setPRep(es, wc, tx)
	case TypeSetRevision:
		err = sim.setRevision(wc, tx)
	case TypeClaimIScore:
		err = sim.claimIScore(es, wc, tx)
	default:
		return errors.Errorf("Unexpected transaction: %v", tx.Type())
	}
	return err
}

func (sim *simulatorImpl) RegisterPRep(from module.Address, info *icstate.PRepInfo) Transaction {
	return NewTransaction(TypeRegisterPRep, []interface{}{from, info})
}

func (sim *simulatorImpl) registerPRep(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	info := args[1].(*icstate.PRepInfo)
	cc := NewCallContext(wc, from)
	if err := cc.Transfer(from, state.SystemAddress, icmodule.BigIntRegPRepFee, module.RegPRep); err != nil {
		return err
	}
	return es.RegisterPRep(cc, info)
}

func (sim *simulatorImpl) UnregisterPRep(from module.Address) Transaction {
	return NewTransaction(TypeUnregisterPRep, []interface{}{from})
}

func (sim *simulatorImpl) unregisterPRep(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	cc := NewCallContext(wc, from)
	return es.UnregisterPRep(cc)
}

func (sim *simulatorImpl) DisqualifyPRep(from module.Address, address module.Address) Transaction {
	return NewTransaction(TypeDisqualifyPRep, []interface{}{from, address})
}

func (sim *simulatorImpl) disqualifyPRep(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	address := args[1].(module.Address)
	cc := NewCallContext(wc, from)
	return es.DisqualifyPRep(cc, address)
}

func (sim *simulatorImpl) SetPRep(from module.Address, info *icstate.PRepInfo) Transaction {
	return NewTransaction(TypeSetPRep, []interface{}{from, info})
}

func (sim *simulatorImpl) setPRep(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	info := args[1].(*icstate.PRepInfo)
	cc := NewCallContext(wc, from)
	return es.SetPRep(cc, info, false)
}

func (sim *simulatorImpl) GetDelegation(from module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ia := es.State.GetAccountSnapshot(from)
	return ia.GetDelegationInJSON()
}

func (sim *simulatorImpl) SetDelegation(from module.Address, ds icstate.Delegations) Transaction {
	return NewTransaction(TypeSetDelegation, []interface{}{from, ds})
}

func (sim *simulatorImpl) setDelegation(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	ds := args[1].(icstate.Delegations)
	cc := NewCallContext(wc, from)
	return es.SetDelegation(cc, ds)
}

func (sim *simulatorImpl) GetBond(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetBond(address)
	return jso
}

func (sim *simulatorImpl) SetBond(from module.Address, bonds icstate.Bonds) Transaction {
	return NewTransaction(TypeSetBond, []interface{}{from, bonds})
}

func (sim *simulatorImpl) setBond(es *iiss.ExtensionStateImpl, wc icmodule.WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	bonds := args[1].(icstate.Bonds)
	return es.SetBond(wc.BlockHeight(), from, bonds)
}

func (sim *simulatorImpl) GetBonderList(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetBonderList(address)
	return jso
}

func (sim *simulatorImpl) SetBonderList(from module.Address, bl icstate.BonderList) Transaction {
	return NewTransaction(TypeSetBonderList, []interface{}{from, bl})
}

func (sim *simulatorImpl) setBonderList(es *iiss.ExtensionStateImpl, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	bl := args[1].(icstate.BonderList)
	return es.SetBonderList(from, bl)
}

func (sim *simulatorImpl) ClaimIScore(from module.Address) Transaction {
	return NewTransaction(TypeClaimIScore, []interface{}{from})
}

func (sim *simulatorImpl) claimIScore(es *iiss.ExtensionStateImpl, wc WorldContext, tx Transaction) error {
	args := tx.Args()
	from := args[0].(module.Address)
	cc := NewCallContext(wc, from)
	return es.ClaimIScore(cc)
}

func (sim *simulatorImpl) QueryIScore(address module.Address) *big.Int {
	es := sim.getExtensionState(true)
	iscore, _ := es.GetIScore(address, sim.revision.Value(), nil)
	return iscore
}

func (sim *simulatorImpl) GetPRepTerm() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetPRepTermInJSON(sim.newCallContext())
	return jso
}

func (sim *simulatorImpl) newCallContext() icmodule.CallContext {
	wss := sim.wss
	ws := newWorldState(wss, false)
	wc := NewWorldContext(ws, sim.blockHeight+1, sim.revision, nil, sim.stepPrice)
	return NewCallContext(wc, state.SystemAddress)
}

func (sim *simulatorImpl) GetPReps() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetPRepsInJSON(sim.newCallContext(), 0, 0)
	return jso
}

func (sim *simulatorImpl) GetMainPReps() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetMainPRepsInJSON(sim.BlockHeight())
	return jso
}

func (sim *simulatorImpl) GetSubPReps() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetSubPRepsInJSON(sim.BlockHeight())
	return jso
}

func (sim *simulatorImpl) GetPRep(address module.Address) *icstate.PRep {
	es := sim.getExtensionState(true)
	return es.State.GetPRepByOwner(address)
}

func (sim *simulatorImpl) GetPRepStats(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ps := es.State.GetPRepStatusByOwner(address, false)
	return ps.GetStatsInJSON(sim.BlockHeight())
}

func (sim *simulatorImpl) GetNetworkInfo() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, err := es.State.GetNetworkInfoInJSON(sim.Revision().Value())
	if err != nil {
		return nil
	}
	return jso
}

func (sim *simulatorImpl) TermSnapshot() *icstate.TermSnapshot {
	es := sim.getExtensionState(true)
	return es.State.GetTermSnapshot()
}

func (sim *simulatorImpl) ValidatorList() []module.Validator {
	vss := sim.wss.GetValidatorSnapshot()
	size := vss.Len()
	vl := make([]module.Validator, size)
	for i := 0; i < size; i++ {
		v, _ := vss.Get(i)
		vl[i] = v
	}
	return vl
}

func NewSimulator(
	revision module.Revision, initValidators []module.Validator, initBalances map[string]*big.Int, config *config,
) Simulator {
	sim := &simulatorImpl{
		logger:      log.GlobalLogger(),
		blockHeight: 0,
		revision:    revision,
		stepPrice:   icmodule.BigIntZero,
		config:      config,
	}
	if err := sim.init(initValidators, initBalances); err != nil {
		return nil
	}
	return sim
}
