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
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icstate/migrate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
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
	if err = s.iissHandleRevision(); err != nil {
		return
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}
	ia := es.State.GetAccountState(s.from)
	v := &value.Int

	usingStake := ia.UsingStake()
	if v.Cmp(usingStake) < 0 {
		return scoreresult.InvalidParameterError.Errorf(
			"Failed to set stake: newStake=%v < usingStake=%v from=%v",
			v, usingStake, s.from,
		)
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
	oldTotalStake := ia.GetTotalStake()

	//update IISS account
	revision := s.cc.Revision().Value()
	expireHeight := s.cc.BlockHeight() + calcUnstakeLockPeriod(es.State, tStake, tSupply).Int64()
	var tl []icstate.TimerJobInfo
	switch stakeInc.Sign() {
	case 1:
		// Condition: stakeInc > 0
		tl, err = ia.DecreaseUnstake(stakeInc, expireHeight, revision)
	case -1:
		slotMax := int(es.State.GetUnstakeSlotMax())
		tl, err = ia.IncreaseUnstake(new(big.Int).Abs(stakeInc), expireHeight, slotMax, revision)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Error while updating unstakes: from=%v",
			s.from,
		)
	}

	for _, t := range tl {
		ts := es.State.GetUnstakingTimerState(t.Height)
		if err = icstate.ScheduleTimerJob(ts, t, s.from); err != nil {
			return scoreresult.UnknownFailureError.Wrapf(
				err,
				"Error while scheduling UnStaking Timer Job: from=%v",
				s.from,
			)
		}
	}
	if err = ia.SetStake(v); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err,
			"Failed to set stake: from=%v stake=%v",
			s.from,
			v,
		)
	}
	if err = es.State.SetTotalStake(new(big.Int).Add(tStake, stakeInc)); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to set totalStake: from=%v totalStake=%v stakeInc=%v",
			s.from,
			tStake,
			stakeInc,
		)
	}

	// update world account
	totalStake := ia.GetTotalStake()
	cmp := oldTotalStake.Cmp(totalStake)
	if cmp > 0 {
		es.Logger().Panicf(
			"Failed to set stake: newTotalStake=%v < oldTotalStake=%v from=%v",
			totalStake, oldTotalStake, s.from,
		)
	} else if cmp < 0 {
		diff := new(big.Int).Sub(totalStake, oldTotalStake)
		account.SetBalance(new(big.Int).Sub(balance, diff))
	}
	if icmodule.RevisionMultipleUnstakes <= revision && revision < icmodule.RevisionFixInvalidUnstake {
		txID := hex.EncodeToString(s.cc.TransactionID())
		if amount, ok := migrate.StakeData[txID]; ok {
			balance = account.GetBalance()
			v := new(big.Int)
			v.SetString(amount, 10)
			account.SetBalance(new(big.Int).Add(balance, v))
			s.log.Tracef("%s get %d illegal icx", s.from, v)
		}

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
	if err := s.iissHandleRevision(); err != nil {
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
	if err = es.SetDelegation(s.cc.BlockHeight(), s.from, ds); err != nil {
		return err
	}
	revision := s.cc.Revision().Value()
	if icmodule.RevisionMultipleUnstakes <= revision && revision < icmodule.RevisionFixInvalidUnstake {
		txID := hex.EncodeToString(s.cc.TransactionID())
		if amount, ok := migrate.DelegationData[txID]; ok {
			account := s.cc.GetAccountState(s.from.ID())
			balance := account.GetBalance()
			v := new(big.Int)
			v.SetString(amount, 10)
			account.SetBalance(new(big.Int).Add(balance, v))
			s.log.Tracef("%s get %d illegal icx", s.from, v)
		}
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
	ia := es.State.GetAccountSnapshot(address)
	if ia == nil {
		ia = icstate.GetEmptyAccountSnapshot()
	}

	return ia.GetDelegationInJSON(), nil
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
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v node=%v",
			s.from,
			nodeAddress,
		)
	}
	if s.value.Cmp(regPRepFee) != 0 {
		return scoreresult.InvalidParameterError.Errorf(
			"Invalid registration fee: value=%v != fee=%v",
			s.value,
			regPRepFee,
		)
	}

	// Subtract regPRepFee from chainScore
	as := s.cc.GetAccountState(state.SystemID)
	balance := new(big.Int).Sub(as.GetBalance(), regPRepFee)
	if balance.Sign() < 0 {
		return scoreresult.UnknownFailureError.Errorf("Not enough balance: %s, value=%v", state.SystemAddress, s.value)
	}
	as.SetBalance(balance)

	// Burn regPRepFee
	if ts, err := icutils.DecreaseTotalSupply(s.cc, regPRepFee); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to burn regPRepFee: from=%v fee=%v",
			s.from,
			regPRepFee,
		)
	} else {
		icutils.OnBurn(s.cc, state.SystemAddress, regPRepFee, ts)
	}

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
	if err := info.Validate(s.cc.Revision().Value(), true); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err,
			"Failed to validate regInfo: from=%v",
			s.from,
		)
	}

	es, err := s.getExtensionState()
	if err != nil {
		return err
	}

	var irep *big.Int
	if es.IsDecentralized() {
		term := es.State.GetTermSnapshot()
		irep = term.Irep()
	} else {
		irep = icstate.BigIntInitialIRep
	}

	if err = es.State.RegisterPRep(s.from, info, irep); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to register PRep: from=%v", s.from,
		)
	}

	term := es.State.GetTermSnapshot()
	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-term.StartHeight()),
		s.from,
		icstage.ESEnable,
	)
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err, "Failed to add EventEnable: from=%v", s.from,
		)
	}

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepRegistered(Address)")},
		[][]byte{s.from.Bytes()},
	)
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
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v", s.from,
		)
	}
	err = es.UnregisterPRep(s.cc.BlockHeight(), s.from)
	if err != nil {
		return err
	}

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepUnregistered(Address)")},
		[][]byte{s.from.Bytes()},
	)
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
		return nil, scoreresult.InvalidInstanceError.Wrap(err, "Failed to get PRep")
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
	jso := es.State.GetPRepManagerInJSON()
	return jso, nil
}

