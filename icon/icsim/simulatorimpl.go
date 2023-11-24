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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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

type mockStateContext struct {
	blockHeight     int64
	revision        int
	termRevision    int
	termIISSVersion int
	activeDSAMask   int64
	br              icmodule.Rate
}

func (m mockStateContext) BlockHeight() int64 {
	return m.blockHeight
}

func (m mockStateContext) RevisionValue() int {
	return m.revision
}

func (m mockStateContext) TermRevisionValue() int {
	return m.termRevision
}

func (m mockStateContext) TermIISSVersion() int {
	return m.termIISSVersion
}

func (m mockStateContext) GetActiveDSAMask() int64 {
	return m.activeDSAMask
}

func (m mockStateContext) GetBondRequirement() icmodule.Rate {
	return m.br
}

func (m mockStateContext) AddEventEnable(module.Address, icmodule.EnableStatus) error {
	return nil
}

type transactionImpl struct {
	txType TxType
	from   module.Address
	args   []interface{}
}

func (t *transactionImpl) Type() TxType {
	return t.txType
}

func (t *transactionImpl) From() module.Address {
	return t.from
}

func (t *transactionImpl) Args() []interface{} {
	return t.args
}

func NewTransaction(txType TxType, from module.Address, args ...interface{}) Transaction {
	return &transactionImpl{txType, from, args}
}

func getExtensionState(ws state.WorldState) *iiss.ExtensionStateImpl {
	return ws.GetExtensionState().(*iiss.ExtensionStateImpl)
}

type simulatorImpl struct {
	config *SimConfig
	plt    platform
	logger log.Logger

	blockHeight int64
	revision    module.Revision
	stepPrice   *big.Int
	wss         state.WorldSnapshot
	revHandlers map[int]RevHandler
}

func (sim *simulatorImpl) init(
	revision module.Revision, validators []module.Validator, balances map[string]*big.Int) error {
	dbase := db.NewMapDB()
	sim.initRevHandler()

	// Initialize WorldState with ValidatorList
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

	// Initialize revision
	rev := icutils.Min(revision.Value(), icmodule.Revision12)
	if err = sim.handleRevisionChange(ws, 0, rev); err != nil {
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

func (sim *simulatorImpl) getExtensionState(readonly bool) *iiss.ExtensionStateImpl {
	return getExtensionState(newWorldState(sim.wss, readonly))
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

func (sim *simulatorImpl) Transfer(from, to module.Address, amount *big.Int) Transaction {
	return NewTransaction(TypeTransfer, from, to, amount)
}

func (sim *simulatorImpl) GoByTransfer(
	csi module.ConsensusInfo, from, to module.Address, amount *big.Int) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeTransfer, from, to, amount)
}

func (sim *simulatorImpl) transfer(_ *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	to := args[0].(module.Address)
	amount := args[1].(*big.Int)
	return cc.Transfer(tx.From(), to, amount, module.Transfer)
}

func (sim *simulatorImpl) SetRevision(from module.Address, revision module.Revision) Transaction {
	return NewTransaction(TypeSetRevision, from, revision)
}

func (sim *simulatorImpl) GoBySetRevision(
	csi module.ConsensusInfo, from module.Address, revision module.Revision) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetRevision, from, revision)
}

func (sim *simulatorImpl) setRevision(_ *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
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

	ws := cc.GetWorldState()
	err := sim.handleRevisionChange(ws, curRev, newRev)
	return err
}

func (sim *simulatorImpl) GetStakeInJSON(from module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ia := es.State.GetAccountSnapshot(from)
	if ia == nil {
		ia = icstate.GetEmptyAccountSnapshot()
	}
	return ia.GetStakeInJSON(sim.BlockHeight())
}

func (sim *simulatorImpl) SetStake(from module.Address, amount *big.Int) Transaction {
	return NewTransaction(TypeSetStake, from, amount)
}

func (sim *simulatorImpl) GoBySetStake(
	csi module.ConsensusInfo, from module.Address, amount *big.Int) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetStake, from, amount)
}

func (sim *simulatorImpl) setStake(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	amount := args[0].(*big.Int)
	return es.SetStake(cc, amount)
}

