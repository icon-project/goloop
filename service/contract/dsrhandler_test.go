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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type testDSData struct {
	name   string
	height int64
	signer string
	nonce  int
	nid    int
}

func (d *testDSData) Type() string {
	return d.name
}

func (d *testDSData) Height() int64 {
	return d.height
}

func (d *testDSData) Signer() []byte {
	return []byte(d.signer)
}

func (d *testDSData) ValidateNetwork(nid int) bool {
	if d.nid == 0 {
		return true
	} else {
		return d.nid == nid
	}
}

func (d *testDSData) Bytes() []byte {
	return codec.MustMarshalToBytes([]interface{}{ d.name, d.height, d.signer, d.nonce })
}

func (d *testDSData) IsConflictWith(other module.DoubleSignData) bool {
	d2 := other.(*testDSData)
	return d.name == d2.name && d.height == d2.height && d.signer == d2.signer && d.nonce != d2.nonce
}

func newSigner(id int) module.Address {
	return common.NewAddressWithTypeAndID(false, []byte(fmt.Sprintf("signer%d", id)))
}

func newTestDSData(height int64, signer int, nonce int) (*testDSData, *testDSData, module.Address) {
	addr := newSigner(signer)
	d1 := &testDSData{
		name: module.DSTVote,
		height: height,
		signer: string(addr.ID()),
		nonce: nonce*2,
	}
	d2 := &testDSData{
		name: module.DSTVote,
		height: height,
		signer: string(addr.ID()),
		nonce: nonce*2+1,
	}
	return d1, d2, addr
}

type testDSContext struct {
	name string
}

func (t *testDSContext) AddressOf(signer []byte) module.Address {
	return common.NewAddressWithTypeAndID(false, signer)
}

func (t *testDSContext) Hash() []byte {
	return crypto.SHA3Sum256([]byte(t.name))
}

func (t *testDSContext) Bytes() []byte {
	return []byte(t.name)
}

type testAccountState struct {
	values map[string][]byte
	state.AccountState
}

func (t *testAccountState) GetValue(key []byte) ([]byte, error) {
	if t.values!=nil {
		return t.values[string(key)], nil
	} else {
		return nil, nil
	}
}

func (t *testAccountState) SetValue(key []byte, value []byte) ([]byte, error){
	if t.values == nil {
		t.values = make(map[string][]byte)
	}
	old,  _ := t.values[string(key)]
	t.values[string(key)] = value
	return old, nil
}

func (t *testAccountState) DeleteValue(key []byte) ([]byte, error){
	if t.values == nil {
		return nil, nil
	}
	old,  ok := t.values[string(key)]
	if ok {
		delete(t.values, string(key))
	}
	return old, nil
}

type testRSCallContext struct {
	CallContext
	height   int64
	ts       int64
	accounts map[string]*testAccountState
	lq       []state.LockRequest
}

func (cc *testRSCallContext) BlockHeight() int64 {
	return cc.height
}

func (cc *testRSCallContext) BlockTimeStamp() int64 {
	return cc.ts
}

func (cc *testRSCallContext) Revision() module.Revision {
	return module.AllRevision
}

func (cc *testRSCallContext) GetAccountState(id []byte) state.AccountState {
	if cc.accounts == nil {
		cc.accounts = make(map[string]*testAccountState)
	}
	if as, ok := cc.accounts[string(id)] ; ok {
		return as
	} else {
		as = new(testAccountState)
		cc.accounts[string(id)] = as
		return as
	}
}

func (cc *testRSCallContext) DecodeDoubleSignContext(tn string, bs []byte) (module.DoubleSignContext, error) {
	return &testDSContext{string(bs)}, nil
}

