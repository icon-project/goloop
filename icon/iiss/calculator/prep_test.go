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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type prep struct {
	owner          module.Address
	status         icmodule.EnableStatus
	bond           int64
	delegate       int64
	pubkey         bool
	commissionRate icmodule.Rate
}

func newTestPRep(p prep) *PRep {
	return NewPRep(p.owner, p.status, big.NewInt(p.delegate), big.NewInt(p.bond), p.commissionRate, p.pubkey)
}

func TestPRep_InitAccumulated(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	bond := int64(100)
	delegate := int64(50)

	type want struct {
		accBonded, accVoted int64
	}
	tests := []struct {
		name       string
		termPeriod int64
		want       want
	}{
		{
			"Init",
			100,
			want{
				bond * 100,
				(bond + delegate) * 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPRep(prep{a1, icmodule.ESEnable, bond, delegate, true, 0})

			p.InitAccumulated(tt.termPeriod)

			assert.Equal(t, tt.want.accBonded, p.AccumulatedBonded().Int64())
			assert.Equal(t, tt.want.accVoted, p.AccumulatedVoted().Int64())
		})
	}
}

func TestPRep_ApplyVote(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	bond := int64(100)
	delegate := int64(0)

	type want struct {
		bonded, delegated, accBonded, accVoted int64
	}
	tests := []struct {
		name   string
		vType  VoteType
		amount int64
		period int
		want   want
	}{
		{
			"bond",
			vtBond,
			20,
			200,
			want{
				bond + 20,
				delegate,
				20 * 200,
				20 * 200,
			},
		},
		{
			"delegate",
			vtDelegate,
			20,
			200,
			want{
				bond,
				delegate + 20,
				0,
				20 * 200,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPRep(prep{a1, icmodule.ESEnable, bond, delegate, true, 0})

			p.ApplyVote(tt.vType, big.NewInt(tt.amount), tt.period)

			assert.Equal(t, tt.want.bonded, p.Bonded().Int64())
			assert.Equal(t, tt.want.delegated, p.Delegated().Int64())
			assert.Equal(t, tt.want.accBonded, p.AccumulatedBonded().Int64())
			assert.Equal(t, tt.want.accVoted, p.AccumulatedVoted().Int64())
		})
	}
}

func TestPRep_Bigger(t *testing.T) {

	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")

	tests := []struct {
		name   string
		p1, p2 prep
		want   bool
	}{
		{
			"address",
			prep{a1, icmodule.ESEnable, 100, 0, true, 0},
			prep{a2, icmodule.ESEnable, 100, 0, true, 0},
			false,
		},
		{
			"delegated",
			prep{a1, icmodule.ESEnable, 99, 1, true, 0},
			prep{a1, icmodule.ESEnable, 100, 0, true, 0},
			true,
		},
		{
			"Power",
			prep{a1, icmodule.ESEnable, 99, 1, true, 0},
			prep{a1, icmodule.ESEnable, 100, 1, true, 0},
			false,
		},
		{
			"public key",
			prep{a1, icmodule.ESEnable, 100, 0, false, 0},
			prep{a1, icmodule.ESEnable, 100, 0, true, 0},
			false,
		},
		{
			"status",
			prep{a1, icmodule.ESEnable, 100, 1, true, 0},
			prep{a1, icmodule.ESJail, 100, 1, true, 0},
			true,
		},
		{
			"status == Unjail",
			prep{a1, icmodule.ESEnable, 99, 1, true, 0},
			prep{a1, icmodule.ESUnjail, 100, 1, true, 0},
			false,
		},
	}
	br := icmodule.ToRate(5)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1 := newTestPRep(tt.p1)
			p1.UpdatePower(br)
			p2 := newTestPRep(tt.p2)
			p2.UpdatePower(br)
			assert.Equal(t, tt.want, p1.Bigger(p2))
		})
	}
}

func TestPRep_ToVoted(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	status := icmodule.ESEnable
	bond := int64(100)
	delegate := int64(0)
	cr := icmodule.Rate(500)
	p := newTestPRep(prep{a1, status, bond, delegate, true, cr})

	voted := p.ToVoted()
	assert.Equal(t, icreward.VotedVersion2, voted.Version())
	assert.Equal(t, status, voted.Status())
	assert.Equal(t, bond, voted.Bonded().Int64())
	assert.Equal(t, delegate, voted.Delegated().Int64())
	assert.Equal(t, 0, voted.BondedDelegation().Sign())
}

