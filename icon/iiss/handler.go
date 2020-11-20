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

package iiss

import (
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type Handler struct {
	state.WorldContext
	from  module.Address
	value *big.Int
	es    *ExtensionStateImpl
}

func (h *Handler) SetStake(v *big.Int) error {
	ia, err := h.es.state.GetAccountState(h.from)
	if err != nil {
		return err
	}

	if ia.GetVotedPower().Cmp(v) == 1 {
		return errors.Errorf("Failed to stake: stake < votedPower")
	}

	prevTotalStake := ia.GetTotalStake()
	stakeInc := new(big.Int).Sub(v, ia.GetStake())
	if stakeInc.Sign() == 0 {
		return nil
	}

	account := h.GetAccountState(h.from.ID())
	balance := account.GetBalance()
	if balance.Cmp(v) == -1 {
		return errors.Errorf("Not enough balance")
	}

	// update IISS account
	expireHeight := calcUnstakeLockPeriod(h.BlockHeight())
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

func (h *Handler) GetStake(address module.Address) (map[string]interface{}, error) {
	ia, err := h.es.state.GetAccountState(address)
	if err != nil {
		return nil, err
	}
	return ia.GetStakeInfo(), nil
}

func calcUnstakeLockPeriod(blockHeight int64) int64 {
	// TODO implement me
	return blockHeight + 10
}

func (h *Handler) SetDelegation(param []interface{}) error {
	ia, err := h.es.state.GetAccountState(h.from)
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

func (h *Handler) GetDelegation(address module.Address) (map[string]interface{}, error) {
	ia, err := h.es.state.GetAccountState(address)
	if err != nil {
		return nil, err
	}
	return ia.GetDelegationInfo(), nil
}

func (h *Handler) RegisterPRep(name string, email string, website string, country string,
	city string, details string, p2pEndpoint string, node module.Address) error {
	ip, err := h.es.state.GetPRepState(h.from)
	if err != nil {
		return err
	}
	return ip.SetPRep(name, email, website, country, city, details, p2pEndpoint, node)
}

func (h *Handler) GetPRep(address module.Address) (map[string]interface{}, error) {
	ip, err := h.es.state.GetPRepState(address)
	if err != nil {
		return nil, err
	}
	return ip.GetPRep(), nil
}

func NewHandler(wc state.WorldContext, from module.Address, value *big.Int, es state.ExtensionState) *Handler {
	return &Handler{
		WorldContext: wc,
		from:         from,
		value:        value,
		es:           es.(*ExtensionStateImpl),
	}
}
