/*
 * Copyright 2020 ICON Foundation
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

package iiss

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
	"github.com/icon-project/goloop/service/state"
)

type RewardType int

const (
	TypeBlockProduce RewardType = iota
	TypeVoted
	TypeVoting
)

const (
	DayBlock     = 24 * 60 * 60 / 2
	DayPerMonth  = 30
	MonthBlock   = DayBlock * DayPerMonth
	MonthPerYear = 12
	YearBlock    = MonthBlock * MonthPerYear

	IScoreICXRatio        = 1_000
	VotedRewardMultiplier = 100
	RrepMultiplier        = 3      // rrep = rrep + eep + dbp = 3 * rrep
	RrepDivider           = 10_000 // rrep(10_000) = 100.00%, rrep(200) = 2.00%
)

var (
	BigIntIScoreICXRatio = big.NewInt(int64(IScoreICXRatio))
)

type Calculator struct {
	log log.Logger

	startHeight int64
	back        *icstage.Snapshot
	base        *icreward.Snapshot
	global      icstage.Global
	temp        *icreward.State
	result      *icreward.Snapshot
	stats       *statistics
}

func (c *Calculator) Result() *icreward.Snapshot {
	return c.result
}

func (c *Calculator) StartHeight() int64 {
	return c.startHeight
}

func (c *Calculator) TotalReward() *big.Int {
	return c.stats.TotalReward()
}

func (c *Calculator) Back() *icstage.Snapshot {
	return c.back
}

func (c *Calculator) Base() *icreward.Snapshot {
	return c.base
}

func (c *Calculator) Temp() *icreward.State {
	return c.temp
}

func (c *Calculator) IsCalcDone(blockHeight int64) bool {
	if c.startHeight == InitBlockHeight {
		return true
	}
	return c.startHeight == blockHeight && c.result != nil
}

func (c *Calculator) CheckToRun(ess state.ExtensionSnapshot) bool {
	ss := ess.(*ExtensionSnapshotImpl)
	if ss.back == nil {
		return false
	}
	global, err := ss.back.GetGlobal()
	if err != nil || global == nil {
		return false
	}
	return c.back == nil || !bytes.Equal(c.back.Bytes(), ss.back.Bytes())
}

func (c *Calculator) Run(ess state.ExtensionSnapshot, logger log.Logger) (err error) {
	c.log = logger
	ss := ess.(*ExtensionSnapshotImpl)
	startTS := time.Now()
	if err = c.prepare(ss); err != nil {
		err = errors.Wrapf(err, "Failed to prepare calculator")
		return
	}
	prepareTS := time.Now()

	if err = c.calculateBlockProduce(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate block produce reward")
		return
	}
	bpTS := time.Now()

	if err = c.calculateVotedReward(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate P-Rep voted reward")
		return
	}
	votedTS := time.Now()

	if err = c.calculateVotingReward(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate ICONist voting reward")
		return
	}
	votingTS := time.Now()

	if err = c.postWork(); err != nil {
		return
	}
	finalTS := time.Now()

	c.log.Infof("Calculation time: total=%s prepare=%s blockProduce=%s voted=%s voting=%s postwork=%s",
		finalTS.Sub(startTS), prepareTS.Sub(startTS), bpTS.Sub(prepareTS),
		votedTS.Sub(bpTS), votingTS.Sub(votedTS), finalTS.Sub(votingTS),
	)
	c.log.Infof("Calculation statistics: BlockProduce=%s Voted=%s Voting=%s",
		c.stats.BlockProduce(), c.stats.Voted(), c.stats.Voting())
	return
}

func (c *Calculator) prepare(ss *ExtensionSnapshotImpl) error {
	var err error
	c.back = ss.back
	c.base = ss.reward
	// make new State with hash value to decoupling base and temp
	c.temp = icreward.NewState(ss.database, c.base.Bytes())
	c.result = nil
	c.stats.Clear()

	// read global variables
	c.global, err = c.back.GetGlobal()
	if err != nil {
		return err
	}
	if c.global == nil {
		return errors.Errorf("There is no Global values for calculator")
	}
	c.startHeight = c.global.GetStartHeight()

	c.log.Infof("Start calculation %d", c.startHeight)
	c.log.Infof("Global Option: %s", c.global)

	// write claim data to temp
	if err = c.processClaim(); err != nil {
		return err
	}

	return nil
}

func (c *Calculator) processClaim() error {
	for iter := c.back.Filter(icstage.IScoreClaimKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		obj := o.(*icobject.Object)
		if obj.Tag().Type() == icstage.TypeIScoreClaim {
			claim := icstage.ToIScoreClaim(o)
			keySplit, err := containerdb.SplitKeys(key)
			if err != nil {
				return nil
			}
			addr, err := common.NewAddress(keySplit[1])
			if err != nil {
				return nil
			}
			iScore, err := c.temp.GetIScore(addr)
			if err != nil {
				return nil
			}
			iScore = iScore.Subtracted(claim.Value())
			if iScore.Value().Sign() == -1 {
				return errors.Errorf("Invalid negative I-Score for %s", addr.String())
			}
			if err = c.temp.SetIScore(addr, iScore); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Calculator) updateIScore(addr module.Address, reward *big.Int, t RewardType) error {
	iScore, err := c.temp.GetIScore(addr)
	if err != nil {
		return err
	}
	if err = c.temp.SetIScore(addr, iScore.Added(reward)); err != nil {
		return err
	}
	c.log.Tracef("Update IScore %s by %d: + %s = %+v", addr.String(), t, reward, iScore)

	switch t {
	case TypeBlockProduce:
		c.stats.IncreaseBlockProduce(reward)
	case TypeVoted:
		c.stats.IncreaseVoted(reward)
	case TypeVoting:
		c.stats.IncreaseVoting(reward)
	}
	return nil
}

// varForBlockProduceReward return variable for block produce reward
// return irep * mainPRepCount * IScoreICXRatio / (2 * 2 * MonthBlock)
func varForBlockProduceReward(irep *big.Int, mainPRepCount int) *big.Int {
	v := new(big.Int)
	v.Mul(irep, big.NewInt(int64(mainPRepCount)*IScoreICXRatio))
	v.Div(v, big.NewInt(int64(MonthBlock*2*2)))
	return v
}

func (c *Calculator) calculateBlockProduce() error {
	if c.global.GetIISSVersion() == icstate.IISSVersion2 {
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
		if err = c.updateIScore(v.Address(), v.IScore(), TypeBlockProduce); err != nil {
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
		return errors.Errorf("Can't find validator with %v", bp)
	}
	beta1Reward := new(big.Int).Set(variable)

	// for proposer
	proposer := validators[pIndex]
	proposer.SetIScore(new(big.Int).Add(proposer.IScore(), beta1Reward))

	// for validator
	if vCount > 0 {
		beta1Validate := new(big.Int).Div(beta1Reward, big.NewInt(int64(vCount)))
		for i := 0; i < maxIndex; i += 1 {
			if (vMask.Bit(i)) != 0 {
				validators[i].SetIScore(new(big.Int).Add(validators[i].IScore(), beta1Validate))
			}
		}
	}

	return nil
}

// varForVotedReward return variable for P-Rep voted reward
// IISS 2.0
// 	multiplier = irep * electedPRepCount * IScoreICXRatio
//	divider = 2 * MonthBlock
// IISS 3.1
// 	multiplier = iglobal * iprep * IScoreICXRatio
//	divider = 100 * TermPeriod
func varForVotedReward(global icstage.Global) (multiplier, divider *big.Int) {
	multiplier = new(big.Int)
	divider = new(big.Int)

	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion1 {
		g := global.(*icstage.GlobalV1)
		multiplier.Mul(g.GetIRep(), big.NewInt(int64(VotedRewardMultiplier*IScoreICXRatio)))
		divider.SetInt64(int64(MonthBlock * 2))
	} else {
		g := global.(*icstage.GlobalV2)
		if g.GetTermPeriod() == 0 {
			return
		}
		multiplier.Mul(g.GetIGlobal(), g.GetIPRep())
		multiplier.Mul(multiplier, BigIntIScoreICXRatio)
		divider.SetInt64(int64(100 * g.GetTermPeriod()))
	}
	return
}

func (c *Calculator) calculateVotedReward() error {
	offset := 0
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
			vInfo.CalculateReward(multiplier, divider, keyOffset-offset)
			offset = keyOffset

			obj := icstage.ToEventEnable(o)
			vInfo.SetEnable(obj.Target(), obj.Status())
			// If revision < 7, do not update totalBondedDelegation with EventEnable
			if c.global.GetRevision() >= icmodule.RevisionFixTotalDelegated {
				vInfo.UpdateTotalBondedDelegation()
			}
		case icstage.TypeEventDelegation:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateDelegated(obj.Votes())
		case icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vInfo.UpdateBonded(obj.Votes())
		}
	}
	if offset < c.global.GetOffsetLimit() {
		vInfo.CalculateReward(multiplier, divider, c.global.GetTermPeriod()-offset)
	}

	// write result to temp and update statistics
	for key, prep := range vInfo.PReps() {
		var addr *common.Address
		addr, err = common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		prep.UpdateToWrite()
		if prep.IsEmpty() {
			if err = c.temp.DeleteVoted(addr); err != nil {
				return err
			}
		} else {
			if err = c.temp.SetVoted(addr, prep.Voted()); err != nil {
				return err
			}
		}

		if prep.IScore().Sign() == 0 {
			continue
		}

		if err = c.updateIScore(addr, prep.IScore(), TypeVoted); err != nil {
			return err
		}
	}
	return nil
}

func (c *Calculator) loadVotedInfo() (*votedInfo, error) {
	electedPRepCount := c.global.GetElectedPRepCount()
	bondRequirement := c.global.GetBondRequirement()
	vInfo := newVotedInfo(electedPRepCount)

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
// 	multiplier = Rrep * IScoreICXRatio
//	divider = YearBlock * RrepDivider
// IISS 3.1
// 	multiplier = Iglobal * Ivoter * IScoreICXRatio
//	divider = 100 * term period * total voting amount
func varForVotingReward(global icstage.Global, totalVotingAmount *big.Int) (multiplier, divider *big.Int) {
	multiplier = new(big.Int)
	divider = new(big.Int)

	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion1 {
		g := global.GetV1()
		if g.GetRRep().Sign() == 0 {
			return
		}
		multiplier.Mul(g.GetRRep(), new(big.Int).SetInt64(IScoreICXRatio*RrepMultiplier))
		divider.SetInt64(int64(YearBlock * RrepDivider))
	} else {
		g := global.GetV2()
		if g.GetTermPeriod() == 0 || totalVotingAmount.Sign() == 0 {
			return
		}
		multiplier.Mul(g.GetIGlobal(), g.GetIVoter())
		multiplier.Mul(multiplier, BigIntIScoreICXRatio)
		divider.SetInt64(int64(100 * g.GetTermPeriod()))
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
			} else {
				prepInfo[idx].SetEndOffset(offset)
			}
			// update vInfo
			vInfo.SetEnable(event.Target(), event.Status())
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

	to := c.global.GetTermPeriod()
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
			reward = c.votingReward(multiplier, divider, 0, to, prepInfo, voting.Iterator())
		}
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}

	return nil
}

// votingReward calculate voting reward with a single voting data
// IISS 2.0
//   reward = Rrep * delegations * period * IScoreICXRatio / YearBlock
//   multiplier = Rrep * IScoreICXRatio
//   divider = YearBlock
// IISS 3.1
//   reward = Iglobal * Ivoter * voting amount * period * IScoreICXRatio / (100 * Term period * total voting amount)
//   multiplier = Iglobal * Ivoter * IScoreICXRatio
//   divider = 100 * Term period * total voting amount
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
	for ; iter.Has(); iter.Next() {
		if voting, err := iter.Get(); err != nil {
			c.log.Errorf("Fail to iterating votings err=%+v", err)
		} else {
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

		votings, err := c.getVoting(_type, addr)
		if err != nil {
			return err
		}

		// Voting took place in the previous period. Add a reward of offset 0.
		start := -1
		end := 0
		offsetLimit := c.global.GetOffsetLimit()
		iissVersion := c.global.GetIISSVersion()
		for i := 0; i < len(events); i += 1 {
			end = offsets[i]
			switch iissVersion {
			case icstate.IISSVersion1:
				ret := c.votingReward(multiplier, divider, start, offsetLimit, prepInfo, votings.Iterator())
				reward.Add(reward, ret)
				c.log.Tracef("VotingEvent %s %d add: %d, %d: %s", addr, i, start, offsetLimit, ret)
				ret = c.votingReward(multiplier, divider, end, offsetLimit, prepInfo, votings.Iterator())
				reward.Sub(reward, ret)
				c.log.Tracef("VotingEvent %s %d sub: %d, %d: %s", addr, i, end, offsetLimit, ret)
			case icstate.IISSVersion2:
				end = offsets[i]
				ret := c.votingReward(multiplier, divider, start, end, prepInfo, votings.Iterator())
				reward.Add(reward, ret)
				c.log.Tracef("VotingEvent %s %d: %d, %d: %s", addr, i, start, end, ret)
			}

			// update Bonding or Delegating
			votes := events[end]
			if err = votings.ApplyVotes(votes); err != nil {
				return err
			}

			start = end
		}
		// calculate reward for last event
		ret := c.votingReward(multiplier, divider, start, offsetLimit, prepInfo, votings.Iterator())
		reward.Add(reward, ret)
		c.log.Tracef("VotingEvent %s last: %d, %d: %s", addr, start, offsetLimit, ret)

		if err = c.writeVoting(addr, votings); err != nil {
			return nil
		}
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
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
		return c.writeDelegating(addr, o)
	case *icreward.Bonding:
		return c.writeBonding(addr, o)
	}
	return nil
}

func (c *Calculator) writeDelegating(addr *common.Address, delegating *icreward.Delegating) error {
	if delegating.IsEmpty() {
		if err := c.temp.DeleteDelegating(addr); err != nil {
			return err
		}
	} else {
		if err := c.temp.SetDelegating(addr, delegating); err != nil {
			return err
		}
	}
	return nil
}

func (c *Calculator) writeBonding(addr *common.Address, bonding *icreward.Bonding) error {
	if bonding.IsEmpty() {
		if err := c.temp.DeleteBonding(addr); err != nil {
			return err
		}
	} else {
		if err := c.temp.SetBonding(addr, bonding); err != nil {
			return err
		}
	}
	return nil
}

func (c *Calculator) postWork() (err error) {
	// check result
	if c.global.GetIISSVersion() == icstate.IISSVersion2 {
		if c.stats.blockProduce.Sign() != 0 {
			return errors.CriticalUnknownError.Errorf("Too much BlockProduce Reward. %s", c.stats.blockProduce.String())
		}
		g := c.global.GetV2()
		maxVotedReward := new(big.Int).Mul(g.GetIGlobal(), g.GetIPRep())
		maxVotedReward.Mul(maxVotedReward, BigIntIScoreICXRatio)
		if c.stats.voted.Cmp(maxVotedReward) == 1 {
			return errors.CriticalUnknownError.Errorf("Too much Voted Reward. %s < %s",
				maxVotedReward, c.stats.voted.String())
		}
		maxVotingReward := new(big.Int).Mul(g.GetIGlobal(), g.GetIVoter())
		maxVotingReward.Mul(maxVotingReward, BigIntIScoreICXRatio)
		if c.stats.voting.Cmp(maxVotingReward) == 1 {
			return errors.CriticalUnknownError.Errorf("Too much Voting Reward. %s < %s",
				maxVotingReward, c.stats.voting.String())
		}
	}

	// save calculation result to MPT
	c.result = c.temp.GetSnapshot()
	if err = c.result.Flush(); err != nil {
		return
	}

	return
}

const InitBlockHeight = -1

func NewCalculator() *Calculator {
	return &Calculator{
		startHeight: InitBlockHeight,
		stats:       newStatistics(),
	}
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
	status icstage.EnableStatus
}

func (vd *votedData) Compare(vd2 *votedData) int {
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
	return vd.status
}

func (vd *votedData) Enable() bool {
	return vd.voted.Enable()
}

func (vd *votedData) SetEnable(status icstage.EnableStatus) {
	vd.voted.SetEnable(status.IsEnabled())
	vd.status = status
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
	if vd.status.IsDisabledTemporarily() {
		vd.voted.SetEnable(true)
	}
}

func (vd *votedData) UpdateBondedDelegation(bondRequirement int) {
	vd.voted.UpdateBondedDelegation(bondRequirement)
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
		vData.SetEnable(status)
	} else {
		voted := icreward.NewVoted()
		vData = newVotedData(voted)
		vData.SetEnable(status)
		vi.AddVotedData(addr, vData)
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
	temp := make(map[votedData]string, size)
	tempKeys := make([]votedData, size)
	i := 0
	for key, data := range vi.preps {
		temp[*data] = key
		tempKeys[i] = *data
		i += 1
	}
	sort.Slice(tempKeys, func(i, j int) bool {
		ret := tempKeys[i].Compare(&tempKeys[j])
		if ret > 0 {
			return true
		} else if ret < 0 {
			return false
		}
		return bytes.Compare([]byte(temp[tempKeys[i]]), []byte(temp[tempKeys[j]])) > 0
	})

	rank := make([]string, size)
	for idx, v := range tempKeys {
		rank[idx] = temp[v]
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
		if vData.Enable() {
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

		reward.Mul(base, prep.GetBondedDelegation())
		reward.Div(reward, divider)
		reward.Div(reward, vi.totalBondedDelegation)

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

type statistics struct {
	blockProduce *big.Int
	voted        *big.Int
	voting       *big.Int
}

func (s *statistics) BlockProduce() *big.Int {
	return s.blockProduce
}

func (s *statistics) Voted() *big.Int {
	return s.voted
}

func (s *statistics) Voting() *big.Int {
	return s.voting
}

func increaseStats(src *big.Int, amount *big.Int) *big.Int {
	n := new(big.Int)
	if src == nil {
		n.Set(amount)
	} else {
		n.Add(src, amount)
	}
	return n
}

func (s *statistics) IncreaseBlockProduce(amount *big.Int) {
	s.blockProduce = increaseStats(s.blockProduce, amount)
}

func (s *statistics) IncreaseVoted(amount *big.Int) {
	s.voted = increaseStats(s.voted, amount)
}

func (s *statistics) IncreaseVoting(amount *big.Int) {
	s.voting = increaseStats(s.voting, amount)
}

func (s *statistics) TotalReward() *big.Int {
	reward := new(big.Int)
	reward.Add(s.BlockProduce(), s.Voted())
	reward.Add(reward, s.Voting())
	return reward
}

func (s *statistics) Clear() {
	s.blockProduce = new(big.Int)
	s.voted = new(big.Int)
	s.voting = new(big.Int)
}

func newStatistics() *statistics {
	return &statistics{
		new(big.Int),
		new(big.Int),
		new(big.Int),
	}
}
