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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func TestVotingEvents(t *testing.T) {
	ve := NewVotingEvents()

	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")
	a3, _ := common.NewAddressFromString("hx3")

	events := []struct {
		vType  VoteType
		from   module.Address
		votes  icstage.VoteList
		offset int
	}{
		{
			vtDelegate,
			a1,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
			},
			10,
		},
		{
			vtDelegate,
			a1,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(-2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
			},
			20,
		},
		{
			vtBond,
			a1,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
			},
			30,
		},
		{
			vtBond,
			a2,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
			},
			30,
		},
		{
			vtDelegate,
			a3,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(1000)),
				icstage.NewVote(a2, big.NewInt(2000)),
				icstage.NewVote(a3, big.NewInt(3000)),
			},
			30,
		},
	}

	// Get(), AddEvent()
	for _, e := range events {
		es := ve.Get(e.from)
		prevLen := len(es)
		ve.AddEvent(e.vType, e.from, e.votes, e.offset)
		es = ve.Get(e.from)
		assert.Equal(t, prevLen+1, len(es))
		event := es[len(es)-1]
		assert.Equal(t, e.vType, event.Type())
		assert.True(t, event.Votes().Equal(e.votes))
		assert.Equal(t, e.offset, event.Offset())
	}
	// Events()
	assert.Equal(t, 3, len(ve.Events()))

	// SetCalculated(), IsCalculated()
	key := icutils.ToKey(a1)
	assert.False(t, ve.IsCalculated(key))
	ve.SetCalculated(a1)
	assert.True(t, ve.IsCalculated(key))
}

type testRW struct {
	voted      map[string]*icreward.Voted
	bonding    map[string]*icreward.Bonding
	delegating map[string]*icreward.Delegating
}

func (t *testRW) GetDelegating(addr module.Address) (*icreward.Delegating, error) {
	if d, ok := t.delegating[icutils.ToKey(addr)]; ok {
		return d, nil
	}
	return nil, nil
}

func (t *testRW) GetBonding(addr module.Address) (*icreward.Bonding, error) {
	if b, ok := t.bonding[icutils.ToKey(addr)]; ok {
		return b, nil
	}
	return nil, nil
}

func (t *testRW) GetVoted(addr module.Address) (*icreward.Voted, error) {
	if v, ok := t.voted[icutils.ToKey(addr)]; ok {
		return v, nil
	}
	return nil, nil
}

func (t *testRW) SetVoted(addr module.Address, voted *icreward.Voted) error {
	key := icutils.ToKey(addr)
	if voted.IsEmpty() {
		delete(t.voted, key)
	} else {
		t.voted[key] = voted
	}
	return nil
}

func (t *testRW) SetDelegating(addr module.Address, delegating *icreward.Delegating) error {
	key := icutils.ToKey(addr)
	if delegating.IsEmpty() {
		delete(t.delegating, key)
	} else {
		t.delegating[key] = delegating
	}
	return nil
}

func (t *testRW) SetBonding(addr module.Address, bonding *icreward.Bonding) error {
	key := icutils.ToKey(addr)
	if bonding.IsEmpty() {
		delete(t.bonding, key)
	} else {
		t.bonding[key] = bonding
	}
	return nil
}

