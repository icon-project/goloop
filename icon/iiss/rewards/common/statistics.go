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

func (s *Stats) TotalReward() *big.Int {
	reward := new(big.Int)
	for _, v := range s.value {
		reward.Add(reward, v)
	}
	return reward
}

func (s *Stats) String() string {
	ret := ""
	total := new(big.Int)
	for k, v := range s.value {
		total.Add(total, v)
		if len(ret) == 0 {
			ret = fmt.Sprintf("%s=%d", k, v)
		} else {
			ret = fmt.Sprintf("%s %s=%d", ret, k, v)
		}
	}
	return fmt.Sprintf("Total=%d %s", total, ret)
}

func NewStats() *Stats {
	return &Stats{value: make(map[RewardType]*big.Int)}
}