func (s *chainScore) Ex_setPRep(name *string, email *string, website *string, country *string,
	city *string, details *string, p2pEndpoint *string, node module.Address) error {
	var err error
	var es *iiss.ExtensionStateImpl

	if err = s.tryChargeCall(true); err != nil {
		return err
	}
	if err = s.iissHandleRevision(); err != nil {
		return err
	}

	if (node != nil && node.IsContract()) || s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf(
			"Invalid address: from=%v node=%v", s.from, node,
		)
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

	if err = info.Validate(s.cc.Revision().Value(), false); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to validate regInfo: from=%v", s.from,
		)
	}

	if es, err = s.getExtensionState(); err != nil {
		return err
	}
	if err = es.State.SetPRep(s.cc.BlockHeight(), s.from, info); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(err, "Failed to set PRep: from=%v", s.from)
	}
	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{s.from.Bytes()},
	)
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
	if s.from.IsContract() {
		return scoreresult.AccessDeniedError.Errorf("Invalid address: from=%s", s.from)
	}
	if err = es.SetGovernanceVariables(s.from, new(big.Int).Set(irep.Value()), s.cc.BlockHeight()); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err, "Failed to set governance variables: from=%v", s.from,
		)
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
	if err := s.iissHandleRevision(); err != nil {
		return err
	}
	if bytes.Compare(s.cc.TransactionID(), skippedClaimTX) == 0 {
		// Skip this TX like ICON1 mainnet.
		s.claimEventLog(s.from, new(big.Int), new(big.Int))
		return nil
	}
	es, err := s.getExtensionState()
	if err != nil {
		return err
	}

	iScore, err := s.getIScore(es, s.from)
	if err != nil {
		return err
	}
	if iScore.Sign() == 0 {
		// there is no IScore to claim
		s.claimEventLog(s.from, new(big.Int), new(big.Int))
		return nil
	}

	icx, remains := new(big.Int).DivMod(iScore, icmodule.BigIntIScoreICXRatio, new(big.Int))
	claim := new(big.Int).Sub(iScore, remains)

	// increase account icx balance
	account := s.cc.GetAccountState(s.from.ID())
	if account == nil {
		return scoreresult.InvalidInstanceError.Errorf("Invalid account: from=%v", s.from)
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
		err = es.Front.AddIScoreClaim(s.from, iScore)
	}
	if err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to add IScore claim event: from=%v",
			s.from,
		)
	}
	s.claimEventLog(s.from, claim, icx)
	return nil
}

