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
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	rc "github.com/icon-project/goloop/icon/iiss/rewards/common"
	"github.com/icon-project/goloop/module"
)

type reward struct {
	c  rc.Calculator
	g  icstage.Global
	pi *PRepInfo
	ve *VotingEvents
}

func NewReward(c rc.Calculator) (rc.Reward, error) {
	global, err := c.Back().GetGlobal()
	if err != nil {
		return nil, err
	}
	return &reward{c: c, g: global}, nil
}

func (r *reward) Global() icstage.Global {
	return r.g
}

func (r *reward) Calculate() error {
	var err error

	if err = r.loadPRepInfo(); err != nil {
		return err
	}

	if err = r.processEvents(); err != nil {
		return err
	}

	if err = r.write(); err != nil {
		return err
	}

	if err = r.prepReward(); err != nil {
		return err
	}

	if err = r.voterReward(); err != nil {
		return err
	}

	return nil
}

// loadPRepInfo make new PRepInfo and load data from base.VotedV1
func (r *reward) loadPRepInfo() error {
	var err error
	var dsa *icreward.DSA
	base := r.c.Base()

	if dsa, err = base.GetDSA(); err != nil {
		return err
	}

	pi := NewPRepInfo(r.g.GetBondRequirement(), r.g.GetElectedPRepCount(), r.g.GetOffsetLimit())

	prefix := icreward.VotedKey.Build()
	for iter := base.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		addr, err := common.NewAddress(keySplit[1])
		if err != nil {
			return err
		}
		voted := icreward.ToVoted(o)
		pubKey, err := base.GetPublicKey(addr)
		if err != nil {
			return err
		}
		pi.Add(addr, voted.Status(), voted.Delegated(), voted.Bonded(), voted.CommissionRate(), pubKey.HasAll(dsa.Mask()))
	}
	pi.Sort()
	pi.InitAccumulated()

	r.pi = pi

	return nil
}

func (r *reward) processEvents() error {
	ve := NewVotingEvents()
	back := r.c.Back()
	eventPrefix := icstage.EventKey.Build()
	for iter := back.Filter(eventPrefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		type_ := o.(*icobject.Object).Tag().Type()
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		keyOffset := int(intconv.BytesToInt64(keySplit[1]))
		switch type_ {
		case icstage.TypeEventEnable:
			obj := icstage.ToEventEnable(o)
			r.pi.SetStatus(obj.Target(), obj.Status())
		case icstage.TypeEventDelegation, icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vType := vtDelegate
			if type_ == icstage.TypeEventBond {
				vType = vtBond
			}
			r.pi.ApplyVote(vType, obj.Votes(), keyOffset)
			ve.AddEvent(vType, obj.From(), obj.Votes(), keyOffset)
		}
	}
	r.pi.UpdateAccumulatedPower()
	r.ve = ve
	return nil
}

func (r *reward) UpdateIScore(addr module.Address, amount *big.Int, t rc.RewardType) error {
	temp := r.c.Temp()
	iScore, err := temp.GetIScore(addr)
	if err != nil {
		return err
	}
	nIScore := iScore.Added(amount)
	if err = temp.SetIScore(addr, nIScore); err != nil {
		return err
	}

	stats := r.c.Stats()
	switch t {
	case rc.RTPRep:
		stats.IncreaseVoted(amount)
	case rc.RTVoter:
		stats.IncreaseVoting(amount)
	default:
		return errors.IllegalArgumentError.Errorf("wrong RewardType %d", t)
	}
	return nil
}

// write writes Voted, Delegating and Bonding to temp
func (r *reward) write() error {
	base := r.c.Base()
	temp := r.c.Temp()
	if err := r.pi.Write(temp); err != nil {
		return err
	}
	if err := r.ve.Write(base, temp); err != nil {
		return err
	}
	return nil
}

func (r *reward) prepReward() error {
	global := r.g.GetV3()
	return r.pi.DistributeReward(
		global.GetRewardFundAmountByKey(icstate.KeyIprep),
		global.GetRewardFundAmountByKey(icstate.KeyIwage),
		global.MinBond(),
		r,
	)
}

func (r *reward) voterReward() error {
	base := r.c.Base()

	prefix := icreward.DelegatingKey.Build()
	for iter := base.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		var keySplit [][]byte
		keySplit, err = containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		var addr *common.Address
		addr, err = common.NewAddress(keySplit[1])
		if err != nil {
			return err
		}
		voter := NewVoter(addr)
		voter.AddVoting(icreward.ToDelegating(o), r.pi.GetTermPeriod())

		b, err := base.GetBonding(addr)
		if err != nil {
			return err
		}
		if b != nil && b.IsEmpty() == false {
			voter.AddVoting(b, r.pi.GetTermPeriod())
		}

		events := r.ve.Get(addr)
		if events != nil {
			for _, event := range events {
				voter.AddEvent(event, r.pi.OffsetLimit()-event.Offset())
			}
			r.ve.SetCalculated(addr)
		}

		amount := voter.CalculateReward(r.pi)
		if err = r.UpdateIScore(voter.Owner(), amount, rc.RTVoter); err != nil {
			return err
		}
	}

	prefix = icreward.BondingKey.Build()
	for iter := base.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		var keySplit [][]byte
		keySplit, err = containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		var addr *common.Address
		addr, err = common.NewAddress(keySplit[1])
		if err != nil {
			return err
		}

		d, err := base.GetDelegating(addr)
		if err != nil {
			return err
		}
		if d != nil && !d.IsEmpty() {
			continue
		}

		voter := NewVoter(addr)
		voter.AddVoting(icreward.ToBonding(o), r.pi.GetTermPeriod())

		events := r.ve.Get(addr)
		if events != nil {
			for _, event := range events {
				voter.AddEvent(event, r.pi.OffsetLimit()-event.Offset())
			}
			r.ve.SetCalculated(addr)
		}

		reward := voter.CalculateReward(r.pi)
		if err = r.UpdateIScore(voter.Owner(), reward, rc.RTVoter); err != nil {
			return err
		}
	}

	for key, events := range r.ve.Events() {
		if r.ve.IsCalculated(key) {
			continue
		}
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		voter := NewVoter(addr)
		for _, event := range events {
			voter.AddEvent(event, r.pi.OffsetLimit()-event.Offset())
		}
		r.ve.SetCalculated(addr)

		reward := voter.CalculateReward(r.pi)
		if err = r.UpdateIScore(voter.Owner(), reward, rc.RTVoter); err != nil {
			return err
		}
	}

	return nil
}