// Go generates as many blocks as the number passed as a parameter "blocks"
func (sim *simulatorImpl) Go(csi module.ConsensusInfo, blocks int64) error {
	if blocks == 0 {
		return nil
	}
	if blocks < 0 {
		return errors.Errorf("Invalid blocks: %d", blocks)
	}

	for i := int64(0); i < blocks; i++ {
		if _, err := sim.GoByBlock(csi, nil); err != nil {
			return err
		}
	}

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

func (sim *simulatorImpl) GoByBlock(csi module.ConsensusInfo, blk Block) ([]Receipt, error) {
	var err error
	var receipts []Receipt

	wss := sim.wss
	blockHeight := sim.blockHeight + 1
	ws := newWorldState(wss, false)
	wc := NewWorldContext(ws, blockHeight, sim.Revision(), csi, sim.stepPrice)

	if err = sim.onExecutionBegin(wc); err != nil {
		return nil, err
	}
	if err = sim.onBaseTx(wc); err != nil {
		return nil, err
	}

	// Execute transactions in a given block
	if blk != nil {
		receipts = make([]Receipt, len(blk.Txs()))

		for i, tx := range blk.Txs() {
			wss = ws.GetSnapshot()
			cc := NewCallContext(wc, tx.From())
			err = sim.executeTx(cc, tx)
			receipts[i] = NewReceipt(blockHeight, err, cc.Events())

			if err != nil {
				if err = ws.Reset(wss); err != nil {
					return nil, err
				}
			}
		}
	}

	if err = sim.onExecutionEnd(wc); err != nil {
		return receipts, err
	}

	wss = ws.GetSnapshot()
	if err = wss.Flush(); err != nil {
		return receipts, err
	}

	sim.onFinalize(wss)

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

func (sim *simulatorImpl) goByOneTransaction(
	csi module.ConsensusInfo, txType TxType, from module.Address, args ...interface{}) (Receipt, error) {
	tx := NewTransaction(txType, from, args...)
	if receipts, err := sim.GoByTransaction(csi, tx); err == nil {
		return receipts[0], err
	} else {
		return nil, err
	}
}

func (sim *simulatorImpl) executeTx(cc *callContext, tx Transaction) error {
	var err error
	es := cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	switch tx.Type() {
	case TypeTransfer:
		err = sim.transfer(es, cc, tx)
	case TypeSetStake:
		err = sim.setStake(es, cc, tx)
	case TypeSetDelegation:
		err = sim.setDelegation(es, cc, tx)
	case TypeSetBond:
		err = sim.setBond(es, cc, tx)
	case TypeSetBonderList:
		err = sim.setBonderList(es, cc, tx)
	case TypeRegisterPRep:
		err = sim.registerPRep(es, cc, tx)
	case TypeUnregisterPRep:
		err = sim.unregisterPRep(es, cc, tx)
	case TypeDisqualifyPRep:
		err = sim.disqualifyPRep(es, cc, tx)
	case TypeSetPRep:
		err = sim.setPRep(es, cc, tx)
	case TypeSetRevision:
		err = sim.setRevision(es, cc, tx)
	case TypeClaimIScore:
		err = sim.claimIScore(es, cc, tx)
	case TypeSetSlashingRates:
		err = sim.setSlashingRates(es, cc, tx)
	case TypeSetMinimumBond:
		err = sim.setMinimumBond(es, cc, tx)
	case TypeInitCommissionRate:
		err = sim.initCommissionRate(es, cc, tx)
	case TypeSetCommissionRate:
		err = sim.setCommissionRate(es, cc, tx)
	case TypeRequestUnjail:
		err = sim.requestUnjail(es, cc, tx)
	case TypeHandleDoubleSignReport:
		err = sim.handleDoubleSignReport(es, cc, tx)
	case TypeSetPRepCountConfig:
		err = sim.setPRepCountConfig(es, cc, tx)
	default:
		return errors.Errorf("Unexpected transaction: %v", tx.Type())
	}
	return err
}

func (sim *simulatorImpl) RegisterPRep(from module.Address, info *icstate.PRepInfo) Transaction {
	return NewTransaction(TypeRegisterPRep, from, info)
}

func (sim *simulatorImpl) GoByRegisterPRep(
	csi module.ConsensusInfo, from module.Address, info *icstate.PRepInfo) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeRegisterPRep, from, info)
}