func (s *chainScore) getIScore(es *iiss.ExtensionStateImpl, from module.Address) (*big.Int, error) {
	iScore := new(big.Int)
	if es.Reward == nil {
		return iScore, nil
	}
	is, err := es.Reward.GetIScore(from)
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to get IScore data: from=%v",
			from,
		)
	}
	if is == nil {
		return iScore, nil
	}

	iScore.Set(is.Value())
	stages := []*icstage.State{es.Front, es.Back1, es.Back2}
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		claim, err := stage.GetIScoreClaim(from)
		if err != nil {
			return nil, scoreresult.UnknownFailureError.Wrapf(
				err,
				"Failed to get claim data from back: from=%v",
				from,
			)
		}
		if claim != nil {
			iScore.Sub(iScore, claim.Value())
		}
	}
	return iScore, nil
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
	var err error
	if err = s.tryChargeCall(true); err != nil {
		return nil, err
	}
	if err = s.iissHandleRevision(); err != nil {
		return nil, err
	}
	es, err := s.getExtensionState()
	if err != nil {
		return nil, err
	}
	is, err := s.getIScore(es, address)
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
	jso["estimatedICX"] = icmodule.IScoreToICX(is)
	return jso, nil
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
	jso := make(map[string]interface{})
	jso["unstakeLockPeriod"] = calcUnstakeLockPeriod(es.State, totalStake, totalSupply)
	return jso, nil
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
		return nil, scoreresult.UnknownFailureError.Wrap(err, "Failed to get PRepTerm")
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
	res, err := es.State.GetNetworkValueInJSON()
	if err != nil {
		return nil, scoreresult.UnknownFailureError.Wrap(err, "Failed to get NetworkValue")
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
	if err = es.DisqualifyPRep(s.cc.BlockHeight(), address); err != nil {
		return scoreresult.UnknownFailureError.Wrapf(
			err,
			"Failed to disqualify PRep: from=%v prep=%v",
			s.from,
			address,
		)
	}

	ps, _ := es.State.GetPRepStatusByOwner(address, false)
	// Record PenaltyImposed eventlog
	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PenaltyImposed(Address,int,int)"), address.Bytes()},
		[][]byte{
			intconv.Int64ToBytes(int64(ps.Status())),
			intconv.Int64ToBytes(iiss.PRepDisqualification),
		},
	)

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
	// Subtract value from chainScore
	as := s.cc.GetAccountState(state.SystemID)
	balance := new(big.Int).Sub(as.GetBalance(), s.value)
	if balance.Sign() < 0 {
		return scoreresult.InvalidParameterError.Errorf(
			"Not enough value: from=%v value=%v", s.from, s.value,
		)
	}
	as.SetBalance(balance)

	// Burn value
	if ts, err := icutils.DecreaseTotalSupply(s.cc, s.value); err != nil {
		return scoreresult.InvalidParameterError.Wrapf(
			err,
			"Failed to decrease totalSupply: from=%v value=%v",
			s.from,
			s.value,
		)
	} else {
		icutils.OnBurn(s.cc, s.from, s.value, ts)
	}
	return nil
}
