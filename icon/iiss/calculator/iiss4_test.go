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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type testCalculator struct {
	stage  *icstage.State
	reward *icreward.State
	log    log.Logger

	back  *icstage.Snapshot
	base  *icreward.Snapshot
	temp  *icreward.State
	stats *Stats
}

func (t *testCalculator) Back() *icstage.Snapshot {
	return t.back
}

func (t *testCalculator) Base() *icreward.Snapshot {
	return t.base
}

func (t *testCalculator) Temp() *icreward.State {
	return t.temp
}

func (t *testCalculator) Stats() *Stats {
	return t.stats
}

func (t *testCalculator) Logger() log.Logger {
	return t.log
}

func (t *testCalculator) UpdateIScore(addr module.Address, reward *big.Int, type_ RewardType) error {
	iScore, err := t.temp.GetIScore(addr)
	if err != nil {
		return err
	}
	nIScore := iScore.Added(reward)
	if err = t.temp.SetIScore(addr, nIScore); err != nil {
		return err
	}
	t.stats.IncreaseReward(type_, reward)
	return nil
}

func (t *testCalculator) AddGlobal(electedPRepCount int) error {
	rFund := icstate.NewRewardFund(icstate.RFVersion2)
	rFund.SetIGlobal(big.NewInt(1_000_000))
	alloc := map[icstate.RFundKey]icmodule.Rate{
		icstate.KeyIprep:  icmodule.ToRate(77),
		icstate.KeyIwage:  icmodule.ToRate(13),
		icstate.KeyIcps:   icmodule.ToRate(10),
		icstate.KeyIrelay: icmodule.ToRate(0),
	}
	rFund.SetAllocation(alloc)
	return t.stage.AddGlobalV3(0, 0, 99, electedPRepCount, icmodule.ToRate(5),
		rFund, big.NewInt(100))
}

func (t *testCalculator) GetGlobalFromBack() (icstage.Global, error) {
	return t.back.GetGlobal()
}

func (t *testCalculator) AddVoted(addr module.Address, voted *icreward.Voted) error {
	return t.reward.SetVoted(addr, voted)
}

func (t *testCalculator) GetVotedFromTemp(addr module.Address) (*icreward.Voted, error) {
	return t.temp.GetVoted(addr)
}

func (t *testCalculator) SetDSA(index int) error {
	dsa := icreward.NewDSA()
	dsa = dsa.Updated(index)
	return t.reward.SetDSA(dsa)
}

func (t *testCalculator) SetPublicKey(addr module.Address, index int) error {
	pubkey := icreward.NewPublicKey()
	pubkey = pubkey.Updated(index)
	return t.reward.SetPublicKey(addr, pubkey)
}

func (t *testCalculator) SetBonding(addr module.Address, bonding *icreward.Bonding) error {
	return t.reward.SetBonding(addr, bonding)
}

func (t *testCalculator) SetDelegating(addr module.Address, delegating *icreward.Delegating) error {
	return t.reward.SetDelegating(addr, delegating)
}

func (t *testCalculator) AddEventEnable(offset int, target module.Address, status icmodule.EnableStatus) (int64, error) {
	return t.stage.AddEventEnable(offset, target, status)
}

func (t *testCalculator) AddEventDelegation(offset int, from module.Address, votes icstage.VoteList) (int64, *icobject.Object, error) {
	return t.stage.AddEventDelegation(offset, from, votes)
}

func (t *testCalculator) AddEventBond(offset int, from module.Address, votes icstage.VoteList) (int64, *icobject.Object, error) {
	return t.stage.AddEventBond(offset, from, votes)
}

func (t *testCalculator) GetBondingFromTemp(addr module.Address) (*icreward.Bonding, error) {
	return t.temp.GetBonding(addr)
}

func (t *testCalculator) GetDelegatingFromTemp(addr module.Address) (*icreward.Delegating, error) {
	return t.temp.GetDelegating(addr)
}

func (t *testCalculator) GetIScoreFromTemp(addr module.Address) (*icreward.IScore, error) {
	return t.temp.GetIScore(addr)
}

