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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	rc "github.com/icon-project/goloop/icon/iiss/rewards/common"
	"github.com/icon-project/goloop/module"
)

type prep struct {
	owner          module.Address
	status         icstage.EnableStatus
	bond           int64
	delegate       int64
	pubkey         bool
	commissionRate int64
}

func newTestPRep(p prep) *PRep {
	return NewPRep(p.owner, p.status, big.NewInt(p.delegate), big.NewInt(p.bond), big.NewInt(p.commissionRate), p.pubkey)
}

func TestPRep_getPower(t *testing.T) {
	tests := []struct {
		name          string
		bonded, voted int64
		br            int
		want          int64
	}{
		{
			"less bond",
			10, 990, 5,
			200,
		},
		{
			"exact bond",
			50, 950, 5,
			1000,
		},
		{
			"more bond",
			500, 500, 5,
			1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := getPower(big.NewInt(tt.bonded), big.NewInt(tt.bonded+tt.voted), tt.br)
			assert.Equal(t, tt.want, ret.Int64())
		})
	}
}

func TestPRep_InitAccumulated(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	bond := int64(100)
	delegate := int64(50)

	type want struct {
		accBonded, accVoted int64
	}
	tests := []struct {
		name        string
		offsetLimit int
		want        want
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
			p := newTestPRep(prep{a1, icstage.ESEnable, bond, delegate, true, 0})

			p.InitAccumulated(tt.offsetLimit)

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
			p := newTestPRep(prep{a1, icstage.ESEnable, bond, delegate, true, 0})

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
			prep{a1, icstage.ESEnable, 100, 0, true, 0},
			prep{a2, icstage.ESEnable, 100, 0, true, 0},
			false,
		},
		{
			"delegated",
			prep{a1, icstage.ESEnable, 99, 1, true, 0},
			prep{a1, icstage.ESEnable, 100, 0, true, 0},
			true,
		},
		{
			"Power",
			prep{a1, icstage.ESEnable, 99, 1, true, 0},
			prep{a1, icstage.ESEnable, 100, 1, true, 0},
			false,
		},
		{
			"public key",
			prep{a1, icstage.ESEnable, 100, 0, false, 0},
			prep{a1, icstage.ESEnable, 100, 0, true, 0},
			false,
		},
		{
			"status",
			prep{a1, icstage.ESEnable, 100, 1, true, 0},
			prep{a1, icstage.ESJail, 100, 1, true, 0},
			true,
		},
		{
			"status == Unjail",
			prep{a1, icstage.ESEnable, 99, 1, true, 0},
			prep{a1, icstage.ESUnjail, 100, 1, true, 0},
			false,
		},
	}
	br := 5

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
	status := icstage.ESEnable
	bond := int64(100)
	delegate := int64(0)
	cr := int64(500)
	ncr := int64(250)
	p := newTestPRep(prep{a1, status, bond, delegate, true, cr})
	p.SetNCommissionRate(big.NewInt(ncr))

	voted := p.ToVoted()
	assert.Equal(t, icreward.VotedVersion2, voted.Version())
	assert.Equal(t, status, voted.Status())
	assert.Equal(t, bond, voted.Bonded().Int64())
	assert.Equal(t, delegate, voted.Delegated().Int64())
	assert.Equal(t, 0, voted.BondedDelegation().Sign())
	assert.Equal(t, ncr, voted.CommissionRate().Int64())
}

func newTestPRepInfo(preps []prep, br, offsetLimit, electedPRepCount int) *PRepInfo {
	pInfo := NewPRepInfo(br, electedPRepCount, offsetLimit)
	for _, p := range preps {
		pInfo.Add(p.owner, p.status, big.NewInt(p.delegate), big.NewInt(p.bond), big.NewInt(p.commissionRate), p.pubkey)
	}
	return pInfo
}

type testRewardUpdater struct {
	iScore map[rc.RewardType]map[string]*big.Int
}

func newTestRewardUpdater() *testRewardUpdater {
	return &testRewardUpdater{
		iScore: make(map[rc.RewardType]map[string]*big.Int),
	}
}

func (tru *testRewardUpdater) UpdateIScore(addr module.Address, reward *big.Int, t rc.RewardType) error {
	key := icutils.ToKey(addr)
	if tru.iScore[t] == nil {
		tru.iScore[t] = make(map[string]*big.Int)
	}
	if is, ok := tru.iScore[t][key]; ok {
		is.Add(is, reward)
	} else {
		tru.iScore[t][key] = reward
	}
	return nil
}

func (tru *testRewardUpdater) GetIScore(addr module.Address, t rc.RewardType) *big.Int {
	if is, ok := tru.iScore[t][icutils.ToKey(addr)]; ok {
		return is
	} else {
		return new(big.Int)
	}
}

