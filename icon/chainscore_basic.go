/*
 * Copyright 2020 ICON Foundation
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

package icon

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

const (
	VarScoreDenyList       = "score_deny_list"
	VarImportAllowList     = "import_allow_list"
	VarImportAllowListKeys = "import_allow_listKeys"
)

func (s *chainScore) tryChargeCall() error {
	if !s.gov {
		if !s.cc.ApplySteps(state.StepTypeContractCall, 1) {
			return scoreresult.OutOfStepError.New("UserCodeError")
		}
	}
	return nil
}

// Destroy : Allowed from score owner
func (s *chainScore) Ex_disableScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.New(StatusNotFound, "NoContract")
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(true)
	return nil
}

func (s *chainScore) Ex_enableScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.New(StatusNotFound, "NoContract")
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(false)
	return nil
}

func (s *chainScore) fromGovernance() bool {
	return s.cc.Governance().Equal(s.from)
}

func (s *chainScore) handleRevisionChange(as state.AccountState, r1, r2 int) error {
	if r1 >= r2 {
		return nil
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if es != nil {
		var err error
		termPeriod := es.State.GetTermPeriod()
		iissVersion := icstate.IISSVersion0

		if r2 >= icmodule.RevisionIISS {
			if termPeriod == defaultTermPeriod {
				if err = es.State.SetTermPeriod(43200); err != nil {
					return err
				}
			}
			iissVersion = icstate.IISSVersion1
		}

		if r2 >= icmodule.RevisionDecentralize {
			if termPeriod == defaultTermPeriod {
				if err = es.State.SetTermPeriod(43120); err != nil {
					return err
				}
			}
		}

		if r2 >= icmodule.RevisionICON2 {
			iissVersion = icstate.IISSVersion2
		}

		if err = es.State.SetIISSVersion(iissVersion); err != nil {
			return err
		}

		if err = es.GenesisTerm(s.cc.BlockHeight(), r2); err != nil {
			return err
		}
	}
	return nil
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *chainScore) Ex_setRevision(code *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if icmodule.MaxRevision < code.Int64() {
		return scoreresult.Errorf(StatusIllegalArgument,
			"IllegalArgument(max=%#x,new=%s)", icmodule.MaxRevision, code)
	}

	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code.Int64() < r {
		return scoreresult.Errorf(StatusIllegalArgument,
			"IllegalArgument(current=%#x,new=%s)", r, code)
	}

	if err := scoredb.NewVarDB(as, state.VarRevision).Set(code); err != nil {
		return err
	}
	if err := s.handleRevisionChange(as, int(r), int(code.Int64())); err != nil {
		return nil
	}
	as.MigrateForRevision(s.cc.ToRevision(int(code.Int64())))
	as.SetAPIInfo(s.GetAPI())
	return nil
}

func (s *chainScore) getScoreAddress(txHash []byte) module.Address {
	sysAs := s.cc.GetAccountState(state.SystemID)
	h2a := scoredb.NewDictDB(sysAs, state.VarTxHashToAddress, 1)
	value := h2a.Get(txHash)
	if value != nil {
		return value.Address()
	}
	return nil
}

func (s *chainScore) Ex_txHashToAddress(txHash []byte) (module.Address, error) {
	if err := s.checkGovernance(false); err != nil {
		return nil, err
	}
	if len(txHash) == 0 {
		return nil, scoreresult.ErrInvalidParameter
	}
	scoreAddr := s.getScoreAddress(txHash)
	return scoreAddr, nil
}

func (s *chainScore) Ex_addressToTxHashes(address module.Address) ([]interface{}, error) {
	if err := s.checkGovernance(false); err != nil {
		return nil, err
	}
	if !address.IsContract() {
		return nil, scoreresult.New(StatusIllegalArgument, "address must be contract")
	}
	as := s.cc.GetAccountState(address.ID())
	if as == nil || !as.IsContract() {
		return nil, scoreresult.New(StatusNotFound, "ContractNotFound")
	}
	result := make([]interface{}, 2)
	if cur := as.Contract(); cur != nil {
		result[0] = cur.DeployTxHash()
	}
	if next := as.NextContract(); next != nil {
		result[1] = next.DeployTxHash()
	}
	return result, nil
}

func (s *chainScore) Ex_acceptScore(txHash []byte) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if len(txHash) == 0 {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	ch := contract.NewCommonHandler(s.from, state.SystemAddress, big.NewInt(0), false, s.log)
	ah := contract.NewAcceptHandler(ch, txHash, auditTxHash)
	status, steps, _, _ := s.cc.Call(ah, s.cc.StepAvailable())
	s.cc.DeductSteps(steps)
	return status
}

func (s *chainScore) Ex_rejectScore(txHash []byte) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if len(txHash) == 0 {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}

	sysAs := s.cc.GetAccountState(state.SystemID)
	h2a := scoredb.NewDictDB(sysAs, state.VarTxHashToAddress, 1)
	value := h2a.Get(txHash)
	if value == nil {
		return scoreresult.Errorf(StatusNotFound, "NoPendingTx")
	}
	scoreAs := s.cc.GetAccountState(value.Address().ID())
	// NOTE : cannot change from reject to accept state because data with address mapped txHash is deleted from DB
	info := s.cc.GetInfo()
	auditTxHash := info[state.InfoTxHash].([]byte)
	if err := h2a.Delete(txHash); err != nil {
		return err
	}
	return scoreAs.RejectContract(txHash, auditTxHash)
}

// Governance score would check the verification of the address
func (s *chainScore) Ex_blockScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == false {
		as.SetBlock(true)
	}
	return nil
}

// Governance score would check the verification of the address
func (s *chainScore) Ex_unblockScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true {
		as.SetBlock(false)
	}
	return nil
}

func (s *chainScore) Ex_setStepPrice(price *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

// User calls icx_call : Functions which can be called by anyone.
func (s *chainScore) Ex_getRevision() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64(), nil
}

func (s *chainScore) Ex_getStepPrice() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Int64(), nil
}

func (s *chainScore) Ex_getStepCosts() (map[string]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)

	stepCosts := make(map[string]interface{})
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		stepCosts[tname] = stepCostDB.Get(tname).Int64()
	}
	return stepCosts, nil
}

func (s *chainScore) Ex_getMaxStepLimit(contextType string) (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if v := stepLimitDB.Get(contextType); v != nil {
		return v.Int64(), nil
	}
	return 0, nil
}

func (s *chainScore) Ex_getScoreStatus(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	if !address.IsContract() {
		return nil, scoreresult.New(StatusIllegalArgument, "address must be contract")
	}
	as := s.cc.GetAccountState(address.ID())
	if as == nil || !as.IsContract() {
		return nil, scoreresult.New(StatusNotFound, "ContractNotFound")
	}
	scoreStatus := make(map[string]interface{})

	scoreStatus["owner"] = as.ContractOwner()

	if cur := as.Contract(); cur != nil {
		curContract := make(map[string]interface{})
		curContract["status"] = cur.Status().String()
		curContract["deployTxHash"] = fmt.Sprintf("%#x", cur.DeployTxHash())
		curContract["auditTxHash"] = fmt.Sprintf("%#x", cur.AuditTxHash())
		scoreStatus["current"] = curContract
	}

	if next := as.NextContract(); next != nil {
		nextContract := make(map[string]interface{})
		nextContract["status"] = next.Status().String()
		nextContract["deployTxHash"] = fmt.Sprintf("%#x", next.DeployTxHash())
		scoreStatus["next"] = nextContract
	}

	if di, err := as.GetDepositInfo(s.cc, module.JSONVersion3); err != nil {
		return nil, scoreresult.New(module.StatusUnknownFailure, "FailOnDepositInfo")
	} else if di != nil {
		scoreStatus["depositInfo"] = di
	}

	// blocked
	if as.IsBlocked() == true {
		scoreStatus["blocked"] = "0x1"
	} else {
		scoreStatus["blocked"] = "0x0"
	}

	// disabled
	if as.IsDisabled() == true {
		scoreStatus["disabled"] = "0x1"
	} else {
		scoreStatus["disabled"] = "0x0"
	}
	return scoreStatus, nil
}

func (s *chainScore) Ex_getServiceConfig() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarServiceConfig).Int64(), nil
}

func (s *chainScore) Ex_getScoreDenyList() ([]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	blDB := scoredb.NewArrayDB(as, VarScoreDenyList)
	size := blDB.Size()
	bl := make([]interface{}, size)
	for i := 0; i < size; i++ {
		a := blDB.Get(i).Address()
		bl = append(bl, common.AddressToPtr(a))
	}
	return bl, nil
}

func (s *chainScore) Ex_getImportAllowList() (map[string]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	wlDB := scoredb.NewDictDB(as, VarImportAllowList, 1)
	kDB := scoredb.NewArrayDB(as, VarImportAllowListKeys)
	size := kDB.Size()
	wl := make(map[string]interface{}, size)
	for i := 0; i < size; i++ {
		k := kDB.Get(i).String()
		wl[k] = wlDB.Get(k).String()
	}
	return wl, nil
}

func (s *chainScore) Ex_addScoreDenyList(score module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if !score.IsContract() {
		return scoreresult.New(StatusNotFound, "NoContract")
	}
	as := s.cc.GetAccountState(state.SystemID)
	dlDB := scoredb.NewArrayDB(as, VarScoreDenyList)
	for i := 0; i < dlDB.Size(); i++ {
		v := dlDB.Get(i).Address()
		if v.Equal(score) {
			return scoreresult.New(StatusIllegalArgument, score.String() + "already in deny list")
		}
	}
	return nil
}

func (s *chainScore) Ex_removeScoreDenyList(score module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if !score.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "NoContract")
	}
	as := s.cc.GetAccountState(state.SystemID)
	dlDB := scoredb.NewArrayDB(as, VarScoreDenyList)
	exist := false
	top := dlDB.Pop().Address()
	if !top.Equal(score) {
		for i := 0; i < dlDB.Size(); i++ {
			v := dlDB.Get(i).Address()
			if v.Equal(score) {
				_ = dlDB.Set(i, top)
				exist = true
			}
		}
	} else {
		exist = true
	}
	if !exist{
		return scoreresult.New(StatusIllegalArgument, score.String() + "not in deny list")
	}
	return nil
}

func (s *chainScore) Ex_disqualifyPRep(prep module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.DisqualifyPRep(prep)
}

func (s *chainScore) Ex_validateIrep(newIrep *common.HexInt) (bool, error) {
	if err := s.checkGovernance(true); err != nil {
		return false, err
	}
	if err := s.tryChargeCall(); err != nil {
		return false, err
	}
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	term := es.State.GetTerm()
	if err := es.ValidateIRep(term.Irep(), &newIrep.Int, 0); err != nil {
		return false, err
	}
	return true, nil
}