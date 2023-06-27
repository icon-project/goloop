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

package rewards

import "math/big"

type RewardType int

const (
	TypeBlockProduce RewardType = iota
	TypeVoted
	TypeVoting
)

type statistics struct {
	blockProduce *big.Int
	voted        *big.Int
	voting       *big.Int
}

func (s *statistics) BlockProduce() *big.Int {
	return s.blockProduce
}

func (s *statistics) Voted() *big.Int {
	return s.voted
}

func (s *statistics) Voting() *big.Int {
	return s.voting
}

func increaseStats(src *big.Int, amount *big.Int) *big.Int {
	n := new(big.Int)
	if src == nil {
		n.Set(amount)
	} else {
		n.Add(src, amount)
	}
	return n
}

func (s *statistics) IncreaseBlockProduce(amount *big.Int) {
	s.blockProduce = increaseStats(s.blockProduce, amount)
}

func (s *statistics) IncreaseVoted(amount *big.Int) {
	s.voted = increaseStats(s.voted, amount)
}

func (s *statistics) IncreaseVoting(amount *big.Int) {
	s.voting = increaseStats(s.voting, amount)
}

func (s *statistics) TotalReward() *big.Int {
	reward := new(big.Int)
	reward.Add(s.blockProduce, s.voted)
	reward.Add(reward, s.voting)
	return reward
}

func (s *statistics) Clear() {
	s.blockProduce = new(big.Int)
	s.voted = new(big.Int)
	s.voting = new(big.Int)
}

func newStatistics() *statistics {
	return &statistics{
		new(big.Int),
		new(big.Int),
		new(big.Int),
	}
}
