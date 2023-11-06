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
	"fmt"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/icon/iiss/rewards/common"
	"github.com/icon-project/goloop/module"
)

type PRep struct {
	status         icmodule.EnableStatus
	delegated      *big.Int
	bonded         *big.Int
	commissionRate icmodule.Rate

	owner             module.Address
	power             *big.Int
	pubkey            bool
	rank              int
	accumulatedBonded *big.Int
	accumulatedVoted  *big.Int
	accumulatedPower  *big.Int
	commission        *big.Int // in IScore
	voterReward       *big.Int // in IScore
	wage              *big.Int // in IScore
}

func (p *PRep) Electable() bool {
	return p.pubkey && (p.status == icmodule.ESEnable || p.status == icmodule.ESUnjail)
}

func (p *PRep) Rewardable(electedPRepCount int) bool {
	return p.status == icmodule.ESEnable && p.rank <= electedPRepCount && p.accumulatedPower.Sign() == 1
}

func (p *PRep) Status() icmodule.EnableStatus {
	return p.status
}

func (p *PRep) SetStatus(status icmodule.EnableStatus) {
	p.status = status
}

func (p *PRep) Bonded() *big.Int {
	return p.bonded
}

func (p *PRep) Delegated() *big.Int {
	return p.delegated
}

func (p *PRep) CommissionRate() icmodule.Rate {
	return p.commissionRate
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

func (p *PRep) Pubkey() bool {
	return p.pubkey
}

func (p *PRep) GetPower(bondRequirement icmodule.Rate) *big.Int {
	return icutils.CalcPower(bondRequirement, p.bonded, p.GetVoted())
}

func (p *PRep) UpdatePower(bondRequirement icmodule.Rate) *big.Int {
	p.power = p.GetPower(bondRequirement)
	return p.power
}

func (p *PRep) Rank() int {
	return p.rank
}

func (p *PRep) SetRank(rank int) {
	p.rank = rank
}

func (p *PRep) AccumulatedPower() *big.Int {
	return p.accumulatedPower
}

func (p *PRep) GetAccumulatedPower(bondRequirement icmodule.Rate) *big.Int {
	return icutils.CalcPower(bondRequirement, p.accumulatedBonded, p.accumulatedVoted)
}

func (p *PRep) UpdateAccumulatedPower(bondRequirement icmodule.Rate) *big.Int {
	p.accumulatedPower = p.GetAccumulatedPower(bondRequirement)
	return p.accumulatedPower
}

func (p *PRep) InitAccumulated(termPeriod int64) {
	period := big.NewInt(termPeriod)
	p.accumulatedVoted = new(big.Int).Mul(p.GetVoted(), period)
	p.accumulatedBonded = new(big.Int).Mul(p.bonded, period)
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

func (p *PRep) Commission() *big.Int {
	return p.commission
}

func (p *PRep) VoterReward() *big.Int {
	return p.voterReward
}

func (p *PRep) SetVoterReward(value *big.Int) {
	p.voterReward = value
}

func (p *PRep) GetReward() *big.Int {
	return new(big.Int).Add(p.commission, p.wage)
}

func (p *PRep) AccumulatedBonded() *big.Int {
	return p.accumulatedBonded
}

func (p *PRep) AccumulatedVoted() *big.Int {
	return p.accumulatedVoted
}

func (p *PRep) CalculateReward(totalPRepReward, totalAccumulatedPower, minBond, minWage *big.Int) {
	prepReward := new(big.Int).Mul(totalPRepReward, p.accumulatedPower)
	prepReward.Div(prepReward, totalAccumulatedPower)

	commission := p.commissionRate.MulBigInt(prepReward)
	p.commission = commission
	p.voterReward = new(big.Int).Sub(prepReward, commission)
	if p.bonded.Cmp(minBond) >= 0 {
		p.wage = minWage
	}
}

func (p *PRep) Bigger(p1 *PRep) bool {
	if p.Electable() != p1.Electable() {
		return p.Electable()
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

func (p *PRep) ToVoted() *icreward.Voted {
	voted := icreward.NewVotedV2()
	if p.Status() == icmodule.ESEnableAtNextTerm {
		voted.SetStatus(icmodule.ESEnable)
	} else {
		voted.SetStatus(p.status)
	}
	voted.SetBonded(p.bonded)
	voted.SetDelegated(p.delegated)
	voted.SetCommissionRate(p.commissionRate)
	return voted
}

func (p *PRep) Equal(p1 *PRep) bool {
	return p.status == p1.status &&
		p.delegated.Cmp(p1.delegated) == 0 &&
		p.bonded.Cmp(p1.bonded) == 0 &&
		p.commissionRate == p1.commissionRate &&
		p.owner.Equal(p1.owner) &&
		p.power.Cmp(p1.power) == 0 &&
		p.pubkey == p1.pubkey &&
		p.rank == p1.rank &&
		p.accumulatedBonded.Cmp(p1.accumulatedBonded) == 0 &&
		p.accumulatedVoted.Cmp(p1.accumulatedVoted) == 0 &&
		p.accumulatedPower.Cmp(p1.accumulatedPower) == 0 &&
		p.commission.Cmp(p1.commission) == 0 &&
		p.voterReward.Cmp(p1.voterReward) == 0 &&
		p.wage.Cmp(p1.wage) == 0
}

func (p *PRep) Clone() *PRep {
	return &PRep{
		owner:             p.owner,
		status:            p.status,
		delegated:         new(big.Int).Set(p.delegated),
		bonded:            new(big.Int).Set(p.bonded),
		commissionRate:    p.commissionRate,
		pubkey:            p.pubkey,
		power:             new(big.Int).Set(p.power),
		accumulatedBonded: new(big.Int).Set(p.accumulatedBonded),
		accumulatedVoted:  new(big.Int).Set(p.accumulatedVoted),
		accumulatedPower:  new(big.Int).Set(p.accumulatedPower),
		commission:        new(big.Int).Set(p.commission),
		voterReward:       new(big.Int).Set(p.voterReward),
		wage:              new(big.Int).Set(p.wage),
	}
}
func (p *PRep) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "PRep{status=%s delegated=%d bonded=%d commissionRate=%d "+
				"owner=%s power=%d pubkey=%v rank=%d accumulatedBonded=%d accumulatedVoted=%d accumulatedPower=%d "+
				"commission=%d voterReward=%d wage=%d}",
				p.status, p.delegated, p.bonded, p.commissionRate,
				p.owner, p.power, p.pubkey, p.rank, p.accumulatedBonded, p.accumulatedVoted, p.accumulatedPower,
				p.commission, p.voterReward, p.wage,
			)
		} else {
			fmt.Fprintf(f, "PRep{%s %d %d %d %s %d %v %d %d %d %d %d %d %d}",
				p.status, p.delegated, p.bonded, p.commissionRate,
				p.owner, p.power, p.pubkey, p.rank, p.accumulatedBonded, p.accumulatedVoted, p.accumulatedPower,
				p.commission, p.voterReward, p.wage,
			)
		}
	}
}

