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
	"encoding/json"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

const (
	prepVersion1  = iota + 1
	prepVersion   = prepVersion1
	bonderListMax = 10
)

type PRepData struct {
	name        string
	country     string
	city        string
	email       string
	website     string
	details     string
	p2pEndpoint string
	node        *common.Address
	bonderList  BonderList
}

func (p *PRepData) Equal(other *PRepData) bool {
	if p == other {
		return true
	}

	return p.name == other.name &&
		p.country == other.country &&
		p.city == other.city &&
		p.email == other.email &&
		p.website == other.website &&
		p.details == other.details &&
		p.p2pEndpoint == other.p2pEndpoint &&
		p.node.Equal(other.node) &&
		p.bonderList.Equal(other.bonderList)
}

func (p *PRepData) Set(other *PRepData) {
	*p = *other
	p.bonderList = other.bonderList.Clone()
}

func (p *PRepData) Clone() *PRepData {
	return &PRepData{
		name:        p.name,
		city:        p.city,
		country:     p.country,
		email:       p.email,
		website:     p.website,
		details:     p.details,
		p2pEndpoint: p.p2pEndpoint,
		node:        p.node,
		bonderList:  p.bonderList.Clone(),
	}
}

func (p *PRepData) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["name"] = p.name
	jso["email"] = p.email
	jso["website"] = p.website
	jso["country"] = p.country
	jso["city"] = p.city
	jso["details"] = p.details
	jso["p2pEndpoint"] = p.p2pEndpoint
	jso["node"] = p.node
	return jso
}

type PRepSnapshot struct {
	icobject.NoDatabase
	owner module.Address
	*PRepData
}

func (p *PRepSnapshot) Owner() module.Address {
	return p.owner
}

func (p *PRepSnapshot) SetOwner(address module.Address) {
	p.owner = address
}

func (p *PRepSnapshot) Version() int {
	return prepVersion
}

func (p *PRepSnapshot) RLPEncodeFields(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(p.name, p.country, p.city, p.email, p.website, p.details, p.p2pEndpoint, p.node, p.bonderList); err != nil {
		return err
	}
	return nil
}

func (p *PRepSnapshot) RLPDecodeFields(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(&p.name, &p.country, &p.city, &p.email, &p.website, &p.details, &p.p2pEndpoint, &p.node, &p.bonderList); err != nil {
		return errors.Wrap(err, "Fail to decode PRepSnapshot")
	}
	return nil
}

func (p *PRepSnapshot) Equal(object icobject.Impl) bool {
	ps, ok := object.(*PRepSnapshot)
	if !ok {
		return false
	}
	if ps == p {
		return true
	}
	return p.PRepData.Equal(ps.PRepData)
}

func newPRepSnapshot(_ icobject.Tag) *PRepSnapshot {
	return &PRepSnapshot{PRepData: &PRepData{}}
}

type PRepState struct {
	owner module.Address
	*PRepData
}

func (p *PRepState) Owner() module.Address {
	return p.owner
}

func newPRepState(owner module.Address) *PRepState {
	return &PRepState{owner: owner}
}

func (p *PRepState) Clear() {
	p.owner = nil
	p.city = ""
	p.country = ""
	p.details = ""
	p.email = ""
	p.name = ""
	p.node = nil
	p.p2pEndpoint = ""
	p.website = ""
}

func (p *PRepState) Reset(ps *PRepSnapshot) {
	p.PRepData.Set(ps.PRepData)
}

func (p *PRepState) GetSnapshot() *PRepSnapshot {
	return &PRepSnapshot{PRepData: p.PRepData.Clone()}
}

func (p PRepState) IsEmpty() bool {
	return p.name == ""
}

func (p *PRepState) SetPRep(name, email, website, country, city, details, endpoint string, node module.Address) error {
	p.name = name
	p.email = email
	p.website = website
	p.country = country
	p.city = city
	p.details = details
	p.p2pEndpoint = endpoint
	p.node = node.(*common.Address)
	return nil
}

func (p *PRepState) SetBonderList(bonderList BonderList) {
	p.bonderList = bonderList
}

func (p *PRepState) BonderList() BonderList {
	return p.bonderList
}

func (p *PRepState) GetBonderListInJSON() []interface{} {
	return p.bonderList.ToJSON()
}

func NewPRepStateWithSnapshot(a module.Address, ss *PRepSnapshot) *PRepState {
	ps := newPRepState(a)
	ps.Reset(ss)
	return ps
}

type BonderList []*common.Address

func getMaxBonderListCount() int {
	return bonderListMax
}

func (bl BonderList) Equal(bl2 BonderList) bool {
	if len(bl) != len(bl2) {
		return false
	}
	for i, d := range bl {
		if !d.Equal(bl2[i]) {
			return false
		}
	}
	return true
}

func (bl BonderList) Contains(a module.Address) bool {
	for _, b := range bl {
		if b.Equal(a) {
			return true
		}
	}
	return false
}

func (bl BonderList) Clone() BonderList {
	size := len(bl)
	if size == 0 {
		return nil
	}

	dst := make([]*common.Address, 0, size)
	copy(dst, bl)

	return dst
}

func (bl BonderList) ToJSON() []interface{} {
	size := len(bl)
	jso := make([]interface{}, size, size)
	for i, b := range bl {
		jso[i] = b
	}
	return jso
}

func NewBonderList(param []interface{}) (BonderList, error) {
	count := len(param)
	if count > getMaxBonderListCount() {
		return nil, errors.Errorf("Too many bonder List %d", count)
	}
	bl := make([]*common.Address, 0)
	for _, p := range param {
		b := new(common.Address)
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get address %v", err)
		}
		if err = json.Unmarshal(bs, b); err != nil {
			return nil, errors.IllegalArgumentError.Errorf("Failed to get address %v", err)
		}
		bl = append(bl, b)
	}
	return bl, nil
}