func (t *testCalculator) isVoterRewardable(addr module.Address, pi *PRepInfo) (bool, error) {
	d, err := t.GetDelegatingFromTemp(addr)
	if err != nil {
		return false, err
	}
	if d != nil {
		for _, v := range d.Delegations {
			k := icutils.ToKey(v.To())
			p := pi.GetPRep(k)
			if p.IsRewardable(pi.ElectedPRepCount()) && p.VoterReward().Sign() == 1 {
				return true, nil
			}
		}
	}

	b, err := t.GetBondingFromTemp(addr)
	if err != nil {
		return false, err
	}
	if b != nil {
		for _, v := range b.Bonds {
			k := icutils.ToKey(v.To())
			p := pi.GetPRep(k)
			if p.IsRewardable(pi.ElectedPRepCount()) && p.VoterReward().Sign() == 1 {
				return true, nil
			}
		}
	}

	return false, nil
}

func (t *testCalculator) Build() {
	t.back = t.stage.GetSnapshot()
	t.temp = t.reward
	t.base = t.reward.GetSnapshot()
}

func newTestCalculator() *testCalculator {
	database := db.NewMapDB()
	tc := &testCalculator{
		stage:  icstage.NewState(database),
		reward: icreward.NewState(database, nil),
		stats:  NewStats(),
		log:    log.New(),
	}
	tc.Build()
	return tc
}

func TestReward_NewReward(t *testing.T) {
	tc := newTestCalculator()

	tc.Build()
	r, err := NewIISS4Reward(tc)
	assert.NotNil(t, r)
	assert.NoError(t, err)
	assert.Nil(t, r.g)

	tc.AddGlobal(0)
	tc.Build()
	r, err = NewIISS4Reward(tc)
	assert.NotNil(t, r)
	assert.NoError(t, err)
	g, err := tc.GetGlobalFromBack()
	assert.NoError(t, err)
	assert.Equal(t, g, r.g)
}

