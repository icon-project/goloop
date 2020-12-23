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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

func TestCalculator(t *testing.T) {
	database := db.NewMapDB()
	c := new(Calculator)

	err := c.Init(database)
	assert.NoError(t, err)
	assert.Equal(t, database, c.dbase)
	assert.Equal(t, int64(0), c.blockHeight)

	c.blockHeight = 100
	err = c.Flush()
	assert.NoError(t, err)

	c2 := new(Calculator)
	err = c2.Init(database)
	assert.NoError(t, err)
	assert.Equal(t, c.dbase, c2.dbase)
	assert.Equal(t, c.blockHeight, c2.blockHeight)
}

func MakeCalculator(database db.Database, back *icstage.Snapshot) *Calculator {
	c := new(Calculator)
	c.back = back
	c.base = icreward.NewSnapshot(database, nil)
	c.temp = c.base.NewState()

	return c
}

func TestCalculator_processClaim(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	v1 := int64(100)
	v2 := int64(200)

	type args struct {
		addr  *common.Address
		value *big.Int
	}

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			"Add Claim 100",
			args{
				addr1,
				big.NewInt(v1),
			},
			v1,
		},
		{
			"Add Claim 200 to new address",
			args{
				addr2,
				big.NewInt(v2),
			},
			v2,
		},
	}

	c := MakeCalculator(database, s.GetSnapshot())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore := icreward.NewIScore()
			iScore.Value.Set(args.value)
			err := c.temp.SetIScore(args.addr, iScore)
			assert.NoError(t, err)

			err = s.AddIScoreClaim(args.addr, args.value)
			assert.NoError(t, err)
		})
	}

	err := c.processClaim()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, 0, args.value.Cmp(iScore.Value))
		})
	}
}

func TestCalculator_processBlockProduce(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)

	offset1 := 0

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	type args struct {
		offset   int
		proposer module.Address
		voters   []module.Address
	}

	datas := []struct {
		name string
		args args
	}{
		{
			"block produce 1",
			args{
				offset:   offset1,
				proposer: addr1,
				voters:   []module.Address{addr1, addr2, addr3, addr4},
			},
		},
		{
			"block produce 2",
			args{
				offset:   offset1 + 1,
				proposer: addr2,
				voters:   []module.Address{addr1, addr2, addr3, addr4},
			},
		},
		{
			"block produce 3",
			args{
				offset:   offset1 + 2,
				proposer: addr5,
				voters:   []module.Address{addr1, addr4, addr5},
			},
		},
	}
	for _, data := range datas {
		a := data.args
		err := s.AddBlockProduce(a.offset, a.proposer, a.voters)
		assert.NoError(t, err)
	}

	c := MakeCalculator(database, s.GetSnapshot())
	irep := big.NewInt(int64(YearBlock * IScoreICXRatio))
	rewardGenerate := new(big.Int).Div(irep, bigIntBeta1Divider).Int64()
	rewardValidate := new(big.Int).Div(irep, bigIntBeta1Divider).Int64()
	vs, err := c.loadValidators()
	assert.NoError(t, err)

	for _, data := range datas {
		a := data.args
		err = c.processBlockProduce(irep, a.offset, vs)
		assert.NoError(t, err)
	}

	tests := []struct {
		name  string
		idx   int
		addr  *common.Address
		wants int64
	}{
		{
			"addr1",
			0,
			addr1,
			rewardGenerate +
				rewardValidate/4 +
				rewardValidate/4 +
				rewardValidate/3,
		},
		{
			"addr2",
			1,
			addr2,
			rewardValidate/4 +
				rewardGenerate +
				rewardValidate/4,
		},
		{
			"addr3",
			2,
			addr3,
			rewardValidate/4 +
				rewardValidate/4,
		},
		{
			"addr4",
			3,
			addr4,
			rewardValidate/4 +
				rewardValidate/4 +
				rewardValidate/3,
		},
		{
			"addr5",
			4,
			addr5,
			rewardGenerate +
				rewardValidate/3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wants, vs[tt.idx].iScore.Int64())
		})
	}
}

func newDelegatedDataForTest(enable bool, current int64, snapshot int64, iScore int64) *delegatedData {
	return &delegatedData{
		delegated: &icreward.Delegated{
			Enable:   enable,
			Current:  big.NewInt(current),
			Snapshot: big.NewInt(snapshot),
		},
		iScore: big.NewInt(iScore),
	}
}

