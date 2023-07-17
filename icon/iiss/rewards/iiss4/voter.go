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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	rc "github.com/icon-project/goloop/icon/iiss/rewards/common"
	"github.com/icon-project/goloop/module"
)

type VoteType int

const (
	vtBond VoteType = iota + 1
	vtDelegate
)

type VoteEvent struct {
	vType  VoteType
	votes  icstage.VoteList
	offset int
}

func (v *VoteEvent) Type() VoteType {
	return v.vType
}

func (v *VoteEvent) Votes() icstage.VoteList {
	return v.votes
}

func (v *VoteEvent) Offset() int {
	return v.offset
}

func NewVoteEvent(vType VoteType, votes icstage.VoteList, offset int) *VoteEvent {
	return &VoteEvent{
		vType:  vType,
		votes:  votes,
		offset: offset,
	}
}

type VotingEvents struct {
	events     map[string][]*VoteEvent
	calculated map[string]struct{}
}

func (v *VotingEvents) Events() map[string][]*VoteEvent {
	return v.events
}

func (v *VotingEvents) Get(from module.Address) []*VoteEvent {
	events, _ := v.events[icutils.ToKey(from)]
	return events
}

func (v *VotingEvents) SetCalculated(from module.Address) {
	v.calculated[icutils.ToKey(from)] = struct{}{}
}

func (v *VotingEvents) IsCalculated(key string) bool {
	_, ok := v.calculated[key]
	return ok
}

func (v *VotingEvents) AddEvent(vType VoteType, from module.Address, votes icstage.VoteList, offset int) {
	key := icutils.ToKey(from)
	if _, ok := v.events[key]; !ok {
		v.events[key] = make([]*VoteEvent, 0)
	}
	v.events[key] = append(v.events[key], NewVoteEvent(vType, votes, offset))
}

// Write writes updated Bonding and Delegating to database
func (v *VotingEvents) Write(reader rc.Reader, writer rc.Writer) error {
	for key, events := range v.events {
		from, err := common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		d, err := reader.GetDelegating(from)
		if err != nil {
			return err
		}
		if d == nil {
			d = icreward.NewDelegating()
		}
		b, err := reader.GetBonding(from)
		if err != nil {
			return err
		}
		if b == nil {
			b = icreward.NewBonding()
		}

		// update with events
		for _, event := range events {
			switch event.Type() {
			case vtBond:
				if err = b.ApplyVotes(event.Votes()); err != nil {
					return err
				}
			case vtDelegate:
				if err = d.ApplyVotes(event.Votes()); err != nil {
					return err
				}
			}
		}

		// write final value
		err = writer.SetBonding(from, b)
		if err != nil {
			return err
		}
		err = writer.SetDelegating(from, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewVotingEvents() *VotingEvents {
	return &VotingEvents{
		events:     make(map[string][]*VoteEvent),
		calculated: make(map[string]struct{}),
	}
}

type Voter struct {
	owner module.Address
	votes map[string]*big.Int
}

func (v *Voter) Owner() module.Address {
	return v.owner
}

func (v *Voter) addVoting(voting icstate.Voting, period *big.Int) {
	key := icutils.ToKey(voting.To())
	amount := new(big.Int).Mul(voting.Amount(), period)
	if value, ok := v.votes[key]; ok {
		value.Add(value, amount)
		v.votes[key] = value
	} else {
		v.votes[key] = amount
	}
}

func (v *Voter) AddVoting(voting icreward.Voting, period int) {
	pr := big.NewInt(int64(period))
	iter := voting.Iterator()
	for ; iter.Has(); iter.Next() {
		if vote, err := iter.Get(); err != nil {
			continue
		} else {
			v.addVoting(vote, pr)
		}
	}
}

func (v *Voter) AddEvent(event *VoteEvent, period int) {
	pr := big.NewInt(int64(period))
	for _, vote := range event.Votes() {
		v.addVoting(vote, pr)
	}
}

func (v *Voter) CalculateReward(pInfo *PRepInfo) *big.Int {
	iScore := new(big.Int)

	for k, vote := range v.votes {
		prep := pInfo.GetPRep(k)
		if prep != nil && prep.Rewardable(pInfo.ElectedPRepCount()) {
			reward := new(big.Int).Mul(vote, prep.VoterReward())
			reward.Div(reward, prep.AccumulatedVoted())
			iScore.Add(iScore, reward)
		}
	}

	return iScore
}

func NewVoter(owner module.Address) *Voter {
	return &Voter{
		owner: owner,
		votes: make(map[string]*big.Int),
	}
}
