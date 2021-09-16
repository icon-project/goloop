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
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

func (s *chainScore) getExtensionState() (*iiss.ExtensionStateImpl, error) {
	es := s.cc.GetExtensionState()
	if es == nil {
		err := errors.Errorf("ExtensionState is nil")
		return nil, s.toScoreResultError(scoreresult.UnknownFailureError, err)
	}
	esi := es.(*iiss.ExtensionStateImpl)
	esi.SetLogger(icutils.NewIconLogger(s.cc.Logger()))
	return esi, nil
}

func (s *chainScore) toScoreResultError(code errors.Code, err error) error {
	msg := err.Error()
	if logger := s.cc.Logger(); logger != nil {
		logger = icutils.NewIconLogger(logger)
		logger.Infof(msg)
	}
	return code.Wrap(err, msg)
}

func (s *chainScore) Ex_setIRep(value *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.State.SetIRep(new(big.Int).Set(&value.Int)); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err,
			"Failed to set IRep: from=%v value=%v",
			s.from,
			value,
		)
	}
	return nil
}

func (s *chainScore) Ex_getIRep() (int64, error) {
	if err := s.tryChargeCall(true); err != nil {
		return 0, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return 0, err
	}
	return es.State.GetIRep().Int64(), nil
}

func (s *chainScore) Ex_getRRep() (int64, error) {
	if err := s.tryChargeCall(true); err != nil {
		return 0, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return 0, nil
	}
	return es.State.GetRRep().Int64(), nil
}

func (s *chainScore) Ex_setStake(value *common.HexInt) (err error) {
	if err = s.tryChargeCall(true); err != nil {
		return
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	return es.SetStake(cc, &value.Int)
}

func (s *chainScore) Ex_getStake(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	ia := es.State.GetAccountSnapshot(address)
	if ia == nil {
		ia = icstate.GetEmptyAccountSnapshot()
	}
	blockHeight := s.cc.BlockHeight()
	return ia.GetStakeInJSON(blockHeight), nil
}

func (s *chainScore) Ex_setDelegation(param []interface{}) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ds, err := icstate.NewDelegations(param, es.State.GetDelegationSlotMax())
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	return es.SetDelegation(cc, ds)
}

func (s *chainScore) Ex_getDelegation(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	ia := es.State.GetAccountSnapshot(address)
	if ia == nil {
		ia = icstate.GetEmptyAccountSnapshot()
	}

	return ia.GetDelegationInJSON(), nil
}

func (s *chainScore) Ex_registerPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, nodeAddress module.Address) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if name == "" || email == "" || website == "" || country == "" || city == "" || details == "" ||
		p2pEndpoint == "" {
		return scoreresult.InvalidParameterError.Errorf("Required param is missed")
	}
	if (nodeAddress != nil && nodeAddress.IsContract()) || s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v node=%v",
			s.from,
			nodeAddress,
		)
	}
	if s.value.Cmp(icmodule.BigIntRegPRepFee) != 0 {
		return scoreresult.InvalidParameterError.Errorf(
			"Invalid registration fee: value=%v != fee=%v",
			s.value,
			icmodule.BigIntRegPRepFee,
		)
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)

	info := &icstate.PRepInfo{
		City:        &city,
		Country:     &country,
		Details:     &details,
		Email:       &email,
		Name:        &name,
		P2PEndpoint: &p2pEndpoint,
		WebSite:     &website,
		Node:        nodeAddress,
	}
	return es.RegisterPRep(cc, info)
}

func (s *chainScore) Ex_unregisterPRep() error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v", s.from,
		)
	}
	cc := s.newCallContext(s.cc)
	return es.UnregisterPRep(cc)
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	res, err := es.GetPRepInJSON(address, s.cc.BlockHeight())
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Wrap(err, "Failed to get PRep")
	} else {
		return res, nil
	}
}

func (s *chainScore) Ex_getPReps(startRanking, endRanking *common.HexInt) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	var start, end = 0, 0
	if startRanking != nil && endRanking != nil {
		start = int(startRanking.Int.Int64())
		end = int(endRanking.Int.Int64())
	}
	if start > end {
		return nil, scoreresult.InvalidParameterError.Errorf(
			"Invalid ranking: start=%d > end=%d", start, end,
		)
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	blockHeight := s.cc.BlockHeight()
	jso, err := es.State.GetPRepsInJSON(blockHeight, start, end)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to get PReps: start=%d end=%d", start, end,
		)
	}
	return jso, nil
}

