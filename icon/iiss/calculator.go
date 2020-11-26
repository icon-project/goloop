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
	"math/big"
	"math/bits"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/service/state"
)

const (
	NumMainPReps    = 22
	NumSubPReps     = 78
	NumMainSubPReps = NumMainPReps + NumSubPReps

	DayBlock   = 24 * 60 * 60 / 2
	MonthBlock = DayBlock * 30
	YearBlock  = MonthBlock * 12

	IScoreICXRatio = 1000
)

var (
	bigIntBeta1Divider = big.NewInt(int64(MonthBlock * 2 * 2))
	bigIntBeta2Divider = big.NewInt(int64(MonthBlock * 2))
	bigIntBeta3Divider = big.NewInt(int64(YearBlock / IScoreICXRatio))
)

type Calculator struct {
	snapshotImpl *extensionSnapshotImpl
	back         *icstage.Snapshot
	base         *icreward.Snapshot
	temp         *icreward.State
	result       *icreward.Snapshot

	irep       *big.Int
	rrep       *big.Int
	validators []*validator

	offsetLimit int
}

func (c *Calculator) run() error {
	if err := c.prepare(); err != nil {
		return errors.Wrapf(err, "Failed to prepare calculator")
	}

	if err := c.calculateBeta1(); err != nil {
		return errors.Wrapf(err, "Failed to calculate beta1")
	}

	if err := c.calculateBeta2(); err != nil {
		return errors.Wrapf(err, "Failed to calculate beta2")
	}

	if err := c.calculateBeta3(); err != nil {
		return errors.Wrapf(err, "Failed to calculate beta3")
	}

	c.postWork()

	return nil
}