func TestDelegatedData_compare(t *testing.T) {
	d1 := newDelegatedDataForTest(true, 10, 10, 10)
	d2 := newDelegatedDataForTest(true, 20, 20, 20)
	d3 := newDelegatedDataForTest(true, 21, 20, 21)
	d4 := newDelegatedDataForTest(false, 30, 30, 30)
	d5 := newDelegatedDataForTest(false, 31, 30, 31)

	type args struct {
		d1 *delegatedData
		d2 *delegatedData
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"x<y",
			args{d1, d2},
			-1,
		},
		{
			"x<y,disable",
			args{d5, d2},
			-1,
		},
		{
			"x==y",
			args{d2, d3},
			0,
		},
		{
			"x==y,disable",
			args{d4, d5},
			0,
		},
		{
			"x>y",
			args{d3, d1},
			1,
		},
		{
			"x>y,disable",
			args{d1, d4},
			1,
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, args.d1.compare(args.d2))
		})
	}
}

func TestDelegated_setEnable(t *testing.T) {
	d := newDelegated()
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)
	}

	enable := true
	for key, dd := range d.preps {
		enable = !enable
		addr, err := common.NewAddress([]byte(key))
		assert.NoError(t, err)
		d.setEnable(addr, enable)
		assert.Equal(t, enable, dd.delegated.Enable)
	}

	addr := common.NewAddressFromString("hx123412341234")
	d.setEnable(addr, false)
	prep, ok := d.preps[string(addr.Bytes())]
	assert.True(t, ok)
	assert.Equal(t, false, prep.delegated.Enable)
	assert.True(t, prep.delegated.IsEmpty())
	assert.Equal(t, 0, prep.iScore.Sign())
}

func TestDelegated_updateCurrent(t *testing.T) {
	d := newDelegated()
	ds := make([]*icstate.Delegation, 0)
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)

		ds = append(
			ds,
			&icstate.Delegation{
				Address: addr,
				Value:   common.NewHexInt(i),
			},
		)
	}
	newAddr := common.NewAddressFromString("hx321321")
	ds = append(
		ds,
		&icstate.Delegation{
			Address: newAddr,
			Value:   common.NewHexInt(100),
		},
	)

	d.updateCurrent(ds)
	for _, v := range ds {
		expect := v.Value.Value().Int64() * 2
		if v.Address.Equal(newAddr) {
			expect = v.Value.Value().Int64()
		}
		assert.Equal(t, expect, d.preps[string(v.Address.Bytes())].delegated.Current.Int64())
	}
}

func TestDelegated_updateSnapshot(t *testing.T) {
	d := newDelegated()
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i*2, i, i)
		d.addDelegatedData(addr, data)
	}

	d.updateSnapshot()

	for _, prep := range d.preps {
		assert.Equal(t, 0, prep.delegated.Current.Cmp(prep.delegated.Snapshot))
	}
}

func TestDelegated_updateTotal(t *testing.T) {
	d := newDelegated()
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward()) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)
		if i > more {
			total += i
		}
	}
	d.updateTotal()
	assert.Equal(t, total, d.total.Int64())

	for i, rank := range d.rank {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", maxIndex-int64(i)))
		assert.Equal(t, string(addr.Bytes()), rank)
	}
}

func TestDelegated_calculateReward(t *testing.T) {
	d := newDelegated()
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward()) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, 0)
		d.addDelegatedData(addr, data)
		if i > more {
			total += i
		}
	}
	d.updateTotal()
	assert.Equal(t, total, d.total.Int64())

	irep := big.NewInt(int64(YearBlock))
	period := MonthBlock
	bigIntPeriod := big.NewInt(int64(period))

	d.calculateReward(irep, period)

	for i, addr := range d.rank {
		expect := big.NewInt(maxIndex - int64(i))
		if i >= d.maxRankForReward() {
			expect.SetInt64(0)
		} else {
			expect.Mul(expect, irep)
			expect.Mul(expect, bigIntPeriod)
			expect.Div(expect, bigIntBeta2Divider)
			expect.Div(expect, d.total)
		}
		assert.Equal(t, expect.Int64(), d.preps[addr].iScore.Int64(), i)
	}
}

