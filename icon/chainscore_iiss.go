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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/service/scoreresult"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func (s *chainScore) iissHandleRevision() error {
	revision := s.cc.Revision().Value()
	if revision < icmodule.RevisionIISS {
		// chain SCORE was added on RevisionIISS
		if !s.cc.ApplySteps(state.StepTypeContractCall, 1) {
			return scoreresult.OutOfStepError.New("ChargeCallFailureOnContractNotFound")
		}
		return scoreresult.ErrContractNotFound
	}
	return nil
}

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
	return code.Errorf(msg)
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
		return scoreresult.InvalidParameterError.Errorf(err.Error())
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
	if err = s.iissHandleRevision(); err != nil {
		return
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ia := es.GetAccount(s.from)
	v := &value.Int
	logger := s.cc.Logger()

	if v.Cmp(ia.UsingStake()) < 0 {
		msg := "Failed to stake: stake < voting"
		logger.Infof(msg)
		return scoreresult.InvalidParameterError.Errorf(msg)
	}

	stakeInc := new(big.Int).Sub(v, ia.Stake())
	if stakeInc.Sign() == 0 {
		return nil
	}

	account := s.cc.GetAccountState(s.from.ID())
	balance := account.GetBalance()
	availableStake := new(big.Int).Add(balance, ia.GetTotalStake())
	if availableStake.Cmp(v) == -1 {
		return scoreresult.OutOfBalanceError.Errorf("Not enough balance")
	}

	tStake := es.State.GetTotalStake()
	tSupply := icutils.GetTotalSupply(s.cc)
	prevTotalStake := ia.GetTotalStake()

	//update IISS account
	var tl []icstate.TimerJobInfo
	switch stakeInc.Sign() {
	case 1:
		// Condition: stakeInc > 0
		tl, err = ia.DecreaseUnstake(stakeInc)
	case -1:
		expireHeight := s.cc.BlockHeight() + calcUnstakeLockPeriod(es.State, tStake, tSupply).Int64()
		slotMax := int(es.State.GetUnstakeSlotMax())
		tl, err = ia.IncreaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf("Error while updating unstakes")
	}

	for _, t := range tl {
		ts := es.GetUnstakingTimerState(t.Height, true)
		if err = icstate.ScheduleTimerJob(ts, t, s.from); err != nil {
			return scoreresult.UnknownFailureError.Errorf("Error while scheduling UnStaking Timer Job")
		}
	}
	if err = ia.SetStake(v); err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}

	if err = es.State.SetTotalStake(new(big.Int).Add(tStake, stakeInc)); err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}

	// update world account
	totalStake := ia.GetTotalStake()
	cmp := prevTotalStake.Cmp(totalStake)
	if cmp != 0 {
		if cmp > 0 {
			logger.Panicf("Fail to setStake: account.totalStake < prevTotalStake(invalid state)")
		}
		diff := new(big.Int).Sub(totalStake, prevTotalStake)
		account.SetBalance(new(big.Int).Sub(balance, diff))
	}
	return
}

func calcUnstakeLockPeriod(state *icstate.State, totalStake *big.Int, totalSupply *big.Int) *big.Int {
	fstake := new(big.Float).SetInt(totalStake)
	fsupply := new(big.Float).SetInt(totalSupply)
	stakeRate := new(big.Float).Quo(fstake, fsupply)
	rPoint := big.NewFloat(rewardPoint)
	termPeriod := new(big.Int).SetInt64(state.GetTermPeriod())
	lMin := new(big.Int).Mul(state.GetLockMinMultiplier(), termPeriod)
	lMax := new(big.Int).Mul(state.GetLockMaxMultiplier(), termPeriod)
	if stakeRate.Cmp(rPoint) == 1 {
		return lMin
	}

	fNumerator := new(big.Float).SetInt(new(big.Int).Sub(lMax, lMin))
	fDenominator := new(big.Float).Mul(rPoint, rPoint)
	firstOperand := new(big.Float).Quo(fNumerator, fDenominator)
	s := new(big.Float).Sub(stakeRate, rPoint)
	secondOperand := new(big.Float).Mul(s, s)

	iResult := new(big.Int)
	fResult := new(big.Float).Mul(firstOperand, secondOperand)
	fResult.Int(iResult)

	return new(big.Int).Add(iResult, lMin)
}

