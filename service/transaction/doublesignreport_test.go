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

package transaction

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
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

type testWContext struct {
	state.WorldContext
	height   int64
	ts       int64
}

func (wc *testWContext) BlockHeight() int64 {
	return wc.height
}

func (wc *testWContext) BlockTimeStamp() int64 {
	return wc.ts
}

func (wc *testWContext) Revision() module.Revision {
	return module.AllRevision
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

func TestNewDoubleSignReportTx(t *testing.T) {
	height1 := 3834 + rand.Int63n(897238947)
	ts1 := 28347328 + rand.Int63n(290384092834)
	nid1 := 7663
	d1a, d1b, signer1 := newTestDSData(height1, 1, 0)
	dsc1 := &testDSContext{ "test" }
	tx := NewDoubleSignReportTx([]module.DoubleSignData{d1a, d1b}, dsc1, nid1, ts1)

	assert.False(t, tx.IsSkippable())
	assert.EqualValues(t, tx.To(), state.SystemAddress)
	assert.EqualValues(t, tx.Version(), Version3)
	assert.EqualValues(t, tx.Group(), module.TransactionGroupNormal)
	assert.EqualValues(t, tx.Timestamp(), ts1)
	assert.Nil(t, tx.Nonce())
	assert.NoError(t, tx.Verify())

	wc := &testWContext{ height: 120, ts: 1000000 }

	assert.NoError(t, tx.PreValidate(wc, false))

	height, signer, ok := TryGetDoubleSignReportInfo(wc, tx)
	assert.True(t, ok)
	assert.EqualValues(t, d1a.Height(), height)
	assert.EqualValues(t, signer1, signer)

	bs := tx.Bytes()
	assert.True(t, checkDSRTxBytes(bs))

	tx2, err := parseDSRTxBytes(bs)
	assert.NoError(t, err)
	height, signer, ok = TryGetDoubleSignReportInfo(wc, tx2)
	assert.True(t, ok)
	assert.EqualValues(t, d1a.Height(), height)
	assert.EqualValues(t, signer1, signer)
}

func TestDoubleSignReportTx_ID(t *testing.T) {
	d1a, d1b, _ := newTestDSData(100, 1, 0)
	dsc1 := &testDSContext{ "test" }
	tx1 := NewDoubleSignReportTx([]module.DoubleSignData{d1a, d1b}, dsc1, 1234, 100)

	id1 := tx1.ID()
	jso, err := tx1.ToJSON(module.JSONVersion3)
	assert.NoError(t, err)
	js, err := json.Marshal(jso)
	assert.NoError(t, err)

	bs, err := SerializeJSON(js, nil, map[string]bool { "txHash" :true })
	assert.NoError(t, err)
	bs = append([]byte("icx_sendTransaction."), bs...)
	id2 := crypto.SHA3Sum256(bs)

	assert.EqualValues(t, id1, id2)
}