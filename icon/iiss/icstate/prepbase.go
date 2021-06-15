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
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	prepVersion1  = iota + 1
	prepVersion   = prepVersion1
	bonderListMax = 10
)

type RegInfo struct {
	city        string
	country     string
	details     string
	email       string
	name        string
	p2pEndpoint string
	website     string
	node        *common.Address
}

func (r *RegInfo) SetNode(node module.Address) {
	r.node = common.AddressToPtr(node)
}

func (r *RegInfo) Node() module.Address {
	if r.node == nil {
		return nil
	}
	return r.node
}

func (r *RegInfo) String() string {
	return fmt.Sprintf(
		"city=%s country=%s details=%s email=%s name=%s p2p=%s website=%s node=%s",
		r.city, r.country, r.details, r.email, r.name, r.p2pEndpoint, r.website, r.node,
	)
}

func (r *RegInfo) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"RegInfo{city=%s country=%s details=%s email=%s p2p=%s website=%s node=%s}",
				r.city, r.country, r.details, r.email, r.p2pEndpoint, r.website, r.node)
		} else {
			fmt.Fprintf(f, "RegInfo{%s %s %s %s %s %s %s}",
				r.city, r.country, r.details, r.email, r.p2pEndpoint, r.website, r.node)
		}
	case 's':
		fmt.Fprint(f, r.String())
	}
}

func (r *RegInfo) Set(other *RegInfo) {
	r.city = other.city
	r.country = other.country
	r.details = other.details
	r.email = other.email
	r.name = other.name
	r.p2pEndpoint = other.p2pEndpoint
	r.website = other.website
	r.node = other.node
}

func (r *RegInfo) Update(other *RegInfo) {
	if len(other.city) != 0 {
		r.city = other.city
	}
	if len(other.country) != 0 {
		r.country = other.country
	}
	if len(other.details) != 0 {
		r.details = other.details
	}
	if len(other.email) != 0 {
		r.email = other.email
	}
	if len(other.name) != 0 {
		r.name = other.name
	}
	if len(other.p2pEndpoint) != 0 {
		r.p2pEndpoint = other.p2pEndpoint
	}
	if len(other.website) != 0 {
		r.website = other.website
	}
	if other.node != nil {
		r.node = other.node
	}
}

func (r *RegInfo) Validate(revision int) error {
	if err := icutils.ValidateEndpoint(r.p2pEndpoint); err != nil {
		return err
	}
	if err := icutils.ValidateURL(r.website); err != nil {
		return err
	}
	if err := icutils.ValidateURL(r.details); err != nil {
		return err
	}
	if err := icutils.ValidateEmail(r.email, revision); err != nil {
		return err
	}
	return nil
}

func (r *RegInfo) Clone() *RegInfo {
	return &RegInfo{
		city:        r.city,
		country:     r.country,
		details:     r.details,
		email:       r.email,
		name:        r.name,
		p2pEndpoint: r.p2pEndpoint,
		website:     r.website,
		node:        r.node,
	}
}

func (r *RegInfo) Equal(other *RegInfo) bool {
	if r == other {
		return true
	}
	return r.city == other.city &&
		r.country == other.country &&
		r.details == other.details &&
		r.email == other.email &&
		r.name == other.name &&
		r.p2pEndpoint == other.p2pEndpoint &&
		r.website == other.website &&
		icutils.EqualAddress(r.node, other.node)
}

func (r *RegInfo) IsEmpty() bool {
	return r.city == "" &&
		r.country == "" &&
		r.details == "" &&
		r.email == "" &&
		r.name == "" &&
		r.p2pEndpoint == "" &&
		r.website == "" &&
		r.node == nil
}

func (r *RegInfo) Clear() {
	r.city = ""
	r.country = ""
	r.details = ""
	r.email = ""
	r.name = ""
	r.p2pEndpoint = ""
	r.website = ""
	r.node = nil
}

func NewRegInfo(city, country, details, email, name, p2pEndpoint, website string, node module.Address) *RegInfo {
	return &RegInfo{
		city:        city,
		country:     country,
		details:     details,
		email:       email,
		name:        name,
		p2pEndpoint: p2pEndpoint,
		website:     website,
		node:        common.AddressToPtr(node),
	}
}

// ===================================================

type PRepBase struct {
	icobject.NoDatabase
	StateAndSnapshot
	// database variables
	RegInfo

	irep       *big.Int
	irepHeight int64
	bonderList BonderList
}

