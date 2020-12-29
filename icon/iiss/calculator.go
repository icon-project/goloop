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
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

const (
	NumMainPReps    = 22
	NumSubPReps     = 78
	NumMainSubPReps = NumMainPReps + NumSubPReps

	DayBlock   = 24 * 60 * 60 / 2
	MonthBlock = DayBlock * 30
	YearBlock  = MonthBlock * 12

	IScoreICXRatio = 1000

	keyCalculator = "iiss.calculator"
)

var (
	BigIntIScoreICXRation = big.NewInt(int64(IScoreICXRatio))
	bigIntBeta1Divider    = big.NewInt(int64(MonthBlock * 2 * 2))
	bigIntBeta2Divider    = big.NewInt(int64(MonthBlock * 2))
	bigIntBeta3Divider    = big.NewInt(int64(YearBlock / IScoreICXRatio))
)

type Calculator struct {
	dbase db.Database

	result      *icreward.Snapshot
	blockHeight int64
	stats       *statistics

	back        *icstage.Snapshot
	base        *icreward.Snapshot
	temp        *icreward.State
	irep        *big.Int
	rrep        *big.Int
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
		c.stats.beta1,
		c.stats.beta2,
		c.stats.beta3,
	)
}

func (c *Calculator) RLPDecodeSelf(d codec.Decoder) error {
	var hash []byte
	if err := d.DecodeListOf(
		&hash,
		&c.blockHeight,
		&c.stats.beta1,
		&c.stats.beta2,
		&c.stats.beta3,
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

	if err = c.calculateBeta1(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate beta1")
		return
	}
	beta1TS := time.Now()

	if err = c.calculateBeta2(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate beta2")
		return
	}
	beta2TS := time.Now()

	if err = c.calculateBeta3(); err != nil {
		err = errors.Wrapf(err, "Failed to calculate beta3")
		return
	}
	beta3TS := time.Now()

	if err = c.postWork(); err != nil {
		err = errors.Wrapf(err, "Failed to save calculation result")
		return
	}
	finalTS := time.Now()

	log.Infof("Calculation time: total=%s prepare=%s beta1=%s beta2=%s beta3=%s",
		finalTS.Sub(startTS), prepareTS.Sub(startTS),
		beta1TS.Sub(prepareTS), beta2TS.Sub(beta1TS), beta3TS.Sub(beta2TS),
	)
	log.Infof("Calculation statistics: Beta1=%s Beta2=%s Beta3=%s",
		c.stats.beta1.String(),
		c.stats.beta2.String(),
		c.stats.beta3.String(),
	)
	return
}

func (c *Calculator) prepare(ss *ExtensionSnapshotImpl) error {
	c.back = icstage.NewSnapshot(ss.database, ss.back.Bytes())
	c.base = icreward.NewSnapshot(ss.database, ss.reward.Bytes())
	c.temp = c.base.NewState()
	c.result = nil
	c.blockHeight = ss.c.currentBH
	c.stats.clear()

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

func (c *Calculator) calculateBeta1() error {
	var err error
	offset := 0
	irep := c.irep
	validators, err := c.loadValidators()
	if err != nil {
		return err
	}

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
				if err := c.processBlockProduce(irep, offset, validators); err != nil {
					return err
				}
			}
			obj := icstage.ToEventPeriod(o)
			irep = obj.Irep
		}
	}
	for ; offset < c.offsetLimit; offset += 1 {
		err = c.processBlockProduce(irep, offset, validators)
		if err != nil {
			return err
		}
	}

	for _, v := range validators {
		is, err := c.temp.GetIScore(v.addr)
		if err != nil {
			return nil
		}
		if err := c.temp.SetIScore(v.addr, is.Added(v.iScore)); err != nil {
			return err
		}
		c.stats.increaseBeta1(v.iScore)
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

func (c *Calculator) processBlockProduce(irep *big.Int, offset int, validators []*validator) error {
	beta1Reward := new(big.Int).Div(irep, bigIntBeta1Divider)
	if irep.Sign() == 0 {
		return nil
	}
	bp, err := c.back.GetBlockProduce(offset)
	if err != nil {
		return err
	}
	if bp == nil {
		return nil
	}

	// for proposer
	proposer := validators[bp.ProposerIndex]
	proposer.iScore.Add(proposer.iScore, beta1Reward)

	// for validator
	if bp.VoteCount > 0 {
		beta1Validate := new(big.Int)
		beta1Validate.Div(beta1Reward, big.NewInt(int64(bp.VoteCount)))
		maxIndex := bp.VoteMask.BitLen()
		for i := 0; i <= maxIndex; i += 1 {
			if (bp.VoteMask.Bit(i)) != 0 {
				validators[i].iScore.Add(validators[i].iScore, beta1Validate)
			}
		}
	}

	return nil
}

func (c *Calculator) calculateBeta2() error {
	offset := 0
	irep := c.irep

	lDelegated, err := c.loadDelegated()
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
			lDelegated.calculateReward(irep, keyOffset-offset)
			offset = keyOffset
			if type_ == icstage.TypeEventPeriod {
				obj := icstage.ToEventPeriod(o)
				irep = obj.Irep
				lDelegated.updateSnapshot()
				lDelegated.updateTotal()
			} else {
				obj := icstage.ToEventEnable(o)
				lDelegated.setEnable(obj.Target, obj.Enable)
				lDelegated.updateTotal()
			}
		} else if type_ == icstage.TypeEventDelegation {
			obj := icstage.ToEventDelegation(o)
			lDelegated.updateCurrent(obj.Delegations)
		}
	}
	if offset < c.offsetLimit {
		lDelegated.calculateReward(irep, c.offsetLimit-offset)
	}

	// write result to temp
	for key, prep := range lDelegated.preps {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		if prep.delegated.IsEmpty() {
			if err = c.temp.DeleteDelegated(addr); err != nil {
				return err
			}
		} else {
			if err = c.temp.SetDelegated(addr, prep.delegated); err != nil {
				return err
			}
		}

		if prep.iScore.Sign() == 0 {
			continue
		}

		is, err := c.temp.GetIScore(addr)
		if err != nil {
			return nil
		}
		if err := c.temp.SetIScore(addr, is.Added(prep.iScore)); err != nil {
			return err
		}
		c.stats.increaseBeta2(prep.iScore)
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

	delegationMap := make(map[string]map[int]icstate.Delegations)

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
			ep := icstage.ToEventPeriod(obj)
			if err := c.processDelegating(rrep, processedOffset, offset, prepInfo, delegationMap); err != nil {
				return err
			}
			rrep = ep.Rrep
			processedOffset = offset
			delegationMap = make(map[string]map[int]icstate.Delegations)
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
			// TODO If delegationsInfo is too big, use other storage
			// gather DELEGATE event
			ed := icstage.ToEventDelegation(obj)
			idx := string(ed.From.Bytes())
			_, ok := delegationMap[idx]
			if !ok {
				delegationMap[idx] = make(map[int]icstate.Delegations)
			}
			delegationMap[idx][offset] = ed.Delegations
		}
	}
	if processedOffset < c.offsetLimit {
		if err = c.processDelegating(rrep, processedOffset, c.offsetLimit, prepInfo, delegationMap); err != nil {
			return err
		}
	}
	c.rrep = rrep
	return nil
}

