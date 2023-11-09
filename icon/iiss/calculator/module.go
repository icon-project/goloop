/*
 * Copyright 2023 ICON Foundation
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

package calculator

import (
	"math/big"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/module"
)

type Context interface {
	Back() *icstage.Snapshot
	Base() *icreward.Snapshot
	Temp() *icreward.State
	Stats() *Stats
	Logger() log.Logger
	UpdateIScore(addr module.Address, reward *big.Int, t RewardType) error
}

// RewardReader reads from icreward.Snapshot
type RewardReader interface {
	GetDelegating(addr module.Address) (*icreward.Delegating, error)
	GetBonding(addr module.Address) (*icreward.Bonding, error)
}

// RewardWriter writes to icreward.State
type RewardWriter interface {
	SetVoted(addr module.Address, voted *icreward.Voted) error
	SetDelegating(addr module.Address, delegating *icreward.Delegating) error
	SetBonding(addr module.Address, bonding *icreward.Bonding) error
}

type RewardCalculator interface {
	Calculate() error
}
