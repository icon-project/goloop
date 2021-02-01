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
	"testing"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/stretchr/testify/assert"
)

type dummyPlatformType struct{}

func (d dummyPlatformType) ToRevision(value int) module.Revision {
	return module.AllRevision
}

type testCallContext struct {
	contract.CallContext
	blockHeight int64
}

func (cc *testCallContext) BlockHeight() int64 {
	return cc.blockHeight
}

func (cc *testCallContext) setBlockHeight(blockHeight int64) {
	cc.blockHeight = blockHeight
}

// test for updatePRepStatus
func TestUpdatePrepStatus(t *testing.T) {
	cc := &testCallContext{}
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), false)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")

	addrArray := []module.Address{addr1, addr2}

	err := s.SetLastValidators(addrArray)
	assert.NoError(t, err)

	// 0 block : vote
	voted := []bool{true, false}
	err = updatePRepStatus(cc, s, addrArray, voted)
	assert.NoError(t, err)

	ps1 := s.GetPRepStatus(addr1)
	ps2 := s.GetPRepStatus(addr2)
	bh := cc.BlockHeight()
	assert.Equal(t, icstate.Success, ps1.LastState())
	assert.Equal(t, icstate.Fail, ps2.LastState())
	assert.Equal(t, int64(0), ps2.LastHeight())
	assert.Equal(t, int64(0), ps2.LastHeight())
	assert.Equal(t, 1, ps1.GetVTotal(bh))
	assert.Equal(t, 1, ps2.GetVTotal(bh))
	assert.Equal(t, 0, ps1.GetVFail(bh))
	assert.Equal(t, 1, ps2.GetVFail(bh))
	assert.Equal(t, 0, ps1.GetVFailCont(bh))
	assert.Equal(t, 1, ps2.GetVFailCont(bh))

	// 9 block : vote
	cc.setBlockHeight(9)
	err = updatePRepStatus(cc, s, addrArray, voted)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.Success, ps1.LastState())
	assert.Equal(t, icstate.Fail, ps2.LastState())
	assert.Equal(t, int64(0), ps2.LastHeight())
	assert.Equal(t, int64(0), ps2.LastHeight())
	assert.Equal(t, 10, ps1.GetVTotal(bh))
	assert.Equal(t, 10, ps2.GetVTotal(bh))
	assert.Equal(t, 0, ps1.GetVFail(bh))
	assert.Equal(t, 10, ps2.GetVFail(bh))
	assert.Equal(t, 0, ps1.GetVFailCont(bh))
	assert.Equal(t, 10, ps2.GetVFailCont(bh))

	// 10 block : no vote
	cc.setBlockHeight(10)
	var emptyArray []module.Address
	err = updatePRepStatus(cc, s, emptyArray, voted)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.None, ps1.LastState())
	assert.Equal(t, icstate.None, ps2.LastState())
	assert.Equal(t, int64(10), ps2.LastHeight())
	assert.Equal(t, int64(10), ps2.LastHeight())
	assert.Equal(t, 10, ps1.GetVTotal(bh))
	assert.Equal(t, 10, ps2.GetVTotal(bh))
	assert.Equal(t, 0, ps1.GetVFail(bh))
	assert.Equal(t, 10, ps2.GetVFail(bh))
	assert.Equal(t, 0, ps1.GetVFailCont(bh))
	assert.Equal(t, 0, ps2.GetVFailCont(bh))

	// 11 block : vote
	cc.setBlockHeight(11)
	voted2 := []bool{false, false}
	err = updatePRepStatus(cc, s, addrArray, voted2)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.Fail, ps1.LastState())
	assert.Equal(t, icstate.Fail, ps2.LastState())
	assert.Equal(t, int64(11), ps2.LastHeight())
	assert.Equal(t, int64(11), ps2.LastHeight())
	assert.Equal(t, 11, ps1.GetVTotal(bh))
	assert.Equal(t, 11, ps2.GetVTotal(bh))
	assert.Equal(t, 1, ps1.GetVFail(bh))
	assert.Equal(t, 11, ps2.GetVFail(bh))
	assert.Equal(t, 1, ps1.GetVFailCont(bh))
	assert.Equal(t, 1, ps2.GetVFailCont(bh))

	// 12 block : vote - false, true
	cc.setBlockHeight(12)
	voted3 := []bool{false, true}
	err = updatePRepStatus(cc, s, addrArray, voted3)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.Fail, ps1.LastState())
	assert.Equal(t, icstate.Success, ps2.LastState())
	assert.Equal(t, int64(12), ps2.LastHeight())
	assert.Equal(t, int64(12), ps2.LastHeight())
	assert.Equal(t, 12, ps1.GetVTotal(bh))
	assert.Equal(t, 12, ps2.GetVTotal(bh))
	assert.Equal(t, 2, ps1.GetVFail(bh))
	assert.Equal(t, 11, ps2.GetVFail(bh))
	assert.Equal(t, 2, ps1.GetVFailCont(bh))
	assert.Equal(t, 0, ps2.GetVFailCont(bh))

	// 13 block : vote - false, false
	cc.setBlockHeight(13)
	voted4 := []bool{false, false}
	err = updatePRepStatus(cc, s, addrArray, voted4)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.Fail, ps1.LastState())
	assert.Equal(t, icstate.Fail, ps2.LastState())
	assert.Equal(t, int64(13), ps2.LastHeight())
	assert.Equal(t, int64(13), ps2.LastHeight())
	assert.Equal(t, 13, ps1.GetVTotal(bh))
	assert.Equal(t, 13, ps2.GetVTotal(bh))
	assert.Equal(t, 3, ps1.GetVFail(bh))
	assert.Equal(t, 12, ps2.GetVFail(bh))
	assert.Equal(t, 3, ps1.GetVFailCont(bh))
	assert.Equal(t, 1, ps2.GetVFailCont(bh))

	// 14 block : vote - true, true
	cc.setBlockHeight(14)
	voted5 := []bool{true, true}
	err = updatePRepStatus(cc, s, addrArray, voted5)
	assert.NoError(t, err)

	ps1 = s.GetPRepStatus(addr1)
	ps2 = s.GetPRepStatus(addr2)
	bh = cc.BlockHeight()
	assert.Equal(t, icstate.Success, ps1.LastState())
	assert.Equal(t, icstate.Success, ps2.LastState())
	assert.Equal(t, int64(14), ps2.LastHeight())
	assert.Equal(t, int64(14), ps2.LastHeight())
	assert.Equal(t, 14, ps1.GetVTotal(bh))
	assert.Equal(t, 14, ps2.GetVTotal(bh))
	assert.Equal(t, 3, ps1.GetVFail(bh))
	assert.Equal(t, 12, ps2.GetVFail(bh))
	assert.Equal(t, 0, ps1.GetVFailCont(bh))
	assert.Equal(t, 0, ps2.GetVFailCont(bh))
}