func NewPRep(owner module.Address, status icmodule.EnableStatus, delegated, bonded *big.Int,
	commissionRate icmodule.Rate, pubkey bool) *PRep {
	return &PRep{
		owner:             owner,
		status:            status,
		delegated:         delegated,
		bonded:            bonded,
		commissionRate:    commissionRate,
		pubkey:            pubkey,
		power:             new(big.Int),
		accumulatedBonded: new(big.Int),
		accumulatedVoted:  new(big.Int),
		accumulatedPower:  new(big.Int),
		commission:        new(big.Int),
		voterReward:       new(big.Int),
		wage:              new(big.Int),
	}
}

// PRepInfo stores information for PRep reward calculation.
type PRepInfo struct {
	preps                 map[string]*PRep
	totalAccumulatedPower *big.Int

	electedPRepCount int
	bondRequirement  icmodule.Rate
	offsetLimit      int
	rank             []string
	log              log.Logger
}

func (p *PRepInfo) PReps() map[string]*PRep {
	return p.preps
}

func (p *PRepInfo) GetPRep(key string) *PRep {
	prep, _ := p.preps[key]
	return prep
}

func (p *PRepInfo) TotalAccumulatedPower() *big.Int {
	return p.totalAccumulatedPower
}

func (p *PRepInfo) ElectedPRepCount() int {
	return p.electedPRepCount
}

func (p *PRepInfo) OffsetLimit() int {
	return p.offsetLimit
}

func (p *PRepInfo) GetTermPeriod() int64 {
	return int64(p.offsetLimit + 1)
}

func (p *PRepInfo) BondRequirement() icmodule.Rate {
	return p.bondRequirement
}

func (p *PRepInfo) Add(target module.Address, status icmodule.EnableStatus, delegated, bonded *big.Int,
	commissionRate icmodule.Rate, pubkey bool) {
	prep := NewPRep(target, status, delegated, bonded, commissionRate, pubkey)
	prep.UpdatePower(p.bondRequirement)
	p.preps[icutils.ToKey(target)] = prep
}

