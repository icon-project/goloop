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
 *
 */

package contract

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type fakeCallContext struct {
	CallContext
	accounts map[string]*fakeAccountState
	revision module.Revision
	events   []*txresult.TestEventLog
}

func (cc *fakeCallContext) GetAccountState(id []byte) state.AccountState{
	if as, ok := cc.accounts[string(id)] ; ok {
		return as
	} else {
		as = &fakeAccountState{
			data: make(map[string][]byte),
		}
		cc.accounts[string(id)] = as
		return as
	}
}

func (cc *fakeCallContext) OnEvent(addr module.Address, indexed [][]byte, data [][]byte) {
	cc.events = append(cc.events, &txresult.TestEventLog{
		Address: addr,
		Indexed: indexed,
		Data:    data,
	})
}

func (cc *fakeCallContext) Revision() module.Revision {
	return cc.revision
}

func newFakeCallContext() *fakeCallContext {
	return &fakeCallContext{
		accounts: make(map[string]*fakeAccountState),
	}
}

type fakeAccountState struct {
	state.AccountState
	data map[string][]byte
}

func (as *fakeAccountState) GetValue(k []byte) ([]byte, error) {
	v, _ := as.data[string(k)]
	return v, nil
}

func (as *fakeAccountState) SetValue(k, v []byte) ([]byte, error) {
	if v == nil {
		return as.DeleteValue(k)
	}
	old, _ := as.data[string(k)]
	as.data[string(k)] = v
	return old, nil
}

func (as *fakeAccountState) DeleteValue(k []byte) ([]byte, error) {
	if old, ok := as.data[string(k)]; ok {
		delete(as.data, string(k))
		return old, nil
	} else {
		return nil, nil
	}
}

func TestRevision(t *testing.T) {
	cc := newFakeCallContext()
	rev := GetRevision(cc)
	assert.Zero(t, rev)

	const (
		Revision0 = 0
		Revision3 = 3
		Revision5 = 5
		Revision6 = 6
	)

	old, err := SetRevision(cc, Revision5, false)
	assert.NoError(t, err)
	assert.Equal(t, Revision0, old)

	old, err = SetRevision(cc, Revision3, false)
	assert.Error(t, err)
	assert.Equal(t, Revision5, old)

	cc.revision = Revision5 | module.ReportConfigureEvents

	old, err = SetRevision(cc, Revision6, false)
	assert.NoError(t, err)
	assert.Equal(t, Revision5, old)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventRevisionSet,
		nil, []any{Revision6},
	))
}

func TestStepPrice(t *testing.T) {
	cc := newFakeCallContext()
	price := GetStepPrice(cc)
	assert.True(t, price.Sign()==0)

	p1 := big.NewInt(20000)

	ok, err := SetStepPrice(cc, p1)
	assert.NoError(t, err)
	assert.True(t, ok)

	px := big.NewInt(-200)
	ok, err = SetStepPrice(cc, px)
	assert.Error(t, err)
	assert.False(t, ok)

	ok, err = SetStepPrice(cc, p1)
	assert.NoError(t, err)
	assert.False(t, ok)

	cc.revision = module.ReportConfigureEvents

	p2 := big.NewInt(30000)

	ok, err = SetStepPrice(cc, p2)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventStepPriceSet,
		nil, []any{p2},
	))
}

func TestStepCost(t *testing.T) {
	cc := newFakeCallContext()
	nx := "my_step"

	cost := GetStepCost(cc, nx)
	assert.EqualValues(t, intconv.BigIntZero, cost)

	n1 := state.StepTypeContractCall
	c1 := big.NewInt(20000)

	ok, err := SetStepCost(cc, nx, c1, false)
	assert.Error(t, err)
	assert.False(t, ok)

	// set n1 as c1
	ok, err = SetStepCost(cc, n1, c1, false)
	assert.NoError(t, err)
	assert.True(t, ok)

	cost = GetStepCost(cc, n1)
	assert.EqualValues(t, c1, cost)

	// set n1 as value overflowing int64
	o1 := big.NewInt(0);
	_, ok = o1.SetString("10000000000000000", 16)
	assert.True(t, ok)
	ok, err = SetStepCost(cc, n1, o1, false)
	assert.Error(t, err)
	assert.False(t, ok)

	c2 := big.NewInt(-4000)

	// set n1 as c2 (update)
	ok, err = SetStepCost(cc, n1, c2, false)
	assert.NoError(t, err)
	assert.True(t, ok)

	cost = GetStepCost(cc, n1)
	assert.EqualValues(t, c2, cost)

	costs := GetStepCosts(cc)
	assert.EqualValues(t, map[string]any{
		state.StepTypeContractCall: c2,
	}, costs)

	cc.revision = module.ReportConfigureEvents

	n2 := state.StepTypeDefault
	c3 := big.NewInt(30000)

	// set n2 as c3
	ok, err = SetStepCost(cc, n2, c3, false)
	assert.NoError(t, err)
	assert.True(t, ok)

	costs = GetStepCosts(cc)
	assert.EqualValues(t, map[string]any{
		state.StepTypeContractCall: c2,
		state.StepTypeDefault: c3,
	}, costs)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventStepCostSet,
		nil, []any{n2, c3},
	))
	cc.events = nil

	// repeat n2 as c3 (no effect)
	ok, err = SetStepCost(cc, n2, c3, false)
	assert.NoError(t, err)
	assert.False(t, ok)

	assert.Zero(t, len(cc.events))

	// update n1 as zero (no delete)
	ok, err = SetStepCost(cc, n1, intconv.BigIntZero, false)
	assert.NoError(t, err)
	assert.True(t, ok)

	costs = GetStepCosts(cc)
	assert.EqualValues(t, map[string]any{
		n2: c3,
		n1: new(big.Int).SetBytes([]byte{0}),
	}, costs)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventStepCostSet,
		nil, []any{n1, intconv.BigIntZero},
	))
	cc.events = nil

	// delete n1 entry
	ok, err = SetStepCost(cc, n1, intconv.BigIntZero, true)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, len(cc.events))
	costs = GetStepCosts(cc)
	assert.EqualValues(t, map[string]any{
		n2: c3,
	}, costs)

	// delete n2 entry
	ok, err = SetStepCost(cc, n2, intconv.BigIntZero, true)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventStepCostSet,
		nil, []any{n2, intconv.BigIntZero},
	))
	cc.events = nil
	costs = GetStepCosts(cc)
	assert.Zero(t, len(costs))
}

