//go:build exclude

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

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type prep struct {
	voted      map[string]*icreward.Voted
	totalPower *big.Int
}

func (p *prep) ApplyVoting(_type int, voting icreward.Voting) {
	iter := voting.Iterator()
	for ; iter.Has(); iter.Next() {
		if vote, err := iter.Get(); err != nil {
			p.updateAmount(_type, vote.To(), vote.Amount())
		}
	}
}

func (p *prep) updateAmount(_type int, address module.Address, amount *big.Int) {
	var nv *icreward.Voted
	var ok bool

	key := icutils.ToKey(address)
	if nv, ok = p.voted[key]; !ok {
		nv = icreward.NewVoted()
	}

	switch _type {
	case icreward.TypeDelegating:
		nv.AddDelegated(amount)
	case icreward.TypeBonding:
		nv.AddBonded(amount)
	}
}

func (p *prep) UpdatePower(br int) {
	for _, v := range p.voted {
		v.UpdateBondedDelegation(br)
		p.totalPower = new(big.Int).Add(p.totalPower, v.BondedDelegation())
	}
}

func (p *prep) SetEnable(address common.Address, enable bool) error {
	key := icutils.ToKey(address)
	if value, ok := p.voted[key]; !ok {
		return errors.NotFoundError.Errorf("there is no Voted info for %s", address)
	} else {
		value.SetEnable(enable)
	}
	return nil
}

func getVotePrefix(_type int) ([]byte, error) {
	switch _type {
	case icreward.TypeBonding:
		return icreward.BondingKey.Build(), nil
	case icreward.TypeDelegating:
		return icreward.DelegatingKey.Build(), nil
	}
	return nil, errors.IllegalArgumentError.Errorf("illegal vote type %d", _type)
}

func (c *calculator.Calculator) phase1(p *prep) error {
	types := []int{icreward.TypeDelegating, icreward.TypeBonding}

	// update with base
	for _, _type := range types {
		prefix, err := getVotePrefix(_type)
		if err != nil {
			return err
		}
		for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
			o, _, err := iter.Get()
			if err != nil {
				return err
			}
			voting := toVoting(_type, o)
			if voting == nil {
				c.log.Errorf("Failed to convert data to voting instance")
				continue
			}
			p.ApplyVoting(_type, voting)
			// TODO update each account
		}
	}

	// update with events
	eventPrefix := icstage.EventKey.Build()
	for iter := c.back.Filter(eventPrefix); iter.Has(); iter.Next() {
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
			if obj.Status().IsEnabled() == false && vInfo.IsElectedPRep(obj.Target()) {
				c.log.Tracef("Calculate voted reward with %+v", obj)
				vInfo.CalculateReward(multiplier, divider, keyOffset-from)
				from = keyOffset
				vInfo.SetEnable(obj.Target(), obj.Status())
				// If revision < 7, do not update totalBondedDelegation with temporarily disabled P-Rep
				if c.global.GetRevision() >= icmodule.RevisionFixTotalDelegated || !obj.Status().IsDisabledTemporarily() {
					vInfo.UpdateTotalBondedDelegation()
				}
			} else {
				vInfo.SetEnable(obj.Target(), obj.Status())
				// do not update total bonded delegation when P-Rep is activated
			}
		case icstage.TypeEventDelegation:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateDelegated(obj.Votes())
		case icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateBonded(obj.Votes())
		default:
			// skip old events
		}
	}

	p.UpdatePower(c.global.GetBondRequirement())
	return nil
}

func (c *calculator.Calculator) phase2() error {
	return nil
}