func (c *Calculator) prepare() error {
	c.base = c.snapshotImpl.reward
	c.back = c.snapshotImpl.back

	// make temp state for calculation
	c.temp = c.snapshotImpl.reward.NewState()

	// read old values from temp
	global, err := c.temp.GetGlobal()
	if err != nil {
		return err
	}
	if global != nil {
		c.irep = global.Irep
		c.rrep = global.Rrep
	} else {
		c.irep = new(big.Int)
		c.rrep = new(big.Int)
	}

	vs, err := c.temp.GetValidators()
	if err != nil {
		return err
	}
	if vs == nil {
		c.validators = make([]*validator, 0)
	} else {
		c.validators = make([]*validator, len(vs.Addresses), len(vs.Addresses))
		for i, addr := range vs.Addresses {
			c.validators[i] = newValidator(addr)
		}
	}

	// read offsetLimit from back
	c.offsetLimit, err = c.back.GetOffsetLimit()
	if err != nil {
		return err
	}

	// write claim data to temp
	if err := c.processClaim(); err != nil {
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
			iScore = iScore.Added(claim.Value)
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

func (c *Calculator) calculateBeta1() error {
	var err error
	offset := 0
	irep := c.irep
	validators := c.validators

	eventPrefix := icstage.EventKey.Build()
	for iter := c.back.Filter(eventPrefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		if o.(*icobject.Object).Tag().Type() == icstage.TypeEventPeriod {
			keySplit, _ := containerdb.SplitKeys(key)
			keyOffset := int(intconv.BytesToInt64(keySplit[1]))
			for ; offset < keyOffset; offset += 1 {
				validators, err = c.processBlockProduce(irep, offset, validators)
				if err != nil {
					return err
				}
			}
			obj := icstage.ToEventPeriod(o)
			irep = obj.Irep
		}
	}
	for ; offset < c.offsetLimit; offset += 1 {
		validators, err = c.processBlockProduce(irep, offset, validators)
		if err != nil {
			return err
		}
	}

	vs := new(icreward.Validators)
	for _, v := range validators {
		is, err := c.temp.GetIScore(v.addr)
		if err != nil {
			return nil
		}
		if err := c.temp.SetIScore(v.addr, is.Added(v.iScore)); err != nil {
			return err
		}
		vs.Add(v.addr)
	}
	if err := c.temp.SetValidators(vs); err != nil {
		return err
	}
	return nil
}

func (c *Calculator) processBlockProduce(irep *big.Int, offset int, validators []*validator) ([]*validator, error) {
	beta1Reward := new(big.Int).Div(irep, bigIntBeta1Divider)
	if irep.Sign() == 0 {
		return validators, nil
	}
	prefix := icstage.BlockProduceKey.Append(offset).Build()
	for iter := c.back.Filter(prefix); iter.Has(); iter.Next() {
		o, _, err := iter.Get()
		if err != nil {
			return validators, err
		}
		type_ := o.(*icobject.Object).Tag().Type()
		if type_ == icstage.TypeValidator {
			// validators will be changed, update temp with gathered I-Score
			for _, v := range validators {
				is, err := c.temp.GetIScore(v.addr)
				if err != nil {
					return validators, err
				}
				if err := c.temp.SetIScore(v.addr, is.Added(v.iScore)); err != nil {
					return validators, err
				}
			}
			// load new validators
			obj := icstage.ToValidators(o)
			nvs := make([]*validator, 0)
			for _, nv := range obj.Addresses {
				v := newValidator(nv)
				nvs = append(nvs, v)
			}
			validators = nvs
		} else {
			v := icstage.ToBlockVotes(o)
			// Beta1 for generator
			proposer := validators[v.ProposerIndex]
			proposer.iScore.Add(proposer.iScore, beta1Reward)
			// Beta1 for validator
			beta1Validate := new(big.Int)
			beta1Validate.Div(beta1Reward, big.NewInt(int64(v.VoteCount)))
			maxIndex := bits.Len(uint(v.VoteMask))
			for i := 0; i <= maxIndex; i += 1 {
				if (v.VoteMask & (1 << i)) != 0 {
					validators[i].iScore.Add(validators[i].iScore, beta1Validate)
				}
			}
		}
	}

	return validators, nil
}

func (c *Calculator) calculateBeta2() error {
	offset := 0
	irep := c.irep

	delegated, err := c.loadDelegated()
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
		if type_ == icstage.TypeEventPeriod || type_ == icstage.TypeEventEnable {
			delegated.calculateReward(irep, keyOffset-offset)
			offset = keyOffset
			if type_ == icstage.TypeEventPeriod {
				obj := icstage.ToEventPeriod(o)
				irep = obj.Irep
				delegated.updateSnapshot()
				delegated.updateTotal()
			} else {
				obj := icstage.ToEventEnable(o)
				delegated.setEnable(obj.Target, obj.Enable)
				delegated.updateTotal()
			}
		} else if type_ == icstage.TypeEventDelegation {
			obj := icstage.ToEventDelegation(o)
			delegated.updateCurrent(obj.Delegations)
		}
	}
	if offset < c.offsetLimit {
		delegated.calculateReward(irep, c.offsetLimit-offset)
	}

	// write result to temp
	for addr, prep := range delegated.preps {
		if prep.delegated.IsEmpty() {
			if err := c.temp.DeleteDelegated(&addr); err != nil {
				return err
			}
		} else {
			if err := c.temp.SetDelegated(&addr, prep.delegated); err != nil {
				return err
			}
		}

		is, err := c.temp.GetIScore(&addr)
		if err != nil {
			return nil
		}
		if err := c.temp.SetIScore(&addr, is.Added(prep.iScore)); err != nil {
			return err
		}
	}

	c.irep = irep
	return nil
}

func (c *Calculator) loadDelegated() (*delegated, error) {
	d := newDelegated()

	prefix := icreward.DelegatedKey.Build()
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
		obj := icreward.ToDelegated(o)
		data := newDelegatedData(obj)
		d.addDelegatedData(addr, data)
	}
	d.updateTotal()

	return d, nil
}