func (s *chainScore) Ex_getStake(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	ia := es.GetAccount(address)
	blockHeight := s.cc.BlockHeight()
	return ia.GetStakeInJSON(blockHeight), nil
}

func (s *chainScore) Ex_setDelegation(param []interface{}) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ds, err := icstate.NewDelegations(param)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	err = es.SetDelegation(s.cc, s.from, ds)
	if err != nil {
		s.cc.Logger().Infof(err.Error())
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getDelegation(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	ia := es.GetAccount(address)
	return ia.GetDelegationInfo(), nil
}

var regPRepFee = icutils.ToLoop(2000)

func (s *chainScore) Ex_registerPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, nodeAddress module.Address) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	if name == "" || email == "" || website == "" || country == "" || city == "" || details == "" ||
		p2pEndpoint == "" {
		return scoreresult.InvalidParameterError.Errorf("Required param is missed")
	}
	if (nodeAddress != nil && nodeAddress.IsContract()) || s.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("nodeAddress must be EOA")
	}
	if s.value.Cmp(regPRepFee) == -1 {
		return scoreresult.InvalidParameterError.Errorf("Not enough value. %v", s.value)
	}

	// Subtract regPRepFee from chainScore
	as := s.cc.GetAccountState(state.SystemID)
	balance := new(big.Int).Sub(as.GetBalance(), regPRepFee)
	if balance.Sign() < 0 {
		return scoreresult.InvalidParameterError.Errorf("Not enough value. %v", s.value)
	}
	as.SetBalance(balance)

	// Burn regPRepFee
	if ts, err := icutils.DecreaseTotalSupply(s.cc, regPRepFee); err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	} else {
		icutils.OnBurn(s.cc, regPRepFee, ts)
	}

	regInfo := iiss.NewRegInfo(city, country, details, email, name, p2pEndpoint, website, nodeAddress, s.from)
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	irep := new(big.Int).Set(iiss.BigIntInitialIRep)
	if es.IsDecentralized() {
		term := es.State.GetTerm()
		irep.Set(term.Irep())
	}

	err = es.RegisterPRep(regInfo, irep)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}

	term := es.State.GetTerm()
	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-term.StartHeight()),
		s.from,
		icstage.EfEnable,
	)

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepRegistered(Address)")},
		[][]byte{s.from.Bytes()},
	)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_unregisterPRep() error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if s.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("nodeAddress must be EOA")
	}
	err = es.UnregisterPRep(s.cc, s.from)
	if err != nil {
		s.cc.Logger().Infof(err.Error())
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	res, err := es.GetPRepInJSON(address, s.cc.BlockHeight())
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Errorf(err.Error())
	} else {
		return res, nil
	}
}

func (s *chainScore) Ex_getPReps(startRanking, endRanking *common.HexInt) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	var start, end int = 0, 0
	if startRanking != nil && endRanking != nil {
		start = int(startRanking.Int.Int64())
		end = int(endRanking.Int.Int64())
	}
	if start > end {
		return nil, scoreresult.InvalidParameterError.Errorf("Invalid parameter")
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	blockHeight := s.cc.BlockHeight()
	jso, err := es.GetPRepsInJSON(blockHeight, start, end)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	return jso, nil
}

func (s *chainScore) Ex_getMainPReps() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
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
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	return es.GetSubPRepsInJSON(s.cc.BlockHeight())
}

func (s *chainScore) Ex_getPRepManager() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	jso := es.GetPRepManagerInJSON()
	return jso, nil
}

