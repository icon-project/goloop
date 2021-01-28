/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

const (
	DayBlock   = 24 * 60 * 60 / 2
	MonthBlock = DayBlock * 30
	YearBlock  = MonthBlock * 12

	IScoreICXRatio = 1000

	keyCalculator = "iiss.calculator"
)

type RewardType int

const (
	TypeBlockProduce RewardType = iota
	TypeVoted
	TypeVoting
)

var (
	BigIntIScoreICXRatio = big.NewInt(int64(IScoreICXRatio))
	bigIntBeta3Divider   = big.NewInt(int64(YearBlock / IScoreICXRatio))
	BigIntTwo            = big.NewInt(2)
)

type Calculator struct {
	dbase db.Database

	result      *icreward.Snapshot
	blockHeight int64
	stats       *statistics

	back        *icstage.Snapshot
	global      *icstage.Global
	base        *icreward.Snapshot
	temp        *icreward.State
	offsetLimit int
}

func (c *Calculator) RLPEncodeSelf(e codec.Encoder) error {
	var hash []byte
	if c.result != nil {
		hash = c.result.Bytes()
	}
	return e.EncodeListOf(
		hash,
		c.blockHeight,
		c.stats.blockProduce,
		c.stats.Voted,
		c.stats.voting,
	)
}

func (c *Calculator) RLPDecodeSelf(d codec.Decoder) error {
	var hash []byte
	if err := d.DecodeListOf(
		&hash,
		&c.blockHeight,
		&c.stats.blockProduce,
		&c.stats.Voted,
		&c.stats.voting,
	); err != nil {
		return err
	}
	c.result = icreward.NewSnapshot(c.dbase, hash)
	return nil
}

func (c *Calculator) Bytes() []byte {
	bs, err := codec.BC.MarshalToBytes(c)
	if err != nil {
		return nil
	}
	return bs
}
func (c *Calculator) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, c)
	return err
}

func (c *Calculator) Flush() error {
	bk, err := c.dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return err
	}
	return bk.Set([]byte(keyCalculator), c.Bytes())
}

func (c *Calculator) Init(dbase db.Database) error {
	c.dbase = dbase
	bk, err := c.dbase.GetBucket(db.ChainProperty)
	if err != nil {
		return err
	}
	bs, err := bk.Get([]byte(keyCalculator))
	if err != nil || bs == nil {
		return err
	}
	return c.SetBytes(bs)
}

func (c *Calculator) isGenesis() bool {
	return c.blockHeight == 0 && c.result == nil
}

func (c *Calculator) isCalculating() bool {
	return c.blockHeight != 0 && c.result == nil
}

func (c *Calculator) isResultSynced(ss *ExtensionSnapshotImpl) bool {
	if c.result == nil {
		return false
	}
	return bytes.Compare(c.result.Bytes(), ss.reward.Bytes()) == 0
}

func (c *Calculator) checkToRun(ss *ExtensionSnapshotImpl) bool {
	if c.isGenesis() {
		return true
	}
	if c.isCalculating() {
		return false
	}
	return c.isResultSynced(ss)
}

func (c *Calculator) Run(ss *ExtensionSnapshotImpl) (err error) {
	if !c.checkToRun(ss) {
		return
	}
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
		err = errors.Wrapf(err, "Failed to save calculation result")
		return
	}
	finalTS := time.Now()

	log.Infof("Calculation time: total=%s prepare=%s blockProduce=%s voted=%s voting=%s postwork=%s",
		finalTS.Sub(startTS), prepareTS.Sub(startTS), bpTS.Sub(prepareTS),
		votedTS.Sub(bpTS), votingTS.Sub(votedTS), finalTS.Sub(votingTS),
	)
	log.Infof("Calculation statistics: BlockProduce=%s Voted=%s Voting=%s",
		c.stats.blockProduce.String(),
		c.stats.Voted.String(),
		c.stats.voting.String(),
	)
	return
}