// loadPRepInfo load P-Rep status from base
func (c *Calculator) loadPRepInfo() (map[string]*pRepEnable, error) {
	prepInfo := make(map[string]*pRepEnable)
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
		prepInfo[string(addr.Bytes())] = new(pRepEnable)
	}

	return prepInfo, nil
}

func (c *Calculator) processDelegating(
	rrep *big.Int,
	from int,
	to int,
	prepInfo map[string]*pRepEnable,
	delegationMap map[string]map[int]icstate.Delegations,
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
		if _, ok := delegationMap[string(addr.Bytes())]; ok {
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
			c.stats.increaseBeta3(reward)
		}
	}
	// calculate reward for account who got DELEGATE event
	for key, ds := range delegationMap { // each account
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			return err
		}
		iScore := icreward.NewIScore()
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
		delegating, err := c.temp.GetDelegating(addr)
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
			if err = c.temp.DeleteDelegating(addr); err != nil {
				return err
			}
		} else {
			if err = c.temp.SetDelegating(addr, delegating); err != nil {
				return err
			}
		}
		if err = c.temp.SetIScore(addr, iScore); err != nil {
			return err
		}
		c.stats.increaseBeta3(iScore.Value)
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
	// beta3 = rrep * delegations * period * IScoreICXRatio / year_block
	total := new(big.Int)
	for _, d := range delegations {
		s := from
		e := to
		if prep, ok := prepInfo[string(d.Address.Bytes())]; ok {
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

func (c *Calculator) postWork() (err error) {
	// save values for next calculation
	g := new(icreward.Global)
	g.Irep = c.irep
	g.Rrep = c.rrep
	if err = c.temp.SetGlobal(g); err != nil {
		return
	}

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
	rank  []string
	preps map[string]*delegatedData
}

func (d *delegated) maxRankForReward() int {
	return NumMainSubPReps
}

func (d *delegated) addDelegatedData(addr *common.Address, data *delegatedData) {
	d.preps[string(addr.Bytes())] = data
}

func (d *delegated) calculateReward(irep *big.Int, period int) {
	if irep.Sign() == 0 || period == 0 {
		return
	}
	if d.total.Sign() == 0 {
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
	if prep, ok := d.preps[string(addr.Bytes())]; ok {
		prep.delegated.Enable = enable
	} else {
		dt := icreward.NewDelegated()
		dt.Enable = enable
		data := newDelegatedData(dt)
		d.addDelegatedData(addr, data)
	}
}

func (d *delegated) updateCurrent(delegations icstate.Delegations) {
	for _, delegation := range delegations {
		if data, ok := d.preps[string(delegation.Address.Bytes())]; ok {
			current := data.delegated.Current
			current.Add(current, delegation.Value.Value())
		} else {
			dt := icreward.NewDelegated()
			dt.Current.Set(delegation.Value.Value())
			data = newDelegatedData(dt)
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
	temp := make(map[delegatedData]string, size)
	tempKeys := make([]delegatedData, size)
	i := 0
	for key, data := range d.preps {
		temp[*data] = key
		tempKeys[i] = *data
		i += 1
	}
	sort.Slice(tempKeys, func(i, j int) bool {
		return tempKeys[i].compare(&tempKeys[j]) > 0
	})

	rank := make([]string, size)
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
		preps: make(map[string]*delegatedData),
	}
}

type pRepEnable struct {
	startOffset int
	endOffset   int
}

type statistics struct {
	beta1 *big.Int
	beta2 *big.Int
	beta3 *big.Int
}

func newStatistics() *statistics {
	return &statistics{
		beta1: new(big.Int),
		beta2: new(big.Int),
		beta3: new(big.Int),
	}
}

func (s *statistics) equal(s2 *statistics) bool {
	return s.beta1.Cmp(s2.beta1) == 0 &&
		s.beta2.Cmp(s2.beta2) == 0 &&
		s.beta3.Cmp(s2.beta3) == 0
}

func (s *statistics) clear() {
	s.beta1.SetInt64(0)
	s.beta2.SetInt64(0)
	s.beta3.SetInt64(0)
}

func increaseStats(src *big.Int, amount *big.Int) *big.Int {
	if src == nil {
		src = new(big.Int).Set(amount)
	} else {
		src.Add(src, amount)
	}
	return src
}

func (s *statistics) increaseBeta1(amount *big.Int) {
	s.beta1 = increaseStats(s.beta1, amount)
}

func (s *statistics) increaseBeta2(amount *big.Int) {
	s.beta2 = increaseStats(s.beta2, amount)
}

func (s *statistics) increaseBeta3(amount *big.Int) {
	s.beta3 = increaseStats(s.beta3, amount)
}
func (s *statistics) totalReward() *big.Int {
	reward := new(big.Int)
	reward.Add(s.beta1, s.beta2)
	reward.Add(reward, s.beta3)
	return reward
}