func TestReward(t *testing.T) {
	a1 := common.MustNewAddressFromString("hx1")
	a2 := common.MustNewAddressFromString("hx2")
	a3 := common.MustNewAddressFromString("hx3")
	a4 := common.MustNewAddressFromString("hx4")
	a5 := common.MustNewAddressFromString("hx5")
	addrs := []module.Address{a1, a2, a3, a4, a5}

	v1 := icreward.NewVotedV2()
	v1.SetStatus(icmodule.ESEnable)
	v1.SetCommissionRate(icmodule.ToRate(10))
	v2 := icreward.NewVotedV2()
	v2.SetStatus(icmodule.ESEnable)
	v2.SetCommissionRate(icmodule.ToRate(5))
	v2.SetBonded(big.NewInt(20))
	v2.SetDelegated(big.NewInt(20))
	v3 := icreward.NewVotedV2()
	v3.SetStatus(icmodule.ESUnjail)
	v3.SetBonded(big.NewInt(30))
	v3.SetDelegated(big.NewInt(30))
	v4 := icreward.NewVotedV2()
	v5 := icreward.NewVotedV2()
	v5.SetStatus(icmodule.ESEnable)
	v5.SetBonded(big.NewInt(50))
	v5.SetDelegated(big.NewInt(50))
	voteds := map[string]*icreward.Voted{
		icutils.ToKey(a1): v1,
		icutils.ToKey(a2): v2,
		icutils.ToKey(a3): v3,
		icutils.ToKey(a4): v4,
		icutils.ToKey(a5): v5,
	}

	tc := newTestCalculator()
	err := tc.AddGlobal(4)
	assert.NoError(t, err)
	for a, v := range voteds {
		if v.IsEmpty() {
			continue
		}
		err = tc.AddVoted(common.MustNewAddress([]byte(a)), v)
		assert.NoError(t, err)
	}

	tc.SetBonding(a1, &icreward.Bonding{Bonds: icstate.Bonds{icstate.NewBond(a1, big.NewInt(100))}})
	tc.SetDelegating(a1, &icreward.Delegating{Delegations: icstate.Delegations{icstate.NewDelegation(a1, big.NewInt(100))}})
	tc.SetDelegating(a2, &icreward.Delegating{Delegations: icstate.Delegations{icstate.NewDelegation(a2, big.NewInt(100))}})
	tc.SetBonding(a3, &icreward.Bonding{Bonds: icstate.Bonds{icstate.NewBond(a3, big.NewInt(100))}})

	dsaIndex := 1
	pubkeys := map[string]int{
		icutils.ToKey(a1): dsaIndex,
		icutils.ToKey(a2): dsaIndex,
		icutils.ToKey(a3): 0,
		icutils.ToKey(a4): dsaIndex,
		icutils.ToKey(a5): dsaIndex,
	}

	err = tc.SetDSA(dsaIndex)
	assert.NoError(t, err)
	for a, p := range pubkeys {
		err = tc.SetPublicKey(common.MustNewAddress([]byte(a)), p)
		assert.NoError(t, err)
	}

	tc.Build()

	r, err := NewIISS4Reward(tc)
	assert.NoError(t, err)

	// loadPRepInfo()
	err = r.loadPRepInfo()
	assert.NoError(t, err)

	for _, a := range addrs {
		t.Run(fmt.Sprintf("loadPRepInfo-voted-%s", a), func(t *testing.T) {
			key := icutils.ToKey(a)
			p := r.pi.GetPRep(key)

			v := voteds[key]
			if v.IsEmpty() {
				assert.Nil(t, p)
			} else {
				assert.True(t, v.Equal(p.ToVoted()))

				pubkey := pubkeys[key]
				assert.Equal(t, pubkey == dsaIndex, p.pubkey, p)
			}
		})
	}

	// check sort
	t.Run("loadPRepInfo-Sort", func(t *testing.T) {
		for i, p := range r.pi.rank {
			assert.Equal(t, i, p.rank)
		}
	})

	// check initAccumulated
	t.Run("loadPRepInfo-InitAccumulated", func(t *testing.T) {
		for k, p := range r.pi.preps {
			if p.rank <= r.pi.ElectedPRepCount() {
				voted := new(big.Int).Mul(p.GetVotedValue(), big.NewInt(r.pi.GetTermPeriod()))
				assert.Equal(t, voted, p.AccumulatedVoted(), fmt.Sprintf("rank%d: %s", p.rank, common.MustNewAddress([]byte(k))))
			} else {
				assert.Equal(t, new(big.Int), p.AccumulatedVoted(), fmt.Sprintf("rank%d: %s", p.rank, common.MustNewAddress([]byte(k))))
			}
		}
	})

	// processEvents()
	enables := []struct {
		status icmodule.EnableStatus
		offset int
		target module.Address
	}{
		{
			icmodule.ESJail,
			10,
			a1,
		},
		{
			icmodule.ESUnjail,
			30,
			a1,
		},
		{
			icmodule.ESEnable,
			50,
			a4,
		},
		{
			icmodule.ESDisablePermanent,
			60,
			a5,
		},
		{
			icmodule.ESEnable,
			100,
			a3,
		},
	}
	for _, e := range enables {
		_, err = tc.AddEventEnable(e.offset, e.target, e.status)
		assert.NoError(t, err)
	}
	votes := []struct {
		vType  VoteType
		offset int
		from   module.Address
		votes  icstage.VoteList
	}{
		{
			vtBond,
			10,
			a1,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
			},
		},
		{
			vtDelegate,
			10,
			a1,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
			},
		},
		{
			vtDelegate,
			50,
			a2,
			icstage.VoteList{
				icstage.NewVote(a2, big.NewInt(1000)),
			},
		},
		{
			vtBond,
			80,
			a3,
			icstage.VoteList{
				icstage.NewVote(a3, big.NewInt(1000)),
			},
		},
		{
			vtBond,
			80,
			a4,
			icstage.VoteList{
				icstage.NewVote(a4, big.NewInt(4000)),
			},
		},
	}
	for _, v := range votes {
		if v.vType == vtBond {
			_, _, err = tc.AddEventBond(v.offset, v.from, v.votes)
		} else {
			_, _, err = tc.AddEventDelegation(v.offset, v.from, v.votes)
		}
		assert.NoError(t, err)
	}

	tc.Build()

	err = r.processEvents()
	assert.NoError(t, err)

	sExpects := []struct {
		name   string
		addr   module.Address
		status icmodule.EnableStatus
	}{
		{"Enable->Jail->Unjail", a1, icmodule.ESUnjail},
		{"Enable->", a2, icmodule.ESEnable},
		{"Unjail->Enable", a3, icmodule.ESEnable},
		{"New", a4, icmodule.ESEnable},
		{"Enable->Disable", a5, icmodule.ESDisablePermanent},
	}
	for _, e := range sExpects {
		t.Run(fmt.Sprintf("processEvent-Status:%s", e.name), func(t *testing.T) {
			key := icutils.ToKey(e.addr)
			p := r.pi.GetPRep(key)
			assert.Equal(t, e.status, p.Status())
		})
	}

	vExpects := []struct {
		addr   module.Address
		events []*VoteEvent
	}{
		{
			a1,
			[]*VoteEvent{
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(1000)),
					},
					10,
				},
				{
					vtDelegate,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(1000)),
					},
					10,
				},
			},
		},
		{
			a2,
			[]*VoteEvent{
				{
					vtDelegate,
					icstage.VoteList{
						icstage.NewVote(a2, big.NewInt(1000)),
					},
					50,
				},
			},
		},
		{
			a3,
			[]*VoteEvent{
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a3, big.NewInt(1000)),
					},
					80,
				},
			},
		},
		{
			a4,
			[]*VoteEvent{
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a4, big.NewInt(4000)),
					},
					80,
				},
			},
		},
	}
	for _, e := range vExpects {
		t.Run(fmt.Sprintf("processEvent-Voting-%s", e.addr), func(t *testing.T) {
			events := r.ve.Get(e.addr)
			assert.Equal(t, len(e.events), len(events))
			for i := 0; i < len(events); i++ {
				assert.True(t, e.events[i].Equal(events[i]))
			}
		})
	}

	// UpdateVoteInfo()
	err = r.UpdateVoteInfo()
	assert.NoError(t, err)
	t.Run("UpdateVoteInfo-PrepInfo", func(t *testing.T) {
		for _, a := range addrs {
			key := icutils.ToKey(a)
			p := r.pi.GetPRep(key)

			voted, err := tc.GetVotedFromTemp(a)
			assert.NoError(t, err)
			assert.True(t, voted.Equal(p.ToVoted()))
		}
	})

	t.Run("UpdateVoteInfo-VoteEvents", func(t *testing.T) {
		for _, a := range addrs {
			delegating := false
			bonding := false
			for _, v := range vExpects {
				if v.addr.Equal(a) {
					for _, e := range v.events {
						if e.vType == vtBond {
							bonding = true
						} else if e.vType == vtDelegate {
							delegating = true
						}
					}
				}
			}
			b, err := tc.GetBondingFromTemp(a)
			assert.NoError(t, err)
			if bonding == false {
				assert.Nil(t, b)
			} else {
				assert.NotNil(t, b)
			}
			d, err := tc.GetDelegatingFromTemp(a)
			assert.NoError(t, err)
			if delegating == false {
				assert.Nil(t, d)
			} else {
				assert.NotNil(t, d)
			}
		}
	})

	// processPrepReward()
	err = r.processPrepReward()
	assert.NoError(t, err)

	for _, p := range r.pi.rank {
		t.Run(fmt.Sprintf("processPrepReward-%s", p.Owner()), func(t *testing.T) {
			rewardable := p.IsRewardable(r.pi.ElectedPRepCount())
			if p.commissionRate == 0 {
				assert.Equal(t, 0, p.commission.Sign())
			} else {
				assert.Equal(t, rewardable, p.commission.Sign() == 1, p)
			}
			assert.Equal(t, rewardable, p.VoterReward().Sign() == 1)
			iScore, err := tc.GetIScoreFromTemp(p.Owner())
			assert.NoError(t, err)
			if rewardable && (p.commissionRate > 0 || p.bonded.Cmp(r.g.GetV3().MinBond()) >= 0) {
				assert.Equal(t, rewardable, iScore.Value().Sign() == 1)
			} else {
				assert.Nil(t, iScore)
			}
		})
	}

	// processVoterReward()
	oldIScore := make(map[string]*icreward.IScore)
	for _, a := range addrs {
		key := icutils.ToKey(a)
		iScore, err := tc.GetIScoreFromTemp(a)
		assert.NoError(t, err)
		oldIScore[key] = iScore
	}
	err = r.processVoterReward()
	assert.NoError(t, err)
	for _, a := range addrs {
		t.Run(fmt.Sprintf("processVoterReward-%s", a), func(t *testing.T) {
			key := icutils.ToKey(a)
			iScore, err := tc.GetIScoreFromTemp(a)
			assert.NoError(t, err)

			rewardable, err := tc.isVoterRewardable(a, r.pi)
			assert.NoError(t, err)

			ois := oldIScore[key]
			if rewardable {
				assert.Equal(t, 1, iScore.Value().Cmp(ois.Value()))
			} else {
				assert.Equal(t, ois, iScore)
			}
		})
	}
}