func TestVotingEvents_Write(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")
	a3, _ := common.NewAddressFromString("hx3")

	rw := &testRW{
		bonding: map[string]*icreward.Bonding{
			icutils.ToKey(a2): {
				Bonds: icstate.Bonds{
					icstate.NewBond(a1, big.NewInt(1000)),
				},
			},
			icutils.ToKey(a3): {
				Bonds: icstate.Bonds{
					icstate.NewBond(a1, big.NewInt(2000)),
				},
			},
		},
		delegating: map[string]*icreward.Delegating{
			icutils.ToKey(a2): {
				Delegations: icstate.Delegations{
					icstate.NewDelegation(a1, big.NewInt(1000)),
					icstate.NewDelegation(a2, big.NewInt(2000)),
				},
			},
			icutils.ToKey(a3): {
				Delegations: icstate.Delegations{
					icstate.NewDelegation(a1, big.NewInt(1000)),
					icstate.NewDelegation(a2, big.NewInt(2000)),
				},
			},
		},
	}

	ve := &VotingEvents{
		events: map[string][]*VoteEvent{
			icutils.ToKey(a1): {
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
						icstage.NewVote(a1, big.NewInt(1000)),
					},
					20,
				},
				{
					vtDelegate,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(-10)),
						icstage.NewVote(a2, big.NewInt(2000)),
						icstage.NewVote(a3, big.NewInt(-3000)),
					},
					30,
				},
			},
			icutils.ToKey(a2): {
				{
					vtDelegate,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(-1000)),
						icstage.NewVote(a2, big.NewInt(2000)),
						icstage.NewVote(a3, big.NewInt(3000)),
					},
					10,
				},
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(-1000)),
					},
					20,
				},
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(1000)),
					},
					30,
				},
			},
			icutils.ToKey(a3): {
				{
					vtBond,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(-2000)),
					},
					10,
				},
				{
					vtDelegate,
					icstage.VoteList{
						icstage.NewVote(a1, big.NewInt(-1000)),
						icstage.NewVote(a2, big.NewInt(-2000)),
					},
					30,
				},
			},
		},
		calculated: nil,
	}

	err := ve.Write(rw, rw)
	assert.NoError(t, err)

	expects := []struct {
		name       string
		owner      module.Address
		delegating *icreward.Delegating
		bonding    *icreward.Bonding
	}{
		{
			"New entry",
			a1,
			&icreward.Delegating{
				Delegations: icstate.Delegations{
					icstate.NewDelegation(a1, big.NewInt(990)),
					icstate.NewDelegation(a2, big.NewInt(4000)),
				}},
			&icreward.Bonding{
				Bonds: icstate.Bonds{
					icstate.NewBond(a1, big.NewInt(1000)),
				}},
		},
		{
			"Update",
			a2,
			&icreward.Delegating{
				Delegations: icstate.Delegations{
					icstate.NewDelegation(a2, big.NewInt(4000)),
					icstate.NewDelegation(a3, big.NewInt(3000)),
				}},
			&icreward.Bonding{
				Bonds: icstate.Bonds{
					icstate.NewBond(a1, big.NewInt(1000)),
				}},
		},
		{
			"Delete",
			a3,
			nil,
			nil,
		},
	}

	for _, e := range expects {
		t.Run(e.name, func(t *testing.T) {
			b, err := rw.GetBonding(e.owner)
			assert.NoError(t, err)
			if e.bonding != nil {
				assert.True(t, b.Equal(e.bonding))
			} else {
				assert.Nil(t, b)
			}

			d, err := rw.GetDelegating(e.owner)
			assert.NoError(t, err)
			if e.delegating != nil {
				assert.True(t, d.Equal(e.delegating))
			} else {
				assert.Nil(t, d)
			}
		})
	}
}

