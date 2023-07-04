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

package iiss4

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/icon/iiss/rewards"
	"github.com/icon-project/goloop/module"
)

type PRep struct {
	enable         bool
	delegated      *big.Int
	bonded         *big.Int
	commissionRate int

	owner             module.Address
	power             *big.Int
	pubkey            bool
	rank              int
	accumulatedBonded *big.Int
	accumulatedVoted  *big.Int
	accumulatedPower  *big.Int
	reward            *big.Int // in IScore
	voterReward       *big.Int // in IScore
}

func (p *PRep) Enable() bool {
	return p.enable
}

func (p *PRep) SetEnable(yn bool) {
	p.enable = yn
}

func (p *PRep) Bonded() *big.Int {
	return p.bonded
}

func (p *PRep) CommissionRate() int {
	return p.commissionRate
}

func (p *PRep) SetCommissionRate(value int) {
	p.commissionRate = value
}

func (p *PRep) GetVoted() *big.Int {
	return new(big.Int).Add(p.delegated, p.bonded)
}

func (p *PRep) Owner() module.Address {
	return p.owner
}

func (p *PRep) Power() *big.Int {
	return p.power
}

func (p *PRep) GetPower(bondRequirement int) *big.Int {
	power := new(big.Int).Mul(p.bonded, big.NewInt(100))
	power.Div(power, big.NewInt(int64(bondRequirement)))
	voted := p.GetVoted()
	if voted.Cmp(power) == -1 {
		power.Set(voted)
	}
	return power
}

func (p *PRep) UpdatePower(bondRequirement int) *big.Int {
	p.power = p.GetPower(bondRequirement)
	return p.power
}

func (p *PRep) SetRank(rank int) {
	p.rank = rank
}

func (p *PRep) Payable(electedPRepCount int) bool {
	// TODO check p.pubkey ?
	return p.enable && p.rank <= electedPRepCount
}

func (p *PRep) AccumulatedPower() *big.Int {
	return p.accumulatedPower
}

func (p *PRep) GetAccumulatedPower(bondRequirement int) *big.Int {
	power := new(big.Int).Mul(p.accumulatedBonded, big.NewInt(100))
	power.Div(power, big.NewInt(int64(bondRequirement)))
	if p.accumulatedVoted.Cmp(power) == -1 {
		power.Set(p.accumulatedVoted)
	}
	return power
}

func (p *PRep) UpdateAccumulatedPower(bondRequirement int) *big.Int {
	p.accumulatedPower = p.GetAccumulatedPower(bondRequirement)
	return p.power
}

func (p *PRep) InitAccumulated(offsetLimit int) {
	ol := big.NewInt(int64(offsetLimit))
	p.accumulatedVoted = new(big.Int).Mul(p.GetVoted(), ol)
	p.accumulatedBonded = new(big.Int).Mul(p.bonded, ol)
}

func (p *PRep) ApplyVote(vType VoteType, amount *big.Int, period int) {
	pr := big.NewInt(int64(period))
	accumulated := new(big.Int).Mul(amount, pr)
	if vType == vtBond {
		p.bonded = new(big.Int).Add(p.bonded, amount)
		p.accumulatedBonded = new(big.Int).Add(p.accumulatedBonded, accumulated)
	} else {
		p.delegated = new(big.Int).Add(p.delegated, amount)
	}
	p.accumulatedVoted = new(big.Int).Add(p.accumulatedVoted, accumulated)
}

func (p *PRep) Reward() *big.Int {
	return p.reward
}

func (p *PRep) SetReward(value *big.Int) {
	p.reward = value
}

func (p *PRep) VoterReward() *big.Int {
	return p.voterReward
}

func (p *PRep) SetVoterReward(value *big.Int) {
	p.voterReward = value
}

func (p *PRep) AccumulatedVoted() *big.Int {
	return p.accumulatedVoted
}

func (p *PRep) Bigger(p1 *PRep) bool {
	if p.enable != p1.enable {
		return p.enable
	}
	if p.pubkey != p1.pubkey {
		return p.pubkey
	}
	c := p.power.Cmp(p1.power)
	if c != 0 {
		return c == 1
	}
	c = p.delegated.Cmp(p1.delegated)
	if c != 0 {
		return c == 1
	}
	return bytes.Compare(p.owner.Bytes(), p1.owner.Bytes()) > 0
}

// TODO with VotedV2
func (p *PRep) ToVoted() *icreward.Voted {
	voted := &icreward.Voted{}
	voted.SetEnable(p.enable)
	voted.SetBonded(p.bonded)
	voted.SetDelegated(p.delegated)
	//commissionRate: p.commissionRate,
	return voted
}

func NewPRep(owner module.Address, enable bool, delegated, bonded *big.Int, pubkey bool) *PRep {
	return &PRep{
		owner:     owner,
		enable:    enable,
		delegated: delegated,
		bonded:    bonded,
		pubkey:    pubkey,
	}
}

