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

func (r *RegInfo) GetNode(owner module.Address) module.Address {
	if r.node == nil {
		return owner
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

func (r *RegInfo) Set(other *RegInfo) *RegInfo {
	*r = *other
	return r
}

func (r *RegInfo) Update(other *RegInfo) bool {
	var dirty bool
	if len(other.city) != 0 && r.city != other.city {
		r.city = other.city
		dirty = true
	}
	if len(other.country) != 0 && r.country != other.country {
		r.country = other.country
		dirty = true
	}
	if len(other.details) != 0 && r.details != other.details {
		r.details = other.details
		dirty = true
	}
	if len(other.email) != 0 && r.email != other.email {
		r.email = other.email
		dirty = true
	}
	if len(other.name) != 0 && r.name != other.name {
		r.name = other.name
		dirty = true
	}
	if len(other.p2pEndpoint) != 0 && r.p2pEndpoint != other.p2pEndpoint {
		r.p2pEndpoint = other.p2pEndpoint
		dirty = true
	}
	if len(other.website) != 0 && r.website != other.website {
		r.website = other.website
		dirty = true
	}
	if other.node != nil && !other.node.Equal(r.node){
		r.node = other.node
		dirty = true
	}
	return dirty
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
	return new(RegInfo).Set(r)
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

type PRepBaseData struct {
	info       RegInfo
	irep       *big.Int
	irepHeight int64
	bonderList BonderList
}

func (p *PRepBaseData) RegInfo() RegInfo {
	return p.info
}

func (p *PRepBaseData) IRep() *big.Int {
	return p.irep
}

func (p *PRepBaseData) IRepHeight() int64 {
	return p.irepHeight
}

func (p *PRepBaseData) equal(p2 *PRepBaseData) bool {
	if p == p2 {
		return true
	}
	return p.info.Equal(&p2.info) &&
		p.irep.Cmp(p2.irep) == 0 &&
		p.irepHeight == p2.irepHeight &&
		p.bonderList.Equal(p2.bonderList)
}

func (p *PRepBaseData) GetNode(owner module.Address) module.Address {
	return p.info.GetNode(owner)
}

func (p *PRepBaseData) IsEmpty() bool {
	return p.info.IsEmpty() &&
		p.irep.Sign() == 0 &&
		p.irepHeight == 0 &&
		p.bonderList.IsEmpty()
}

func (p *PRepBaseData) BonderList() BonderList {
	return p.bonderList
}

func (p *PRepBaseData) GetBonderListInJSON() []interface{} {
	return p.bonderList.ToJSON()
}

func (p PRepBaseData) clone() PRepBaseData {
	return PRepBaseData{
		info:       p.info,
		irep:       p.irep,
		irepHeight: p.irepHeight,
		bonderList: p.bonderList.Clone(),
	}
}

func (p *PRepBaseData) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["name"] = p.info.name
	jso["country"] = p.info.country
	jso["city"] = p.info.city
	jso["email"] = p.info.email
	jso["website"] = p.info.website
	jso["details"] = p.info.details
	jso["p2pEndpoint"] = p.info.p2pEndpoint
	if p.info.node != nil {
		jso["nodeAddress"] = p.info.node
	}
	jso["irep"] = p.irep
	jso["irepUpdateBlockHeight"] = p.irepHeight
	return jso
}

func (p *PRepBaseData) String() string {
	return fmt.Sprintf("PRepBase{RegInfo{%s} irep=%d irepHeight=%d}",
		p.info.String(), p.irep, p.irepHeight)
}

func (p *PRepBaseData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "PRepBase{%+v irep=%d irepHeight=%d}", &p.info, p.irep, p.irepHeight)
		} else {
			fmt.Fprintf(f, "PRepBase{%v %d %d}", &p.info, p.irep, p.irepHeight)
		}
	case 's':
		fmt.Fprint(f, p.String())
	}
}

type PRepBaseSnapshot struct {
	icobject.NoDatabase
	PRepBaseData
}

func (p *PRepBaseSnapshot) Version() int {
	return prepVersion
}

func (p *PRepBaseSnapshot) RLPEncodeFields(e codec.Encoder) error {
	return e.EncodeMulti(
		p.info.name,
		p.info.country,
		p.info.city,
		p.info.email,
		p.info.website,
		p.info.details,
		p.info.p2pEndpoint,
		p.info.node,
		p.irep,
		p.irepHeight,
		p.bonderList,
	)
}

func (p *PRepBaseSnapshot) RLPDecodeFields(d codec.Decoder) error {
	return d.DecodeAll(
		&p.info.name,
		&p.info.country,
		&p.info.city,
		&p.info.email,
		&p.info.website,
		&p.info.details,
		&p.info.p2pEndpoint,
		&p.info.node,
		&p.irep,
		&p.irepHeight,
		&p.bonderList,
	)
}

func (p *PRepBaseSnapshot) Equal(object icobject.Impl) bool {
	other, ok := object.(*PRepBaseSnapshot)
	if !ok {
		return false
	}
	if p == other {
		return true
	}

	return p.PRepBaseData.equal(&other.PRepBaseData)
}


type PRepBaseState struct {
	PRepBaseData
	last *PRepBaseSnapshot
}

func (p *PRepBaseState) GetSnapshot() *PRepBaseSnapshot {
	if p.last == nil {
		p.last = &PRepBaseSnapshot {
			PRepBaseData: p.PRepBaseData.clone(),
		}
	}
	return p.last
}

func (p *PRepBaseState) setDirty() {
	if p.last != nil {
		p.last = nil
	}
}

func (p *PRepBaseState) Clear() {
	p.Reset(emptyPRepBaseSnapshot)
}

func (p *PRepBaseState) UpdateRegInfo(ri *RegInfo) error {
	if p.info.Update(ri) {
		p.setDirty()
	}
	return nil
}

func (p *PRepBaseState) SetRegInfo(ri *RegInfo) error {
	p.info.Set(ri)
	p.setDirty()
	return nil
}

func (p *PRepBaseState) SetIrep(irep *big.Int, irepHeight int64) {
	p.irep = irep
	p.irepHeight = irepHeight
	p.setDirty()
}

func (p *PRepBaseState) SetBonderList(bonderList BonderList) {
	p.bonderList = bonderList
	p.setDirty()
}

func (p *PRepBaseState) Reset(snapshot *PRepBaseSnapshot) *PRepBaseState {
	if p.last != snapshot {
		p.last = snapshot
		p.PRepBaseData = snapshot.PRepBaseData.clone()
	}
	return p
}

var emptyPRepBaseSnapshot = &PRepBaseSnapshot{
	PRepBaseData: PRepBaseData{
		irep: new(big.Int),
	},
}

func newPRepBaseWithTag(_ icobject.Tag) *PRepBaseSnapshot {
	return new(PRepBaseSnapshot)
}

func NewPRepBaseState() *PRepBaseState {
	return new(PRepBaseState).Reset(emptyPRepBaseSnapshot)
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

func (bl BonderList) IsEmpty() bool {
	return len(bl) == 0
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