func (sim *simulatorImpl) registerPRep(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	info := args[0].(*icstate.PRepInfo)
	if err := cc.Transfer(tx.From(), state.SystemAddress, icmodule.BigIntRegPRepFee, module.RegPRep); err != nil {
		return err
	}
	return es.RegisterPRep(cc, info)
}

func (sim *simulatorImpl) UnregisterPRep(from module.Address) Transaction {
	return NewTransaction(TypeUnregisterPRep, from)
}

func (sim *simulatorImpl) GoByUnregisterPRep(
	csi module.ConsensusInfo, from module.Address) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeUnregisterPRep, from)
}

func (sim *simulatorImpl) unregisterPRep(es *iiss.ExtensionStateImpl, cc *callContext, _ Transaction) error {
	return es.UnregisterPRep(cc)
}

func (sim *simulatorImpl) DisqualifyPRep(from module.Address, address module.Address) Transaction {
	return NewTransaction(TypeDisqualifyPRep, from, address)
}

func (sim *simulatorImpl) GoByDisqualifyPRep(csi module.ConsensusInfo, from, address module.Address) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeDisqualifyPRep, from, address)
}

func (sim *simulatorImpl) disqualifyPRep(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	address := args[0].(module.Address)
	return es.DisqualifyPRep(cc, address)
}

func (sim *simulatorImpl) SetPRep(from module.Address, info *icstate.PRepInfo) Transaction {
	return NewTransaction(TypeSetPRep, from, info)
}

func (sim *simulatorImpl) GoBySetPRep(
	csi module.ConsensusInfo, from module.Address, info *icstate.PRepInfo) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetPRep, from, info)
}

func (sim *simulatorImpl) setPRep(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	info := args[0].(*icstate.PRepInfo)
	return es.SetPRep(cc, info, false)
}

func (sim *simulatorImpl) GetDelegationInJSON(from module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ia := es.State.GetAccountSnapshot(from)
	return ia.GetDelegationInJSON()
}

func (sim *simulatorImpl) SetDelegation(from module.Address, ds icstate.Delegations) Transaction {
	return NewTransaction(TypeSetDelegation, from, ds)
}

func (sim *simulatorImpl) GoBySetDelegation(
	csi module.ConsensusInfo, from module.Address, ds *icstate.Delegations) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetDelegation, from, ds)
}

func (sim *simulatorImpl) setDelegation(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	ds := args[0].(icstate.Delegations)
	return es.SetDelegation(cc, ds)
}

func (sim *simulatorImpl) GetBondInJSON(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetBond(address)
	return jso
}

func (sim *simulatorImpl) SetBond(from module.Address, bonds icstate.Bonds) Transaction {
	return NewTransaction(TypeSetBond, from, bonds)
}

func (sim *simulatorImpl) GoBySetBond(
	csi module.ConsensusInfo, from module.Address, bonds icstate.Bonds) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetBond, from, bonds)
}

func (sim *simulatorImpl) setBond(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	bonds := args[0].(icstate.Bonds)
	return es.SetBond(cc, bonds)
}

func (sim *simulatorImpl) GetBonderListInJSON(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetBonderList(address)
	return jso
}

func (sim *simulatorImpl) GetBonderList(address module.Address) icstate.BonderList {
	es := sim.getExtensionState(true)
	pb := es.State.GetPRepBaseByOwner(address, false)
	return pb.BonderList()
}

func (sim *simulatorImpl) SetBonderList(from module.Address, bl icstate.BonderList) Transaction {
	return NewTransaction(TypeSetBonderList, from, bl)
}

func (sim *simulatorImpl) GoBySetBonderList(
	csi module.ConsensusInfo, from module.Address, bl icstate.BonderList) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetBonderList, from, bl)
}

func (sim *simulatorImpl) setBonderList(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	bl := args[0].(icstate.BonderList)
	return es.SetBonderList(cc.From(), bl)
}

func (sim *simulatorImpl) ClaimIScore(from module.Address) Transaction {
	return NewTransaction(TypeClaimIScore, from)
}

func (sim *simulatorImpl) GoByClaimIScore(
	csi module.ConsensusInfo, from module.Address) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeClaimIScore, from)
}

func (sim *simulatorImpl) claimIScore(es *iiss.ExtensionStateImpl, cc *callContext, _ Transaction) error {
	return es.ClaimIScore(cc)
}

func (sim *simulatorImpl) QueryIScore(address module.Address) *big.Int {
	es := sim.getExtensionState(true)
	iscore, _ := es.GetIScore(address, sim.revision.Value(), nil)
	return iscore
}