func (c *Calculator) prepare(ss *ExtensionSnapshotImpl) error {
	var err error
	c.back = icstage.NewSnapshot(ss.database, ss.back.Bytes())
	c.base = icreward.NewSnapshot(ss.database, ss.reward.Bytes())
	c.temp = c.base.NewState()
	c.result = nil
	c.blockHeight = ss.c.currentBH
	c.stats.clear()

	// read global variables
	c.global, err = c.back.GetGlobal()
	if err != nil {
		return err
	}

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
			iScore = iScore.Added(claim.Value.Neg(claim.Value))
			if iScore.Value.Sign() == -1 {
				return errors.Errorf("Invalid negative I-Score for %s", addr.String())
			}
			if err := c.temp.SetIScore(addr, iScore); err != nil {
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

	switch t {
	case TypeBlockProduce:
		c.stats.increaseBlockProduce(reward)
	case TypeVoted:
		c.stats.increaseVoted(reward)
	case TypeVoting:
		c.stats.increaseVoting(reward)
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
	variable := varForBlockProduceReward(global.Irep, global.MainPRepCount)
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
		if err = c.updateIScore(v.addr, v.iScore, TypeBlockProduce); err != nil {
			return err
		}
	}

	return nil
}

func (c *Calculator) loadValidators() ([]*validator, error) {
	vs := make([]*validator, 0)
	count := 0

	prefix := icstage.ValidatorKey.Build()
	for iter := c.back.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return nil, err
		}
		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return nil, err
		}
		idx := int(intconv.BytesToInt64(keySplit[1]))
		if idx != count {
			return nil, errors.ErrExecutionFail
		}
		obj := icstage.ToValidator(o)
		vs = append(vs, newValidator(obj.Address))
		count += 1
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
	maxIndex := bp.VoteMask.BitLen()
	if bp.ProposerIndex >= vLen || maxIndex > vLen {
		return errors.Errorf("Can't find validator with %v", bp)
	}
	beta1Reward := new(big.Int).Set(variable)

	// for proposer
	proposer := validators[bp.ProposerIndex]
	proposer.iScore.Add(proposer.iScore, beta1Reward)

	// for validator
	if bp.VoteCount > 0 {
		beta1Validate := new(big.Int).Div(beta1Reward, big.NewInt(int64(bp.VoteCount)))
		for i := 0; i < maxIndex; i += 1 {
			if (bp.VoteMask.Bit(i)) != 0 {
				validators[i].iScore.Add(validators[i].iScore, beta1Validate)
			}
		}
	}

	return nil
}

// varForVotedReward return variable for P-Rep voted reward
// IISS 2.0
// 	variable = irep * electedPRepCount * IScoreICXRatio / (2 * MonthBlock)
// IISS 3.1
// 	variable = iglobal * iprep * IScoreICXRatio / (100 * TermPeriod)
func varForVotedReward(global *icstage.Global) *big.Int {
	v := new(big.Int)
	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion1 {
		g := global.GetV1()
		v.Mul(g.Irep, big.NewInt(int64(g.ElectedPRepCount)))
		v.Div(v, big.NewInt(int64(MonthBlock*2/IScoreICXRatio)))
	} else {
		g := global.GetV2()
		if g.OffsetLimit == 0 {
			return v
		}
		v.Mul(g.Iglobal, g.Iprep)
		v.Mul(v, BigIntIScoreICXRatio)
		v.Div(v, big.NewInt(int64(100*g.OffsetLimit)))
	}
	return v
}

