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
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func (es *ExtensionStateImpl) handlePenalty(cc icmodule.CallContext, owner module.Address) error {
	var err error

	ps := es.State.GetPRepStatusByOwner(owner, false)
	if ps == nil {
		return nil
	}

	blockHeight := cc.BlockHeight()

	// Penalty check
	if !es.State.CheckValidationPenalty(ps, blockHeight) {
		return nil
	}

	// Impose penalty
	if err = es.State.ImposePenalty(owner, ps, blockHeight); err != nil {
		return err
	}

	// Record PenaltyImposed eventlog
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("PenaltyImposed(Address,int,int)"), owner.Bytes()},
		[][]byte{
			intconv.Int64ToBytes(int64(ps.Status())),
			intconv.Int64ToBytes(int64(icmodule.PenaltyBlockValidation)),
		},
	)

	// Slashing
	revision := cc.Revision().Value()
	if es.State.CheckConsistentValidationPenalty(revision, ps) {
		slashRatio := es.State.GetConsistentValidationPenaltySlashRatio()
		if err = es.slash(cc, owner, slashRatio); err != nil {
			return err
		}
	}

	// Record event for reward calculation
	return es.addEventEnable(blockHeight, owner, icstage.ESDisableTemp)
}

func (es *ExtensionStateImpl) slash(cc icmodule.CallContext, owner module.Address, ratio int) error {
	if ratio < 0 || 100 < ratio {
		return errors.Errorf("Invalid slash ratio %d", ratio)
	}

	logger := cc.FrameLogger()
	logger.TSystemf("IISS slash start owner=%s ratio=%d", owner, ratio)

	pb := es.State.GetPRepBaseByOwner(owner, false)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	bonders := pb.BonderList()
	slashedBondSum := new(big.Int)
	slashedStakeSum := new(big.Int)

	// slash bonds deposited by all bonders
	for i, bonder := range bonders {
		logger.TSystemf("IISS bonder slash loop start idx=%d bonder=%s", i, bonder)

		var expire int64
		slashedBond := new(big.Int)
		slashedUnbond := new(big.Int)
		slashedStake := new(big.Int)
		account := es.State.GetAccountState(bonder)

		if ratio > 0 {
			// bond
			slashedBond = account.SlashBond(owner, ratio)
			slashedBondSum.Add(slashedBondSum, slashedBond)

			// unbond
			slashedUnbond, expire = account.SlashUnbond(owner, ratio)
			if expire != -1 {
				timer := es.State.GetUnbondingTimerState(expire)
				if timer != nil {
					timer.Delete(owner)
				} else {
					return errors.Errorf("timer doesn't exist for height %d", expire)
				}
			}

			// stake
			slashedStake.Add(slashedBond, slashedUnbond)
			if err := account.SlashStake(slashedStake); err != nil {
				return err
			}
			slashedStakeSum.Add(slashedStakeSum, slashedStake)

			// add icstage.EventBond
			delta := map[string]*big.Int{
				icutils.ToKey(owner): new(big.Int).Neg(slashedBond),
			}
			if err := es.AddEventBond(cc.BlockHeight(), bonder, delta); err != nil {
				return err
			}
		}

		// Record Slashed eventlog
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{[]byte("Slashed(Address,Address,int)"), owner.Bytes()},
			[][]byte{bonder.Bytes(), intconv.BigIntToBytes(slashedStake)},
		)
		// slashedStake is the same as the sum of slashedBond and slashedUnbond
		logger.TSystemf(
			"IISS bonder slash loop end bonder=%s slashedBond=%v slashedUnbond=%v slashedStake=%v",
			bonder, slashedBond, slashedUnbond, slashedStake,
		)
	}

	// newTotalStake = oldTotalStake - slashedStakeSum
	oldTotalStake := es.State.GetTotalStake()
	newTotalStake := new(big.Int).Sub(oldTotalStake, slashedStakeSum)
	if err := es.State.SetTotalStake(newTotalStake); err != nil {
		return err
	}
	if err := es.State.ReducePRepBonded(owner, slashedBondSum); err != nil {
		return err
	}
	err := cc.HandleBurn(state.SystemAddress, slashedStakeSum)

	logger.TSystemf(
		"IISS slash end owner=%s slashedBondSum=%v slashedStakeSum=%v oldTotalStake=%v newTotalStake=%v",
		owner, slashedBondSum, slashedStakeSum, oldTotalStake, newTotalStake,
	)
	return err
}
