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

const (
	ValidationPenaltyCondition  int = 660
	ValidationPenaltySlashRatio     = 10

	ConsistentValidationPenaltyCondition  int = 5
	ConsistentValidationPenaltyMask           = 0x3fffffff
	ConsistentValidationPenaltySlashRatio     = 100
)

func validationPenalty(cc contract.CallContext, ps *icstate.PRepStatus) error {
	if ps.LastState() != icstate.Fail {
		return nil
	}
	owner := ps.Owner()
	blockHeight := cc.BlockHeight()

	// check and apply penalties
	if checkValidationPenalty(ps, blockHeight) {
		// Validation Penalty
		ps.SetVPenaltyMask(ps.VPenaltyMask() | 1)
		if err := Slash(cc, owner, ValidationPenaltySlashRatio); err != nil {
			return err
		}
		ps.SetLastHeight(blockHeight)
		// TODO IC2-35 notify to PRepManager. PRepManager must add icstage.EventEnable

		// Consistent Penalty
		if checkConsistentValidationPenalty(ps) {
			if err := Slash(cc, owner, ConsistentValidationPenaltySlashRatio); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkValidationPenalty(ps *icstate.PRepStatus, blockHeight int64) bool {
	return (ps.VPenaltyMask()&1 == 0) && ps.GetVFailCont(blockHeight) >= int64(ValidationPenaltyCondition)
}

func checkConsistentValidationPenalty(ps *icstate.PRepStatus) bool {
	return ps.GetVPenaltyCount() >= ConsistentValidationPenaltyCondition
}

func Slash(cc contract.CallContext, address module.Address, ratio int) error {
	if ratio < 0 || 100 < ratio {
		return errors.Errorf("Too big slash ratio %d", ratio)
	}
	es := cc.GetExtensionState().(*ExtensionStateImpl)
	pm := es.pm
	bonders := pm.GetPRepByOwner(address).BonderList()
	// slash all bonder
	for _, bonder := range bonders {
		account := es.GetAccount(bonder)
		totalSlash := new(big.Int)

		// from bonds
		slashBond := account.SlashBond(address, ratio)
		totalSlash.Add(totalSlash, slashBond)

		// from unbondings
		slashUnbond, expire := account.SlashUnbond(address, ratio)
		totalSlash.Add(totalSlash, slashUnbond)
		if expire != -1 {
			timer := es.GetUnbondingTimerState(expire, false)
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
		totalStake := es.State.GetTotalStake()
		totalStake.Sub(totalStake, totalSlash)
		if err := es.State.SetTotalStake(totalStake); err != nil {
			return err
		}

		// add icstage.EventBond
		delta := make(map[string]*big.Int)
		key := icutils.ToKey(bonder)
		delta[key] = slashBond
		if err := es.AddEventBond(cc.BlockHeight(), bonder, delta); err != nil {
			return err
		}

		// event log
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{[]byte("Slash(Address,Address,int)"), address.Bytes()},
			[][]byte{bonder.Bytes(), intconv.BigIntToBytes(totalSlash)},
		)
	}

	return nil
}

func DisqualifyPRep(cc contract.CallContext, address module.Address) error {
	// TODO implement me
	// event log
	return nil
}