func TestMaxStepLimit(t *testing.T) {
	cc := newFakeCallContext()
	nx := "my_type"

	// get with invalid (no error, return zero)
	cost := GetMaxStepLimit(cc, nx)
	assert.EqualValues(t, intconv.BigIntZero, cost)

	n1 := state.StepLimitTypeInvoke
	c1 := big.NewInt(20000)

	// set nx as c1 (invalid)
	ok, err := SetMaxStepLimit(cc, nx, c1)
	assert.Error(t, err)
	assert.False(t, ok)

	// set n1 as c1
	ok, err = SetMaxStepLimit(cc, n1, c1)
	assert.NoError(t, err)
	assert.True(t, ok)

	cost = GetMaxStepLimit(cc, n1)
	assert.EqualValues(t, c1, cost)

	cx := big.NewInt(-4000)

	// set n1 as cx (invalid)
	ok, err = SetMaxStepLimit(cc, n1, cx)
	assert.Error(t, err)
	assert.False(t, ok)

	// set n1 as value overflowing int64
	o1 := big.NewInt(0);
	_, ok = o1.SetString("10000000000000000", 16)
	assert.True(t, ok)
	ok, err = SetMaxStepLimit(cc, n1, o1)
	assert.Error(t, err)
	assert.False(t, ok)

	c2 := big.NewInt(40000)

	// set n1 as c2 (update)
	ok, err = SetMaxStepLimit(cc, n1, c2)
	assert.NoError(t, err)
	assert.True(t, ok)

	cost = GetMaxStepLimit(cc, n1)
	assert.EqualValues(t, c2, cost)

	cc.revision = module.ReportConfigureEvents

	n2 := state.StepLimitTypeQuery
	c3 := big.NewInt(30000)

	// set n2 as c3
	ok, err = SetMaxStepLimit(cc, n2, c3)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventMaxStepLimitSet,
		nil, []any{n2, c3},
	))
	cc.events = nil

	// repeat n2 as c3 (no effect)
	ok, err = SetMaxStepLimit(cc, n2, c3)
	assert.NoError(t, err)
	assert.False(t, ok)

	assert.Zero(t, len(cc.events))

	// set n1 as zero
	ok, err = SetMaxStepLimit(cc, n1, intconv.BigIntZero)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventMaxStepLimitSet,
		nil, []any{n1, intconv.BigIntZero},
	))
	cc.events = nil

	// set n1 as zero again (no effect)
	ok, err = SetMaxStepLimit(cc, n1, intconv.BigIntZero)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, len(cc.events))

	// set n2 as zero
	ok, err = SetMaxStepLimit(cc, n2, intconv.BigIntZero)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestTimestampThreshold(t *testing.T) {
	cc := newFakeCallContext()

	// initial is zero
	th := GetTimestampThreshold(cc)
	assert.EqualValues(t, 0, th)

	t1 := int64(1000)

	// set as t1
	ok, err := SetTimestampThreshold(cc, t1)
	assert.NoError(t, err)
	assert.True(t, ok)

	th = GetTimestampThreshold(cc)
	assert.EqualValues(t, t1, th)

	t2 := int64(2000)

	cc.revision |= module.ReportConfigureEvents

	// set as t2
	ok, err = SetTimestampThreshold(cc, t2)
	assert.NoError(t, err)
	assert.True(t, ok)

	th = GetTimestampThreshold(cc)
	assert.EqualValues(t, t2, th)

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventTimestampThresholdSet,
		nil, []any{t2},
	))
	cc.events = nil

	// set as same
	ok, err = SetTimestampThreshold(cc, t2)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, len(cc.events))

	// set as negative (invalid)
	ok, err = SetTimestampThreshold(cc, -100)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, len(cc.events))

	// set as zero (delete)
	ok, err = SetTimestampThreshold(cc, 0)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.EqualValues(t, 0, GetTimestampThreshold(cc))

	assert.Equal(t, 1, len(cc.events))
	assert.NoError(t, cc.events[0].Assert(
		state.SystemAddress,
		EventTimestampThresholdSet,
		nil, []any{intconv.BigIntZero},
	))
	cc.events = nil
}