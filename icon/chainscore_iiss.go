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

func (s *chainScore) checkNetworkScore(charge bool) error {
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ns := es.State.GetNetworkScores(s.newCallContext(s.cc))
	for _, address := range ns {
		if address.Equal(s.from) {
			return nil
		}
	}
	if charge {
		if err := s.cc.ApplyCallSteps(); err != nil {
			return err
		}
	}
	return scoreresult.New(module.StatusAccessDenied, "NoPermission")
}

func (s *chainScore) checkQueryMode() error {
	if s.cc.TransactionID() != nil {
		return scoreresult.AccessDeniedError.Errorf("NotAllowedInTransaction")
	}
	return nil
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
	res, err := es.GetPRepInJSON(s.newCallContext(s.cc), address)
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
	jso, err := es.GetPRepsInJSON(s.newCallContext(s.cc), start, end)
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
	return es.SetPRep(s.newCallContext(s.cc), info, false)
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
	bonds, err := icstate.NewBonds(bondList, s.cc.Revision().Value())
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
	return es.GetBond(address)
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
	is, err := es.GetIScore(address, s.cc.Revision().Value(), s.cc.TransactionID())
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
		"unstakeLockPeriod": es.State.GetUnstakeLockPeriod(cc.Revision().Value(), cc.GetTotalSupply()),
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
	res, err := es.State.GetNetworkInfoInJSON(s.cc.Revision().Value())
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

	var iissVariables map[string]interface{}
	if iissVersion == icstate.IISSVersion2 {
		iissVariables = make(map[string]interface{})
		iissVariables["irep"] = term.Irep()
		iissVariables["rrep"] = term.Rrep()
	} else if iissVersion >= icstate.IISSVersion3 {
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

	rev := s.cc.Revision().Value()
	if rev >= icmodule.RevisionUpdatePRepStats {
		if err := s.checkQueryMode(); err != nil {
			return nil, err
		}
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.State.GetPRepStatsInJSON(rev, s.cc.BlockHeight())
}

func (s *chainScore) Ex_getPRepStatsOf(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.checkQueryMode(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.State.GetPRepStatsOfInJSON(s.cc.Revision().Value(), s.cc.BlockHeight(), address)
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
	var err error
	if err = s.tryChargeCall(true); err != nil {
		return err
	}

	cc := s.newCallContext(s.cc)
	from := s.from
	amount := s.value

	if err = cc.Withdraw(state.SystemAddress, amount, module.Burn); err != nil {
		return scoreresult.InvalidParameterError.Errorf(
			"Not enough value: from=%v value=%v", from, amount,
		)
	}
	if err = cc.HandleBurn(from, amount); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to burn: from=%v value=%v", from, amount,
		)
	}
	return nil
}

func (s *chainScore) Ex_validateRewardFund(iglobal *common.HexInt) (bool, error) {
	if err := s.checkGovernance(true); err != nil {
		return false, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return false, err
	}
	cc := s.newCallContext(s.cc)
	if err = es.ValidateRewardFund(iglobal.Value(), cc.GetTotalSupply(), cc.Revision().Value()); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (s *chainScore) Ex_setRewardFund(iglobal *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	revision := s.cc.Revision().Value()
	if revision <= icmodule.RevisionPreIISS4 {
		rf := es.State.GetRewardFundV1()
		rf.SetIGlobal(iglobal.Value())
		if err = es.State.SetRewardFund(rf); err != nil {
			return err
		}
	}

	if revision >= icmodule.RevisionPreIISS4 {
		rf := es.State.GetRewardFundV2()
		rf.SetIGlobal(iglobal.Value())
		return es.State.SetRewardFund(rf)
	}
	return nil
}

func (s *chainScore) Ex_setRewardFundAllocation(iprep *common.HexInt, icps *common.HexInt, irelay *common.HexInt, ivoter *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	rf := es.State.GetRewardFundV1()
	if err = rf.SetAllocation(
		map[icstate.RFundKey]icmodule.Rate{
			icstate.KeyIprep:  icmodule.ToRate(iprep.Int64()),
			icstate.KeyIcps:   icmodule.ToRate(icps.Int64()),
			icstate.KeyIrelay: icmodule.ToRate(irelay.Int64()),
			icstate.KeyIvoter: icmodule.ToRate(ivoter.Int64()),
		},
	); err != nil {
		return err
	}
	return es.State.SetRewardFund(rf)
}

func (s *chainScore) Ex_setRewardFundAllocation2(values []interface{}) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	alloc, err := icstate.NewRewardFund2Allocation(values)
	if err != nil {
		return err
	}
	rf := es.State.GetRewardFundV2()
	rf.SetAllocation(alloc)
	return es.State.SetRewardFund(rf)
}

func (s *chainScore) Ex_getScoreOwner(score module.Address) (module.Address, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	return s.newCallContext(s.cc).GetScoreOwner(score)
}

func (s *chainScore) Ex_setScoreOwner(score module.Address, owner module.Address) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	return s.newCallContext(s.cc).SetScoreOwner(s.from, score, owner)
}

