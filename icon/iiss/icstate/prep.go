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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	prepVersion1 = iota + 1
	prepVersion  = prepVersion1
)

type PRepSnapshot struct {
	NoDatabaseObject
	name        string
	country     string
	city        string
	email       string
	website     string
	details     string
	p2pEndpoint string
	node        *common.Address
}

func (p *PRepSnapshot) Version() int {
	return prepVersion
}

func (p *PRepSnapshot) RLPEncodeFields(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(p.name, p.country, p.city, p.email, p.website, p.details, p.p2pEndpoint, p.node); err != nil {
		return err
	}
	return nil
}

func (p *PRepSnapshot) RLPDecodeFields(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(&p.name, &p.country, &p.city, &p.email, &p.website, &p.details, &p.p2pEndpoint, &p.node); err != nil {
		return errors.Wrap(err, "Fail to decode PRepSnapshot")
	}
	return nil
}

func (p *PRepSnapshot) Equal(object ObjectImpl) bool {
	ps, ok := object.(*PRepSnapshot)
	if !ok {
		return false
	}
	if ps == p {
		return true
	}
	return p.name == ps.name &&
		p.country == ps.country &&
		p.city == ps.city &&
		p.email == ps.email &&
		p.website == ps.website &&
		p.details == ps.details &&
		p.p2pEndpoint == ps.p2pEndpoint
}

func newPRepSnapshot(tag Tag) *PRepSnapshot {
	return &PRepSnapshot{}
}

type PRepState struct {
	address     module.Address
	name        string
	country     string
	city        string
	email       string
	website     string
	details     string
	p2pEndpoint string
	//registerBlockHeight uint
	//registerTxIndex     uint
	//iRep                common.HexInt
	//iRepBlockHeight     uint
	node *common.Address
	//bondList            []common.Address
}

func newPRepState(address module.Address) *PRepState {
	return &PRepState{
		address: address,
	}
}

func (p *PRepState) Clear() {
	p.name = ""
	p.country = ""
	p.city = ""
	p.email = ""
	p.website = ""
	p.details = ""
	p.p2pEndpoint = ""
	p.node = nil
}

func (p *PRepState) Reset(ps *PRepSnapshot) {
	p.name = ps.name
	p.country = ps.country
	p.city = ps.city
	p.email = ps.email
	p.website = ps.website
	p.details = ps.details
	p.p2pEndpoint = ps.p2pEndpoint
	p.node = ps.node
}

func (p *PRepState) GetSnapshot() *PRepSnapshot {
	ps := &PRepSnapshot{}
	ps.name = p.name
	ps.country = p.country
	ps.city = p.city
	ps.email = p.email
	ps.website = p.website
	ps.details = p.details
	ps.p2pEndpoint = p.p2pEndpoint
	ps.node = p.node
	return ps
}

func (p PRepState) IsEmpty() bool {
	return p.name == ""
}

func (p PRepState) GetAddress() module.Address {
	return p.address
}

func (p *PRepState) SetPRep(name string, email string, website string, country string,
	city string, details string, endpoint string, node module.Address) error {
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

func (p *PRepState) GetPRep() map[string]interface{} {
	data := make(map[string]interface{})
	data["name"] = p.name
	data["email"] = p.email
	data["website"] = p.website
	data["country"] = p.country
	data["city"] = p.city
	data["details"] = p.details
	data["p2pEndpoint"] = p.p2pEndpoint
	data["node"] = p.node
	return data
}

func NewPRepStateWithSnapshot(a module.Address, ss *PRepSnapshot) *PRepState {
	ps := newPRepState(a)
	ps.Reset(ss)
	return ps
}