func TestVoter(t *testing.T) {
	a1, _ := common.NewAddressFromString("hx1")
	a2, _ := common.NewAddressFromString("hx2")
	a3, _ := common.NewAddressFromString("hx3")
	a4, _ := common.NewAddressFromString("hx4")

	preps := []struct {
		owner       module.Address
		status      icmodule.EnableStatus
		accVoted    int64
		accPower    int64
		voterReward int64
	}{
		{a1, icmodule.ESEnable, 100_000, 100_000, 1_000_000},
		{a2, icmodule.ESJail, 200_000, 200_000, 0},
		{a3, icmodule.ESUnjail, 300_000, 300_000, 0},
		{a4, icmodule.ESDisablePermanent, 400_000, 400_000, 0},
	}
	pInfo := NewPRepInfo(5, 3, 100, log.New())
	for _, p := range preps {
		k := icutils.ToKey(p.owner)
		np := NewPRep(p.owner, p.status, new(big.Int), new(big.Int), 0, true)
		np.accumulatedVoted = big.NewInt(p.accVoted)
		np.accumulatedPower = big.NewInt(p.accPower)
		np.SetVoterReward(big.NewInt(p.voterReward))
		pInfo.preps[k] = np
	}

	voter := NewVoter(a1, log.New())
	assert.Equal(t, a1, voter.Owner())

	// AddVoting()
	votings := []icreward.Voting{
		&icreward.Bonding{
			Bonds: icstate.Bonds{
				icstate.NewBond(a1, big.NewInt(100)),
				icstate.NewBond(a2, big.NewInt(200)),
				icstate.NewBond(a3, big.NewInt(300)),
				icstate.NewBond(a4, big.NewInt(400)),
			}},
		&icreward.Delegating{
			Delegations: icstate.Delegations{
				icstate.NewDelegation(a1, big.NewInt(10)),
				icstate.NewDelegation(a2, big.NewInt(20)),
				icstate.NewDelegation(a3, big.NewInt(30)),
				icstate.NewDelegation(a4, big.NewInt(40)),
			}},
	}
	expectVotes := map[string]*big.Int{
		icutils.ToKey(a1): big.NewInt(110 * pInfo.GetTermPeriod()),
		icutils.ToKey(a2): big.NewInt(220 * pInfo.GetTermPeriod()),
		icutils.ToKey(a3): big.NewInt(330 * pInfo.GetTermPeriod()),
		icutils.ToKey(a4): big.NewInt(440 * pInfo.GetTermPeriod()),
	}
	for _, voting := range votings {
		voter.AddVoting(voting, pInfo.GetTermPeriod())
	}
	for key, amount := range voter.accumulatedVotes {
		v, _ := expectVotes[key]
		assert.Equal(t, v, amount, common.MustNewAddress([]byte(key)))
	}

	// AddEvent()
	events := []*VoteEvent{
		{
			vtBond,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(10)),
				icstage.NewVote(a2, big.NewInt(20)),
				icstage.NewVote(a3, big.NewInt(30)),
			},
			10,
		},
		{
			vtDelegate,
			icstage.VoteList{
				icstage.NewVote(a1, big.NewInt(-10)),
				icstage.NewVote(a2, big.NewInt(-20)),
				icstage.NewVote(a3, big.NewInt(-30)),
			},
			50,
		},
	}
	expectVotes = map[string]*big.Int{
		icutils.ToKey(a1): big.NewInt(
			110*pInfo.GetTermPeriod() + int64(10*(pInfo.OffsetLimit()-10)+-10*(pInfo.OffsetLimit()-50)),
		),
		icutils.ToKey(a2): big.NewInt(
			220*pInfo.GetTermPeriod() + int64(20*(pInfo.OffsetLimit()-10)+-20*(pInfo.OffsetLimit()-50)),
		),
		icutils.ToKey(a3): big.NewInt(
			330*pInfo.GetTermPeriod() + int64(30*(pInfo.OffsetLimit()-10)+-30*(pInfo.OffsetLimit()-50)),
		),
		icutils.ToKey(a4): big.NewInt(440 * pInfo.GetTermPeriod()),
	}
	for _, event := range events {
		voter.AddEvent(event, pInfo.OffsetLimit()-event.Offset())
	}
	for key, amount := range voter.accumulatedVotes {
		v, _ := expectVotes[key]
		assert.Equal(t, v, amount, common.MustNewAddress([]byte(key)))
	}

	// CalculateReward
	key := icutils.ToKey(a1)
	prep1 := pInfo.GetPRep(key)
	expectReward := big.NewInt(prep1.VoterReward().Int64() * voter.accumulatedVotes[key].Int64() / prep1.AccumulatedVoted().Int64())
	r := voter.CalculateReward(pInfo)
	assert.Equal(t, expectReward, r)
}
