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
	"bytes"
	"math/big"
	"sort"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func (c *Calculator) calculateRewardV3() (err error) {
	startTS := time.Now()
	if err = c.calculateBlockProduce(); err != nil {
		err = icmodule.CalculationFailedError.Wrapf(err, "Failed to calculate block produce reward")
		return
	}
	bpTS := time.Now()

	if err = c.calculateVotedReward(); err != nil {
		err = icmodule.CalculationFailedError.Wrapf(err, "Failed to calculate P-Rep voted reward")
		return
	}
	votedTS := time.Now()

	if err = c.calculateVotingReward(); err != nil {
		err = icmodule.CalculationFailedError.Wrapf(err, "Failed to calculate ICONist voting reward")
		return
	}
	finalTS := time.Now()

	c.log.Infof("Calculation time: total=%s blockProduce=%s voted=%s voting=%s",
		finalTS.Sub(startTS), bpTS.Sub(startTS), votedTS.Sub(bpTS), finalTS.Sub(votedTS),
	)
	return nil
}

func (c *Calculator) replayBugDisabledPRep() error {
	revision := c.global.GetRevision()
	if c.global.GetIISSVersion() != icstate.IISSVersion2 ||
		revision < icmodule.RevisionDecentralize || revision >= icmodule.RevisionFixBugDisabledPRep {
		return nil
	}
	for iter := c.base.Filter(icreward.BugDisabledPRepKey.Build()); iter.Has(); iter.Next() {
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
		obj := icreward.ToBugDisabledPRep(o)
		if err = c.UpdateIScore(addr, obj.Value(), TypeVoting); err != nil {
			return err
		}
		if err = c.temp.DeleteBugDisabledPRep(addr); err != nil {
			return err
		}
	}
	return nil
}

// varForBlockProduceReward return variable for block produce reward
// return (((irep * MonthPerYear) / (YearBlock * 2)) * mainPRepCount * IScoreICXRatio) / 2
func varForBlockProduceReward(irep *big.Int, mainPRepCount int) *big.Int {
	v := new(big.Int)
	v.Mul(irep, big.NewInt(icmodule.MonthPerYear))
	v.Div(v, big.NewInt(int64(icmodule.YearBlock*2)))
	v.Mul(v, big.NewInt(int64(mainPRepCount)*icmodule.IScoreICXRatio))
	v.Div(v, big.NewInt(int64(2)))
	return v
}

func (c *Calculator) calculateBlockProduce() error {
	if c.global.GetIISSVersion() == icstate.IISSVersion3 {
		return nil
	}
	var err error
	var validators []*validator
	global := c.global.GetV1()
	variable := varForBlockProduceReward(global.GetIRep(), global.GetMainRepCount())
	validators, err = c.loadValidators()
	if err != nil {
		return err
	}

	prefix := icstage.BlockProduceKey.Build()
	for iter := c.back.Filter(prefix); iter.Has(); iter.Next() {
		var obj trie.Object
		obj, _, err = iter.Get()
		if err != nil {
			return err
		}
		bp := icstage.ToBlockProduce(obj)
		if err = processBlockProduce(bp, variable, validators); err != nil {
			return err
		}
	}

	for _, v := range validators {
		if err = c.UpdateIScore(v.Address(), v.IScore(), TypeBlockProduce); err != nil {
			return err
		}
	}

	return nil
}

func (c *Calculator) loadValidators() ([]*validator, error) {
	vl, err := c.back.GetValidators()
	if err != nil {
		return nil, err
	}
	vs := make([]*validator, len(vl))
	for i, a := range vl {
		vs[i] = newValidator(common.AddressToPtr(a))
	}
	return vs, nil
}

// processBlockProduce calculate blockProduce reward with Block Produce Info.
// reward for proposer per block = irep * mainPRepCount / (2 * 2 * MonthBlock)
// reward for validator per block = irep * mainPRepCount / (2 * 2 * MonthBlock * validatorCount)
// variable = irep * mainPRepCount / (2 * 2 * MonthBlock)
func processBlockProduce(bp *icstage.BlockProduce, variable *big.Int, validators []*validator) error {
	if variable.Sign() == 0 {
		return nil
	}

	vLen := len(validators)
	pIndex := bp.ProposerIndex()
	vCount := bp.VoteCount()
	vMask := bp.VoteMask()
	maxIndex := vMask.BitLen()
	if pIndex >= vLen || maxIndex > vLen {
		return errors.Errorf("Can't find validator with %+v", bp)
	}
	// ICON1 did not give validate reward to proposer
	if vMask.Bit(pIndex) == uint(1) {
		vCount -= 1
	}
	beta1Reward := new(big.Int).Set(variable)

	// for proposer
	proposer := validators[pIndex]
	proposer.SetIScore(new(big.Int).Add(proposer.IScore(), beta1Reward))

	// for validator
	if vCount > 0 {
		beta1Validate := new(big.Int).Div(beta1Reward, big.NewInt(int64(vCount)))
		for i := 0; i < maxIndex; i += 1 {
			if pIndex != i && vMask.Bit(i) == uint(1) {
				validators[i].SetIScore(new(big.Int).Add(validators[i].IScore(), beta1Validate))
			}
		}
	}

	return nil
}

