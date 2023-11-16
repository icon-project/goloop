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

package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
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

type testAccountSnapshot struct {
	values map[string][]byte
	state.AccountSnapshot
}

func (t *testAccountSnapshot) GetValue(key []byte) ([]byte, error) {
	if t.values!=nil {
		return t.values[string(key)], nil
	} else {
		return nil, nil
	}
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

func (as *testAccountState) GetSnapshot() state.AccountSnapshot {
	var values map[string][]byte
	if as.values != nil {
		values = make(map[string][]byte)
		for k, v := range as.values {
			values[k] = v
		}
	}
	return &testAccountSnapshot{
		values: values,
	}
}

type testWContext struct {
	state.WorldContext
	height   int64
	ts       int64
	accounts map[string]*testAccountState
}

func (wc *testWContext) BlockHeight() int64 {
	return wc.height
}

func (wc *testWContext) BlockTimeStamp() int64 {
	return wc.ts
}

func (wc *testWContext) GetAccountState(id []byte) state.AccountState {
	if wc.accounts == nil {
		wc.accounts = make(map[string]*testAccountState)
	}
	if as, ok := wc.accounts[string(id)] ; ok {
		return as
	} else {
		as = new(testAccountState)
		wc.accounts[string(id)] = as
		return as
	}
}

func (wc *testWContext) DecodeDoubleSignContext(tn string, bs []byte) (module.DoubleSignContext, error) {
	return &testDSContext{string(bs)}, nil
}

func (wc *testWContext) DecodeDoubleSignData(tn string, bs []byte) (module.DoubleSignData, error) {
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

func TestDSRManager_Basic(t *testing.T) {
	const NID = 0x234
	mgr := newDSRManager(log.GlobalLogger())

	dsc1 := &testDSContext { name: "dsc1" }

	wc := &testWContext{ height: 100, ts: 1000000 }

	d1a, d1b, signer1 := newTestDSData(wc.BlockHeight(), 1, 0)

	// invalid sign data
	assert.Error(t, mgr.Add(nil, dsc1))
	assert.Error(t, mgr.Add([]module.DoubleSignData{}, dsc1))
	assert.Error(t, mgr.Add([]module.DoubleSignData{d1a}, dsc1))
	assert.Error(t, mgr.Add([]module.DoubleSignData{d1a, d1b}, nil))

	// not ready to accept DSR
	assert.Error(t, mgr.Add([]module.DoubleSignData{d1a, d1b}, dsc1))

	as := wc.GetAccountState(state.SystemID)
	dsch, err := contract.NewDSContextHistoryDB(as)
	assert.NoError(t, err)
	assert.NoError(t, dsch.Push(wc.BlockHeight(), dsc1.Hash()))
	mgr.OnFinalizeState(as.GetSnapshot())

	wc.height += 2

	// in-conflict case
	assert.Error(t, mgr.Add([]module.DoubleSignData{d1a, d1a}, dsc1))

	// just report (should not in DONE list)
	assert.NoError(t, mgr.Add([]module.DoubleSignData{d1a, d1b}, dsc1))
	assert.False(t, mgr.Has(wc.BlockHeight(), signer1))

	wc.height += 1
	d2a, d2b, _ := newTestDSData(wc.BlockHeight(), 1, 1)
	d3a, d3b, _ := newTestDSData(wc.BlockHeight(), 1, 2)

	// just report (should not in DONE list)
	assert.NoError(t, mgr.Add([]module.DoubleSignData{d2a, d2b}, dsc1))
	assert.False(t, mgr.Has(wc.BlockHeight(), signer1))

	// it should be ignored.
	assert.NoError(t, mgr.Add([]module.DoubleSignData{d3a, d3b}, dsc1))
	assert.False(t, mgr.Has(wc.BlockHeight(), signer1))

	tr := mgr.NewTracker()

	// on the tracker, it candidates two of them
	trs, err := mgr.Candidate(tr, wc, NID)
	assert.NoError(t, err)
	assert.Len(t, trs, 2)
	height, signer, ok := transaction.TryGetDoubleSignReportInfo(wc, trs[0])
	assert.True(t, ok)
	assert.Equal(t, d1a.Height(), height)
	assert.EqualValues(t, signer1, signer)
	height, signer, ok = transaction.TryGetDoubleSignReportInfo(wc, trs[1])
	assert.True(t, ok)
	assert.Equal(t, d2a.Height(), height)
	assert.EqualValues(t, signer1, signer)

	tr2 := tr.New()
	tr2.Add(d1a.Height(), signer1)          // from mine
	tr2.Add(wc.BlockHeight(), newSigner(2)) // from other

	// after adding first, then it candidates the second one
	wc.height += 1
	trs, err = mgr.Candidate(tr2, wc, NID)
	assert.NoError(t, err)
	assert.Len(t, trs, 1)
	height, signer, ok = transaction.TryGetDoubleSignReportInfo(wc, trs[0])
	assert.True(t, ok)
	assert.Equal(t, d2a.Height(), height)
	assert.EqualValues(t, signer1, signer)

	tr3 := tr2.New()
	tr3.Add(d2a.Height(), signer1)

	assert.True(t, tr3.Has(d1a.Height(), signer1))
	assert.True(t, tr3.Has(d2a.Height(), signer1))

	tr2.Commit()

	assert.True(t, mgr.Has(d1a.Height(), signer1))
	assert.False(t, mgr.Has(d2a.Height(), signer1))

	wc.height += 1
	trs, err = mgr.Candidate(tr2, wc, NID)
	assert.NoError(t, err)
	assert.Len(t, trs, 1)
	height, signer, ok = transaction.TryGetDoubleSignReportInfo(wc, trs[0])
	assert.True(t, ok)
	assert.Equal(t, d2a.Height(), height)
	assert.EqualValues(t, signer1, signer)


	for i := 0 ; i<contract.DSContextHistoryLimit ; i++ {
		dsc1.name = fmt.Sprintf("DSContext %d", i)
		wc.height += 1
		assert.NoError(t, dsch.Push(wc.BlockHeight(), dsc1.Hash()))

		mgr.OnFinalizeState(as.GetSnapshot())
	}

	assert.False(t, tr3.Has(d1a.Height(), signer1))
	assert.True(t, tr3.Has(d2a.Height(), signer1))

	trs, err = mgr.Candidate(tr2, wc, NID)
	assert.NoError(t, err)
	assert.Len(t, trs, 0)
}