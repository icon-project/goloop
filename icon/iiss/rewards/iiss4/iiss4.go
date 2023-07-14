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
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	rc "github.com/icon-project/goloop/icon/iiss/rewards/common"
)

type IISS4 struct {
	c  rc.Calculator
	g  icstage.Global
	pi *PRepInfo
	ve *VotingEvents
}

func NewIISS4(c rc.Calculator) (*IISS4, error) {
	global, err := c.Back().GetGlobal()
	if err != nil {
		return nil, err
	}
	return &IISS4{c: c, g: global}, nil
}

func (i *IISS4) CalculateReward() error {
	var err error

	if err = i.loadPRepInfo(); err != nil {
		return err
	}

	if err = i.processEvents(); err != nil {
		return err
	}

	if err = i.write(); err != nil {
		return err
	}

	if err = i.prepReward(); err != nil {
		return err
	}

	if err = i.voterReward(); err != nil {
		return err
	}

	return nil
}

// loadPRepInfo make new PRepInfo and load data from base.VotedV1
func (i *IISS4) loadPRepInfo() error {
	var err error
	var dsa *icreward.DSA
	base := i.c.Base()

	if dsa, err = base.GetDSA(); err != nil {
		return err
	}

	pi := NewPRepInfo(i.g.GetBondRequirement(), i.g.GetElectedPRepCount(), i.g.GetOffsetLimit())

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

	i.pi = pi

	return nil
}

func (i *IISS4) processEvents() error {
	ve := NewVotingEvents()
	back := i.c.Back()
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
			i.pi.SetStatus(obj.Target(), obj.Status())
		case icstage.TypeEventDelegation, icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vType := vtDelegate
			if type_ == icstage.TypeEventBond {
				vType = vtBond
			}
			i.pi.ApplyVote(vType, obj.Votes(), i.pi.OffsetLimit()-keyOffset)
			ve.AddEvent(vType, obj.From(), obj.Votes(), keyOffset)
		case icstage.TypeEventCommissionRate:
			obj := icstage.ToEventCommissionRate(o)
			i.pi.SetCommissionRate(obj.Target(), obj.Value())
		}
	}
	i.pi.UpdateAccumulatedPower()
	i.ve = ve
	return nil
}

// write set Voted, Delegating and Bonding to temp
func (i *IISS4) write() error {
	base := i.c.Base()
	temp := i.c.Temp()
	if err := i.pi.Write(temp); err != nil {
		return err
	}
	if err := i.ve.Write(base, temp); err != nil {
		return err
	}
	return nil
}

func (i *IISS4) prepReward() error {
	global := i.g.GetV3()
	return i.pi.DistributeReward(
		new(big.Int).Mul(global.GetIGlobal(), global.GetIPRep()),
		new(big.Int).Mul(global.GetIGlobal(), global.GetIWage()),
		global.MinBond(),
		i.c,
	)
}

func (i *IISS4) voterReward() error {
	base := i.c.Base()

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
		voter.AddVoting(icreward.ToDelegating(o), i.pi.OffsetLimit())

		b, err := base.GetBonding(addr)
		if err != nil {
			return err
		}
		if b != nil && b.IsEmpty() == false {
			voter.AddVoting(b, i.pi.OffsetLimit())
		}

		events := i.ve.Get(addr)
		if events != nil {
			for _, event := range events {
				voter.AddEvent(event, i.pi.OffsetLimit()-event.Offset())
			}
			i.ve.SetCalculated(addr)
		}

		reward := voter.CalculateReward(i.pi)
		if err = i.c.UpdateIScore(voter.Owner(), reward, rc.RTVoter); err != nil {
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
		voter.AddVoting(icreward.ToBonding(o), i.pi.OffsetLimit())

		events := i.ve.Get(addr)
		if events != nil {
			for _, event := range events {
				voter.AddEvent(event, i.pi.OffsetLimit()-event.Offset())
			}
			i.ve.SetCalculated(addr)
		}

		reward := voter.CalculateReward(i.pi)
		if err = i.c.UpdateIScore(voter.Owner(), reward, rc.RTVoter); err != nil {
			return err
		}
	}

	for key, events := range i.ve.Events() {
		if i.ve.IsCalculated(key) {
			continue
		}
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		voter := NewVoter(addr)
		for _, event := range events {
			voter.AddEvent(event, i.pi.OffsetLimit()-event.Offset())
		}
		i.ve.SetCalculated(addr)

		reward := voter.CalculateReward(i.pi)
		if err = i.c.UpdateIScore(voter.Owner(), reward, rc.RTVoter); err != nil {
			return err
		}
	}

	return nil
}