func (s *chainScore) Ex_setNetworkScore(role string, address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if address != nil {
		cc := s.newCallContext(s.cc)
		owner, err := cc.GetScoreOwner(address)
		if err != nil {
			return err
		}
		if !common.AddressEqual(owner, s.cc.Governance()) {
			return icmodule.IllegalArgumentError.Errorf("Only scores owned by governance can be designated")
		}
	}
	return es.State.SetNetworkScore(role, address)
}

func (s *chainScore) Ex_getNetworkScores() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	ns := es.State.GetNetworkScores(s.newCallContext(s.cc))
	jso := make(map[string]interface{})
	for k, v := range ns {
		jso[k] = v
	}
	return jso, nil
}

func (s *chainScore) Ex_addTimer(blockHeight *common.HexInt) error {
	if err := s.checkNetworkScore(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ts := es.State.GetNetworkScoreTimerState(blockHeight.Int64())
	ts.Add(s.from)
	return nil
}

func (s *chainScore) Ex_removeTimer(blockHeight *common.HexInt) error {
	if err := s.checkNetworkScore(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ts := es.State.GetNetworkScoreTimerState(blockHeight.Int64())
	ts.Delete(s.from)
	return nil
}

func (s *chainScore) Ex_penalizeNonvoters(params []interface{}) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	cc := s.newCallContext(s.cc)
	for _, p := range params {
		prep, ok := p.(module.Address)
		if !ok {
			return scoreresult.InvalidParameterError.Errorf("invalid parameter. not an address")
		}
		if err := es.PenalizeNonVoters(cc, prep); err != nil {
			return err
		}
	}
	return nil
}

func (s *chainScore) Ex_setConsistentValidationSlashingRate(slashingRate *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if !slashingRate.IsInt64() {
		return icmodule.IllegalArgumentError.Errorf("Invalid range")
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	rate := icmodule.ToRate(slashingRate.Int64())
	if err = es.State.SetConsistentValidationPenaltySlashRate(s.cc.Revision().Value(), rate); err != nil {
		if errors.IllegalArgumentError.Equals(err) {
			return icmodule.IllegalArgumentError.Errorf("Invalid range")
		}
		return err
	}
	s.onSlashingRateChangedEvent("ConsistentValidationPenalty", rate)
	return nil
}

func (s *chainScore) Ex_setNonVoteSlashingRate(slashingRate *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if !slashingRate.IsInt64() {
		return icmodule.IllegalArgumentError.Errorf("Invalid range")
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	rate := icmodule.ToRate(slashingRate.Int64())
	if err = es.State.SetNonVotePenaltySlashRate(s.cc.Revision().Value(), rate); err != nil {
		if errors.IllegalArgumentError.Equals(err) {
			return icmodule.IllegalArgumentError.Errorf("Invalid range")
		}
		return err
	}
	s.onSlashingRateChangedEvent("NonVotePenalty", rate)
	return nil
}

func (s *chainScore) Ex_setSlashingRates(values []interface{}) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rates := make(map[string]icmodule.Rate)
	for _, v := range values {
		pair, ok := v.(map[string]interface{})
		name, ok := pair["name"].(string)
		if !ok {
			return scoreresult.InvalidParameterError.New("InvalidNameType")
		}
		value, ok := pair["value"].(*common.HexInt)
		if !ok {
			return scoreresult.InvalidParameterError.New("InvalidRateType")
		}
		if _, ok = rates[name]; ok {
			return icmodule.DuplicateError.Errorf("DuplicatePenaltyName(%s)", name)
		}
		rates[name] = icmodule.Rate(value.Int64())
	}
	return es.SetSlashingRates(s.newCallContext(s.cc), rates)
}

func (s *chainScore) Ex_getSlashingRates(values []interface{}) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}

	var penaltyTypes []icmodule.PenaltyType
	for _, v := range values {
		name, ok := v.(string)
		if !ok {
			return nil, scoreresult.InvalidParameterError.New("InvalidPenaltyNameType")
		}
		if pt := icmodule.ToPenaltyType(name); pt == icmodule.PenaltyNone {
			return nil, scoreresult.InvalidParameterError.Errorf("InvalidPenaltyName(%s)", name)
		} else {
			penaltyTypes = append(penaltyTypes, pt)
		}
	}
	return es.GetSlashingRates(penaltyTypes)
}

func (s *chainScore) onSlashingRateChangedEvent(name string, rate icmodule.Rate) {
	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("SlashingRateChanged(str,int)"), []byte(name)},
		[][]byte{intconv.Int64ToBytes(rate.Percent())},
	)
}

func (s *chainScore) newCallContext(cc contract.CallContext) icmodule.CallContext {
	return iiss.NewCallContext(cc, s.from)
}

func (s *chainScore) Ex_initCommissionRate(rate, maxRate, maxChangeRate *common.HexInt) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	return es.InitCommissionInfo(
		s.newCallContext(s.cc),
		icmodule.Rate(rate.Int64()),
		icmodule.Rate(maxRate.Int64()),
		icmodule.Rate(maxChangeRate.Int64()))
}

func (s *chainScore) Ex_setCommissionRate(rate *common.HexInt) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	return es.SetCommissionRate(s.newCallContext(s.cc), icmodule.Rate(rate.Int64()))
}