func (p *PRepInfo) SetStatus(target module.Address, status icmodule.EnableStatus) {
	p.log.Debugf("SetStatus: %s to %d", target, status)
	key := icutils.ToKey(target)
	if prep, ok := p.preps[key]; ok {
		prep.SetStatus(status)
	} else {
		p.Add(target, status, new(big.Int), new(big.Int), 0, false)
	}
}

func (p *PRepInfo) Sort() {
	size := len(p.preps)
	pSlice := make([]*PRep, size)
	i := 0
	for _, data := range p.preps {
		pSlice[i] = data
		i += 1
	}
	sort.Slice(pSlice, func(i, j int) bool {
		return pSlice[i].Bigger(pSlice[j])
	})
	rank := make([]string, size)
	for idx, prep := range pSlice {
		key := icutils.ToKey(prep.Owner())
		rank[idx] = key
		p.preps[key].SetRank(idx + 1)
	}
	p.rank = rank
}

func (p *PRepInfo) InitAccumulated() {
	for i, key := range p.rank {
		if i >= p.electedPRepCount {
			break
		}
		prep := p.preps[key]
		prep.InitAccumulated(p.GetTermPeriod())
		p.preps[key] = prep
	}
}

func (p *PRepInfo) ApplyVote(vType VoteType, votes icstage.VoteList, offset int) {
	for _, vote := range votes {
		key := icutils.ToKey(vote.To())
		if prep, ok := p.preps[key]; !ok {
			continue
		} else {
			prep.ApplyVote(vType, vote.Amount(), p.offsetLimit-offset)
			p.log.Debugf("ApplyVote %+v: by %d, %d %+v, %d * %d",
				prep, vType, offset, vote, vote.Amount(), p.offsetLimit-offset)
			p.preps[key] = prep
		}
	}
}

// UpdateAccumulatedPower update accumulatedPower of elected PRep and totalAccumulatedPower of PRepInfo.
func (p *PRepInfo) UpdateAccumulatedPower() {
	for i, key := range p.rank {
		if i >= p.electedPRepCount {
			break
		}
		prep := p.preps[key]
		power := prep.UpdateAccumulatedPower(p.bondRequirement)
		p.preps[key] = prep
		p.totalAccumulatedPower = new(big.Int).Add(p.totalAccumulatedPower, power)
		p.log.Debugf("[%d] totalAccumulatedPower %d = old + %d by %s", i, p.totalAccumulatedPower, power, prep.owner)
	}
}

func fundToPeriodIScore(reward *big.Int, period int64) *big.Int {
	value := new(big.Int).Mul(reward, big.NewInt(period*icmodule.IScoreICXRatio))
	return value.Div(value, big.NewInt(icmodule.MonthBlock))
}

// CalculateReward calculates commission, wage and voter reward of the PRep.
func (p *PRepInfo) CalculateReward(totalReward, totalMinWage, minBond *big.Int) error {
	p.log.Debugf("CalculateReward()")
	tReward := fundToPeriodIScore(totalReward, p.GetTermPeriod())
	minWage := fundToPeriodIScore(totalMinWage, p.GetTermPeriod())
	p.log.Debugf("RewardFund: PRep: %d, wage: %d", tReward, minWage)
	minWage.Div(minWage, big.NewInt(int64(p.electedPRepCount)))
	p.log.Debugf("wage to a prep: %d", minWage)
	p.log.Debugf("TotalAccumulatedPower: %d", p.totalAccumulatedPower)
	for rank, key := range p.rank {
		prep, _ := p.preps[key]
		if rank >= p.electedPRepCount {
			break
		}
		if !prep.Rewardable(p.electedPRepCount) {
			continue
		}
		prep.CalculateReward(tReward, p.totalAccumulatedPower, minBond, minWage)

		p.log.Debugf("rank#%d: %+v", rank, prep)
	}
	return nil
}

// Write writes updated Voted to database
func (p *PRepInfo) Write(writer common.Writer) error {
	for _, prep := range p.preps {
		err := writer.SetVoted(prep.Owner(), prep.ToVoted())
		if err != nil {
			return err
		}
	}
	return nil
}

func NewPRepInfo(bondRequirement icmodule.Rate, electedPRepCount, offsetLimit int, logger log.Logger) *PRepInfo {
	return &PRepInfo{
		preps:                 make(map[string]*PRep),
		totalAccumulatedPower: new(big.Int),
		electedPRepCount:      electedPRepCount,
		bondRequirement:       bondRequirement,
		offsetLimit:           offsetLimit,
		log:                   logger,
	}
}
