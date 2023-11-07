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

package calculator

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
)

func MakeCalculator(database db.Database, back *icstage.Snapshot) *calculator {
	c := new(calculator)
	c.back = back
	c.base = icreward.NewSnapshot(database, nil)
	c.temp = c.base.NewState()
	c.log = log.New()

	return c
}

func TestCalculator_processClaim(t *testing.T) {
	database := db.NewMapDB()
	front := icstage.NewState(database)

	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
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
			"Add Claim 200",
			args{
				addr2,
				big.NewInt(v2),
			},
			v2,
		},
	}

	// initialize data
	c := MakeCalculator(database, nil)
	for _, tt := range tests {
		args := tt.args
		// temp IScore : args.value * 2
		iScore := icreward.NewIScore(new(big.Int).Mul(args.value, big.NewInt(2)))
		err := c.temp.SetIScore(args.addr, iScore)
		assert.NoError(t, err)

		// add Claim : args.value
		_, err = front.AddIScoreClaim(args.addr, args.value)
		assert.NoError(t, err)
	}
	c.back = front.GetSnapshot()

	err := c.processClaim()
	assert.NoError(t, err)

	// check result
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, 0, args.value.Cmp(iScore.Value()))
		})
	}
}

func TestCalculator_WaitResult(t *testing.T) {
	c := &calculator{
		startHeight: InitBlockHeight,
	}
	err := c.WaitResult(1234)
	assert.NoError(t, err)

	c = &calculator{
		startHeight: 3414,
	}
	err = c.WaitResult(1234)
	assert.Error(t, err)

	toTC := make(chan string, 2)

	go func() {
		err := c.WaitResult(3414)
		assert.True(t, err == errors.ErrInvalidState)
		toTC <- "done"
	}()
	time.Sleep(time.Millisecond * 10)

	c.setResult(nil, errors.ErrInvalidState)
	assert.Equal(t, "done", <-toTC)

	c = &calculator{
		startHeight: 3414,
	}
	go func() {
		err := c.WaitResult(3414)
		assert.NoError(t, err)
		toTC <- "done"
	}()
	go func() {
		err := c.WaitResult(3414)
		assert.NoError(t, err)
		toTC <- "done"
	}()
	time.Sleep(time.Millisecond * 20)

	mdb := db.NewMapDB()
	rss := icreward.NewSnapshot(mdb, nil)
	c.setResult(rss, nil)

	assert.Equal(t, "done", <-toTC)
	assert.Equal(t, "done", <-toTC)

	assert.True(t, c.Result() == rss)
}

func TestCalculator_processCommissionRate(t *testing.T) {
	database := db.NewMapDB()
	front := icstage.NewState(database)

	addr1 := common.MustNewAddressFromString("hx1")
	addr2 := common.MustNewAddressFromString("hx2")
	v1 := icmodule.Rate(100)
	v2 := icmodule.Rate(200)

	type args struct {
		addr  *common.Address
		value icmodule.Rate
	}

	tests := []struct {
		name string
		args args
		want icmodule.Rate
	}{
		{
			"Set 100",
			args{
				addr1,
				v1,
			},
			v1,
		},
		{
			"Address has no Voted",
			args{
				addr2,
				v2,
			},
			icmodule.Rate(-1),
		},
	}

	// initialize data
	c := MakeCalculator(database, nil)
	voted := icreward.NewVotedV2()
	voted.SetStatus(icmodule.ESEnable)
	err := c.temp.SetVoted(addr1, voted)
	assert.NoError(t, err)

	for _, tt := range tests {
		args := tt.args
		err = front.AddCommissionRate(args.addr, args.value)
		assert.NoError(t, err)
	}
	c.back = front.GetSnapshot()

	err = c.processCommissionRate()
	assert.NoError(t, err)

	// check result
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			voted, err = c.temp.GetVoted(args.addr)
			assert.NoError(t, err)
			if tt.want != icmodule.Rate(-1) {
				assert.Equal(t, tt.want, voted.CommissionRate())
			} else {
				assert.Nil(t, voted)
			}
		})
	}
}