// varForVotedReward return variable for P-Rep voted reward
// IISS 2.0
//
//	multiplier = (((irep * MonthPerYear) / (YearBlock * 2)) * 100 * IScoreICXRatio) / 2
//	divider = 1
//
// IISS 3.1
//
//	multiplier = iglobal * iprep * IScoreICXRatio
//	divider = 100 * MonthBlock
func varForVotedReward(global icstage.Global) (multiplier, divider *big.Int) {
	multiplier = new(big.Int)
	divider = new(big.Int).SetInt64(1)

	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion2 {
		g := global.GetV1()
		multiplier.Mul(g.GetIRep(), big.NewInt(icmodule.MonthPerYear))
		multiplier.Div(multiplier, big.NewInt(int64(icmodule.YearBlock*2)))
		multiplier.Mul(multiplier, big.NewInt(int64(icmodule.VotedRewardMultiplier*icmodule.IScoreICXRatio)))
	} else {
		g := global.GetV2()
		if g.GetTermPeriod() == 0 {
			return
		}
		multiplier.Mul(g.GetIGlobal(), g.GetIPRep())
		multiplier.Mul(multiplier, icmodule.BigIntIScoreICXRatio)
		divider.SetInt64(int64(100 * icmodule.MonthBlock))
	}
	return
}

func (c *Calculator) calculateVotedReward() error {
	// Calculate reward with a new configuration from next block
	from := -1
	multiplier, divider := varForVotedReward(c.global)
	vInfo, err := c.loadVotedInfo()
	if err != nil {
		return err
	}

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
		case icstage.TypeEventDelegation, icstage.TypeEventDelegated:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateDelegated(obj.Votes())
		case icstage.TypeEventDelegationV2:
			obj := icstage.ToEventDelegationV2(o)
			vInfo.UpdateDelegated(obj.Delegated())
		case icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateBonded(obj.Votes())
		case icstage.TypeEventVotedReward:
			vInfo.CalculateReward(multiplier, divider, keyOffset-from)
			from = keyOffset
		case icstage.TypeEventCommissionRate:
			obj := icstage.ToEventCommissionRate(o)
			vInfo.SetCommissionRate(obj.Target(), big.NewInt(int64(obj.Value())))
		}
	}
	if from < c.global.GetOffsetLimit() {
		vInfo.CalculateReward(multiplier, divider, c.global.GetOffsetLimit()-from)
	}

	// write result to temp and update statistics
	for key, prep := range vInfo.PReps() {
		var addr *common.Address
		addr, err = common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		prep.UpdateToWrite()
		if err = c.temp.SetVoted(addr, prep.Voted()); err != nil {
			return err
		}

		if prep.IScore().Sign() == 0 {
			continue
		}

		if err = c.UpdateIScore(addr, prep.IScore(), TypeVoted); err != nil {
			return err
		}
	}
	return nil
}

func (c *Calculator) loadVotedInfo() (*votedInfo, error) {
	electedPRepCount := c.global.GetElectedPRepCount()
	bondRequirement := c.global.GetBondRequirement()
	vInfo := newVotedInfo(electedPRepCount)

	var dsa *icreward.DSA
	var err error
	if dsa, err = c.base.GetDSA(); err != nil {
		return nil, err
	}

	prefix := icreward.VotedKey.Build()
	for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return nil, err
		}
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return nil, err
		}
		addr, err := common.NewAddress(keySplit[1])
		if err != nil {
			return nil, err
		}
		obj := icreward.ToVoted(o)
		data := newVotedData(obj.Clone()) // Clone Voted instance as we will modify it later
		data.UpdateBondedDelegation(bondRequirement)
		pubKey, err := c.base.GetPublicKey(addr)
		if err != nil {
			return nil, err
		}
		// if dsa is not set, all data.Pubkey() will be true
		data.SetPubKey(pubKey.HasAll(dsa.Mask()))
		vInfo.AddVotedData(addr, data)
	}
	vInfo.Sort()
	vInfo.UpdateTotalBondedDelegation()

	return vInfo, nil
}

