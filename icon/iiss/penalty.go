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
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"math/big"
)

type PenaltyType int
const (
	PenaltyNone PenaltyType = iota
	PenaltyValidationFailure
	PenaltyAccumulatedValidationFailure
)

const (
	ValidationPenaltyCondition  int = 3
	ValidationPenaltySlashRatio     = 0

	ConsistentValidationPenaltyCondition  int = 5
	ConsistentValidationPenaltyMask           = 0x3fffffff
	ConsistentValidationPenaltySlashRatio     = 10
)

func (s *ExtensionStateImpl) UpdateBlockVoteStats(
	cc contract.CallContext, owner module.Address, voted bool) error {
	blockHeight := cc.BlockHeight()
	if err := s.pm.UpdateBlockVoteStats(owner, voted, blockHeight); err != nil {
		return err
	}
	return nil
}

func (s *ExtensionStateImpl) handlePenalty(cc contract.CallContext, owner module.Address) (error, bool) {
	prep := s.pm.GetPRepByOwner(owner)
	if prep == nil {
		return nil, false
	}
	if prep.LastState() != icstate.Failure {
		return nil, false
	}

	blockHeight := cc.BlockHeight()
	penalty := checkPenalty(prep.PRepStatus, blockHeight)
	if penalty == PenaltyNone {
		return nil, false
	}

	var err error
	var slashed bool
	if err, slashed = s.imposePenalty(cc, owner, penalty); err != nil {
		return err, false
	}
	if err = s.selectNewValidator(); err != nil {
		return err, false
	}
	return nil, slashed
}

func (s *ExtensionStateImpl) imposePenalty(
	cc contract.CallContext, owner module.Address, penalty PenaltyType) (error, bool) {
	var err error = nil

	var slashRatio int
	bh := cc.BlockHeight()

	switch penalty {
	case PenaltyValidationFailure:
		slashRatio = ValidationPenaltySlashRatio
	case PenaltyAccumulatedValidationFailure:
		slashRatio = ConsistentValidationPenaltySlashRatio
	default:
		return errors.Errorf("Unknown penalty: %d", penalty), false
	}

	if err = s.pm.ImposePenalty(owner, bh); err != nil {
		return err, false
	}
	if err = s.slash(cc, owner, slashRatio); err != nil {
		return err, false
	}
	return nil, slashRatio > 0
}

func checkPenalty(ps *icstate.PRepStatus, blockHeight int64) PenaltyType {
	if checkValidationPenalty(ps, blockHeight) {
		if checkConsistentValidationPenalty(ps) {
			return PenaltyAccumulatedValidationFailure
		}
		return PenaltyValidationFailure
	}
	return PenaltyNone
}

func checkValidationPenalty(ps *icstate.PRepStatus, blockHeight int64) bool {
	return (ps.VPenaltyMask()&1 == 0) && ps.GetVFailCont(blockHeight) >= int64(ValidationPenaltyCondition)
}

func checkConsistentValidationPenalty(ps *icstate.PRepStatus) bool {
	return ps.GetVPenaltyCount() >= ConsistentValidationPenaltyCondition
}

func (s *ExtensionStateImpl) slash(cc contract.CallContext, address module.Address, ratio int) error {
	if ratio < 0 || 100 < ratio {
		return errors.Errorf("Too big slash ratio %d", ratio)
	}

	pm := s.pm
	bonders := pm.GetPRepByOwner(address).BonderList()
	totalSlashBond := new(big.Int)

	// slash all bonder
	for _, bonder := range bonders {
		account := s.GetAccount(bonder)
		totalSlash := new(big.Int)

		// from bonds
		slashBond := account.SlashBond(address, ratio)
		totalSlash.Add(totalSlash, slashBond)
		totalSlashBond.Add(totalSlashBond, slashBond)

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
	}

	return s.pm.Slash(address, totalSlashBond, false)
}
