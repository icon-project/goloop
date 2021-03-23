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
	if r1 >= r2 || r2 < icmodule.RevisionIISS {
		return nil
	}
	if r1 < icmodule.RevisionIISS && r2 >= icmodule.RevisionIISS {
		iconConfig := s.loadIconConfig()

		s.cc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(s.cc.Database(), nil))
		es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
		if err := es.State.SetIISSVersion(int(iconConfig.IISSVersion.Int64())); err != nil {
			return err
		}
		if err := es.State.SetIISSBlockHeight(iconConfig.IISSBlockHeight.Int64()); err != nil {
			return err
		}
		if err := es.State.SetTermPeriod(iconConfig.TermPeriod.Int64()); err != nil {
			return err
		}
		if err := es.State.SetIRep(iconConfig.Irep.Value()); err != nil {
			return err
		}
		if err := es.State.SetRRep(iconConfig.Rrep.Value()); err != nil {
			return err
		}
		if err := es.State.SetMainPRepCount(iconConfig.MainPRepCount.Int64()); err != nil {
			return err
		}
		if err := es.State.SetSubPRepCount(iconConfig.SubPRepCount.Int64()); err != nil {
			return err
		}
		if err := es.State.SetBondRequirement(iconConfig.BondRequirement.Int64()); err != nil {
			return err
		}
		if err := es.State.SetLockVariables(iconConfig.LockMin.Value(), iconConfig.LockMax.Value()); err != nil {
			return err
		}
		if err := es.State.SetUnbondingPeriodMultiplier(iconConfig.UnbondingPeriodMultiplier.Int64()); err != nil {
			return err
		}
		if err := applyRewardFund(iconConfig, es.State); err != nil {
			return err
		}
		if err := es.State.SetUnstakeSlotMax(iconConfig.UnstakeSlotMax.Int64()); err != nil {
			return err
		}
		if err := es.State.SetUnbondingMax(iconConfig.UnbondingMax.Value()); err != nil {
			return err
		}
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

func (s *chainScore) Ex_setStepCost(costType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if stepCostDB.Get(costType) == nil {
		stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
		if err := stepTypes.Put(costType); err != nil {
			return err
		}
	}
	return stepCostDB.Set(costType, cost)
}

func (s *chainScore) Ex_setMaxStepLimit(contextType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if stepLimitDB.Get(contextType) == nil {
		stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
		if err := stepLimitTypes.Put(contextType); err != nil {
			return err
		}
	}
	return stepLimitDB.Set(contextType, cost)
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

func (s *chainScore) Ex_getStepCost(t string) (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if v := stepCostDB.Get(t); v != nil {
		return v.Int64(), nil
	}
	return 0, nil
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