func (p *PRepBase) IRep() *big.Int {
	return p.irep
}

func (p *PRepBase) IRepHeight() int64 {
	return p.irepHeight
}

func (p *PRepBase) GetNode(owner module.Address) module.Address {
	if p.node != nil {
		return p.node
	}
	return owner
}

func (p *PRepBase) equal(other *PRepBase) bool {
	if p == other {
		return true
	}

	return p.RegInfo.Equal(&other.RegInfo) &&
		p.irep.Cmp(other.irep) == 0 &&
		p.irepHeight == other.irepHeight &&
		p.bonderList.Equal(other.bonderList)
}

func (p *PRepBase) Set(other *PRepBase) {
	p.checkWritable()

	p.RegInfo.Set(&other.RegInfo)
	p.irep = other.irep
	p.irepHeight = other.irepHeight
	p.bonderList = other.bonderList.Clone()
}

func (p *PRepBase) Clone() *PRepBase {
	pb := &PRepBase{
		irep:       p.irep,
		irepHeight: p.irepHeight,
		bonderList: p.bonderList.Clone(),
	}
	pb.RegInfo.Set(&p.RegInfo)
	return pb
}

func (p *PRepBase) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["name"] = p.name
	jso["country"] = p.country
	jso["city"] = p.city
	jso["email"] = p.email
	jso["website"] = p.website
	jso["details"] = p.details
	jso["p2pEndpoint"] = p.p2pEndpoint
	jso["nodeAddress"] = p.node
	jso["irep"] = p.irep
	jso["irepUpdateBlockHeight"] = p.irepHeight
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
		p.irep,
		p.irepHeight,
		p.bonderList); err != nil {
		return err
	}
	return nil
}

func (p *PRepBase) RLPDecodeFields(d codec.Decoder) error {
	p.checkWritable()

	if err := d.DecodeListOf(
		&p.name,
		&p.country,
		&p.city,
		&p.email,
		&p.website,
		&p.details,
		&p.p2pEndpoint,
		&p.node,
		&p.irep,
		&p.irepHeight,
		&p.bonderList); err != nil {
		return errors.Wrap(err, "Fail to decode PRepBase")
	}
	return nil
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
	p.RegInfo.Clear()
	p.irep = new(big.Int)
	p.irepHeight = 0
}

func (p *PRepBase) IsEmpty() bool {
	return p.RegInfo.IsEmpty() &&
		p.irep.Sign() == 0 &&
		p.irepHeight == 0
}

func (p *PRepBase) SetRegInfo(ri *RegInfo) error {
	p.checkWritable()
	p.RegInfo.Set(ri)
	return nil
}

func (p *PRepBase) ApplyRegInfo(ri *RegInfo) error {
	p.checkWritable()
	p.RegInfo.Update(ri)
	return nil
}

func (p *PRepBase) SetNode(node module.Address) {
	p.checkWritable()
	p.RegInfo.SetNode(node)
}

func (p *PRepBase) SetIrep(irep *big.Int, irepHeight int64) {
	p.checkWritable()
	p.irep = irep
	p.irepHeight = irepHeight
}

func (p *PRepBase) SetBonderList(bonderList BonderList) {
	p.checkWritable()
	p.bonderList = bonderList
}

func (p *PRepBase) BonderList() BonderList {
	return p.bonderList
}

func (p *PRepBase) GetBonderListInJSON() []interface{} {
	return p.bonderList.ToJSON()
}

func newPRepBaseWithTag(_ icobject.Tag) *PRepBase {
	return new(PRepBase)
}

func NewPRepBase() *PRepBase {
	return &PRepBase{
		irep: new(big.Int),
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
		return nil, scoreresult.InvalidParameterError.Errorf("Too many bonder List %d", count)
	}
	bonderList := make([]*common.Address, count)
	bonderMap := make(map[string]int)

	for i, p := range param {
		b := new(common.Address)
		bs, err := json.Marshal(p)
		if err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "Failed to get bonder list")
		}
		if err = json.Unmarshal(bs, b); err != nil {
			return nil, scoreresult.IllegalFormatError.Wrapf(err, "Failed to get bonder list")
		}

		key := icutils.ToKey(b)
		if bonderMap[key] > 0 {
			return nil, scoreresult.InvalidParameterError.Errorf("Duplicate bonder: %v", b)
		}
		bonderMap[key]++
		bonderList[i] = b
	}
	return bonderList, nil
}
