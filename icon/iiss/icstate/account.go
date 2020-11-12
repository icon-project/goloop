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

package icstate

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
)

type AccountSnapshot struct {
	NoDatabaseObject
	stake       *big.Int
	delegated   *big.Int
	delegations Delegations
	unstakes    Unstakes
}

func (a *AccountSnapshot) Version() int {
	return 0
}

func (a *AccountSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&a.stake,
		&a.delegated,
		&a.delegations,
		&a.unstakes,
	)
	return err
}

func (a *AccountSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		a.stake,
		a.delegated,
		a.delegations,
		a.unstakes,
	)
}

func (a *AccountSnapshot) Equal(object ObjectImpl) bool {
	aa, ok := object.(*AccountSnapshot)
	if !ok {
		return false
	}
	if aa == a {
		return true
	}
	return a.stake.Cmp(aa.stake) == 0 &&
		a.delegated.Cmp(aa.delegated) == 0 &&
		a.delegations.Equal(aa.delegations) &&
		a.unstakes.Equal(aa.unstakes)
}

func newAccountSnapshot(tag Tag) *AccountSnapshot {
	return &AccountSnapshot{
		stake:     new(big.Int),
		delegated: new(big.Int),
	}
}

type AccountState struct {
	stake       *big.Int
	delegated   *big.Int
	delegations Delegations
	unstakes    Unstakes
}

func (as *AccountState) Reset(ass *AccountSnapshot) {
	as.stake = ass.stake
	as.delegated = ass.delegated
	as.delegations = ass.delegations.Clone()
}

func (as *AccountState) GetSnapshot() *AccountSnapshot {
	ass := &AccountSnapshot{}
	ass.stake = as.stake
	ass.delegated = as.delegated
	ass.delegations = as.delegations.Clone()
	return ass
}

func NewAccountStateWithSnapshot(ss *AccountSnapshot) *AccountState {
	return &AccountState{
		stake:       ss.stake,
		delegated:   ss.delegated,
		delegations: ss.delegations,
	}
}
