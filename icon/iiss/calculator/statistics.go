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
	"fmt"
	"math/big"
	"strings"
)

type Stats struct {
	value map[RewardType]*big.Int
	count map[RewardType]int64
}

func (s *Stats) GetValue(t RewardType) *big.Int {
	if v, ok := s.value[t]; ok {
		return v
	} else {
		return new(big.Int)
	}
}

func (s *Stats) GetCount(t RewardType) int64 {
	if v, ok := s.count[t]; ok {
		return v
	} else {
		return 0
	}
}

func (s *Stats) IncreaseReward(t RewardType, amount *big.Int) {
	if v, ok := s.value[t]; ok {
		s.value[t] = new(big.Int).Add(v, amount)
		s.count[t]++
	} else {
		s.value[t] = amount
		s.count[t] = 1
	}
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
		fmt.Fprintf(&sb, " %s: (count=%d, value=%s)", k, s.count[k], v)
	}
	return sb.String()
}

func NewStats() *Stats {
	return &Stats{
		value: make(map[RewardType]*big.Int),
		count: make(map[RewardType]int64),
	}
}