func Test_applyPRepStatus(t *testing.T) {
	addr1 := common.NewAddressFromString("hx1")
	status1 := icstate.NewPRepStatus(addr1)

	type args struct {
		lastState   icstate.ValidationState
		blockHeight int64
	}

	type want struct {
		vTotal    int
		getVTotal int
		vFail     int
		getVFail  int
		vFailCont int
		lastState icstate.ValidationState
		lastBH    int64
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"Same state - None",
			args{
				icstate.None,
				0,
			},
			want{
				0,
				0,
				0,
				0,
				0,
				icstate.None,
				0,
			},
		},
		{
			"Fail first",
			args{
				icstate.Fail,
				10,
			},
			want{
				0,
				1,
				0,
				1,
				1,
				icstate.Fail,
				10,
			},
		},
		{
			"Fail again",
			args{
				icstate.Fail,
				20,
			},
			want{
				0,
				11,
				0,
				11,
				11,
				icstate.Fail,
				10,
			},
		},
		{
			"Success first",
			args{
				icstate.Success,
				30,
			},
			want{
				20,
				21,
				20,
				20,
				0,
				icstate.Success,
				30,
			},
		},
		{
			"Success again",
			args{
				icstate.Success,
				40,
			},
			want{
				20,
				31,
				20,
				20,
				0,
				icstate.Success,
				30,
			},
		},
		{
			"None",
			args{
				icstate.None,
				50,
			},
			want{
				40,
				40,
				20,
				20,
				0,
				icstate.None,
				50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.args
			out := tt.want
			applyPRepStatus(status1, in.lastState, in.blockHeight)

			assert.Equal(t, out.vTotal, status1.VTotal())
			assert.Equal(t, out.getVTotal, status1.GetVTotal(in.blockHeight))
			assert.Equal(t, out.vFail, status1.VFail())
			assert.Equal(t, out.getVFail, status1.GetVFail(in.blockHeight))
			assert.Equal(t, out.vFailCont, status1.GetVFailCont(in.blockHeight))
			assert.Equal(t, out.lastState, status1.LastState())
			assert.Equal(t, out.lastBH, status1.LastHeight())
		})
	}
}
