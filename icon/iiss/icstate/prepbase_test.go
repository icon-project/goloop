/*
 * Copyright 2020 ICON Foundation
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

package icstate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

func TestPRepBase_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	pb := newDummyPRepBase(0)
	pbs1 := pb.GetSnapshot()

	o1 := icobject.New(TypePRepBase, pbs1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	pbs2 := ToPRepBase(o2)
	assert.Equal(t, true, pbs1.Equal(pbs2))
	assert.Equal(t, true, pbs2.Equal(pbs1))
}

func TestPRepBaseState_UpdateInfo(t *testing.T) {
	info1 := &PRepInfo{
		City:        NewStringPtr("Seoul"),
		Country:     NewStringPtr("KOR"),
		Details:     NewStringPtr("https://test.url/test/"),
		Email:       NewStringPtr("test@google.com"),
		Name:        NewStringPtr("Test User"),
		P2PEndpoint: NewStringPtr("192.168.1.1:9080"),
		WebSite:     NewStringPtr("http://test.url/home"),
	}
	info2 := &PRepInfo{
		City:        NewStringPtr("Chuncheon"),
		Country:     NewStringPtr("USA"),
		Details:     NewStringPtr("https://test.host/info/"),
		Email:       NewStringPtr("test2@apple.com"),
		Name:        NewStringPtr("Test User2"),
		P2PEndpoint: NewStringPtr("192.168.1.2:9080"),
		WebSite:     NewStringPtr("http://test.host/home2"),
		Node:        common.MustNewAddressFromString("hx02"),
	}
	type args struct {
		initial *PRepInfo
		update  *PRepInfo
	}
	tests := []struct {
		name    string
		args    args
		want    *PRepInfo
	}{
		{
			"City",
			args{ info1, &PRepInfo{
				City: info2.City,
			}},
			&PRepInfo{
				City:        info2.City,
				Country:     info1.Country,
				Details:     info1.Details,
				Email:       info1.Email,
				Name:        info1.Name,
				P2PEndpoint: info1.P2PEndpoint,
				WebSite:     info1.WebSite,
			},
		},
		{
			"Country",
			args{info1, &PRepInfo{
				Country: info2.Country,
			}},
			&PRepInfo{
				City:        info1.City,
				Country:     info2.Country,
				Details:     info1.Details,
				Email:       info1.Email,
				Name:        info1.Name,
				P2PEndpoint: info1.P2PEndpoint,
				WebSite:     info1.WebSite,
			},
		},
		{
			"Details",
			args{info1, &PRepInfo{
				Details: info2.Details,
			}},
			&PRepInfo{
				City:        info1.City,
				Country:     info1.Country,
				Details:     info2.Details,
				Email:       info1.Email,
				Name:        info1.Name,
				P2PEndpoint: info1.P2PEndpoint,
				WebSite:     info1.WebSite,
			},
		},
		{
			"Email&Name",
			args{info1, &PRepInfo{
				Email: info2.Email,
				Name:  info2.Name,
			}},
			&PRepInfo{
				City:        info1.City,
				Country:     info1.Country,
				Details:     info1.Details,
				Email:       info2.Email,
				Name:        info2.Name,
				P2PEndpoint: info1.P2PEndpoint,
				WebSite:     info1.WebSite,
			},
		},
		{
			"EndPoint&Web",
			args{info1, &PRepInfo{
				P2PEndpoint: info2.P2PEndpoint,
				WebSite:     info2.WebSite,
			}},
			&PRepInfo{
				City:        info1.City,
				Country:     info1.Country,
				Details:     info1.Details,
				Email:       info1.Email,
				Name:        info1.Name,
				P2PEndpoint: info2.P2PEndpoint,
				WebSite:     info2.WebSite,
			},
		},
		{
			"Node1",
			args{info1, &PRepInfo{
				Node: info2.Node,
			}},
			&PRepInfo{
				City:        info1.City,
				Country:     info1.Country,
				Details:     info1.Details,
				Email:       info1.Email,
				Name:        info1.Name,
				P2PEndpoint: info1.P2PEndpoint,
				WebSite:     info1.WebSite,
				Node:        info2.Node,
			},
		},
		{
			"Node2",
			args{info2, &PRepInfo{
				Node: common.MustNewAddressFromString("hx03"),
			}},
			&PRepInfo{
				City:        info2.City,
				Country:     info2.Country,
				Details:     info2.Details,
				Email:       info2.Email,
				Name:        info2.Name,
				P2PEndpoint: info2.P2PEndpoint,
				WebSite:     info2.WebSite,
				Node:       common.MustNewAddressFromString("hx03"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// make initial
			pb := NewPRepBaseState()
			pb.UpdateInfo(tt.args.initial)
			assert.True(t, pb.info().equal(tt.args.initial))

			// apply update
			pb.UpdateInfo(tt.args.update)

			// confirm state
			assert.True(t, pb.info().equal(tt.want))

			// confirm snapshot
			pbs2 := pb.GetSnapshot()
			assert.True(t, pbs2.info().equal(tt.want))
		})
	}
}

func TestPRepBaseSnapshot_RLPEncodeFields(t *testing.T) {
	const (
		Rate = icmodule.Rate(1000)
		MaxRate = icmodule.Rate(2000)
		MaxChangeRate = icmodule.Rate(100)
	)

	pbs := NewPRepBaseState()
	err := pbs.InitCommissionInfo(Rate, MaxRate, MaxChangeRate)
	assert.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	e := codec.BC.NewEncoder(buf)

	pbss := pbs.GetSnapshot()
	assert.Equal(t, PRepBaseVersion2, pbss.Version())

	err = pbss.RLPEncodeFields(e)
	assert.NoError(t, err)

	err = e.Close()
	assert.NoError(t, err)

	pbss2 := NewPRepBaseSnapshot(PRepBaseVersion2)
	assert.Zero(t, pbss2.CommissionRate())
	assert.Zero(t, pbss2.MaxCommissionRate())
	assert.Zero(t, pbss2.MaxCommissionChangeRate())

	d := codec.BC.NewDecoder(bytes.NewReader(buf.Bytes()))
	err = pbss2.RLPDecodeFields(d)
	assert.NoError(t, err)
	assert.True(t, pbss.Equal(pbss2))

	assert.Equal(t, Rate, pbss2.CommissionRate())
	assert.Equal(t, MaxRate, pbss2.MaxCommissionRate())
	assert.Equal(t, MaxChangeRate, pbss2.MaxCommissionChangeRate())
}

func TestPRepInfo_Validate1(t *testing.T) {
	type fields struct {
		City        *string
		Country     *string
		Details     *string
		Email       *string
		Name        *string
		P2PEndpoint *string
		WebSite     *string
		Node        module.Address
	}
	type args struct {
		revision int
		reg      bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"RegisterNormal",
			fields{
				City:        NewStringPtr("Seoul"),
				Country:     NewStringPtr("KOR"),
				Details:     NewStringPtr("https://test.url/test/"),
				Email:       NewStringPtr("test@google.com"),
				Name:        NewStringPtr("Test User"),
				P2PEndpoint: NewStringPtr("192.168.1.1:9080"),
				WebSite:     NewStringPtr("http://test.url/home"),
			},
			args{icmodule.RevisionDecentralize, true},
			false,
		},
		{
			"UpdateWithAField1",
			fields {
				Country: NewStringPtr("USA"),
			},
			args { icmodule.RevisionDecentralize, false },
			false,
		},
		{
			"RegWithMissingField1",
			fields {
				Country:     NewStringPtr("KOR"),
				Details:     NewStringPtr("https://test.url/test/"),
				Email:       NewStringPtr("test@google.com"),
				Name:        NewStringPtr("Test User"),
				P2PEndpoint: NewStringPtr("192.168.1.1:9080"),
				WebSite:     NewStringPtr("https://test.url/home"),
			},
			args { icmodule.RevisionDecentralize, true },
			true,
		},
		{
			"RegWithEmptyField1",
			fields {
				City:        NewStringPtr("  "),
				Country:     NewStringPtr("KOR"),
				Details:     NewStringPtr("https://test.url/test/"),
				Email:       NewStringPtr("test@google.com"),
				Name:        NewStringPtr("Test User"),
				P2PEndpoint: NewStringPtr("192.168.1.1:9080"),
				WebSite:     NewStringPtr("http://test.url/home"),
			},
			args { icmodule.RevisionDecentralize, true },
			true,
		},
		{
			"UpdateWithEmptyField1",
			fields {
				City: NewStringPtr("  "),
			},
			args { icmodule.RevisionDecentralize, false },
			true,
		},
		{
			"InvalidCountryField",
			fields {
				Country: NewStringPtr("PUH"),
			},
			args { icmodule.RevisionDecentralize, false },
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PRepInfo{
				City:        tt.fields.City,
				Country:     tt.fields.Country,
				Details:     tt.fields.Details,
				Email:       tt.fields.Email,
				Name:        tt.fields.Name,
				P2PEndpoint: tt.fields.P2PEndpoint,
				WebSite:     tt.fields.WebSite,
				Node:        tt.fields.Node,
			}
			if err := r.Validate(tt.args.revision, tt.args.reg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