func newTestPRepInfo(preps []prep, br icmodule.Rate, offsetLimit, electedPRepCount int) *PRepInfo {
	pInfo := NewPRepInfo(br, electedPRepCount, offsetLimit, log.New())
	for _, p := range preps {
		pInfo.Add(p.owner, p.status, big.NewInt(p.delegate), big.NewInt(p.bond), p.commissionRate, p.pubkey)
	}
	return pInfo
}

func TestPRepInfo(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")
	a3, _ := common.NewAddressFromString("hx3")
	a4, _ := common.NewAddressFromString("hx4")
	a5, _ := common.NewAddressFromString("hx5")
	a6, _ := common.NewAddressFromString("hx6")
	preps := []prep{
		{a1, icmodule.ESEnable, 100, 1000, true, 100},
		{a2, icmodule.ESJail, 200, 2000, true, 200},
		{a3, icmodule.ESUnjail, 300, 3000, true, 300},
		{a4, icmodule.ESEnable, 40, 4000, true, 400},
		{a5, icmodule.ESUnjail, 50, 5000, true, 500},
	}

	ranks := []module.Address{a3, a1, a5, a4, a2}

	// Add() and GetPRep()
	pInfo := newTestPRepInfo(preps, icmodule.ToRate(5), 100, 4)
	for _, p := range preps {
		e := newTestPRep(p)
		r := pInfo.GetPRep(icutils.ToKey(e.Owner()))
		assert.False(t, e.Equal(r))

		e.UpdatePower(pInfo.BondRequirement())
		assert.True(t, e.Equal(r))
	}

	pInfo.Sort()
	for i, r := range ranks {
		p := pInfo.GetPRep(icutils.ToKey(r))
		assert.Equal(t, i, p.Rank())
	}

	pInfo.InitAccumulated()
	for i, r := range ranks {
		p := pInfo.GetPRep(icutils.ToKey(r))
		if p.rank < pInfo.ElectedPRepCount() && p.IsElectable() {
			accBonded := new(big.Int).Mul(p.Bonded(), big.NewInt(pInfo.GetTermPeriod()))
			accVoted := new(big.Int).Mul(new(big.Int).Add(p.Bonded(), p.Delegated()), big.NewInt(pInfo.GetTermPeriod()))
			assert.Equal(t, accBonded, p.AccumulatedBonded(), i)
			assert.Equal(t, accVoted, p.AccumulatedVoted(), i)
		} else {
			assert.Equal(t, 0, p.AccumulatedBonded().Sign())
			assert.Equal(t, 0, p.AccumulatedVoted().Sign())
		}
	}

	votes := []struct {
		vType  VoteType
		vl     icstage.VoteList
		offset int
	}{
		{
			vtDelegate,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
				icstage.NewVote(a6, big.NewInt(6000)),
			},
			10,
		},
		{
			vtBond,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(100)),
				icstage.NewVote(a2, big.NewInt(200)),
				icstage.NewVote(a3, big.NewInt(300)),
			},
			30,
		},
		{
			vtDelegate,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(-1000)),
				icstage.NewVote(a2, big.NewInt(-2000)),
				icstage.NewVote(a3, big.NewInt(-3000)),
				icstage.NewVote(a5, big.NewInt(-3000)),
			},
			80,
		},
	}

	// ApplyVote()
	prev := make(map[string]*PRep)
	for _, vote := range votes {
		for _, v := range vote.vl {
			k := icutils.ToKey(v.To())
			p := pInfo.GetPRep(k)
			if p == nil {
				continue
			} else {
				prev[k] = p.Clone()
			}
		}

		pInfo.ApplyVote(vote.vType, vote.vl, vote.offset)

		period := big.NewInt(int64(pInfo.OffsetLimit() - vote.offset))
		for _, v := range vote.vl {
			k := icutils.ToKey(v.To())
			p := pInfo.GetPRep(k)
			if p == nil {
				continue
			}
			accuAmount := new(big.Int).Mul(v.Amount(), period)
			if vote.vType == vtBond {
				e := new(big.Int).Add(prev[k].Bonded(), v.Amount())
				assert.Equal(t, e, p.Bonded())
				e = new(big.Int).Add(prev[k].AccumulatedBonded(), accuAmount)
				assert.Equal(t, e, p.AccumulatedBonded())
				e = new(big.Int).Add(prev[k].AccumulatedVoted(), accuAmount)
				assert.Equal(t, e, p.AccumulatedVoted())
			} else if vote.vType == vtDelegate {
				e := new(big.Int).Add(prev[k].Delegated(), v.Amount())
				assert.Equal(t, e, p.Delegated())
				e = new(big.Int).Add(prev[k].AccumulatedVoted(), accuAmount)
				assert.Equal(t, e, p.AccumulatedVoted())
			}
		}
	}

	status := []struct {
		target module.Address
		es     icmodule.EnableStatus
	}{
		{a3, icmodule.ESEnable},
		{a5, icmodule.ESJail},
		{a4, icmodule.ESJail},
		{a6, icmodule.ESEnable}, // will add new PRep
	}
	for _, s := range status {
		old := pInfo.GetPRep(icutils.ToKey(s.target))
		pInfo.SetStatus(s.target, s.es)
		p := pInfo.GetPRep(icutils.ToKey(s.target))
		assert.Equal(t, s.es, p.Status())
		if old == nil {
			bigZero := new(big.Int)
			assert.Equal(t, bigZero, p.Bonded())
			assert.Equal(t, bigZero, p.Delegated())
			assert.Equal(t, bigZero, p.Power())
			assert.False(t, p.Pubkey())
		}
	}

	pInfo.UpdateAccumulatedPower()
	totalPower := new(big.Int)
	for _, r := range ranks {
		p := pInfo.GetPRep(icutils.ToKey(r))
		if p.rank < pInfo.ElectedPRepCount() {
			power := icutils.CalcPower(pInfo.BondRequirement(), p.AccumulatedBonded(), p.AccumulatedVoted())
			assert.Equal(t, power, p.AccumulatedPower())
			totalPower.Add(totalPower, p.AccumulatedPower())
		}
	}
	assert.Equal(t, totalPower, pInfo.TotalAccumulatedPower())

	// CalculateReward
	totalReward := int64(1_000_000_000)
	totalMinWage := int64(10_000_000)
	minWage := totalMinWage * int64(pInfo.OffsetLimit()+1) * icmodule.IScoreICXRatio / icmodule.MonthBlock
	minWage = minWage / int64(pInfo.ElectedPRepCount())
	minBond := int64(300)

	p1Reward, p1Commission := prepReward(pInfo.GetPRep(icutils.ToKey(a1)), totalReward, pInfo.TotalAccumulatedPower().Int64(), pInfo.OffsetLimit())
	p3Reward, p3Commission := prepReward(pInfo.GetPRep(icutils.ToKey(a3)), totalReward, pInfo.TotalAccumulatedPower().Int64(), pInfo.OffsetLimit())

	iScores := []struct {
		target      module.Address
		commission  *big.Int
		minWage     *big.Int
		voterReward *big.Int
	}{
		{a1, big.NewInt(p1Commission), big.NewInt(0), big.NewInt(p1Reward - p1Commission)},
		{a2, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{a3, big.NewInt(p3Commission), big.NewInt(minWage), big.NewInt(p3Reward - p3Commission)},
		{a4, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{a5, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
	}

	err := pInfo.CalculateReward(big.NewInt(totalReward), big.NewInt(totalMinWage), big.NewInt(minBond))
	assert.NoError(t, err)
	for _, is := range iScores {
		p := pInfo.GetPRep(icutils.ToKey(is.target))
		assert.Equal(t, is.commission, p.Commission(), p)
		assert.Equal(t, is.voterReward, p.VoterReward(), p)
		assert.Equal(t, new(big.Int).Add(is.commission, is.minWage), p.GetReward(), p)
	}
}

func prepReward(prep *PRep, totalReward, totalPower int64, offsetLimit int) (reward, commission int64) {
	reward = totalReward * int64(offsetLimit+1) * icmodule.IScoreICXRatio / icmodule.MonthBlock
	reward = reward * prep.AccumulatedPower().Int64() / totalPower
	commission = prep.CommissionRate().MulInt64(reward)
	return
}