func (c *Calculator) calculateBeta3() error {
	rrep := c.rrep
	processedOffset := 0

	prepInfo, err := c.loadPRepInfo()
	if err != nil {
		return err
	}

	delegationMap := make(map[common.Address]map[int]icstate.Delegations)

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
		case icstage.TypeEventPeriod:
			if err := c.processDelegating(rrep, processedOffset, offset, prepInfo, delegationMap); err != nil {
				return err
			}
			processedOffset = offset
			delegationMap = make(map[common.Address]map[int]icstate.Delegations)
		case icstage.TypeEventEnable:
			// update prepInfo
			ee := icstage.ToEventEnable(obj)
			if _, ok := prepInfo[*ee.Target]; !ok {
				pe := new(pRepEnable)
				prepInfo[*ee.Target] = pe
			}
			if ee.Enable {
				prepInfo[*ee.Target].startOffset = offset
			} else {
				prepInfo[*ee.Target].endOffset = offset
			}
		case icstage.TypeEventDelegation:
			// TODO If delegationsInfo is too big, use other storage
			// gather DELEGATE event
			ed := icstage.ToEventDelegation(obj)
			if delegationMap[*ed.From] == nil {
				delegationMap[*ed.From] = make(map[int]icstate.Delegations)
			}
			delegationMap[*ed.From][offset] = ed.Delegations
		}
	}
	if processedOffset < c.offsetLimit {
		if err := c.processDelegating(rrep, processedOffset, c.offsetLimit, prepInfo, delegationMap); err != nil {
			return err
		}
	}
	c.rrep = rrep
	return nil
}

// loadPRepInfo load P-Rep status from base
func (c *Calculator) loadPRepInfo() (map[common.Address]*pRepEnable, error) {
	prepInfo := make(map[common.Address]*pRepEnable)
	for iter := c.base.Filter(icreward.DelegatedKey.Build()); iter.Has(); iter.Next() {
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
		obj := icreward.ToDelegated(o)
		if obj.Enable == false {
			// do not collect disabled P-Rep
			continue
		}
		prepInfo[*addr] = new(pRepEnable)
	}

	return prepInfo, nil
}

func (c *Calculator) processDelegating(
	rrep *big.Int,
	from int,
	to int,
	prepInfo map[common.Address]*pRepEnable,
	delegationMap map[common.Address]map[int]icstate.Delegations,
) error {
	if rrep.Sign() == 0 {
		return nil
	}
	// calc reward for delegatings
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
		if _, ok := delegationMap[*addr]; ok {
			// if delegation is modified, calculate later
			continue
		} else {
			reward := delegatingReward(rrep, from, to, prepInfo, delegating.Delegations)

			iScore, err := c.temp.GetIScore(addr)
			if err != nil {
				return err
			}
			if err := c.temp.SetIScore(addr, iScore.Added(reward)); err != nil {
				return err
			}
		}
	}
	// calculate reward for account who got DELEGATE event
	for addr, ds := range delegationMap { // each account
		iScore := new(icreward.IScore)
		iScore.Value = new(big.Int)
		offsets := make([]int, 0, len(ds))
		for offset, _ := range ds {
			offsets = append(offsets, offset)
		}
		// sort with offset
		sort.Ints(offsets)
		end := to
		// calculate from last DELEGATE event
		for i := len(ds) - 1; i >= 0; i -= 1 {
			start := offsets[i]
			delegations := ds[start]
			ret := delegatingReward(rrep, start, end, prepInfo, delegations)
			iScore.Value.Add(iScore.Value, ret)
			end = start
		}
		// calculate reward from temp.delegations to first DELEGATE event
		delegating, err := c.temp.GetDelegating(&addr)
		if err != nil {
			return err
		}
		if delegating != nil {
			ret := delegatingReward(
				rrep, from, offsets[0], prepInfo, delegating.Delegations)
			iScore.Value.Add(iScore.Value, ret)
		}
		// write last DELEGATE event and iScore
		if delegating == nil {
			delegating = icreward.NewDelegating()
		}
		delegating.Delegations = ds[offsets[len(offsets)-1]]
		if delegating.IsEmpty() {
			if err := c.temp.DeleteDelegating(&addr); err != nil {
				return err
			}
		} else {
			if err := c.temp.SetDelegating(&addr, delegating); err != nil {
				return err
			}
		}
		if err := c.temp.SetIScore(&addr, iScore); err != nil {
			return err
		}
	}
	return nil
}

