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
	"github.com/icon-project/goloop/common/intconv"
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
func (s *chainScore) Ex_setRevision(code int64) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if icmodule.MaxRevision < code {
		return scoreresult.Errorf(StatusIllegalArgument,
			"IllegalArgument(max=%d,new=%d)", icmodule.MaxRevision, code)
	}
	rev := int(code)
	old, err := contract.SetRevision(s.cc, rev, false)
	if err != nil {
		if scoreresult.InvalidParameterError.Equals(err) {
			return scoreresult.Wrapf(err, StatusIllegalArgument,
				"IllegalArgument(current=%d,new=%d)", old, rev)
		} else {
			return err
		}
	}
	if err = s.handleRevisionChange(old, rev); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	_ = as.MigrateForRevision(s.cc.ToRevision(rev))
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
		if s.cc.Revision().Value() >= icmodule.RevisionChainScoreEventLog {
			s.emitAccountBlockedSet(address, true)
		}
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
		if s.cc.Revision().Value() >= icmodule.RevisionChainScoreEventLog {
			s.emitAccountBlockedSet(address, false)
		}
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

func (s *chainScore) emitAccountBlockedSet(address module.Address, yn bool) {
	var ynBytes []byte
	if yn == true {
		ynBytes = intconv.Int64ToBytes(1)
	} else {
		ynBytes = intconv.Int64ToBytes(0)
	}
	s.cc.OnEvent(
		state.SystemAddress,
		[][]byte{
			[]byte("AccountBlockedSet(Address,bool)"),
			address.Bytes(),
		},
		[][]byte{
			ynBytes,
		},
	)
}

func (s *chainScore) Ex_blockAccount(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if address == nil || address.IsContract() {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if address.IsContract() != as.IsContract() {
		return scoreresult.InvalidParameterError.New("AddressTypeIsMismatch")
	}
	if as.IsBlocked() == false {
		as.SetBlock(true)
		s.emitAccountBlockedSet(address, true)
	}
	return nil
}

func (s *chainScore) Ex_unblockAccount(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if address == nil || address.IsContract() {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if address.IsContract() != as.IsContract() {
		return scoreresult.InvalidParameterError.New("AddressTypeIsMismatch")
	}
	if as.IsBlocked() == true {
		as.SetBlock(false)
		s.emitAccountBlockedSet(address, false)
	}
	return nil
}

func (s *chainScore) Ex_isBlocked(address module.Address) (bool, error) {
	if err := s.tryChargeCall(false); err != nil {
		return false, err
	}
	if address == nil {
		return false, scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if address.IsContract() != as.IsContract() {
		return false, scoreresult.InvalidParameterError.New("AddressTypeIsMismatch")
	}
	return as.IsBlocked(), nil
}

func (s *chainScore) Ex_setStepPrice(price *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}

	_, err := contract.SetStepPrice(s.cc, price.Value())
	return err
}

func (s *chainScore) Ex_setStepCost(costType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if _, err := contract.SetStepCost(s.cc, costType, cost.Value(), true); err != nil {
		if scoreresult.InvalidParameterError.Equals(err) {
			return scoreresult.IllegalFormatError.Wrapf(err, "InvalidStepType(%s)", costType)
		} else {
			return err
		}
	}
	return nil
}

func (s *chainScore) Ex_setMaxStepLimit(contextType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	_, err := contract.SetMaxStepLimit(s.cc, contextType, cost.Value())
	return err
}

func (s *chainScore) Ex_getRevision() (int64, error) {
	if err := s.tryChargeCall(false); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64(), nil
}

func (s *chainScore) Ex_getStepPrice() (*big.Int, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	return contract.GetStepPrice(s.cc), nil
}

func (s *chainScore) Ex_getStepCost(t string) (*big.Int, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	return contract.GetStepCost(s.cc, t), nil
}

func (s *chainScore) Ex_getStepCosts() (map[string]interface{}, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	return contract.GetStepCosts(s.cc), nil
}

func (s *chainScore) Ex_getMaxStepLimit(contextType string) (*big.Int, error) {
	if err := s.tryChargeCall(false); err != nil {
		return nil, err
	}
	return contract.GetMaxStepLimit(s.cc, contextType), nil
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
