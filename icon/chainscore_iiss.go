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
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func (s *chainScore) Ex_setIRep(value *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return icstate.SetIRep(es.State, new(big.Int).Set(&value.Int))
}

func (s *chainScore) Ex_getIRep() (int64, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return icstate.GetIRep(es.State).Int64(), nil
}

func (s *chainScore) Ex_getRRep() (int64, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return icstate.GetRRep(es.State).Int64(), nil
}

func (s *chainScore) Ex_setStake(value *common.HexInt) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccount(s.from)
	if err != nil {
		return err
	}

	v := &value.Int

	if ia.GetVotedPower().Cmp(v) == 1 {
		return errors.Errorf("Failed to stake: stake < votedPower")
	}

	prevTotalStake := ia.GetTotalStake()
	stakeInc := new(big.Int).Sub(v, ia.Stake())
	if stakeInc.Sign() == 0 {
		return nil
	}

	account := s.cc.GetAccountState(s.from.ID())
	balance := account.GetBalance()
	availableStake := new(big.Int).Add(balance, ia.GetVotingPower())
	if availableStake.Cmp(v) == -1 {
		return errors.Errorf("Not enough balance")
	}

	// update IISS account
	expireHeight := calcUnstakeLockPeriod(s.cc.BlockHeight())
	tl, err := ia.UpdateUnstake(stakeInc, expireHeight)
	if err != nil {
		return err
	}
	for _, t := range tl {
		ts, e := es.GetUnstakingTimerState(t.Height)
		if e != nil {
			return errors.Errorf("Error while getting Timer")
		} else if ts == nil {
			ts = es.AddUnstakingTimerToState(t.Height)
		}
		if err = icstate.ScheduleTimerJob(ts, t, s.from); err != nil {
			return errors.Errorf("Error while scheduling UnStaking Timer Job")
		}
	}
	if err = ia.SetStake(v); err != nil {
		return err
	}

	// update world account
	totalStake := ia.GetTotalStake()
	if prevTotalStake.Cmp(totalStake) != 0 {
		diff := new(big.Int).Sub(totalStake, prevTotalStake)
		account.SetBalance(new(big.Int).Sub(balance, diff))
	}

	return nil
}

func calcUnstakeLockPeriod(blockHeight int64) int64 {
	// TODO implement me
	return blockHeight + 10
}

func (s *chainScore) Ex_getStake(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccount(address)
	if err != nil {
		return nil, err
	}
	return ia.GetStakeInfo(), nil
}

func (s *chainScore) Ex_setDelegation(param []interface{}) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ds, err := icstate.NewDelegations(param)
	if err != nil {
		return err
	}
	return es.SetDelegation(s.cc, s.from, ds)
}

func (s *chainScore) Ex_getDelegation(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccount(address)
	if err != nil {
		return nil, err
	}
	return ia.GetDelegationInfo(), nil
}

func (s *chainScore) Ex_registerPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	size := 7
	params := make([]string, size, size)
	params[icstate.IdxName] = name
	params[icstate.IdxCountry] = country
	params[icstate.IdxCity] = city
	params[icstate.IdxDetails] = details
	params[icstate.IdxEmail] = email
	params[icstate.IdxWebsite] = website
	params[icstate.IdxP2pEndpoint] = p2pEndpoint

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	err := es.RegisterPRep(s.from, node, params)
	if err != nil {
		return err
	}

	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-es.CalculationBlockHeight()),
		s.from,
		true,
	)

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepRegistered(Address)")},
		[][]byte{s.from.Bytes()},
	)

	return err
}

func (s *chainScore) Ex_unregisterPRep() error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.UnregisterPRep(s.cc, s.from)
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.GetPRepInJSON(address)
}

func (s *chainScore) Ex_getPReps() (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	jso := es.GetPRepsInJSON()
	jso["blockHeight"] = s.cc.BlockHeight()
	return jso, nil
}

func (s *chainScore) Ex_setPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	size := 7
	params := make([]string, size, size)
	params[icstate.IdxName] = name
	params[icstate.IdxCountry] = country
	params[icstate.IdxCity] = city
	params[icstate.IdxDetails] = details
	params[icstate.IdxEmail] = email
	params[icstate.IdxWebsite] = website
	params[icstate.IdxP2pEndpoint] = p2pEndpoint

	s.cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PRepSet(Address)")},
		[][]byte{s.from.Bytes()},
	)

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.SetPRep(s.from, node, params)
}

func (s *chainScore) Ex_setBond(bondList []interface{}) error {
	bonds, err := icstate.NewBonds(bondList)
	if err != nil {
		return err
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.SetBond(s.cc, s.from, bonds)
}

func (s *chainScore) Ex_getBond(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	account, err := es.GetAccount(address)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	data["bonds"] = account.GetBondsInfo()
	data["unbonds"] = account.GetUnbondsInfo()
	return data, nil
}

func (s *chainScore) Ex_setBonderList(bonderList []interface{}) error {
	bl, err := icstate.NewBonderList(bonderList)
	if err != nil {
		return err
	}

	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.SetBonderList(s.from, bl)
}

func (s *chainScore) Ex_getBonderList(address module.Address) ([]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	return es.GetBonderList(address)
}

func (s *chainScore) Ex_claimIScore() error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	claimed, err := es.Front.GetIScoreClaim(s.from)
	if err != nil {
		return err
	}
	if claimed != nil {
		// claim already in this calculation period
		return nil
	}

	iScore, err := es.Reward.GetIScore(s.from)
	if err != nil {
		return err
	}
	if iScore == nil {
		// there is no iScore to claim
		return nil
	}
	claimed, err = es.Back.GetIScoreClaim(s.from)
	if err != nil {
		return err
	}
	if claimed != nil {
		iScore.Value.Sub(iScore.Value, claimed.Value)
	}

	if iScore.IsEmpty() {
		// there is no IScore to claim
		return nil
	}

	icx, remains := new(big.Int).DivMod(iScore.Value, iiss.BigIntIScoreICXRation, new(big.Int))
	claim := new(big.Int).Sub(iScore.Value, remains)

	// increase account icx balance
	account := s.cc.GetAccountState(s.from.ID())
	if account == nil {
		return nil
	}
	balance := account.GetBalance()
	account.SetBalance(balance.Add(balance, icx))

	// decrease treasury icx balance
	tr := s.cc.GetAccountState(s.cc.Treasury().ID())
	tb := tr.GetBalance()
	tr.SetBalance(new(big.Int).Sub(tb, icx))

	// write claim data to front
	if err = es.Front.AddIScoreClaim(s.from, claim); err != nil {
		return err
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
		return nil, err
	}
	is := new(big.Int)
	if fClaim == nil {
		iScore, err := es.Reward.GetIScore(address)
		if err != nil {
			return nil, err
		}
		if iScore == nil || iScore.IsEmpty() {
			is.SetInt64(0)
		} else {
			is = iScore.Value
		}
		bClaim, err := es.Back.GetIScoreClaim(address)
		if err != nil {
			return nil, err
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