func (c *Calculator) calculateVotedReward() error {
	offset := 0
	variable := varForVotedReward(c.global)
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
			vInfo.calculateReward(variable, keyOffset-offset)
			offset = keyOffset

			obj := icstage.ToEventEnable(o)
			vInfo.setEnable(obj.Target, obj.Enable)
			vInfo.updateTotalBondedDelegation()
		case icstage.TypeEventDelegation:
			obj := icstage.ToEventVote(o)
			vInfo.updateDelegated(obj.Votes)
		case icstage.TypeEventBond:
			obj := icstage.ToEventVote(o)
			vInfo.updateBonded(obj.Votes)
		}
	}
	if offset < c.global.GetOffsetLimit() {
		vInfo.calculateReward(variable, c.global.GetOffsetLimit()-offset)
	}

	// write result to temp and update statistics
	for key, prep := range vInfo.preps {
		var addr *common.Address
		addr, err = common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		if prep.voted.IsEmpty() {
			if err = c.temp.DeleteVoted(addr); err != nil {
				return err
			}
		} else {
			if err = c.temp.SetVoted(addr, prep.voted); err != nil {
				return err
			}
		}

		if prep.iScore.Sign() == 0 {
			continue
		}

		if err = c.updateIScore(addr, prep.iScore, TypeVoted); err != nil {
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
		data := newVotedData(obj)
		data.voted.UpdateBondedDelegation(bondRequirement)
		vInfo.addVotedData(addr, data)
	}
	vInfo.sort()
	vInfo.updateTotalBondedDelegation()

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
		if obj.Enable == false {
			// do not collect disabled P-Rep
			continue
		}
		prepInfo[string(addr.Bytes())] = new(pRepEnable)
	}

	return prepInfo, nil
}

// varForPRepDelegatingReward return variable for ICONist delegating reward
// IISS 2.0
// 	variable = rrep * IScoreICXRatio / YearBlock
// IISS 3.1
// 	variable = iglobal * ivoter * IScoreICXRatio / (100 * TermPeriod)
func varForVotingReward(global *icstage.Global) *big.Int {
	v := new(big.Int)
	iissVersion := global.GetIISSVersion()
	if iissVersion == icstate.IISSVersion1 {
		g := global.GetV1()
		v.Div(g.Rrep, big.NewInt(int64(YearBlock/IScoreICXRatio)))
	} else {
		g := global.GetV2()
		if g.OffsetLimit == 0 {
			return v
		}
		v.Mul(g.Iglobal, g.Ivoter)
		v.Mul(v, BigIntIScoreICXRatio)
		v.Div(v, big.NewInt(int64(100*g.OffsetLimit)))
	}
	return v
}

func (c *Calculator) calculateVotingReward() error {
	var err error

	if c.global.GetIISSVersion() == icstate.IISSVersion1 {
		err = c.calculateVotingRewardV1()
	} else {
		err = c.calculateVotingRewardV2()
	}

	return err
}

func (c *Calculator) calculateVotingRewardV1() error {
	prepInfo, err := c.loadPRepInfo()
	if err != nil {
		return err
	}
	variable := varForVotingReward(c.global)
	processedOffset := 0
	delegateMap := make(map[string]map[int]icstage.VoteList)

	for iter := c.back.Filter(icstage.EventKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}

		obj := o.(*icobject.Object)
		type_ := obj.Tag().Type()

		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		offset := int(intconv.BytesToInt64(keySplit[1]))
		switch type_ {
		case icstage.TypeEventEnable:
			// update prepInfo
			ee := icstage.ToEventEnable(obj)
			idx := string(ee.Target.Bytes())
			if _, ok := prepInfo[idx]; !ok {
				pe := new(pRepEnable)
				prepInfo[idx] = pe
			}
			if ee.Enable {
				prepInfo[idx].startOffset = offset
			} else {
				prepInfo[idx].endOffset = offset
			}
		case icstage.TypeEventDelegation:
			// TODO If delegateMap is too big, use other storage
			e := icstage.ToEventVote(obj)
			idx := string(e.From.Bytes())
			_, ok := delegateMap[idx]
			if !ok {
				delegateMap[idx] = make(map[int]icstage.VoteList)
			}
			delegateMap[idx][offset] = e.Votes
		}
	}
	if err := c.processDelegating(variable, processedOffset, c.offsetLimit, prepInfo, delegateMap); err != nil {
		return err
	}

	// from DELEGATE event
	if err := c.processDelegateEvent(variable, c.offsetLimit, prepInfo, delegateMap); err != nil {
		return err
	}
	return nil
}

