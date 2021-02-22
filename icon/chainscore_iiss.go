/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icon

import (
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

const VarUnstakeSlotMax = "unstake_slot_max"

func (s *chainScore) Ex_setIRep(value *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err // this is already formatted inside the method
	}
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err := es.State.SetIRep(new(big.Int).Set(&value.Int))
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getIRep() (int64, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.State.GetIRep().Int64(), nil
}

func (s *chainScore) Ex_getRRep() (int64, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.State.GetRRep().Int64(), nil
}

func (s *chainScore) Ex_setStake(value *common.HexInt) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia := es.GetAccount(s.from)
	v := &value.Int

	if ia.GetVoting().Cmp(v) == 1 {
		return scoreresult.InvalidParameterError.Errorf("Failed to stake: stake < voting")
	}

	prevTotalStake := ia.GetTotalStake()
	stakeInc := new(big.Int).Sub(v, ia.Stake())
	if stakeInc.Sign() == 0 {
		return nil
	}

	account := s.cc.GetAccountState(s.from.ID())
	balance := account.GetBalance()
	availableStake := new(big.Int).Add(balance, ia.Stake())
	if availableStake.Cmp(v) == -1 {
		return scoreresult.OutOfBalanceError.Errorf("Not enough balance")
	}

	tStake := es.State.GetTotalStake()
	tsupply := icutils.GetTotalSupply(s.cc)

	// update IISS account
	expireHeight := s.cc.BlockHeight() + calcUnstakeLockPeriod(es.State, tStake, tsupply).Int64()
	sa := s.cc.GetAccountState(state.SystemID)
	slotMax := int(scoredb.NewVarDB(sa, VarUnstakeSlotMax).Int64())
	tl, err := ia.UpdateUnstake(stakeInc, expireHeight, slotMax)
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

	// update world account
	totalStake := ia.GetTotalStake()
	if prevTotalStake.Cmp(totalStake) != 0 {
		diff := new(big.Int).Sub(totalStake, prevTotalStake)
		account.SetBalance(new(big.Int).Sub(balance, diff))
	}
	if err := es.State.SetTotalStake(new(big.Int).Add(tStake, stakeInc)); err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}

	return nil
}

func calcUnstakeLockPeriod(state *icstate.State, totalStake *big.Int, totalSupply *big.Int) *big.Int {
	fstake := new(big.Float).SetInt(totalStake)
	fsupply := new(big.Float).SetInt(totalSupply)
	stakeRate := new(big.Float).Quo(fstake, fsupply)
	rPoint := big.NewFloat(rewardPoint)
	lMin := state.GetLockMin()
	lMax := state.GetLockMax()
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
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia := es.GetAccount(address)
	return ia.GetStakeInfo(), nil
}

func (s *chainScore) Ex_setDelegation(param []interface{}) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ds, err := icstate.NewDelegations(param)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	err = es.SetDelegation(s.cc, s.from, ds)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getDelegation(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia := es.GetAccount(address)
	return ia.GetDelegationInfo(), nil
}

func (s *chainScore) Ex_registerPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	regInfo := iiss.NewRegInfo(city, country, details, email, name, p2pEndpoint, website, node, s.from)

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err := es.RegisterPRep(regInfo)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}

	term := es.State.GetTerm()
	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-term.StartHeight()),
		s.from,
		true,
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
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err := es.UnregisterPRep(s.cc, s.from)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	res, err := es.GetPRepInJSON(address, s.cc.BlockHeight())
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Errorf(err.Error())
	} else {
		return res, nil
	}
}

func (s *chainScore) Ex_getPReps(startRanking, endRanking *common.HexInt) (map[string]interface{}, error) {
	var start, end int = 0, 0
	if startRanking != nil && endRanking != nil {
		start = int(startRanking.Int.Int64())
		end = int(endRanking.Int.Int64())
	}
	if start > end {
		return nil, scoreresult.InvalidParameterError.Errorf("Invalid parameter")
	}
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	blockHeight := s.cc.BlockHeight()
	jso, err := es.GetPRepsInJSON(blockHeight, start, end)
	if err != nil {
		return nil, scoreresult.InvalidParameterError.Errorf(err.Error())
	}
	return jso, nil
}

func (s *chainScore) Ex_getPRepManager() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	jso := es.GetPRepManagerInJSON()
	return jso, nil
}

