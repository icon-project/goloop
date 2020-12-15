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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"math/big"
)

func (s *chainScore) Ex_setStake(value *common.HexInt) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccountState(s.from)
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
	if balance.Cmp(v) == -1 {
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
	ia, err := es.GetAccountState(address)
	if err != nil {
		return nil, err
	}
	return ia.GetStakeInfo(), nil
}

func (s *chainScore) Ex_setDelegation(param []interface{}) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccountState(s.from)
	if err != nil {
		return err
	}

	ds, err := icstate.NewDelegations(param)
	if err != nil {
		return err
	}

	if ia.Stake().Cmp(new(big.Int).Add(ds.GetDelegationAmount(), ia.Bond())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	ia.SetDelegation(ds)

	bonds := ia.Bonds()
	event := make([]*icstate.Delegation, 0, len(ds)+len(bonds))
	for _, d := range ds {
		event = append(event, d)
	}
	for _, b := range bonds {
		d := new(icstate.Delegation)
		d.Address = b.Address
		d.Value = b.Value
		event = append(event, d)
	}
	_, err = es.Front.AddEventDelegation(
		int(s.cc.BlockHeight()-es.CalculationBlockHeight()),
		s.from,
		event,
	)
	return nil
}

func (s *chainScore) Ex_getDelegation(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ia, err := es.GetAccountState(address)
	if err != nil {
		return nil, err
	}
	return ia.GetDelegationInfo(), nil
}

func (s *chainScore) Ex_registerPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ip, err := es.GetPRepState(s.from)
	if err != nil {
		return err
	}
	ips, err := es.GetPRepStatusState(s.from)
	if err != nil {
		return err
	}
	ips.SetGrade(icstate.Candidate)
	ips.SetStatus(icstate.Active)
	err = ip.SetPRep(name, email, website, country, city, details, p2pEndpoint, node)
	if err != nil {
		return err
	}
	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-es.CalculationBlockHeight()),
		s.from,
		true,
	)
	return err
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)

	prep, err := es.GetPRepState(address)
	if err != nil {
		return nil, err
	}

	prepStatus, err := es.GetPRepStatusState(address)
	if err != nil {
		return nil, err
	}

	return icutils.MergeMaps(prep.ToJSON(), prepStatus.ToJSON()), nil
}

func (s *chainScore) Ex_unregisterPRep() error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ips, err := es.GetPRepStatusState(s.from)
	if err != nil {
		return err
	}
	ips.SetGrade(icstate.Candidate)
	ips.SetStatus(icstate.Unregistered)

	_, err = es.Front.AddEventEnable(
		int(s.cc.BlockHeight()-es.CalculationBlockHeight()),
		s.from,
		false,
	)
	return err
}

func (s *chainScore) Ex_setPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ip, err := es.GetPRepState(s.from)
	if err != nil {
		return err
	}
	return ip.SetPRep(name, email, website, country, city, details, p2pEndpoint, node)
}

func (s *chainScore) Ex_setBond(bondList []interface{}) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	account, err := es.GetAccountState(s.from)
	if err != nil {
		return err
	}
	bonds, err := icstate.NewBonds(bondList)
	if err != nil {
		return err
	}
	bondAmount := big.NewInt(0)
	for _, b := range bonds {
		bondAmount = bondAmount.Add(bondAmount, b.Amount())
		prep, err := es.GetPRepState(b.To())
		if err != nil {
			return err
		}
		if !prep.BonderList().Contains(s.from) {
			return errors.Errorf("%s is not in bonder List of %s", s.from.String(), b.Address.String())
		}
		prepStatus, err := es.GetPRepStatusState(b.To())
		if err != nil {
			return err
		}
		prepStatus.SetBonded(b.Amount())
	}
	if account.Stake().Cmp(new(big.Int).Add(bondAmount, account.Delegating())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	ubToAdd, ubToMod, ubDiff := account.GetUnbondingInfo(bonds, s.cc.BlockHeight()+icstate.UnbondingPeriod)
	votingAmount := new(big.Int).Add(account.Delegating(), bondAmount)
	votingAmount.Sub(votingAmount, account.Bond())
	unbondingAmount := new(big.Int).Add(account.Unbonds().GetUnbondAmount(), ubDiff)
	if account.Stake().Cmp(new(big.Int).Add(votingAmount, unbondingAmount)) == -1 {
		return errors.Errorf("Not enough voting power")
	}
	account.SetBonds(bonds)
	tl := account.UpdateUnbonds(ubToAdd, ubToMod)
	for _, t := range tl {
		ts, e := es.GetUnbondingTimerState(t.Height)
		if e != nil {
			return errors.Errorf("Error while getting unbonding Timer")
		}
		if err = icstate.ScheduleTimerJob(ts, t, s.from); err != nil {
			return errors.Errorf("Error while scheduling Unbonding Timer Job")
		}
	}

	ds := account.Delegations()
	event := make([]*icstate.Delegation, 0, len(ds)+len(bonds))
	for _, d := range ds {
		event = append(event, d)
	}
	for _, b := range bonds {
		d := new(icstate.Delegation)
		d.Address = b.Address
		d.Value = b.Value
		event = append(event, d)
	}
	_, err = es.Front.AddEventDelegation(
		int(s.cc.BlockHeight()-es.CalculationBlockHeight()),
		s.from,
		event,
	)
	return nil
}

func (s *chainScore) Ex_getBond(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	account, err := es.GetAccountState(address)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	data["bonds"] = account.GetBondsInfo()
	data["unbonds"] = account.GetUnbondsInfo()
	return data, nil
}

func (s *chainScore) Ex_setBonderList(bonderList []interface{}) error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	prep, err := es.GetPRepState(s.from)
	if err != nil {
		return err
	}
	bl, err := icstate.NewBonderList(bonderList)
	if err != nil {
		return err
	}

	var b *icstate.AccountState
	for _, old := range prep.BonderList() {
		if !bl.Contains(old) {
			b, err = es.GetAccountState(old)
			if err != nil {
				return err
			}
			if len(b.Bonds()) > 0 || len(b.Unbonds()) > 0 {
				return errors.Errorf("Bonding/Unbonding exist. bonds : %d, unbonds : %d", len(b.Bonds()), len(b.Unbonds()))
			}
		}
	}

	prep.SetBonderList(bl)
	return nil
}

func (s *chainScore) Ex_getBonderList(address module.Address) ([]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	prep, err := es.GetPRepState(address)
	if err != nil {
		return nil, err
	}
	return prep.GetBonderListInJSON(), nil
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
	if iScore == nil || iScore.IsEmpty() {
		// there is no iScore to claim
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
	tr.SetBalance(new(big.Int).Add(tb, icx))

	// write claim data to front
	if err = es.Front.AddIScoreClaim(s.from, claim); err != nil {
		return err
	}

	return nil
}

func (s *chainScore) Ex_queryIScore(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	claimed, err := es.Front.GetIScoreClaim(address)
	if err != nil {
		return nil, err
	}
	is := new(big.Int)
	if claimed == nil {
		iScore, err := es.Reward.GetIScore(address)
		if err != nil {
			return nil, err
		}
		if iScore == nil || iScore.IsEmpty() {
			is.SetInt64(0)
		} else {
			is = iScore.Value
		}
	}

	data := make(map[string]interface{})
	data["blockheight"] = intconv.FormatInt(es.PrevCalculationBlockHeight())
	data["iscore"] = intconv.FormatBigInt(is)
	data["estimatedICX"] = intconv.FormatBigInt(is.Div(is, big.NewInt(iiss.IScoreICXRatio)))

	return data, nil
}