func (cc *testRSCallContext) DecodeDoubleSignData(tn string, bs []byte) (module.DoubleSignData, error) {
	var dsData struct {
		Name   string
		Height int64
		Signer string
		Nonce  int
	}
	remain := codec.MustUnmarshalFromBytes(bs, &dsData)
	if len(remain) > 0 {
		return nil, errors.IllegalArgumentError.New("RemainingBytes")
	}
	if dsData.Name != tn {
		return nil, errors.IllegalArgumentError.New("InvalidTypeValue")
	}
	return &testDSData{
		name:   dsData.Name,
		height: dsData.Height,
		signer: dsData.Signer,
		nonce:  dsData.Nonce,
	}, nil
}
func (cc *testRSCallContext) GetFuture(lq []state.LockRequest) state.WorldContext {
	cc.lq = lq
	return &testRSCallContext{}
}

func (cc *testRSCallContext) StepAvailable() *big.Int {
	return big.NewInt(10_000_000)
}

func (cc *testRSCallContext) Call(handler ContractHandler, limit *big.Int) (error, *big.Int, *codec.TypedObj, module.Address) {
	return nil, big.NewInt(1000), nil, nil
}

func newTestDSR(height int64, dsc string) *DoubleSignReport {
	d1a, d1b, _ := newTestDSData(height, 1, 0)
	dsc1 := &testDSContext{name: dsc}
	return NewDoubleSignReport([]module.DoubleSignData{d1a, d1b}, dsc1)
}

func newTestDSRHandler(height int64, dsc string) (*DSRHandler, *DoubleSignReport) {
	dsr := newTestDSR(height, dsc)
	return NewDSRHandler(nil, dsr, log.GlobalLogger()), dsr
}

func TestNewDoubleSignReport(t *testing.T) {
	d1a, d1b, _ := newTestDSData(3334, 1, 0)
	dsc1 := &testDSContext{name: "TEST"}

	cc := &testRSCallContext{}

	dsr := NewDoubleSignReport([]module.DoubleSignData{d1a, d1b}, dsc1)
	d2, dsc2, err := dsr.Decode(cc, false)
	assert.NoError(t, err)
	assert.EqualValues(t, d1a, d2[0])
	assert.EqualValues(t, d1b, d2[1])
	assert.EqualValues(t, dsc1, dsc2)

	dsr = NewDoubleSignReport([]module.DoubleSignData{d1b, d1a}, dsc1)
	d2, dsc2, err = dsr.Decode(cc, false)
	assert.NoError(t, err)
	assert.EqualValues(t, d1a, d2[0])
	assert.EqualValues(t, d1b, d2[1])
	assert.EqualValues(t, dsc1, dsc2)
}

func TestDSRHandler_Prepare(t *testing.T) {
	handler, _ := newTestDSRHandler(1234, "TEST")

	ctx := &testRSCallContext{ }
	_, err := handler.Prepare(ctx)
	assert.NoError(t, err)
	elq := []state.LockRequest{
		{ state.WorldIDStr, state.AccountWriteLock },
	}
	assert.EqualValues(t, elq, ctx.lq)
}

func TestDSRHandler_ExecuteSync(t *testing.T) {
	height := int64(4567)

	dsr := newTestDSR(height, "TEST1")
	handler := NewDSRHandler(state.SystemAddress, dsr, log.GlobalLogger())

	cc := &testRSCallContext{
		height: height,
		ts:     8888,
	}
	as := cc.GetAccountState(state.SystemID)
	dsch, err := NewDSContextHistoryDB(as)
	assert.NoError(t, err)
	assert.NoError(t, dsch.Push(height-2, dsr.dsc.Hash()))

	err, ro, addr := handler.ExecuteSync(cc)
	assert.NoError(t, err)
	assert.Nil(t, ro)
	assert.Nil(t, addr)

	dsr2 := newTestDSR(height, "TEST2")
	handler2 := NewDSRHandler(newSigner(2), dsr2, log.GlobalLogger())
	err, _, _ = handler2.ExecuteSync(cc)
	assert.Error(t, err)

	handler3 := NewDSRHandler(newSigner(2), dsr, log.GlobalLogger())
	err, ro, addr = handler3.ExecuteSync(cc)
	assert.Error(t, err)
	assert.Nil(t, ro)
	assert.Nil(t, addr)
}