func TestPRepInfo(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")
	a3, _ := common.NewAddressFromString("hx3")
	a4, _ := common.NewAddressFromString("hx4")
	a5, _ := common.NewAddressFromString("hx5")
	preps := []prep{
		{a1, icstage.ESEnable, 100, 1000, true, 1},
		{a2, icstage.ESJail, 200, 2000, true, 2},
		{a3, icstage.ESUnjail, 300, 3000, true, 3},
		{a4, icstage.ESEnable, 40, 4000, true, 4},
		{a5, icstage.ESUnjail, 50, 5000, true, 5},
	}

	ranks := []module.Address{a3, a1, a5, a4, a2}

	// Add() and GetPRep()
	pInfo := newTestPRepInfo(preps, 5, 100, 4)
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
		assert.Equal(t, i+1, p.Rank())
	}

	pInfo.InitAccumulated()
	for i, r := range ranks {
		p := pInfo.GetPRep(icutils.ToKey(r))
		if p.rank <= pInfo.ElectedPRepCount() && p.Electable() {
			accBonded := new(big.Int).Mul(p.Bonded(), big.NewInt(int64(pInfo.OffsetLimit())))
			accVoted := new(big.Int).Mul(new(big.Int).Add(p.Bonded(), p.Delegated()), big.NewInt(int64(pInfo.OffsetLimit())))
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
			prev[k] = pInfo.GetPRep(k).Clone()
		}

		pInfo.ApplyVote(vote.vType, vote.vl, vote.offset)

		period := big.NewInt(int64(pInfo.OffsetLimit() - vote.offset))
		for _, v := range vote.vl {
			k := icutils.ToKey(v.To())
			p := pInfo.GetPRep(k)
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
		es     icstage.EnableStatus
	}{
		{a3, icstage.ESEnable},
		{a5, icstage.ESJail},
		{a4, icstage.ESJail},
	}
	for _, s := range status {
		pInfo.SetStatus(s.target, s.es)
		p := pInfo.GetPRep(icutils.ToKey(s.target))
		assert.Equal(t, s.es, p.Status())
	}

	pInfo.UpdateAccumulatedPower()
	totalPower := new(big.Int)
	for _, r := range ranks {
		p := pInfo.GetPRep(icutils.ToKey(r))
		if p.rank <= pInfo.ElectedPRepCount() {
			power := new(big.Int).Mul(p.AccumulatedBonded(), big.NewInt(100))
			power.Div(power, big.NewInt(int64(pInfo.BondRequirement())))
			if power.Cmp(p.AccumulatedVoted()) == 1 {
				power.Set(p.AccumulatedVoted())
			}
			assert.Equal(t, power, p.AccumulatedPower())
			totalPower.Add(totalPower, p.AccumulatedPower())
		}
	}
	assert.Equal(t, totalPower, pInfo.TotalAccumulatedPower())

	// DistributeReward
	tru := newTestRewardUpdater()
	totalReward := int64(1_000_000_000)
	minWage := int64(10_000)
	totalMinWage := int64(pInfo.ElectedPRepCount()) * minWage
	minBond := int64(300)

	p1Reward, p1Commission := prepReward(pInfo.GetPRep(icutils.ToKey(a1)), totalReward, pInfo.TotalAccumulatedPower().Int64())
	p3Reward, p3Commission := prepReward(pInfo.GetPRep(icutils.ToKey(a3)), totalReward, pInfo.TotalAccumulatedPower().Int64())

	iScores := []struct {
		target      module.Address
		commission  *big.Int
		minWage     *big.Int
		voterReward *big.Int
	}{
		{a1, big.NewInt(p1Commission), big.NewInt(0), big.NewInt(p1Reward - p1Commission)},
		{a2, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{a3, big.NewInt(p3Commission), big.NewInt(minWage * 1000), big.NewInt(p3Reward - p3Commission)},
		{a4, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{a5, big.NewInt(0), big.NewInt(0), big.NewInt(0)},
	}

	err := pInfo.DistributeReward(big.NewInt(totalReward), big.NewInt(totalMinWage), big.NewInt(minBond), tru)
	assert.NoError(t, err)
	for _, is := range iScores {
		p := pInfo.GetPRep(icutils.ToKey(is.target))
		assert.Equal(t, is.commission, p.Commission(), p)
		assert.Equal(t, is.voterReward, p.VoterReward(), p)
		assert.Equal(t, new(big.Int).Add(is.commission, is.minWage), tru.GetIScore(is.target, rc.RTPRep), p)
	}
}

func prepReward(prep *PRep, totalReward, totalPower int64) (reward, commission int64) {
	reward = totalReward * prep.AccumulatedPower().Int64() * 1000 / totalPower
	commission = reward * prep.CommissionRate().Int64() / 100
	return
}
