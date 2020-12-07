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
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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
	stakeInc := new(big.Int).Sub(v, ia.GetStake())
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
	if err := ia.UpdateUnstake(stakeInc, expireHeight); err != nil {
		return err
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

	if ia.GetStake().Cmp(new(big.Int).Add(ds.GetDelegationAmount(), ia.GetBond())) == -1 {
		return errors.Errorf("Not enough voting power")
	}

	ia.SetDelegation(ds)

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
	return ip.SetPRep(name, email, website, country, city, details, p2pEndpoint, node)
}

func (s *chainScore) Ex_getPRep(address module.Address) (map[string]interface{}, error) {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ip, err := es.GetPRepState(address)
	if err != nil {
		return nil, err
	}
	ips, err := es.GetPRepStatusState(address)
	if err != nil {
		return nil, err
	}
	prepInfo := ip.GetPRep()
	for k, v := range ips.GetPRepStatusInfo() {
		prepInfo[k] = v
	}
	return prepInfo, nil
}

func (s *chainScore) Ex_unregisterPRep() error {
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	ips, err := es.GetPRepStatusState(s.from)
	if err != nil {
		return err
	}
	ips.SetGrade(icstate.Candidate)
	ips.SetStatus(icstate.Unregistered)
	return nil
}