// processDelegating calculator voting reward for delegating data.
// Do not calculate rewards for accounts that have modified their voting configuration with setDelegate and setBond.
// processDelegateEvent will calculate rewards for that account.
func (c *Calculator) processDelegating(
	variable *big.Int,
	from int,
	to int,
	prepInfo map[string]*pRepEnable,
	delegateMap map[string]map[int]icstage.VoteList,
) error {
	if variable.Sign() == 0 {
		return nil
	}

	prefix := icreward.DelegatingKey.Build()
	for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
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
		if _, ok := delegateMap[string(addr.Bytes())]; ok {
			continue
		}
		delegating := icreward.ToDelegating(o)
		reward := delegatingReward(variable, from, to, prepInfo, delegating.Delegations)
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}
	return nil
}

// processDelegateEvent calculate reward for account who got DELEGATE event
func (c *Calculator) processDelegateEvent(
	rrep *big.Int,
	to int,
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

		delegating, err := c.temp.GetDelegating(addr)
		if err != nil {
			return err
		}
		if delegating == nil {
			delegating = icreward.NewDelegating()
		}

		var start, end int
		for i := 0; i < len(events); i += 1 {
			end = offsets[i]
			ret := delegatingReward(rrep, start, end, prepInfo, delegating.Delegations)
			reward.Add(reward, ret)

			// update delegating
			votes := events[end]
			if err = delegating.ApplyVotes(votes); err != nil {
				return err
			}

			start = end
		}
		// calculate reward for last event
		ret := delegatingReward(rrep, start, to, prepInfo, delegating.Delegations)
		reward.Add(reward, ret)

		if err = c.writeDelegating(addr, delegating); err != nil {
			return nil
		}
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}
	return nil
}

func delegatingReward(
	rrep *big.Int,
	from int,
	to int,
	prepInfo map[string]*pRepEnable,
	delegations icstate.Delegations,
) *big.Int {
	// voting = rrep * delegations * period * IScoreICXRatio / year_block
	total := new(big.Int)
	for _, d := range delegations {
		s := from
		e := to
		if prep, ok := prepInfo[string(d.To().Bytes())]; ok {
			if prep.startOffset > s {
				s = prep.startOffset
			}
			if prep.endOffset != 0 && prep.endOffset < e {
				e = prep.endOffset
			}
			period := e - s
			reward := new(big.Int).Mul(rrep, d.Value.Value())
			reward.Mul(reward, big.NewInt(int64(period)))
			reward.Div(reward, bigIntBeta3Divider)
			total.Add(total, reward)
		}
	}
	return total
}

func (c *Calculator) calculateVotingRewardV2() error {
	lastOffset := 0
	variable := varForVotingReward(c.global)
	vInfo, err := c.loadVotedInfo()
	if err != nil {
		return err
	}

	for iter := c.back.Filter(icstage.EventKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}

		obj := o.(*icobject.Object)
		type_ := obj.Tag().Type()

		keySplit, err := containerdb.SplitKeys(key)
		if err != nil {
			return err
		}
		offset := int(intconv.BytesToInt64(keySplit[1]))
		if lastOffset != offset {
			if err = c.processVoting(variable, lastOffset, c.offsetLimit, vInfo); err != nil {
				return err
			}
			lastOffset = offset
		}
		switch type_ {
		case icstage.TypeEventEnable:
			event := icstage.ToEventEnable(o)
			vInfo.setEnable(event.Target, event.Enable)
		case icstage.TypeEventDelegation:
			event := icstage.ToEventVote(o)
			vInfo.updateDelegated(event.Votes)

			var delegating *icreward.Delegating
			delegating, err = c.temp.GetDelegating(event.From)
			if err != nil {
				return err
			}
			if delegating == nil {
				delegating = icreward.NewDelegating()
			}
			if err = delegating.ApplyVotes(event.Votes); err != nil {
				return err
			}
			if err = c.writeDelegating(event.From, delegating); err != nil {
				return nil
			}
		case icstage.TypeEventBond:
			event := icstage.ToEventVote(o)
			vInfo.updateBonded(event.Votes)
			var bonding *icreward.Bonding
			bonding, err = c.temp.GetBonding(event.From)
			if err != nil {
				return err
			}
			if bonding == nil {
				bonding = icreward.NewBonding()
			}
			if err = bonding.ApplyVotes(event.Votes); err != nil {
				return err
			}
			if err = c.writeBonding(event.From, bonding); err != nil {
				return nil
			}
		}
	}
	return nil
}