func (s *chainScore) Ex_setPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	regInfo := iiss.NewRegInfo(city, country, details, email, name, p2pEndpoint, website, node, s.from)
	if (node != nil && node.IsContract()) || s.from.IsContract() {
		return scoreresult.InvalidParameterError.Errorf("nodeAddress must be EOA")
	}

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{s.from.Bytes()},
	)

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	err = es.SetPRep(regInfo)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_setGovernanceVariables(irep *common.HexInt) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	return es.SetGovernanceVariables(s.from, new(big.Int).Set(irep.Value()), s.cc.BlockHeight())
}

func (s *chainScore) Ex_setBond(bondList []interface{}) error {
	logger := s.cc.Logger()
	logger.Tracef("Ex_setBond() start: from=%s", s.from)

	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	bonds, err := icstate.NewBonds(bondList)
	if err != nil {
		return s.toScoreResultError(scoreresult.InvalidParameterError, err)
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.SetBond(s.cc, s.from, bonds); err != nil {
		return s.toScoreResultError(scoreresult.UnknownFailureError, err)
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
	account := es.GetAccount(address)
	data := make(map[string]interface{})
	data["bonds"] = account.GetBondsInfo()
	data["unbonds"] = account.GetUnbondsInfo()
	return data, nil
}

func (s *chainScore) Ex_setBonderList(bonderList []interface{}) error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	bl, err := icstate.NewBonderList(bonderList)
	if err != nil {
		return s.toScoreResultError(scoreresult.InvalidParameterError, err)
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.SetBonderList(s.from, bl); err != nil {
		return s.toScoreResultError(scoreresult.UnknownFailureError, err)
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
		return nil, scoreresult.InvalidInstanceError.Errorf(err.Error())
	} else {
		return res, nil
	}
}

func (s *chainScore) Ex_claimIScore() error {
	if err := s.tryChargeCall(true); err != nil {
		return err
	}
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}

	fClaimed, err := es.Front.GetIScoreClaim(s.from)
	if err != nil {
		return scoreresult.InvalidInstanceError.Errorf(err.Error())
	}
	if fClaimed != nil {
		// claim already in this calculation period. there is no IScore to claim
		s.claimEventLog(s.from, new(big.Int), new(big.Int))
		return nil
	}

	is, err := es.Reward.GetIScore(s.from)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf("Failed to get IScore data(%v)", err)
	}
	if is == nil {
		// there is no IScore to claim.
		s.claimEventLog(s.from, new(big.Int), new(big.Int))
		return nil
	}
	iScore := is.Clone()
	bClaimed, err := es.Back.GetIScoreClaim(s.from)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf("Failed to get claim data from back (%s)", err.Error())
	}
	if bClaimed != nil {
		iScore.Value.Sub(iScore.Value, bClaimed.Value)
	}

	if iScore.IsEmpty() {
		// there is no IScore to claim
		s.claimEventLog(s.from, new(big.Int), new(big.Int))
		return nil
	}

	icx, remains := new(big.Int).DivMod(iScore.Value, iiss.BigIntIScoreICXRatio, new(big.Int))
	claim := new(big.Int).Sub(iScore.Value, remains)

	// increase account icx balance
	account := s.cc.GetAccountState(s.from.ID())
	if account == nil {
		return scoreresult.InvalidInstanceError.Errorf("Invalid account %s", s.from.String())
	}
	balance := account.GetBalance()
	account.SetBalance(new(big.Int).Add(balance, icx))

	// decrease treasury icx balance
	tr := s.cc.GetAccountState(s.cc.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Sub(tb, icx))

	// write claim data to front
	// IISS 2.0 : do not burn iScore < 1000
	// IISS 3.1 : burn iScore < 1000. To burn remains, set full iScore
	revision := s.cc.Revision().Value()
	if revision < icmodule.RevisionICON2 {
		err = es.Front.AddIScoreClaim(s.from, claim)
	} else {
		err = es.Front.AddIScoreClaim(s.from, iScore.Value)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf("Failed to add IScore claim event. (%s)", err.Error())
	}

	s.claimEventLog(s.from, claim, icx)

	return nil
}

