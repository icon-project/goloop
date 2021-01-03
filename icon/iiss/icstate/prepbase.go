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
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

const (
	prepVersion1  = iota + 1
	prepVersion   = prepVersion1
	bonderListMax = 10
)

type PRepBase struct {
	icobject.NoDatabase
	StateAndSnapshot

	// memory variables
	readonly bool
	owner    module.Address

	// database variables
	name        string
	country     string
	city        string
	email       string
	website     string
	details     string
	p2pEndpoint string
	node        module.Address
	bonderList  BonderList
}

func (p *PRepBase) equal(other *PRepBase) bool {
	if p == other {
		return true
	}

	return icutils.EqualAddress(p.owner, other.owner) &&
		p.name == other.name &&
		p.country == other.country &&
		p.city == other.city &&
		p.email == other.email &&
		p.website == other.website &&
		p.details == other.details &&
		p.p2pEndpoint == other.p2pEndpoint &&
		icutils.EqualAddress(p.node, other.node) &&
		p.bonderList.Equal(other.bonderList)
}

func (p *PRepBase) Owner() module.Address {
	return p.owner
}

func (p *PRepBase) SetOwner(owner module.Address) {
	p.checkWritable()
	p.owner = owner
}

func (p *PRepBase) GetNode() module.Address {
	if p.node != nil {
		return p.node
	}
	return p.owner
}

func (p *PRepBase) Set(other *PRepBase) {
	p.checkWritable()

	p.owner = other.owner
	p.name = other.name
	p.country = other.country
	p.city = other.city
	p.email = other.email
	p.website = other.website
	p.details = other.details
	p.p2pEndpoint = other.p2pEndpoint
	p.node = other.node
	p.bonderList = other.bonderList.Clone()
}

func (p *PRepBase) Clone() *PRepBase {
	return &PRepBase{
		readonly:    false,
		owner:       p.owner,
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

func (p *PRepBase) GetSnapshot() *PRepBase {
	if p.IsReadonly() {
		return p
	}
	ret := p.Clone()
	ret.freeze()
	return ret
}

func (p *PRepBase) ToJSON() map[string]interface{} {
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

func (p *PRepBase) RLPEncodeFields(e codec.Encoder) error {
	if err := e.EncodeListOf(
		p.name,
		p.country,
		p.city,
		p.email,
		p.website,
		p.details,
		p.p2pEndpoint,
		p.node,
		p.bonderList); err != nil {
		return err
	}
	return nil
}

func (p *PRepBase) RLPDecodeFields(d codec.Decoder) error {
	p.checkWritable()

	var node *common.Address

	if err := d.DecodeListOf(
		&p.name,
		&p.country,
		&p.city,
		&p.email,
		&p.website,
		&p.details,
		&p.p2pEndpoint,
		&node,
		&p.bonderList); err != nil {
		return errors.Wrap(err, "Fail to decode PRepBase")
	}
	p.node = node
	return nil
}

func (p *PRepBase) freeze() {
	p.readonly = true
}

func (p *PRepBase) Version() int {
	return prepVersion
}

func (p *PRepBase) Equal(object icobject.Impl) bool {
	other, ok := object.(*PRepBase)
	if !ok {
		return false
	}
	if p == other {
		return true
	}

	return p.equal(other)
}

func (p *PRepBase) Clear() {
	p.checkWritable()

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

func (p *PRepBase) IsEmpty() bool {
	return p == nil || p.owner == nil
}

func (p *PRepBase) SetPRep(name, email, website, country, city, details, endpoint string, node module.Address) error {
	p.checkWritable()

	p.name = name
	p.email = email
	p.website = website
	p.country = country
	p.city = city
	p.details = details
	p.p2pEndpoint = endpoint
	p.node = node
	return nil
}

func (p *PRepBase) SetBonderList(bonderList BonderList) {
	p.bonderList = bonderList
}

func (p *PRepBase) BonderList() BonderList {
	return p.bonderList
}

func (p *PRepBase) GetBonderListInJSON() []interface{} {
	return p.bonderList.ToJSON()
}

func newPRepBaseWithTag(_ icobject.Tag) *PRepBase {
	return &PRepBase{}
}

func NewPRepBase(owner module.Address) *PRepBase {
	return &PRepBase{
		owner: owner,
	}
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

	dst := make([]*common.Address, size)
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