// loadPRepInfo load P-Rep status from base
func (c *Calculator) loadPRepInfo() (map[string]*pRepEnable, error) {
	prepInfo := make(map[string]*pRepEnable)
	for iter := c.base.Filter(icreward.VotedKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return nil, err
		}
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return nil, err
		}
		addr, err := common.NewAddress(keySplit[1])
		if err != nil {
			return nil, err
		}
		obj := icreward.ToVoted(o)
		if obj.Enable() == false {
			// do not collect disabled P-Rep
			continue
		}
		prepInfo[string(addr.Bytes())] = new(pRepEnable)
	}

	return prepInfo, nil
}

// varForPRepDelegatingReward return variables for ICONist delegating reward
// IISS 2.0
//
//	multiplier = Rrep * IScoreICXRatio
//	divider = YearBlock * RrepDivider
//
// IISS 3.1
//
//	multiplier = Iglobal * Ivoter * IScoreICXRatio
//	divider = 100 * MonthBlock * total voting amount
func varForVotingReward(global icstage.Global, totalVotingAmount *big.Int) (multiplier, divider *big.Int) {
	multiplier = new(big.Int)
	divider = new(big.Int)

	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion2 {
		g := global.GetV1()
		if g.GetRRep().Sign() == 0 {
			return
		}
		multiplier.Mul(g.GetRRep(), new(big.Int).SetInt64(icmodule.IScoreICXRatio*icmodule.RrepMultiplier))
		divider.SetInt64(int64(icmodule.YearBlock * icmodule.RrepDivider))
	} else {
		g := global.GetV2()
		if g.GetTermPeriod() == 0 || totalVotingAmount.Sign() == 0 {
			return
		}
		multiplier.Mul(g.GetIGlobal(), g.GetIVoter())
		multiplier.Mul(multiplier, icmodule.BigIntIScoreICXRatio)
		divider.SetInt64(int64(100 * icmodule.MonthBlock))
		divider.Mul(divider, totalVotingAmount)
	}
	return
}

func (c *Calculator) calculateVotingReward() error {
	totalVotingAmount := new(big.Int)
	delegatingMap := make(map[string]map[int]icstage.VoteList)
	bondingMap := make(map[string]map[int]icstage.VoteList)
	prepInfo, err := c.loadPRepInfo()
	if err != nil {
		return err
	}
	vInfo, err := c.loadVotedInfo()
	if err != nil {
		return err
	}
	totalVotingAmount.Set(vInfo.TotalVoted())

	for iter := c.back.Filter(icstage.EventKey.Build()); iter.Has(); iter.Next() {
		var o trie.Object
		var key []byte
		o, key, err = iter.Get()
		if err != nil {
			return err
		}

		obj := o.(*icobject.Object)
		_type := obj.Tag().Type()

		var keySplit [][]byte
		keySplit, err = containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		offset := int(intconv.BytesToInt64(keySplit[1]))
		switch _type {
		case icstage.TypeEventEnable:
			// update prepInfo
			event := icstage.ToEventEnable(obj)
			idx := icutils.ToKey(event.Target())
			if _, ok := prepInfo[idx]; !ok {
				pe := new(pRepEnable)
				prepInfo[idx] = pe
			}
			if event.Status().IsEnabled() {
				prepInfo[idx].SetStartOffset(offset)
			} else if event.Status().IsDisabledPermanently() {
				// ICONist can't get voting reward when target PRep was unregistered or disqualified
				prepInfo[idx].SetEndOffset(offset)
			}
			// update vInfo
			status := event.Status()
			if c.global.GetRevision() >= icmodule.RevisionFixVotingReward && event.Status().IsDisabledTemporarily() {
				// ICONist get voting reward when target PRep got turn skipping penalty
				status = icstage.ESEnable
			}
			vInfo.SetEnable(event.Target(), status)
		case icstage.TypeEventDelegation, icstage.TypeEventBond:
			// update eventMap and vInfo
			event := icstage.ToEventVote(obj)
			idx := icutils.ToKey(event.From())
			if _type == icstage.TypeEventDelegation {
				_, ok := delegatingMap[idx]
				if !ok {
					delegatingMap[idx] = make(map[int]icstage.VoteList)
				}
				votes, ok := delegatingMap[idx][offset]
				if ok {
					votes.Update(event.Votes())
					delegatingMap[idx][offset] = votes
				} else {
					delegatingMap[idx][offset] = event.Votes()
				}
				vInfo.UpdateDelegated(event.Votes())
			} else {
				_, ok := bondingMap[idx]
				if !ok {
					bondingMap[idx] = make(map[int]icstage.VoteList)
				}
				votes, ok := bondingMap[idx][offset]
				if ok {
					votes.Update(event.Votes())
					bondingMap[idx][offset] = votes
				} else {
					bondingMap[idx][offset] = event.Votes()
				}
				vInfo.UpdateBonded(event.Votes())
			}
		case icstage.TypeEventDelegated:
			// update delegated only
			event := icstage.ToEventVote(obj)
			vInfo.UpdateDelegated(event.Votes())
		case icstage.TypeEventDelegationV2:
			event := icstage.ToEventDelegationV2(obj)
			idx := icutils.ToKey(event.From())
			_, ok := delegatingMap[idx]
			if !ok {
				delegatingMap[idx] = make(map[int]icstage.VoteList)
			}
			votes, ok := delegatingMap[idx][offset]
			if ok {
				votes.Update(event.Delegating())
				delegatingMap[idx][offset] = votes
			} else {
				delegatingMap[idx][offset] = event.Delegating()
			}
			vInfo.UpdateDelegated(event.Delegated())
		}
		// find MAX totalVotingAmount
		if totalVotingAmount.Cmp(vInfo.TotalVoted()) == -1 {
			totalVotingAmount.Set(vInfo.TotalVoted())
		}
	}

	// get variables for calculation
	multiplier, divider := varForVotingReward(c.global, totalVotingAmount)
	if multiplier.Sign() == 0 || divider.Sign() == 0 {
		return nil
	}

	inputs := []struct {
		_type    int
		eventMap map[string]map[int]icstage.VoteList
	}{
		{icreward.TypeDelegating, delegatingMap},
		{icreward.TypeBonding, bondingMap},
	}

	// calculate voting reward
	for _, i := range inputs {
		if err = c.processVoting(
			i._type,
			multiplier,
			divider,
			prepInfo,
			i.eventMap,
		); err != nil {
			return err
		}
		if err = c.processVotingEvent(
			i._type,
			multiplier,
			divider,
			prepInfo,
			i.eventMap,
		); err != nil {
			return err
		}
	}
	// add preprocessed data for BugDisabledPRep
	c.addDataForBugDisabledPRep(prepInfo, multiplier, divider)
	return nil
}