func (s *chainScore) Ex_getMainPReps() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.GetMainPRepsInJSON(s.cc.BlockHeight())
}

func (s *chainScore) Ex_getSubPReps() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.GetSubPRepsInJSON(s.cc.BlockHeight())
}

func (s *chainScore) Ex_setPRep(name *string, email *string, website *string, country *string,
	city *string, details *string, p2pEndpoint *string, node module.Address) error {
	var err error
	var es *iiss.ExtensionStateImpl

	if err = s.tryChargeCall(true); err != nil {
		return err
	}
	if (node != nil && node.IsContract()) || s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v node=%v", s.from, node,
		)
	}
	if es, err = s.getExtensionState(); err != nil {
		return err
	}
	info := &icstate.PRepInfo{
		City:        city,
		Country:     country,
		Details:     details,
		Email:       email,
		Name:        name,
		P2PEndpoint: p2pEndpoint,
		WebSite:     website,
		Node:        node,
	}
	return es.SetPRep(s.newCallContext(s.cc), info)
}

func (s *chainScore) Ex_setGovernanceVariables(irep *common.HexInt) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf("Invalid address: from=%s", s.from)
	}
	if err = es.SetGovernanceVariables(s.from, new(big.Int).Set(irep.Value()), s.cc.BlockHeight()); err != nil {
		return err
	}
	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("GovernanceVariablesSet(Address,int)"), s.from.Bytes()},
		[][]byte{intconv.BigIntToBytes(irep.Value())},
	)
	return nil
}

func (s *chainScore) Ex_setBond(bondList []interface{}) error {
	logger := s.cc.Logger()
	logger.Tracef("Ex_setBond() start: from=%v", s.from)

	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	bonds, err := icstate.NewBonds(bondList)
	if err != nil {
		return err
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.SetBond(s.cc.BlockHeight(), s.from, bonds); err != nil {
		return err
	}
	logger.Tracef("Ex_setBond() end")
	return nil
}

func (s *chainScore) Ex_getBond(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	account := es.State.GetAccountSnapshot(address)
	if account == nil {
		account = icstate.GetEmptyAccountSnapshot()
	}
	data := make(map[string]interface{})
	data["bonds"] = account.GetBondsInJSON()
	data["unbonds"] = account.GetUnbondsInJSON()
	return data, nil
}

func (s *chainScore) Ex_setBonderList(bonderList []interface{}) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	bl, err := icstate.NewBonderList(bonderList)
	if err != nil {
		return err
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.SetBonderList(s.from, bl); err != nil {
		return err
	}
	return nil
}

func (s *chainScore) Ex_getBonderList(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	res, err := es.GetBonderList(address)
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Wrapf(
			err, "Failed to get bonderList: address=%v", address,
		)
	}
	return res, nil
}

var skippedClaimTX, _ = hex.DecodeString("b9eeb235f715b166cf4b91ffcf8cc48a81913896086d30104ffc0cf47eed1cbd")

func (s *chainScore) Ex_claimIScore() error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	if bytes.Compare(cc.TransactionID(), skippedClaimTX) == 0 {
		// Skip this TX like ICON1 mainnet.
		iiss.ClaimEventLog(cc, s.from, new(big.Int), new(big.Int))
		return nil
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	return es.ClaimIScore(cc)
}

func (s *chainScore) Ex_queryIScore(address module.Address) (map[string]interface{}, error) {
	var err error
	if err = s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	is, err := es.GetIScore(address)
	if err != nil {
		return nil, err
	}
	bh := int64(0)
	if is.Sign() != 0 {
		bh = es.CalculationBlockHeight() - 1
	}

	jso := make(map[string]interface{})
	jso["blockHeight"] = bh
	jso["iscore"] = is
	jso["estimatedICX"] = icutils.IScoreToICX(is)
	return jso, nil
}

func (s *chainScore) Ex_estimateUnstakeLockPeriod() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	cc := s.newCallContext(s.cc)
	return map[string]interface{}{
		"unstakeLockPeriod": es.State.GetUnstakeLockPeriod(cc.GetTotalSupply()),
	}, nil
}

func (s *chainScore) Ex_getPRepTerm() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	blockHeight := s.cc.BlockHeight()
	jso, err := es.GetPRepTermInJSON(blockHeight)
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrap(err, "Failed to get PRepTerm")
	}
	return jso, nil
}

