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
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

func (s *chainScore) tryChargeCall(iiss bool) error {
	if s.gov {
		return nil
	}
	noCharge := (s.flags & SysNoCharge) != 0
	if !noCharge {
		if err := s.cc.ApplyCallSteps(); err != nil {
			return err
		}
	}
	if iiss {
		if (s.flags & IISSDisabled) != 0 {
			return scoreresult.ContractNotFoundError.New("IISSIsDisabled")
		}
	} else {
		if (s.flags & BasicHidden) != 0 {
			return scoreresult.MethodNotFoundError.New("BasicIsHidden")
		}
	}
	return nil
}

// Ex_disableScore disables the given score. Allowed only from score owner.
func (s *chainScore) Ex_disableScore(address module.Address) error {
	if err := s.tryChargeCall(false); err != nil {
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

// Ex_enableScore enables the given score. Allowed only from score owner.
func (s *chainScore) Ex_enableScore(address module.Address) error {
	if err := s.tryChargeCall(false); err != nil {
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

func (s *chainScore) blockAccounts() {
	for target, _ := range icmodule.BlockedAccount {
		addr := common.MustNewAddressFromString(target)
		as := s.cc.GetAccountState(addr.ID())
		as.SetBlock(true)
	}
}

func (s *chainScore) blockAccounts2() {
	targets := []string{
		"hxb8edf10e2d415f49d8598187e53f146111f549cf",
	}
	for _, target := range targets {
		addr := common.MustNewAddressFromString(target)
		as := s.cc.GetAccountState(addr.ID())
		as.SetBlock(true)
	}
}

// Ex_setRevision sets the system revision to the given number.
// This can only be called by the governance SCORE.
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
	if err := s.handleRevisionChange(int(r), int(code.Int64())); err != nil {
		return err
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
	if err := s.checkGovernance(true); err != nil {
		return nil, err
	}
	if len(txHash) == 0 {
		return nil, scoreresult.ErrInvalidParameter
	}
	scoreAddr := s.getScoreAddress(txHash)
	return scoreAddr, nil
}

func (s *chainScore) Ex_addressToTxHashes(address module.Address) ([]interface{}, error) {
	if err := s.checkGovernance(true); err != nil {
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
	if err := s.tryChargeCall(false); err != nil {
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
	if err := s.tryChargeCall(false); err != nil {
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

// Ex_blockScore blocks the given score address.
// Governance score would check the verification of the address
func (s *chainScore) Ex_blockScore(address module.Address) error {
	if err := s.tryChargeCall(false); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == false && as.IsContract() {
		as.SetBlock(true)
		// add to blocked score list
		sas := s.cc.GetAccountState(state.SystemID)
		db := scoredb.NewArrayDB(sas, state.VarBlockedScores)
		return db.Put(address)
	}
	return nil
}

// Ex_unblockScore unblocks the given score address.
// Governance score would check the verification of the address
func (s *chainScore) Ex_unblockScore(address module.Address) error {
	if err := s.tryChargeCall(false); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true && as.IsContract() {
		as.SetBlock(false)
		// remove from blocked score list
		sas := s.cc.GetAccountState(state.SystemID)
		db := scoredb.NewArrayDB(sas, state.VarBlockedScores)
		for i := 0; i < db.Size(); i++ {
			if db.Get(i).Address().Equal(address) == true {
				rAddr := db.Pop().Address()
				if i < db.Size() { // addr is not rAddr
					if err := db.Set(i, rAddr); err != nil {
						return err
					}
				}
				break
			}
		}
	}
	return nil
}

func (s *chainScore) Ex_getBlockedScores() ([]interface{}, error) {
	if err := s.checkGovernance(true); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarBlockedScores)
	scores := make([]interface{}, db.Size())
	for i := 0; i < db.Size(); i++ {
		scores[i] = db.Get(i).Address()
	}
	return scores, nil
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
	if !state.IsValidStepType(costType) {
		return scoreresult.IllegalFormatError.Errorf("InvalidStepType(%s)", costType)
	}
	costZero := cost.Sign() == 0
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	if stepCostDB.Get(costType) == nil && !costZero {
		if err := stepTypes.Put(costType); err != nil {
			return err
		}
	}
	if costZero {
		// remove the step type and cost
		for i := 0; i < stepTypes.Size(); i++ {
			if stepTypes.Get(i).String() == costType {
				last := stepTypes.Pop().String()
				if i < stepTypes.Size() {
					if err := stepTypes.Set(i, last); err != nil {
						return err
					}
				}
				return stepCostDB.Delete(costType)
			}
		}
		return nil
	} else {
		return stepCostDB.Set(costType, cost)
	}
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

func (s *chainScore) Ex_getRevision() (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64(), nil
}

func (s *chainScore) Ex_getStepPrice() (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Int64(), nil
}

func (s *chainScore) Ex_getStepCost(t string) (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
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
	if err := s.tryChargeCall(false); err != nil {
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
	if err := s.tryChargeCall(false); err != nil {
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
	if err := s.tryChargeCall(false); err != nil {
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

func (s *chainScore) Ex_getScoreDepositInfo(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	if err := s.checkGovernance(true); err != nil {
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

	return as.GetDepositInfo(s.cc, module.JSONVersion3)
}

func (s *chainScore) Ex_getServiceConfig() (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarServiceConfig).Int64(), nil
}

func (s *chainScore) Ex_getFeeSharingConfig() (map[string]interface{}, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	systemConfig := scoredb.NewVarDB(as, state.VarServiceConfig).Int64()
	fsConfig := make(map[string]interface{})
	fsConfig["feeSharingEnabled"] = systemConfig&state.SysConfigFeeSharing != 0
	fsConfig["depositTerm"] = s.cc.DepositTerm()
	fsConfig["depositIssueRate"] = s.cc.DepositIssueRate()
	return fsConfig, nil
}

func (s *chainScore) Ex_setUseSystemDeposit(address module.Address, yn bool) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() != address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "InvalidPrefixForAddress")
	}
	if !as.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "NotContract")
	}
	return as.SetUseSystemDeposit(yn)
}

func (s *chainScore) Ex_getUseSystemDeposit(address module.Address) (bool, error) {
	if err := s.tryChargeCall(false); err != nil {
		return false, err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() != address.IsContract() {
		return false, scoreresult.New(StatusIllegalArgument, "InvalidPrefixForAddress")
	}
	return as.UseSystemDeposit(), nil
}