func (s *chainScore) Ex_setPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	regInfo := iiss.NewRegInfo(city, country, details, email, name, p2pEndpoint, website, node, s.from)

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{s.from.Bytes()},
	)

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err := es.SetPRep(regInfo)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_setBond(bondList []interface{}) error {
	bonds, err := icstate.NewBonds(bondList)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err = es.SetBond(s.cc, s.from, bonds)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getBond(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	account := es.GetAccount(address)
	data := make(map[string]interface{})
	data["bonds"] = account.GetBondsInfo()
	data["unbonds"] = account.GetUnbondsInfo()
	return data, nil
}

func (s *chainScore) Ex_setBonderList(bonderList []interface{}) error {
	bl, err := icstate.NewBonderList(bonderList)
	if err != nil {
		return scoreresult.InvalidParameterError.Errorf(err.Error())
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err = es.SetBonderList(s.from, bl)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return nil
}

func (s *chainScore) Ex_getBonderList(address module.Address) ([]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	res, err := es.GetBonderList(address)
	if err != nil {
		return nil, scoreresult.InvalidInstanceError.Errorf(err.Error())
	} else {
		return res, nil
	}
}

func (s *chainScore) Ex_claimIScore() error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	claimed, err := es.Front.GetIScoreClaim(s.from)
	if err != nil {
		return scoreresult.InvalidInstanceError.Errorf(err.Error())
	}
	if claimed != nil {
		// claim already in this calculation period
		return scoreresult.UnknownFailureError.Errorf("claim already in this calculation period")
	}

	iScore, err := es.Reward.GetIScore(s.from)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf("cannot find IScore data")
	}
	if iScore == nil {
		// there is no iScore to claim
		return scoreresult.UnknownFailureError.Errorf("no IScore data to claim")
	}
	claimed, err = es.Back.GetIScoreClaim(s.from)
	if err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	if claimed != nil {
		iScore.Value.Sub(iScore.Value, claimed.Value)
	}

	if iScore.IsEmpty() {
		// there is no IScore to claim
		return scoreresult.OutOfBalanceError.Errorf("no IScore to claim")
	}

	icx, remains := new(big.Int).DivMod(iScore.Value, iiss.BigIntIScoreICXRatio, new(big.Int))
	claim := new(big.Int).Sub(iScore.Value, remains)

	// increase account icx balance
	account := s.cc.GetAccountState(s.from.ID())
	if account == nil {
		return scoreresult.InvalidInstanceError.Errorf("Invalid account")
	}
	balance := account.GetBalance()
	account.SetBalance(balance.Add(balance, icx))

	// decrease treasury icx balance
	tr := s.cc.GetAccountState(s.cc.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Sub(tb, icx))

	// write claim data to front
	if err = es.Front.AddIScoreClaim(s.from, claim); err != nil {
		return scoreresult.UnknownFailureError.Errorf(err.Error())
	}

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{
			[]byte("IScoreClaimedV2(Address,int,int)"),
			s.from.Bytes(),
		},
		[][]byte{
			intconv.BigIntToBytes(claim),
			intconv.BigIntToBytes(icx),
		},
	)

	return nil
}

func (s *chainScore) Ex_queryIScore(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
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
			is = iScore.Value
		}
		bClaim, err := es.Back.GetIScoreClaim(address)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Errorf("error while querying IScore")
		}
		if bClaim != nil {
			is.Sub(is, bClaim.Value)
		}
	}

	data := make(map[string]interface{})
	data["blockheight"] = intconv.FormatInt(es.PrevCalculationBlockHeight())
	data["iscore"] = intconv.FormatBigInt(is)
	data["estimatedICX"] = intconv.FormatBigInt(is.Div(is, big.NewInt(iiss.IScoreICXRatio)))

	return data, nil
}

func (s *chainScore) Ex_estimateUnstakeLockPeriod() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	totalStake := es.State.GetTotalStake()
	totalSupply := icutils.GetTotalSupply(s.cc)
	result := make(map[string]interface{})
	result["unstakeLockPeriod"] = calcUnstakeLockPeriod(es.State, totalStake, totalSupply)
	return result, nil
}

func (s *chainScore) Ex_getPRepTerm() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	jso, err := es.GetPRepTermInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	jso["blockHeight"] = s.cc.BlockHeight()
	return jso, nil
}

func (s *chainScore) Ex_getNetworkValue() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	res, err := es.GetNetworkValueInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Errorf(err.Error())
	}
	return res, nil
}

func (s *chainScore) Ex_getIISSInfo() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	term := es.State.GetTerm()
	iissVersion := es.State.GetIISSVersion()

	iissVariables := make(map[string]interface{})
	if iissVersion == icstate.IISSVersion1 {
		iissVariables["irep"] = intconv.FormatBigInt(term.Irep())
		iissVariables["rrep"] = intconv.FormatBigInt(term.Rrep())
	} else {
		iissVariables = term.RewardFund().ToJSON()
	}

	rcInfo, err := es.State.GetRewardCalcInfo()
	if err != nil {
		return nil, err
	}
	rcResult := make(map[string]interface{})
	rcResult["iscore"] = intconv.FormatBigInt(rcInfo.PrevCalcReward())
	rcResult["estimatedICX"] = intconv.FormatBigInt(new(big.Int).Div(rcInfo.PrevCalcReward(), iiss.BigIntIScoreICXRatio))
	rcResult["startBlockHeight"] = intconv.FormatInt(rcInfo.StartHeight())
	rcResult["endBlockHeight"] = intconv.FormatInt(rcInfo.GetEndHeight())
	rcResult["stateHash"] = es.Reward.GetSnapshot().Bytes()

	jso := make(map[string]interface{})
	jso["blockHeight"] = intconv.FormatInt(s.cc.BlockHeight())
	jso["nextCalculation"] = intconv.FormatInt(term.GetEndBlockHeight() + 1)
	jso["nextPRepTerm"] = intconv.FormatInt(term.GetEndBlockHeight() + 1)
	jso["variable"] = iissVariables
	jso["rcResult"] = rcResult
	return jso, nil
}