func delegatingReward(
	rrep *big.Int,
	from int,
	to int,
	prepInfo map[common.Address]*pRepEnable,
	delegations icstate.Delegations,
) *big.Int {
	// beta3 = rrep * delegations * period * IScoreICXRatio / year_block
	total := new(big.Int)
	for _, d := range delegations {
		s := from
		e := to
		if prep, ok := prepInfo[*d.Address]; ok {
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

func (c *Calculator) postWork() {
	g := new(icreward.Global)
	g.Irep = c.irep
	g.Rrep = c.rrep
	c.temp.SetGlobal(g)
	c.result = c.temp.GetSnapshot()
	// TODO return hash to caller via channel
}

func newCalculator(ess state.ExtensionSnapshot) *Calculator {
	return &Calculator{
		snapshotImpl: ess.(*extensionSnapshotImpl),
	}
}

func RunCalculator(ess state.ExtensionSnapshot) error {
	calculator := newCalculator(ess)
	if err := calculator.run(); err != nil {
		return err
	}

	return nil
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

type delegatedData struct {
	delegated *icreward.Delegated
	iScore    *big.Int
}

func (dd *delegatedData) compare(dd2 *delegatedData) int {
	dv := new(big.Int)
	if dd.delegated.Enable {
		dv = dd.delegated.Snapshot
	}
	dv2 := new(big.Int)
	if dd2.delegated.Enable {
		dv2 = dd2.delegated.Snapshot
	}
	return dv.Cmp(dv2)
}

func newDelegatedData(d *icreward.Delegated) *delegatedData {
	return &delegatedData{
		delegated: d,
		iScore:    new(big.Int),
	}
}

type delegated struct {
	total *big.Int // total delegated amount of top 100 P-Reps
	rank  []common.Address
	preps map[common.Address]*delegatedData
}

func (d *delegated) maxRankForReward() int {
	return NumMainSubPReps
}

func (d *delegated) addDelegatedData(addr *common.Address, data *delegatedData) {
	d.preps[*addr] = data
}

func (d *delegated) calculateReward(irep *big.Int, period int) {
	if irep.Sign() == 0 {
		return
	}
	// beta2 = irep * delegated * period / (2 * month_block * total_delegated)
	base := new(big.Int).Mul(irep, big.NewInt(int64(period)))
	for i, addr := range d.rank {
		if i == d.maxRankForReward() {
			break
		}
		prep := d.preps[addr]

		reward := new(big.Int).Mul(base, prep.delegated.Snapshot)
		reward.Div(reward, bigIntBeta2Divider)
		reward.Div(reward, d.total)

		prep.iScore.Add(prep.iScore, reward)
	}
}

func (d *delegated) setEnable(addr *common.Address, enable bool) {
	if _, ok := d.preps[*addr]; ok {
		d.preps[*addr].delegated.Enable = enable
	} else {
		dt := icreward.NewDelegated()
		dt.Enable = enable
		data := newDelegatedData(dt)
		d.addDelegatedData(addr, data)
	}
}

func (d *delegated) updateCurrent(delegations icstate.Delegations) {
	for _, delegation := range delegations {
		if _, ok := d.preps[*delegation.Address]; ok {
			current := d.preps[*delegation.Address].delegated.Current
			current.Add(current, delegation.Value.Value())
		} else {
			dt := icreward.NewDelegated()
			dt.Current.Set(delegation.Value.Value())
			data := newDelegatedData(dt)
			d.addDelegatedData(delegation.Address, data)
		}
	}
}

func (d *delegated) updateSnapshot() {
	for _, prep := range d.preps {
		prep.delegated.Snapshot.Set(prep.delegated.Current)
	}
}

func (d *delegated) updateTotal() {
	// sort prep list
	size := len(d.preps)
	temp := make(map[delegatedData]common.Address, size)
	tempKeys := make([]delegatedData, size)
	i := 0
	for address, data := range d.preps {
		temp[*data] = address
		tempKeys[i] = *data
		i += 1
	}
	sort.Slice(tempKeys, func(i, j int) bool {
		return tempKeys[i].compare(&tempKeys[j]) > 0
	})

	rank := make([]common.Address, size)
	for i, v := range tempKeys {
		rank[i] = temp[v]
	}
	d.rank = rank

	// update total
	total := new(big.Int)
	for i, address := range d.rank {
		if i == d.maxRankForReward() {
			break
		}
		total.Add(total, d.preps[address].delegated.Snapshot)
	}
	d.total = total
}

func newDelegated() *delegated {
	return &delegated{
		total: new(big.Int),
		preps: make(map[common.Address]*delegatedData),
	}
}

type pRepEnable struct {
	startOffset int
	endOffset   int
}
