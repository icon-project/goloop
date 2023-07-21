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

package common

import (
	"fmt"
	"math/big"
	"strings"
)

type Stats struct {
	value map[RewardType]*big.Int
}

func (s *Stats) GetValue(t RewardType) *big.Int {
	if v, ok := s.value[t]; ok {
		return v
	} else {
		return new(big.Int)
	}
}

func (s *Stats) BlockProduce() *big.Int {
	return s.GetValue(RTBlockProduce)
}

func (s *Stats) Voted() *big.Int {
	return s.GetValue(RTPRep)
}

func (s *Stats) Voting() *big.Int {
	return s.GetValue(RTVoter)
}

func (s *Stats) IncreaseValue(t RewardType, amount *big.Int) {
	n := new(big.Int)
	if v, ok := s.value[t]; ok {
		n.Add(v, amount)
	} else {
		n.Set(amount)
	}
	s.value[t] = n
}

func (s *Stats) IncreaseBlockProduce(amount *big.Int) {
	s.IncreaseValue(RTBlockProduce, amount)
}

func (s *Stats) IncreaseVoted(amount *big.Int) {
	s.IncreaseValue(RTPRep, amount)
}

func (s *Stats) IncreaseVoting(amount *big.Int) {
	s.IncreaseValue(RTVoter, amount)
}

func (s *Stats) Total() *big.Int {
	reward := new(big.Int)
	for _, v := range s.value {
		reward.Add(reward, v)
	}
	return reward
}

func (s *Stats) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Total=%d", s.Total())
	for k, v := range s.value {
		fmt.Fprintf(&sb, " %s=%d", k, v)
	}
	return sb.String()
}

func NewStats() *Stats {
	return &Stats{value: make(map[RewardType]*big.Int)}
}