func (s *chainScore) claimEventLog(address module.Address, claim *big.Int, icx *big.Int) {
	revision := s.cc.Revision().Value()
	if revision < icmodule.Revision9 {
		s.cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimed(int,int)"),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	} else {
		s.cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("IScoreClaimedV2(Address,int,int)"),
				address.Bytes(),
			},
			[][]byte{
				intconv.BigIntToBytes(claim),
				intconv.BigIntToBytes(icx),
			},
		)
	}
}

func (s *chainScore) Ex_queryIScore(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	fClaim, err := es.Front.GetIScoreClaim(address)
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Errorf("Invalid account")
	}
	is := new(big.Int)
	if fClaim == nil {
		iScore, err := es.Reward.GetIScore(address)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Errorf("error while querying IScore")
		}
		if iScore == nil || iScore.IsEmpty() {
			is.SetInt64(0)
		} else {
			is.Set(iScore.Value)
		}
		bClaim, err := es.Back.GetIScoreClaim(address)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Errorf("error while querying IScore")
		}
		if bClaim != nil {
			is.Sub(is, bClaim.Value)
		}
	}

	bh := int64(0)
	if is.Sign() != 0 {
		bh = es.CalculationBlockHeight() - 1
	}

	data := make(map[string]interface{})
	data["blockHeight"] = bh
	data["iscore"] = is
	data["estimatedICX"] = new(big.Int).Div(is, big.NewInt(iiss.IScoreICXRatio))

	return data, nil
}

func (s *chainScore) Ex_estimateUnstakeLockPeriod() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	totalStake := es.State.GetTotalStake()
	totalSupply := icutils.GetTotalSupply(s.cc)
	result := make(map[string]interface{})
	result["unstakeLockPeriod"] = calcUnstakeLockPeriod(es.State, totalStake, totalSupply)
	return result, nil
}

func (s *chainScore) Ex_getPRepTerm() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	jso, err := es.GetPRepTermInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	jso["blockHeight"] = s.cc.BlockHeight()
	return jso, nil
}

func (s *chainScore) Ex_getNetworkValue() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	res, err := es.GetNetworkValueInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return res, nil
}

func (s *chainScore) Ex_getIISSInfo() (map[string]interface{}, error) {
	if err := s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err := s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	term := es.State.GetTerm()
	iissVersion := es.State.GetIISSVersion()

	iissVariables := make(map[string]interface{})
	if iissVersion == icstate.IISSVersion1 {
		iissVariables["irep"] = term.Irep()
		iissVariables["rrep"] = term.Rrep()
	} else if iissVersion == icstate.IISSVersion2 {
		iissVariables = term.RewardFund().ToJSON()
	}

	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil {
		return nil, err
	}
	rcResult := make(map[string]interface{})
	rcResult["iscore"] = rcInfo.PrevCalcReward()
	rcResult["estimatedICX"] = new(big.Int).Div(rcInfo.PrevCalcReward(), iiss.BigIntIScoreICXRatio)
	rcResult["startBlockHeight"] = rcInfo.StartHeight()
	rcResult["endBlockHeight"] = rcInfo.GetEndHeight()
	rcResult["stateHash"] = es.Reward.GetSnapshot().Bytes()

	jso := make(map[string]interface{})
	jso["blockHeight"] = s.cc.BlockHeight()
	jso["nextCalculation"] = term.GetEndBlockHeight() + 1
	jso["nextPRepTerm"] = term.GetEndBlockHeight() + 1
	jso["variable"] = iissVariables
	jso["rcResult"] = rcResult
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
	return es.GetPRepStatsInJSON(s.cc.BlockHeight())
}

func (s *chainScore) Ex_disqualifyPRep(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	if err = es.DisqualifyPRep(address); err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
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
	term := es.State.GetTerm()
	if err := es.ValidateIRep(term.Irep(), new(big.Int).Set(irep.Value()), 0); err != nil {
		return false, scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	return true, nil
}