func (sim *simulatorImpl) GetPRepTermInJSON() map[string]interface{} {
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

func (sim *simulatorImpl) GetPRepsInJSON() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetPRepsInJSON(sim.newCallContext(), 0, 0)
	return jso
}

func (sim *simulatorImpl) NewDefaultConsensusInfo() module.ConsensusInfo {
	vl := sim.ValidatorList()
	voted := make([]bool, len(vl))
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}
	return NewConsensusInfo(sim.Database(), vl, voted)
}

func (sim *simulatorImpl) NewConsensusInfo(voted []bool) (module.ConsensusInfo, error) {
	vl := sim.ValidatorList()
	if len(vl) != len(voted) {
		return nil, errors.Errorf("len(vl) != len(voted): len(vl)=%d len(voted)=%d", len(vl), len(voted))
	}
	return NewConsensusInfo(sim.Database(), vl, voted), nil
}

func (sim *simulatorImpl) GetMainPRepsInJSON() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetMainPRepsInJSON(sim.BlockHeight())
	return jso
}

func (sim *simulatorImpl) GetSubPRepsInJSON() map[string]interface{} {
	es := sim.getExtensionState(true)
	jso, _ := es.GetSubPRepsInJSON(sim.BlockHeight())
	return jso
}

func (sim *simulatorImpl) GetPRep(address module.Address) *icstate.PRep {
	es := sim.getExtensionState(true)
	return es.State.GetPRepByOwner(address)
}

func (sim *simulatorImpl) GetPRepStatsInJSON(address module.Address) map[string]interface{} {
	es := sim.getExtensionState(true)
	ps := es.State.GetPRepStatusByOwner(address, false)
	sc := sim.GetStateContext()
	return ps.GetStatsInJSON(sc)
}

func (sim *simulatorImpl) GetNetworkInfoInJSON() map[string]interface{} {
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

func (sim *simulatorImpl) GetPReps(grade icstate.Grade) []*icstate.PRep {
	cfg := sim.config
	es := sim.getExtensionState(true)
	activePReps := es.State.GetPReps(true)

	preps := make([]*icstate.PRep, 0, cfg.SubPRepCount)
	for _, prep := range activePReps {
		if prep.Grade() == grade {
			preps = append(preps, prep)
		}
	}
	return preps
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

func (sim *simulatorImpl) GetAccountSnapshot(address module.Address) *icstate.AccountSnapshot {
	es := sim.getExtensionState(true)
	return es.State.GetAccountSnapshot(address)
}

func (sim *simulatorImpl) GetStateContext() icmodule.StateContext {
	return &mockStateContext{
		sim.BlockHeight(),
		sim.Revision().Value(),
		sim.Revision().Value(),
		icstate.IISSVersion3,
		int64(0),
		icmodule.ToRate(5),
	}
}

func (sim *simulatorImpl) GetSlashingRates() (map[string]interface{}, error) {
	es := sim.getExtensionState(true)
	cc := sim.newCallContext()
	return es.GetSlashingRates(cc)
}

func (sim *simulatorImpl) SetSlashingRates(from module.Address, rates map[string]icmodule.Rate) Transaction {
	return NewTransaction(TypeSetSlashingRates, from, rates)
}

func (sim *simulatorImpl) GoBySetSlashingRates(
	csi module.ConsensusInfo, from module.Address, rates map[string]icmodule.Rate) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetSlashingRates, from, rates)
}

func (sim *simulatorImpl) setSlashingRates(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	rates := args[0].(map[string]icmodule.Rate)
	return es.SetSlashingRates(cc, rates)
}

func (sim *simulatorImpl) GetMinimumBond() *big.Int {
	es := sim.getExtensionState(true)
	return es.State.GetMinimumBond()
}

func (sim *simulatorImpl) SetMinimumBond(from module.Address, bond *big.Int) Transaction {
	return NewTransaction(TypeSetMinimumBond, from, bond)
}

func (sim *simulatorImpl) GoBySetMinimumBond(
	csi module.ConsensusInfo, from module.Address, bond *big.Int) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetMinimumBond, from, bond)
}

func (sim *simulatorImpl) setMinimumBond(es *iiss.ExtensionStateImpl, _ *callContext, tx Transaction) error {
	args := tx.Args()
	bond := args[0].(*big.Int)
	return es.State.SetMinimumBond(bond)
}