func (s *chainScore) Ex_getNetworkInfo() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}

	if s.from != nil && s.from.IsContract() {
		 return nil, scoreresult.AccessDeniedError.Errorf("Invalid address: from=%s", s.from)
	}

	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	res, err := es.State.GetNetworkInfoInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrap(err, "Failed to get NetworkValue")
	}
	return res, nil
}

func (s *chainScore) Ex_getIISSInfo() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	term := es.State.GetTermSnapshot()
	iissVersion := es.State.GetIISSVersion()

	iissVariables := make(map[string]interface{})
	if iissVersion == icstate.IISSVersion2 {
		iissVariables["irep"] = term.Irep()
		iissVariables["rrep"] = term.Rrep()
	} else if iissVersion == icstate.IISSVersion3 {
		iissVariables = term.RewardFund().ToJSON()
	}

	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrap(
			err, "Failed to get RewardCalcInfo",
		)
	}

	endBlockHeight := term.GetEndHeight()
	jso := make(map[string]interface{})
	jso["blockHeight"] = s.cc.BlockHeight()
	jso["nextCalculation"] = endBlockHeight + 1
	jso["nextPRepTerm"] = endBlockHeight + 1
	jso["variable"] = iissVariables
	jso["rcResult"] = rcInfo.GetResultInJSON()
	return jso, nil
}

func (s *chainScore) Ex_getPRepStats() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.State.GetPRepStatsInJSON(s.cc.BlockHeight())
}

func (s *chainScore) Ex_disqualifyPRep(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	if err = es.DisqualifyPRep(cc, address); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to disqualify PRep: from=%v prep=%v",
			s.from,
			address,
		)
	}
	return nil
}

func (s *chainScore) Ex_validateIRep(irep *common.HexInt) (bool, error) {
	if err := s.checkGovernance(true); err != nil {
		return false, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return false, err
	}
	term := es.State.GetTermSnapshot()
	if err = es.ValidateIRep(term.Irep(), irep.Value(), 0); err != nil {
		return false, scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate IRep: irep=%v", irep.Value(),
		)
	}
	return true, nil
}

func (s *chainScore) Ex_burn() error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	return es.Burn(cc, s.value)
}

func (s *chainScore) Ex_validateRewardFund(iglobal *common.HexInt) (bool, error) {
	if err := s.checkGovernance(true); err != nil {
		return false, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return false, err
	}
	rewardFund := es.State.GetRewardFund()
	currentIglobal := rewardFund.Iglobal
	min := new(big.Int).Mul(currentIglobal, big.NewInt(3))
	min.Div(min, big.NewInt(4))
	max := new(big.Int).Mul(currentIglobal, big.NewInt(5))
	max.Div(max, big.NewInt(4))
	if (iglobal.Cmp(min) < 0) || (iglobal.Cmp(max) > 0) {
		return false, scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate IGlobal: iglobal=%v", iglobal.Value(),
		)
	}
	cc := s.newCallContext(s.cc)
	totalSupply := cc.GetTotalSupply()
	rewardPerYear := new(big.Int).Mul(iglobal.Value(), big.NewInt(12))
	maxRewardPerYear := new(big.Int).Mul(totalSupply, big.NewInt(115))
	maxRewardPerYear.Div(totalSupply, big.NewInt(100))

	if rewardPerYear.Cmp(maxRewardPerYear) > 0 {
		return false, scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate IGlobal: iglobal=%v", iglobal.Value(),
		)
	}
	return true, nil
}

func (s *chainScore) Ex_setRewardFund(iglobal *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	rewardFund := es.State.GetRewardFund()
	rewardFund.Iglobal = iglobal.Value()
	return es.State.SetRewardFund(rewardFund)
}

func (s *chainScore) Ex_setRewardFundAllocation(iprep *common.HexInt, icps *common.HexInt, irelay *common.HexInt, ivoter *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	rewardFund := es.State.GetRewardFund()
	rewardFund.Iprep = &iprep.Int
	rewardFund.Icps = &icps.Int
	rewardFund.Irelay = &irelay.Int
	rewardFund.Ivoter = &ivoter.Int
	return es.State.SetRewardFund(rewardFund)
}

func (s *chainScore) newCallContext(cc contract.CallContext) icmodule.CallContext {
	return iiss.NewCallContext(cc, s.from)
}
