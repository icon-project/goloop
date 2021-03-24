/*
 * Copyright 2021 ICON Foundation
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

package iiss

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

func (s *ExtensionStateImpl) UpdateBlockVoteStats(
	cc contract.CallContext, owner module.Address, voted bool) error {
	blockHeight := cc.BlockHeight()
	if !voted {
		s.logger.Debugf("Nil vote: bh=%d addr=%s", blockHeight, owner)
	}
	if err := s.pm.UpdateBlockVoteStats(owner, voted, blockHeight); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) handlePenalty(cc contract.CallContext, owner module.Address) error {
	var err error = nil

	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return nil
	}
	if prep.LastState() != icstate.Failure {
		return nil
	}

	blockHeight := cc.BlockHeight()

	penaltyCondition := s.State.GetValidationPenaltyCondition().Int64()
	if !checkValidationPenalty(prep.PRepStatus, blockHeight, penaltyCondition) {
		return nil
	}

	if err = s.pm.ImposePenalty(owner, blockHeight); err != nil {
		return err
	}

	penaltyCondition = s.State.GetConsistentValidationPenaltyCondition().Int64()
	if checkConsistentValidationPenalty(prep.PRepStatus, int(penaltyCondition)) {
		slashRatio := int(s.State.GetConsistentValidationPenaltySlashRatio().Int64())
		if err = s.slash(cc, owner, slashRatio); err != nil {
			return err
		}
	}

	if err = s.replaceValidator(owner); err != nil {
		return err
	}
	if err = s.addEventEnable(blockHeight, owner, icstage.EfDisableTemp); err != nil {
		return err
	}
	return nil
}

func checkValidationPenalty(ps *icstate.PRepStatus, blockHeight, condition int64) bool {
	return (ps.VPenaltyMask()&1 == 0) && ps.GetVFailCont(blockHeight) >= condition
}

func checkConsistentValidationPenalty(ps *icstate.PRepStatus, condition int) bool {
	return ps.GetVPenaltyCount() >= condition
}

func (s *ExtensionStateImpl) slash(cc contract.CallContext, address module.Address, ratio int) error {
	if ratio == 0 {
		return nil
	}
	if ratio < 0 || 100 < ratio {
		return errors.Errorf("Invalid slash ratio %d", ratio)
	}

	logger := cc.Logger().WithFields(log.Fields{log.FieldKeyModule: "ICON"})
	logger.Debugf("slash() start: addr=%s ratio=%d", address, ratio)

	pm := s.pm
	bonders := pm.GetPRepByOwner(address).BonderList()
	totalSlashBond := new(big.Int)

	// slash all bonder
	for _, bonder := range bonders {
		account := s.GetAccount(bonder)
		totalSlash := new(big.Int)

		logger.Debugf("Before slashing: %s", account)

		// from bonds
		slashBond := account.SlashBond(address, ratio)
		totalSlash.Add(totalSlash, slashBond)
		totalSlashBond.Add(totalSlashBond, slashBond)
		logger.Debugf("addr=%s ratio=%d slashBond=%s", address, ratio, slashBond)

		// from unbondings
		slashUnbond, expire := account.SlashUnbond(address, ratio)
		totalSlash.Add(totalSlash, slashUnbond)
		if expire != -1 {
			timer := s.GetUnbondingTimerState(expire, false)
			if timer != nil {
				if err := timer.Delete(address); err != nil {
					return err
				}
			} else {
				return errors.Errorf("timer doesn't exist for height %d", expire)
			}
		}

		// from stake
		if err := account.SlashStake(totalSlash); err != nil {
			return err
		}
		totalStake := s.State.GetTotalStake()
		totalStake.Sub(totalStake, totalSlash)
		if err := s.State.SetTotalStake(totalStake); err != nil {
			return err
		}

		// add icstage.EventBond
		delta := make(map[string]*big.Int)
		key := icutils.ToKey(bonder)
		delta[key] = slashBond
		if err := s.AddEventBond(cc.BlockHeight(), bonder, delta); err != nil {
			return err
		}

		// event log
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{[]byte("Slash(Address,Address,int)"), address.Bytes()},
			[][]byte{bonder.Bytes(), intconv.BigIntToBytes(totalSlash)},
		)

		logger.Debugf("After slashing: %s", account)
	}

	ret := s.pm.Slash(address, totalSlashBond)
	logger.Debugf("slash() end")
	return ret
}

func buildPenaltyMask(input *big.Int) (res uint32) {
	var mid uint32
	mid = 0x00000001
	for i := 0; i < int(input.Int64()) ; i++ {
		res = res | mid
		mid = mid << 1
	}
	return
}