func TestCalculator_DelegatingReward(t *testing.T) {
	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	prepInfo := map[string]*pRepEnable{
		string(addr1.Bytes()): {0, 0},
		string(addr2.Bytes()): {10, 0},
		string(addr3.Bytes()): {100, 200},
	}

	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(100),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(100),
	}
	d3 := &icstate.Delegation{
		Address: addr3,
		Value:   common.NewHexInt(100),
	}
	d4 := &icstate.Delegation{
		Address: addr4,
		Value:   common.NewHexInt(100),
	}

	type args struct {
		rrep       int
		from       int
		to         int
		delegating icstate.Delegations
	}

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "PRep-full",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1},
			},
			want: 100 * 100 * 1000 * 1000 / YearBlock,
		},
		{
			name: "PRep-enabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d2},
			},
			want: 100 * 100 * (1000 - 10) * 1000 / YearBlock,
		},
		{
			name: "PRep-disabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d3},
			},
			want: 100 * 100 * (200 - 100) * 1000 / YearBlock,
		},
		{
			name: "PRep-None",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d4},
			},
			want: 0,
		},
		{
			name: "PRep-combination",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1, d2, d3, d4},
			},
			want: (100 * 100 * 1000 * 1000 / YearBlock) +
				(100 * 100 * (1000 - 10) * 1000 / YearBlock) +
				(100 * 100 * (200 - 100) * 1000 / YearBlock),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			reward := delegatingReward(
				big.NewInt(int64(args.rrep)),
				args.from,
				args.to,
				prepInfo,
				args.delegating,
			)
			assert.Equal(t, tt.want, reward.Int64())
		})
	}
}

func TestCalculator_processDelegating(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database)
	c := MakeCalculator(database, s.GetSnapshot())

	rrep := 100
	rrepBigInt := big.NewInt(100)
	from := 0
	to := 100
	offset := 50

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")

	d1Value := 100
	d2Value := 200
	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d1Value)),
	}
	d2 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	ds1 := icstate.Delegations{d1}
	ds2 := icstate.Delegations{d2}

	// make pRepInfo. all enabled
	prepInfo := make(map[string]*pRepEnable)
	prepInfo[string(addr1.Bytes())] = &pRepEnable{0, 0}

	// write delegating data to base
	dting1 := icreward.NewDelegating()
	dting1.Delegations = ds1
	dting2 := icreward.NewDelegating()
	dting2.Delegations = ds2
	c.temp.SetDelegating(addr2, dting2.Clone())
	c.temp.SetDelegating(addr3, dting1.Clone())
	c.temp.SetDelegating(addr4, dting2.Clone())
	c.base = c.temp.GetSnapshot()

	// make delegationMap
	delegationMap := make(map[string]map[int]icstate.Delegations)
	delegationMap[string(addr1.Bytes())] = make(map[int]icstate.Delegations)
	delegationMap[string(addr1.Bytes())][from+offset] = ds2
	delegationMap[string(addr3.Bytes())] = make(map[int]icstate.Delegations)
	delegationMap[string(addr3.Bytes())][from+offset] = ds2
	delegationMap[string(addr4.Bytes())] = make(map[int]icstate.Delegations)
	delegationMap[string(addr4.Bytes())][from+offset] = icstate.Delegations{}

	err := c.processDelegating(rrepBigInt, from, to, prepInfo, delegationMap)
	assert.NoError(t, err)

	type args struct {
		addr *common.Address
	}

	tests := []struct {
		name       string
		args       args
		want       int64
		delegating *icreward.Delegating
	}{
		{
			name:       "Delegate New",
			args:       args{addr1},
			want:       int64(rrep * d2Value * (to - offset) * IScoreICXRatio / YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegated and no modification",
			args:       args{addr2},
			want:       int64(rrep * d2Value * (to - from) * IScoreICXRatio / YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegated and modified",
			args:       args{addr3},
			want:       int64(rrep*d1Value*(offset-from)*IScoreICXRatio/YearBlock) + int64(rrep*d2Value*(to-offset)*IScoreICXRatio/YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegating removed",
			args:       args{addr4},
			want:       int64(rrep * d2Value * (offset - from) * IScoreICXRatio / YearBlock),
			delegating: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args

			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, iScore.Value.Int64())

			delegating, err := c.temp.GetDelegating(args.addr)
			assert.NoError(t, err)
			if tt.delegating != nil {
				assert.NotNil(t, delegating)
				assert.True(t, delegating.Equal(tt.delegating))
			} else {
				assert.Nil(t, delegating)
			}
		})
	}
}