func (c *Calculator) addDataForBugDisabledPRep(prepInfo map[string]*pRepEnable, multiplier, divider *big.Int) error {
	revision := c.global.GetRevision()
	if c.global.GetIISSVersion() != icstate.IISSVersion2 ||
		revision < icmodule.RevisionDecentralize || revision >= icmodule.RevisionFixBugDisabledPRep {
		return nil
	}
	for iter := c.back.Filter(icstage.EventKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}

		obj := o.(*icobject.Object)
		_type := obj.Tag().Type()

		var keySplit [][]byte
		keySplit, err = containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		// DisabledPRep bug condition
		// - got a disabled event
		// - had a delegating
		if _type == icstage.TypeEventEnable {
			event := icstage.ToEventEnable(obj)
			if !event.Status().IsEnabled() {
				delegating, err := c.temp.GetDelegating(event.Target())
				if err != nil {
					return err
				}
				if delegating != nil {
					offset := int(intconv.BytesToInt64(keySplit[1]))
					reward := c.votingReward(multiplier, divider, offset, c.global.GetOffsetLimit(), prepInfo, delegating.Iterator())
					bug := icreward.NewBugDisabledPRep(reward)
					if err = c.temp.AddBugDisabledPRep(event.Target(), bug); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// processVoting calculator voting reward with delegating and bonding data.
func (c *Calculator) processVoting(
	_type int,
	multiplier *big.Int,
	divider *big.Int,
	prepInfo map[string]*pRepEnable,
	eventMap map[string]map[int]icstage.VoteList,
) error {
	if multiplier.Sign() == 0 {
		return nil
	}

	// voting took place in the previous period
	from := -1
	to := c.global.GetOffsetLimit()
	var prefix []byte
	if _type == icreward.TypeDelegating {
		prefix = icreward.DelegatingKey.Build()
	} else {
		prefix = icreward.BondingKey.Build()
	}
	for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
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
		var reward *big.Int
		if _, ok := eventMap[string(addr.Bytes())]; ok {
			continue
		} else {
			voting := toVoting(_type, o)
			if voting == nil {
				c.log.Errorf("Failed to convert data to voting instance")
				continue
			}
			reward = c.votingReward(multiplier, divider, from, to, prepInfo, voting.Iterator())
		}
		if err = c.UpdateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}

	return nil
}

// votingReward calculate voting reward with a single voting data
// IISS 2.0
//
//	reward = Rrep * delegations * period * IScoreICXRatio / YearBlock
//	multiplier = Rrep * IScoreICXRatio
//	divider = YearBlock
//
// IISS 3.1
//
//	reward = Iglobal * Ivoter * voting amount * period * IScoreICXRatio / (100 * Term period * total voting amount)
//	multiplier = Iglobal * Ivoter * IScoreICXRatio
//	divider = 100 * Term period * total voting amount
//
// reward = multiplier * voting amount * period / divider
func (c *Calculator) votingReward(
	multiplier *big.Int,
	divider *big.Int,
	from int,
	to int,
	prepInfo map[string]*pRepEnable,
	iter icstate.VotingIterator,
) *big.Int {
	total := new(big.Int)
	checkMinVoting := c.global.GetIISSVersion() == icstate.IISSVersion2
	for ; iter.Has(); iter.Next() {
		if voting, err := iter.Get(); err != nil {
			c.log.Errorf("Failed to iterate votings err=%+v", err)
		} else {
			if checkMinVoting && voting.Amount().Cmp(icmodule.BigIntMinDelegation) < 0 {
				continue
			}
			s := from
			e := to
			if prep, ok := prepInfo[icutils.ToKey(voting.To())]; ok {
				if prep.StartOffset() != 0 && prep.StartOffset() > s {
					s = prep.StartOffset()
				}
				if prep.EndOffset() != 0 && prep.EndOffset() < e {
					e = prep.EndOffset()
				}
				period := e - s
				if period <= 0 {
					continue
				}
				reward := new(big.Int).Mul(multiplier, voting.Amount())
				reward.Mul(reward, big.NewInt(int64(period)))
				reward.Div(reward, divider)
				total.Add(total, reward)
				c.log.Tracef("VotingReward %s: %s = %s * %s * %d / %s",
					voting.To(), reward, multiplier, voting.Amount(), period, divider)
			}
		}
	}
	return total
}

// processVotingEvent calculate reward for account who got DELEGATE event
func (c *Calculator) processVotingEvent(
	_type int,
	multiplier *big.Int,
	divider *big.Int,
	prepInfo map[string]*pRepEnable,
	eventMap map[string]map[int]icstage.VoteList,
) error {
	for key, events := range eventMap { // each account
		addr, _ := common.NewAddress([]byte(key))
		reward := new(big.Int)
		offsets := make([]int, 0, len(events))
		for offset, _ := range events {
			offsets = append(offsets, offset)
		}
		// sort with offset
		sort.Ints(offsets)

		voting, err := c.getVoting(_type, addr)
		if err != nil {
			return err
		}

		// initial voting took place in the previous period
		// New configuration works from the next block
		from := -1
		offsetLimit := c.global.GetOffsetLimit()
		iissVersion := c.global.GetIISSVersion()
		for i := 0; i < len(events); i += 1 {
			to := offsets[i]
			switch iissVersion {
			case icstate.IISSVersion2:
				ret := c.votingReward(multiplier, divider, from, offsetLimit, prepInfo, voting.Iterator())
				reward.Add(reward, ret)
				c.log.Tracef("VotingEvent %s %d add: %d-%d %s", addr, i, from, offsetLimit, ret)
				ret = c.votingReward(multiplier, divider, to, offsetLimit, prepInfo, voting.Iterator())
				reward.Sub(reward, ret)
				c.log.Tracef("VotingEvent %s %d sub: %d-%d %s", addr, i, to, offsetLimit, ret)
			case icstate.IISSVersion3:
				to = offsets[i]
				ret := c.votingReward(multiplier, divider, from, to, prepInfo, voting.Iterator())
				reward.Add(reward, ret)
				c.log.Tracef("VotingEvent %s %d: %d-%d %s", addr, i, from, to, ret)
			}

			// update Bonding or Delegating
			votes := events[to]
			if err = voting.ApplyVotes(votes); err != nil {
				errors.Wrapf(err, "Failed to apply vote of %s, offset=%d, votes=%+v", addr, to, votes)
				return err
			}

			from = to
		}
		// calculate reward for last event
		ret := c.votingReward(multiplier, divider, from, offsetLimit, prepInfo, voting.Iterator())
		reward.Add(reward, ret)
		c.log.Tracef("VotingEvent %s last: %d, %d: %s", addr, from, offsetLimit, ret)

		if err = c.writeVoting(addr, voting); err != nil {
			return nil
		}
		if err = c.UpdateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}
	return nil
}

func toVoting(_type int, o trie.Object) icreward.Voting {
	switch _type {
	case icreward.TypeDelegating:
		return icreward.ToDelegating(o)
	case icreward.TypeBonding:
		return icreward.ToBonding(o)
	}
	return nil
}

// getVoting read Voting object from MPT and return cloned object
func (c *Calculator) getVoting(_type int, addr *common.Address) (icreward.Voting, error) {
	switch _type {
	case icreward.TypeDelegating:
		delegating, err := c.temp.GetDelegating(addr)
		if err != nil {
			return nil, err
		}
		if delegating == nil {
			delegating = icreward.NewDelegating()
		} else {
			delegating = delegating.Clone()
		}
		return delegating, nil
	case icreward.TypeBonding:
		bonding, err := c.temp.GetBonding(addr)
		if err != nil {
			return nil, err
		}
		if bonding == nil {
			bonding = icreward.NewBonding()
		} else {
			bonding = bonding.Clone()
		}
		return bonding, nil
	}

	return nil, nil
}

func (c *Calculator) writeVoting(addr *common.Address, data interface{}) error {
	switch o := data.(type) {
	case *icreward.Delegating:
		return c.temp.SetDelegating(addr, o)
	case *icreward.Bonding:
		return c.temp.SetBonding(addr, o)
	}
	return nil
}

func (c *Calculator) postWork() (err error) {
	// check result
	if c.global.GetIISSVersion() == icstate.IISSVersion3 {
		if c.stats.blockProduce.Sign() != 0 {
			return errors.Errorf("Too much BlockProduce Reward. %d", c.stats.blockProduce)
		}
		g := c.global.GetV2()
		maxVotedReward := new(big.Int).Mul(g.GetIGlobal(), g.GetIPRep())
		maxVotedReward.Mul(maxVotedReward, icmodule.BigIntIScoreICXRatio)
		if c.stats.voted.Cmp(maxVotedReward) == 1 {
			return errors.Errorf("Too much Voted Reward. %d < %d", maxVotedReward, c.stats.voted)
		}
		maxVotingReward := new(big.Int).Mul(g.GetIGlobal(), g.GetIVoter())
		maxVotingReward.Mul(maxVotingReward, icmodule.BigIntIScoreICXRatio)
		if c.stats.voting.Cmp(maxVotingReward) == 1 {
			return errors.Errorf("Too much Voting Reward. %d < %d", maxVotingReward, c.stats.voting)
		}
	}

	// write BTP data to temp. Use BTP data in the next term
	if err = c.processBTP(); err != nil {
		return err
	}
	return nil
}

func (c *Calculator) processBTP() error {
	for iter := c.back.Filter(icstage.BTPKey.Build()); iter.Has(); iter.Next() {
		o, _, err := iter.Get()
		if err != nil {
			return err
		}
		obj := o.(*icobject.Object)
		switch obj.Tag().Type() {
		case icstage.TypeBTPDSA:
			value := icstage.ToBTPDSA(o)
			dsa, err := c.temp.GetDSA()
			if err != nil {
				return err
			}
			nDSA := dsa.Updated(value.Index())
			if err = c.temp.SetDSA(nDSA); err != nil {
				return err
			}
		case icstage.TypeBTPPublicKey:
			value := icstage.ToBTPPublicKey(o)
			pubKey, err := c.temp.GetPublicKey(value.From())
			if err != nil {
				return nil
			}
			nPubKey := pubKey.Updated(value.Index())
			if err = c.temp.SetPublicKey(value.From(), nPubKey); err != nil {
				return err
			}
		}
	}
	return nil
}

type validator struct {
	addr   *common.Address
	iScore *big.Int
}

func (v *validator) Address() module.Address {
	return v.addr
}

func (v *validator) IScore() *big.Int {
	return v.iScore
}

func (v *validator) SetIScore(value *big.Int) {
	v.iScore = value
}

func newValidator(addr *common.Address) *validator {
	return &validator{
		addr:   addr,
		iScore: new(big.Int),
	}
}

type votedData struct {
	voted  *icreward.Voted
	iScore *big.Int
	pubKey bool
}

func (vd *votedData) Compare(vd2 *votedData) int {
	if vd.pubKey != vd2.pubKey {
		if vd.pubKey {
			return 1
		} else {
			return -1
		}
	}

	dv := new(big.Int)
	if vd.Enable() {
		dv = vd.GetBondedDelegation()
	}
	dv2 := new(big.Int)
	if vd2.Enable() {
		dv2 = vd2.GetBondedDelegation()
	}
	ret := dv.Cmp(dv2)
	if ret != 0 {
		return ret
	}

	if vd.Enable() {
		dv = vd.GetDelegated()
	}
	if vd2.Enable() {
		dv2 = vd2.GetDelegated()
	}
	return dv.Cmp(dv2)
}

func (vd *votedData) Voted() *icreward.Voted {
	return vd.voted
}

func (vd *votedData) IScore() *big.Int {
	return vd.iScore
}

func (vd *votedData) SetIScore(value *big.Int) {
	vd.iScore = value
}

func (vd *votedData) Status() icstage.EnableStatus {
	return vd.voted.Status()
}

func (vd *votedData) Enable() bool {
	return vd.voted.Enable()
}

func (vd *votedData) SetStatus(status icstage.EnableStatus) {
	vd.voted.SetStatus(status)
}

func (vd *votedData) SetCommissionRate(value *big.Int) {
	vd.voted.SetCommissionRate(value)
	vd.voted.SetVersion(icreward.VotedVersion2)
}

func (vd *votedData) GetDelegated() *big.Int {
	return vd.voted.Delegated()
}

func (vd *votedData) SetDelegated(value *big.Int) {
	vd.voted.SetDelegated(value)
}

func (vd *votedData) GetBonded() *big.Int {
	return vd.voted.Bonded()
}

func (vd *votedData) SetBonded(value *big.Int) {
	vd.voted.SetBonded(value)
}

func (vd *votedData) GetBondedDelegation() *big.Int {
	return vd.voted.BondedDelegation()
}

func (vd *votedData) GetVotedAmount() *big.Int {
	return vd.voted.GetVotedAmount()
}

func (vd *votedData) IsEmpty() bool {
	return vd.voted.IsEmpty()
}

func (vd *votedData) UpdateToWrite() {
	if vd.voted.Status().IsDisabledTemporarily() {
		vd.voted.SetStatus(icstage.ESEnable)
	}
}

func (vd *votedData) UpdateBondedDelegation(bondRequirement int) {
	vd.voted.UpdateBondedDelegation(bondRequirement)
}

func (vd *votedData) SetPubKey(yn bool) {
	vd.pubKey = yn
}

func (vd *votedData) PubKey() bool {
	return vd.pubKey
}

func newVotedData(d *icreward.Voted) *votedData {
	return &votedData{
		voted:  d,
		iScore: new(big.Int),
	}
}

type votedInfo struct {
	totalBondedDelegation *big.Int // total bondedDelegation amount of top 100 P-Reps
	totalVoted            *big.Int // total delegated + bonded amount of all P-Reps
	maxRankForReward      int
	rank                  []string
	preps                 map[string]*votedData
}

func (vi *votedInfo) TotalBondedDelegation() *big.Int {
	return vi.totalBondedDelegation
}

func (vi *votedInfo) TotalVoted() *big.Int {
	return vi.totalVoted
}

func (vi *votedInfo) MaxRankForReward() int {
	return vi.maxRankForReward
}

func (vi *votedInfo) Rank() []string {
	return vi.rank
}

func (vi *votedInfo) PReps() map[string]*votedData {
	return vi.preps
}

func (vi *votedInfo) GetPRepByAddress(addr module.Address) *votedData {
	key := icutils.ToKey(addr)
	return vi.preps[key]
}

func (vi *votedInfo) IsElectedPRep(addr module.Address) bool {
	key := icutils.ToKey(addr)
	for i, addrKey := range vi.rank {
		if i == vi.maxRankForReward {
			return false
		}
		if key == addrKey {
			return true
		}
	}
	return false
}

func (vi *votedInfo) AddVotedData(addr module.Address, data *votedData) {
	vi.preps[icutils.ToKey(addr)] = data
	if data.Enable() {
		vi.updateTotalVoted(data.GetVotedAmount())
	}
}

func (vi *votedInfo) SetEnable(addr module.Address, status icstage.EnableStatus) {
	if vData, ok := vi.preps[icutils.ToKey(addr)]; ok {
		if status.IsEnabled() != vData.Enable() {
			if status.IsEnabled() {
				vi.updateTotalVoted(vData.GetVotedAmount())
			} else {
				vi.updateTotalVoted(new(big.Int).Neg(vData.GetVotedAmount()))
			}
		}
		vData.SetStatus(status)
	} else {
		voted := icreward.NewVoted()
		vData = newVotedData(voted)
		vData.SetStatus(status)
		vi.AddVotedData(addr, vData)
	}
}

func (vi *votedInfo) SetCommissionRate(addr module.Address, value *big.Int) {
	if vData, ok := vi.preps[icutils.ToKey(addr)]; ok {
		vData.SetCommissionRate(value)
	}
}

func (vi *votedInfo) UpdateDelegated(votes icstage.VoteList) {
	for _, vote := range votes {
		// vote got diff value
		if data, ok := vi.preps[icutils.ToKey(vote.To())]; ok {
			data.SetDelegated(new(big.Int).Add(data.GetDelegated(), vote.Amount()))
			if data.Enable() {
				vi.updateTotalVoted(vote.Amount())
			}
		} else {
			voted := icreward.NewVoted()
			data = newVotedData(voted)
			data.SetDelegated(vote.Amount())
			vi.AddVotedData(vote.To(), data)
		}
	}
}

func (vi *votedInfo) UpdateBonded(votes icstage.VoteList) {
	for _, vote := range votes {
		if data, ok := vi.preps[icutils.ToKey(vote.To())]; ok {
			data.SetBonded(new(big.Int).Add(data.GetBonded(), vote.Amount()))
			if data.Enable() {
				vi.updateTotalVoted(vote.Amount())
			}
		} else {
			voted := icreward.NewVoted()
			data = newVotedData(voted)
			data.SetBonded(vote.Amount())
			vi.AddVotedData(vote.To(), data)
		}
	}
}

func (vi *votedInfo) Sort() {
	// sort prep list with bondedDelegation amount
	size := len(vi.preps)
	keys := make(map[votedData]string, size)
	vDataSlice := make([]votedData, size)
	i := 0
	for key, data := range vi.preps {
		keys[*data] = key
		vDataSlice[i] = *data
		i += 1
	}
	sort.Slice(vDataSlice, func(i, j int) bool {
		ret := vDataSlice[i].Compare(&vDataSlice[j])
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}
		return bytes.Compare([]byte(keys[vDataSlice[i]]), []byte(keys[vDataSlice[j]])) > 0
	})

	rank := make([]string, size)
	for idx, v := range vDataSlice {
		rank[idx] = keys[v]
	}
	vi.rank = rank
}

func (vi *votedInfo) UpdateTotalBondedDelegation() {
	total := new(big.Int)
	for i, addrKey := range vi.rank {
		if i == vi.maxRankForReward {
			break
		}
		vData := vi.preps[addrKey]
		if vData.Enable() && vData.PubKey() {
			total.Add(total, vData.GetBondedDelegation())
		}
	}
	vi.totalBondedDelegation = total
}

func (vi *votedInfo) updateTotalVoted(amount *big.Int) {
	vi.totalVoted = new(big.Int).Add(vi.totalVoted, amount)
}

// CalculateReward calculate P-Rep voted reward
func (vi *votedInfo) CalculateReward(multiplier, divider *big.Int, period int) {
	if multiplier.Sign() == 0 || period == 0 {
		return
	}
	if divider.Sign() == 0 || vi.totalBondedDelegation.Sign() == 0 {
		return
	}
	// reward = multiplier * period * bondedDelegation / (divider * totalBondedDelegation)
	base := new(big.Int).Mul(multiplier, big.NewInt(int64(period)))
	reward := new(big.Int)
	for i, addrKey := range vi.rank {
		if i == vi.maxRankForReward {
			break
		}
		prep := vi.preps[addrKey]
		if prep.Enable() == false {
			continue
		}
		if prep.PubKey() == false {
			continue
		}

		reward.Mul(base, prep.GetBondedDelegation())
		reward.Div(reward, divider)
		reward.Div(reward, vi.totalBondedDelegation)

		log.Tracef("VOTED REWARD %s %d = %d * %d * %d / (%d * %d)",
			common.MustNewAddress([]byte(addrKey)),
			reward, multiplier, period, prep.GetBondedDelegation(), divider, vi.totalBondedDelegation)

		prep.SetIScore(new(big.Int).Add(prep.IScore(), reward))
	}
}

func newVotedInfo(maxRankForReward int) *votedInfo {
	return &votedInfo{
		totalBondedDelegation: new(big.Int),
		totalVoted:            new(big.Int),
		maxRankForReward:      maxRankForReward,
		preps:                 make(map[string]*votedData),
	}
}

type pRepEnable struct {
	startOffset int
	endOffset   int
}

func (p *pRepEnable) StartOffset() int {
	return p.startOffset
}

func (p *pRepEnable) EndOffset() int {
	return p.endOffset
}

func (p *pRepEnable) SetStartOffset(value int) {
	p.startOffset = value
}

func (p *pRepEnable) SetEndOffset(value int) {
	p.endOffset = value
}