func (c *Calculator) processVoting(variable *big.Int, from int, to int, vInfo *votedInfo) error {
	if variable.Sign() == 0 {
		return nil
	}

	prefix := icreward.DelegatingKey.Build()
	for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
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
		delegating := icreward.ToDelegating(o)
		reward := votingReward(variable, from, to, vInfo, delegating.Delegations.Iterator())
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}

	prefix = icreward.BondingKey.Build()
	for iter := c.base.Filter(prefix); iter.Has(); iter.Next() {
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
		bonding := icreward.ToBonding(o)
		reward := votingReward(variable, from, to, vInfo, bonding.Bonds.Iterator())
		if err = c.updateIScore(addr, reward, TypeVoting); err != nil {
			return err
		}
	}
	return nil
}

func votingReward(
	variable *big.Int,
	from int,
	to int,
	vInfo *votedInfo,
	iter icstate.VotingIterator,
) *big.Int {
	// reward = variable * voting * period / total_voting
	//	variable = iglobal * ivoter * IScoreICXRatio / (100 * TermPeriod)
	if variable.Sign() == 0 ||
		from == to ||
		vInfo == nil || vInfo.totalVoted.Sign() == 0 ||
		iter == nil || !iter.Has() {
		return new(big.Int)
	}
	total := new(big.Int)
	reward := new(big.Int)
	period := big.NewInt(int64(to - from))
	base := new(big.Int).Mul(variable, period)
	for ; iter.Has(); iter.Next() {
		if voting, err := iter.Get(); err != nil {
			log.Errorf("Fail to iterating votings err=%+v", err)
		} else {
			if prep, ok := vInfo.preps[string(voting.To().Bytes())]; ok {
				if prep.voted.Enable != true {
					continue
				}
				reward.Mul(base, voting.Amount())
				reward.Div(reward, vInfo.totalVoted)
				total.Add(total, reward)
			}
		}
	}
	return total
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
	// save calculation result to MPT
	c.result = c.temp.GetSnapshot()
	if err = c.result.Flush(); err != nil {
		return
	}

	// save calculator Info.
	if err = c.Flush(); err != nil {
		return
	}
	return
}

func NewCalculator() *Calculator {
	return &Calculator{
		stats: newStatistics(),
	}
}

type validator struct {
	addr   *common.Address
	iScore *big.Int
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
}

func (vd *votedData) compare(vd2 *votedData) int {
	dv := new(big.Int)
	if vd.voted.Enable {
		dv = vd.voted.BondedDelegation
	}
	dv2 := new(big.Int)
	if vd2.voted.Enable {
		dv2 = vd2.voted.BondedDelegation
	}
	return dv.Cmp(dv2)
}

func (vd *votedData) Enable() bool {
	return vd.voted.Enable
}