func (sim *simulatorImpl) InitCommissionRate(
	from module.Address, rate, maxRate, maxChangeRate icmodule.Rate) Transaction {
	return NewTransaction(TypeInitCommissionRate, from, rate, maxRate, maxChangeRate)
}

func (sim *simulatorImpl) GoByInitCommissionRate(
	csi module.ConsensusInfo, from module.Address, rate, maxRate, maxChangeRate icmodule.Rate) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeInitCommissionRate, from, rate, maxRate, maxChangeRate)
}

func (sim *simulatorImpl) initCommissionRate(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	rate := args[0].(icmodule.Rate)
	maxRate := args[1].(icmodule.Rate)
	maxChangeRate := args[2].(icmodule.Rate)
	return es.InitCommissionInfo(cc, rate, maxRate, maxChangeRate)
}

func (sim *simulatorImpl) SetCommissionRate(from module.Address, rate icmodule.Rate) Transaction {
	return NewTransaction(TypeSetCommissionRate, from, rate)
}

func (sim *simulatorImpl) GoBySetCommissionRate(
	csi module.ConsensusInfo, from module.Address, rate icmodule.Rate) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetCommissionRate, from, rate)
}

func (sim *simulatorImpl) setCommissionRate(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	rate := args[0].(icmodule.Rate)
	return es.SetCommissionRate(cc, rate)
}

func (sim *simulatorImpl) RequestUnjail(from module.Address) Transaction {
	return NewTransaction(TypeSetCommissionRate, from)
}

func (sim *simulatorImpl) GoByRequestUnjail(
	csi module.ConsensusInfo, from module.Address) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeRequestUnjail, from)
}

func (sim *simulatorImpl) requestUnjail(es *iiss.ExtensionStateImpl, cc *callContext, _ Transaction) error {
	return es.RequestUnjail(cc)
}

func (sim *simulatorImpl) HandleDoubleSignReport(
	from module.Address, dsType string, dsBlockHeight int64, signer module.Address) Transaction {
	return NewTransaction(TypeHandleDoubleSignReport, from, dsType, dsBlockHeight, signer)
}

func (sim *simulatorImpl) GoByHandleDoubleSignReport(csi module.ConsensusInfo,
	from module.Address, dsType string, dsBlockHeight int64, signer module.Address) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeHandleDoubleSignReport, from, dsType, dsBlockHeight, signer)
}

func (sim *simulatorImpl) handleDoubleSignReport(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	dsType := args[0].(string)
	dsBlockHeight := args[1].(int64)
	signer := args[2].(module.Address)
	return es.HandleDoubleSignReport(cc, dsType, dsBlockHeight, signer)
}

func (sim *simulatorImpl) GetPRepCountConfig() (map[string]interface{}, error) {
	es := sim.getExtensionState(true)
	return es.GetPRepCountConfig()
}

func (sim *simulatorImpl) SetPRepCountConfig(from module.Address, counts map[string]int64) Transaction {
	return NewTransaction(TypeSetPRepCountConfig, from, counts)
}

func (sim *simulatorImpl) GoBySetPRepCountConfig(
	csi module.ConsensusInfo, from module.Address, counts map[string]int64) (Receipt, error) {
	return sim.goByOneTransaction(csi, TypeSetPRepCountConfig, from, counts)
}

func (sim *simulatorImpl) setPRepCountConfig(es *iiss.ExtensionStateImpl, cc *callContext, tx Transaction) error {
	args := tx.Args()
	counts := args[0].(map[string]int64)
	return es.SetPRepCountConfig(cc, counts)
}

func NewSimulator(
	revision module.Revision, initValidators []module.Validator, initBalances map[string]*big.Int, config *SimConfig,
) (Simulator, error) {
	sim := &simulatorImpl{
		logger:      log.GlobalLogger(),
		blockHeight: 0,
		stepPrice:   icmodule.BigIntZero,
		config:      config,
	}
	if err := sim.init(revision, initValidators, initBalances); err != nil {
		return nil, err
	}
	return sim, nil
}

func CheckReceiptSuccess(receipts ...Receipt) bool {
	for _, rcpt := range receipts {
		if rcpt.Status() != 1 {
			return false
		}
	}
	return true
}