type PRepInfo struct {
	preps                 map[string]*PRep
	totalAccumulatedPower *big.Int

	electedPRepCount int
	bondRequirement  int
	offsetLimit      int
	rank             []string
}

func (p *PRepInfo) GetPRep(key string) *PRep {
	prep, _ := p.preps[key]
	return prep
}

func (p *PRepInfo) ElectedPRepCount() int {
	return p.electedPRepCount
}

func (p *PRepInfo) OffsetLimit() int {
	return p.offsetLimit
}

func (p *PRepInfo) Add(target module.Address, enable bool, delegated, bonded *big.Int, pubkey bool) {
	prep := NewPRep(target, enable, delegated, bonded, pubkey)
	prep.UpdatePower(p.bondRequirement)
	p.preps[icutils.ToKey(target)] = prep
}

func (p *PRepInfo) SetEnable(target module.Address, enable bool) {
	key := icutils.ToKey(target)
	if prep, ok := p.preps[key]; ok {
		prep.SetEnable(enable)
	} else {
		p.Add(target, enable, nil, nil, false)
	}
}

func (p *PRepInfo) Sort() {
	size := len(p.preps)
	pSlice := make([]*PRep, size, size)
	i := 0
	for _, data := range p.preps {
		pSlice[i] = data
		i += 1
	}
	sort.Slice(pSlice, func(i, j int) bool {
		return pSlice[i].Bigger(pSlice[j])
	})
	rank := make([]string, size, size)
	for idx, prep := range pSlice {
		key := icutils.ToKey(prep.Owner())
		rank[idx] = key
		p.preps[key].SetRank(idx + 1)
	}
	p.rank = rank
}

func (p *PRepInfo) InitAccumulated() {
	i := 0
	for key, prep := range p.preps {
		if i >= p.electedPRepCount {
			break
		}
		i += 1
		prep.InitAccumulated(p.offsetLimit)
		p.preps[key] = prep
	}
}

func (p *PRepInfo) ApplyVote(vType VoteType, votes icstage.VoteList, offset int) {
	for _, vote := range votes {
		key := icutils.ToKey(vote.To())
		if prep, ok := p.preps[key]; !ok {
			continue
		} else {
			prep.ApplyVote(vType, vote.Amount(), offset)
			p.preps[key] = prep
		}
	}
}

// UpdateAccumulatedPower update accumulatedPower of elected PRep and totalAccumulatedPower of PRepInfo
func (p *PRepInfo) UpdateAccumulatedPower() {
	i := 0
	for key, prep := range p.preps {
		if i >= p.electedPRepCount {
			break
		}
		i += 1
		power := prep.UpdateAccumulatedPower(p.bondRequirement)
		p.preps[key] = prep
		p.totalAccumulatedPower = new(big.Int).Add(p.totalAccumulatedPower, power)
	}
}

func (p *PRepInfo) DistributeReward(totalReward, totalMinWage, minBond *big.Int, c *rewards.Calculator) error {
	count := 0
	minWage := new(big.Int).Mul(totalMinWage, big.NewInt(1000))
	minWage.Div(minWage, big.NewInt(int64(p.electedPRepCount)))
	for _, prep := range p.preps {
		if count >= p.electedPRepCount {
			break
		}
		count += 1
		if prep.Enable() == false {
			continue
		}

		reward := new(big.Int).Mul(totalReward, prep.AccumulatedPower())
		reward.Mul(reward, big.NewInt(1000))
		reward.Div(reward, p.totalAccumulatedPower)

		prepReward := new(big.Int).Mul(reward, big.NewInt(int64(prep.CommissionRate())))
		prepReward.Div(prepReward, big.NewInt(100))
		prep.SetReward(prepReward)
		prep.SetVoterReward(new(big.Int).Sub(reward, prepReward))

		iScore := new(big.Int).Set(prepReward)
		if prep.Bonded().Cmp(minBond) >= 0 {
			iScore.Add(iScore, minWage)
		}
		if err := c.UpdateIScore(prep.Owner(), iScore, rewards.TypeVoted); err != nil {
			return err
		}
	}
	return nil
}

func (p *PRepInfo) Write(temp *icreward.State) error {
	for _, p := range p.preps {
		err := temp.SetVoted(p.Owner(), p.ToVoted())
		if err != nil {
			return err
		}
	}
	return nil
}

func NewPRepInfo(bondRequirement, electedPRepCount, offsetLimit int) *PRepInfo {
	return &PRepInfo{
		preps:                 make(map[string]*PRep),
		totalAccumulatedPower: new(big.Int),
		electedPRepCount:      electedPRepCount,
		bondRequirement:       bondRequirement,
		offsetLimit:           offsetLimit,
	}
}