func (vd *votedData) GetVotedAmount() *big.Int {
	return vd.voted.GetVoted()
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

func (vi *votedInfo) addVotedData(addr module.Address, data *votedData) {
	vi.preps[string(addr.Bytes())] = data
	if data.Enable() {
		vi.updateTotalVoted(data.GetVotedAmount())
	}
}

func (vi *votedInfo) setEnable(addr module.Address, enable bool) {
	if prep, ok := vi.preps[string(addr.Bytes())]; ok {
		if enable != prep.voted.Enable {
			if enable {
				vi.updateTotalVoted(prep.GetVotedAmount())
			} else {
				vi.updateTotalVoted(new(big.Int).Neg(prep.GetVotedAmount()))
			}
		}
		prep.voted.Enable = enable
	} else {
		dt := icreward.NewVoted()
		dt.Enable = enable
		data := newVotedData(dt)
		vi.addVotedData(addr, data)
	}
}

func (vi *votedInfo) updateDelegated(votes icstage.VoteList) {
	for _, vote := range votes {
		if data, ok := vi.preps[string(vote.To().Bytes())]; ok {
			current := data.voted.Delegated
			current.Add(current, vote.Amount())
			if data.Enable() {
				vi.updateTotalVoted(vote.Amount())
			}
		} else {
			voted := icreward.NewVoted()
			voted.Delegated.Set(vote.Value)
			data = newVotedData(voted)
			vi.addVotedData(vote.To(), data)
		}
	}
}

func (vi *votedInfo) updateBonded(votes icstage.VoteList) {
	for _, vote := range votes {
		if data, ok := vi.preps[string(vote.To().Bytes())]; ok {
			current := data.voted.Bonded
			current.Add(current, vote.Value)
			if data.Enable() {
				vi.updateTotalVoted(vote.Amount())
			}
		} else {
			voted := icreward.NewVoted()
			voted.Bonded.Set(vote.Value)
			data = newVotedData(voted)
			vi.addVotedData(vote.To(), data)
		}
	}
}

func (vi *votedInfo) sort() {
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
		return tempKeys[i].compare(&tempKeys[j]) > 0
	})

	rank := make([]string, size)
	for idx, v := range tempKeys {
		rank[idx] = temp[v]
	}
	vi.rank = rank
}

func (vi *votedInfo) updateTotalBondedDelegation() {
	total := new(big.Int)
	for i, address := range vi.rank {
		if i == vi.maxRankForReward {
			break
		}
		delegated := vi.preps[address].voted
		if delegated.Enable {
			total.Add(total, delegated.BondedDelegation)
		}
	}
	vi.totalBondedDelegation = total
}

func (vi *votedInfo) updateTotalVoted(amount *big.Int) {
	vi.totalVoted.Add(vi.totalVoted, amount)
}

// calculateReward calculate P-Rep voted reward
func (vi *votedInfo) calculateReward(variable *big.Int, period int) {
	if variable.Sign() == 0 || period == 0 {
		return
	}
	if vi.totalBondedDelegation.Sign() == 0 {
		return
	}
	// reward = variable * period * bondedDelegation / totalBondedDelegation
	base := new(big.Int).Mul(variable, big.NewInt(int64(period)))
	for i, addr := range vi.rank {
		if i == vi.maxRankForReward {
			break
		}
		prep := vi.preps[addr]

		reward := new(big.Int).Set(base)
		reward.Mul(reward, prep.voted.BondedDelegation)
		reward.Div(reward, vi.totalBondedDelegation)

		prep.iScore.Add(prep.iScore, reward)
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

type statistics struct {
	blockProduce *big.Int
	Voted        *big.Int
	voting       *big.Int
}

func newStatistics() *statistics {
	return &statistics{
		blockProduce: new(big.Int),
		Voted:        new(big.Int),
		voting:       new(big.Int),
	}
}

func (s *statistics) equal(s2 *statistics) bool {
	return s.blockProduce.Cmp(s2.blockProduce) == 0 &&
		s.Voted.Cmp(s2.Voted) == 0 &&
		s.voting.Cmp(s2.voting) == 0
}

func (s *statistics) clear() {
	s.blockProduce.SetInt64(0)
	s.Voted.SetInt64(0)
	s.voting.SetInt64(0)
}

func increaseStats(src *big.Int, amount *big.Int) *big.Int {
	if src == nil {
		src = new(big.Int).Set(amount)
	} else {
		src.Add(src, amount)
	}
	return src
}

func (s *statistics) increaseBlockProduce(amount *big.Int) {
	s.blockProduce = increaseStats(s.blockProduce, amount)
}

func (s *statistics) increaseVoted(amount *big.Int) {
	s.Voted = increaseStats(s.Voted, amount)
}

func (s *statistics) increaseVoting(amount *big.Int) {
	s.voting = increaseStats(s.voting, amount)
}
func (s *statistics) totalReward() *big.Int {
	reward := new(big.Int)
	reward.Add(s.blockProduce, s.Voted)
	reward.Add(reward, s.voting)
	return reward
}
