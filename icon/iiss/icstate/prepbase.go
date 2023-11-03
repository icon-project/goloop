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
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	PRepBaseVersion1 = iota + 1
	PRepBaseVersion2

	bonderListMax = 10
)

type PRepInfo struct {
	City        *string
	Country     *string
	Details     *string
	Email       *string
	Name        *string
	P2PEndpoint *string
	WebSite     *string
	Node        module.Address
}

// checkStringPtrValue check field value.
// mandatory should be true if it's required field.
// n is name of field for errors.
func checkStringPtrValue(s *string, name string, mandatory bool) (bool, error) {
	if s == nil {
		if mandatory {
			return false, errors.IllegalArgumentError.Errorf("MandatoryField(field=%s)", name)
		} else {
			return false, nil
		}
	}
	if len(strings.TrimSpace(*s)) == 0 {
		return false, errors.IllegalArgumentError.Errorf("EmptyField(field=%s)", name)
	}
	return true, nil
}

// Validate check validity of fields
// reg: whether it's for registration
// revision: revision value
func (r *PRepInfo) Validate(revision int, reg bool) error {
	if _, err := checkStringPtrValue(r.Name, "name", reg); err != nil {
		return err
	}
	if _, err := checkStringPtrValue(r.City, "city", reg); err != nil {
		return err
	}
	if has, err := checkStringPtrValue(r.Country, "country", reg); err != nil {
		return err
	} else if has {
		if err = icutils.ValidateCountryAlpha3(*r.Country); err != nil {
			return err
		}
	}
	if has, err := checkStringPtrValue(r.Details, "details", reg); err != nil {
		return err
	} else if has {
		if err = icutils.ValidateURL(*r.Details); err != nil {
			return errors.IllegalArgumentError.Wrap(err, "InvalidURL(field=details)")
		}
	}
	if has, err := checkStringPtrValue(r.P2PEndpoint, "p2pEndpoint", reg); err != nil {
		return err
	} else if has {
		if err = icutils.ValidateEndpoint(*r.P2PEndpoint); err != nil {
			return err
		}
	}
	if has, err := checkStringPtrValue(r.WebSite, "website", reg); err != nil {
		return err
	} else if has {
		if err = icutils.ValidateURL(*r.WebSite); err != nil {
			return errors.IllegalArgumentError.Wrap(err, "InvalidURL(field=website)")
		}
	}
	if has, err := checkStringPtrValue(r.Email, "email", reg); err != nil {
		return err
	} else if has {
		if err = icutils.ValidateEmail(*r.Email, revision); err != nil {
			return err
		}
	}
	return nil
}

func (r *PRepInfo) GetNode(owner module.Address) module.Address {
	if r.Node != nil {
		return r.Node
	}
	return owner
}

func toString(sp *string) string {
	if sp == nil {
		return "nil"
	} else {
		return *sp
	}
}

func (r *PRepInfo) String() string {
	return fmt.Sprintf(
		"PRepInfo{name=%s country=%s city=%s email=%s website=%s detail=%s p2pEndpoint=%s node=%v}",
		toString(r.Name), toString(r.Country), toString(r.City),
		toString(r.Email), toString(r.WebSite), toString(r.Details), toString(r.P2PEndpoint), r.Node,
	)
}

func equalStrPtr(s1, s2 *string) bool {
	if s1 == s2 {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	return *s1 == *s2
}

func (r *PRepInfo) equal(r2 *PRepInfo) bool {
	if r == r2 {
		return true
	}
	return equalStrPtr(r.City, r2.City) &&
		equalStrPtr(r.Country, r2.Country) &&
		equalStrPtr(r.Details, r2.Details) &&
		equalStrPtr(r.Email, r2.Email) &&
		equalStrPtr(r.Name, r2.Name) &&
		equalStrPtr(r.P2PEndpoint, r2.P2PEndpoint) &&
		equalStrPtr(r.WebSite, r2.WebSite) &&
		common.AddressEqual(r.Node, r2.Node)
}

type PRepBaseData struct {
	version int

	// Fields in version1
	city        string
	country     string
	details     string
	email       string
	name        string
	p2pEndpoint string
	website     string
	node        *common.Address
	irep        *big.Int
	irepHeight  int64
	bonderList  BonderList
	// Fields in version2
	ci *CommissionInfo
}

func (p *PRepBaseData) Version() int {
	return p.version
}

func (p *PRepBaseData) migrateVersion(version int) {
	if version > p.version {
		p.version = version
	}
}

func (p *PRepBaseData) Name() string {
	return p.name
}

func (p *PRepBaseData) P2PEndpoint() string {
	return p.p2pEndpoint
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
	if p.Version() != p2.Version() {
		return false
	}
	ret := p.city == p2.city &&
		p.country == p2.country &&
		p.details == p2.details &&
		p.email == p2.email &&
		p.name == p2.name &&
		p.p2pEndpoint == p2.p2pEndpoint &&
		p.website == p2.website &&
		common.AddressEqual(p.node, p2.node) &&
		p.irep.Cmp(p2.irep) == 0 &&
		p.irepHeight == p2.irepHeight &&
		p.bonderList.Equal(p2.bonderList)
	if !ret {
		return false
	}
	if p.ci != nil {
		if !p.ci.Equal(p2.ci) {
			return false
		}
	}
	return true
}

func (p *PRepBaseData) GetNode(owner module.Address) module.Address {
	if p.node != nil {
		return p.node
	}
	return owner
}

func (p *PRepBaseData) IsEmpty() bool {
	return p.city == "" &&
		p.country == "" &&
		p.details == "" &&
		p.email == "" &&
		p.name == "" &&
		p.p2pEndpoint == "" &&
		p.website == "" &&
		p.node == nil &&
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
	p.bonderList = p.bonderList.Clone()
	if p.ci != nil {
		p.ci = p.ci.Clone()
	}
	return p
}

func (p *PRepBaseData) initCommissionInfo(rate, maxRate, maxChangeRate icmodule.Rate) error {
	ci, err := NewCommissionInfo(rate, maxRate, maxChangeRate)
	if err != nil {
		return err
	}
	if p.ci != nil {
		return icmodule.IllegalArgumentError.New("CommissionInfoAlreadyExists")
	}
	p.ci = ci
	return nil
}

func (p *PRepBaseData) CommissionRate() icmodule.Rate {
	if p.ci != nil {
		return p.ci.Rate()
	}
	return 0
}

func (p *PRepBaseData) MaxCommissionRate() icmodule.Rate {
	if p.ci != nil {
		return p.ci.MaxRate()
	}
	return 0
}

func (p *PRepBaseData) MaxCommissionChangeRate() icmodule.Rate {
	if p.ci != nil {
		return p.ci.MaxChangeRate()
	}
	return 0
}

func NewStringPtr(s string) *string {
	return &s
}

func (p *PRepBaseData) info() *PRepInfo {
	return &PRepInfo{
		City:        NewStringPtr(p.city),
		Country:     NewStringPtr(p.country),
		Details:     NewStringPtr(p.details),
		Email:       NewStringPtr(p.email),
		Name:        NewStringPtr(p.name),
		P2PEndpoint: NewStringPtr(p.p2pEndpoint),
		WebSite:     NewStringPtr(p.website),
		Node:        p.GetNode(nil),
	}
}

func (p *PRepBaseData) ToJSON(owner module.Address) map[string]interface{} {
	jso := make(map[string]interface{})
	jso["name"] = p.name
	jso["country"] = p.country
	jso["city"] = p.city
	jso["email"] = p.email
	jso["website"] = p.website
	jso["details"] = p.details
	jso["p2pEndpoint"] = p.p2pEndpoint
	if p.node != nil {
		jso["nodeAddress"] = p.node
	} else {
		jso["nodeAddress"] = owner
	}
	jso["irep"] = p.irep
	jso["irepUpdateBlockHeight"] = p.irepHeight

	if p.ci != nil {
		ci := p.ci
		ci.ToJSON(jso)
	}
	return jso
}

func (p *PRepBaseData) String() string {
	return fmt.Sprintf("PRepBase{city=%s country=%s details=%s email=%s p2p=%s website=%s node=%s irep=%d irepHeight=%d}",
		p.city, p.country, p.details, p.email, p.p2pEndpoint, p.website, p.node, p.irep, p.irepHeight)
}

func (p *PRepBaseData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f,
				"PRepBase{city=%s country=%s details=%s email=%s p2p=%s website=%s node=%s irep=%d irepHeight=%d}",
				p.city, p.country, p.details, p.email, p.p2pEndpoint, p.website, p.node, p.irep, p.irepHeight,
			)
		} else {
			fmt.Fprintf(f,
				"PRepBase{%s %s %s %s %s %s %s %d %d}",
				p.city, p.country, p.details, p.email, p.p2pEndpoint, p.website, p.node, p.irep, p.irepHeight,
			)
		}
	case 's':
		fmt.Fprint(f, p.String())
	}
}

type PRepBaseSnapshot struct {
	icobject.NoDatabase
	PRepBaseData
}

func (p *PRepBaseSnapshot) RLPEncodeFields(e codec.Encoder) error {
	err := e.EncodeMulti(
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
		p.bonderList,
	)
	if err == nil && p.Version() == PRepBaseVersion2 {
		err = e.Encode(p.ci)
	}
	return err
}

func (p *PRepBaseSnapshot) RLPDecodeFields(d codec.Decoder) error {
	err := d.DecodeAll(
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
		&p.bonderList,
	)
	if err == nil && p.Version() == PRepBaseVersion2 {
		err = d.Decode(&p.ci)
	}
	return err
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

func NewPRepBaseSnapshot(version int) *PRepBaseSnapshot {
	return &PRepBaseSnapshot{
		PRepBaseData: PRepBaseData{
			version: version,
		},
	}
}

type PRepBaseState struct {
	PRepBaseData
	last *PRepBaseSnapshot
}

func (p *PRepBaseState) GetSnapshot() *PRepBaseSnapshot {
	if p.last == nil {
		p.last = &PRepBaseSnapshot{
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

func (p *PRepBaseState) UpdateInfo(info *PRepInfo) {
	dirty := false
	if info.City != nil && *info.City != p.city {
		p.city = *info.City
		dirty = true
	}
	if info.Country != nil && *info.Country != p.country {
		p.country = *info.Country
		dirty = true
	}
	if info.Details != nil && *info.Details != p.details {
		p.details = *info.Details
		dirty = true
	}
	if info.Email != nil && *info.Email != p.email {
		p.email = *info.Email
		dirty = true
	}
	if info.Name != nil && *info.Name != p.name {
		p.name = *info.Name
		dirty = true
	}
	if info.P2PEndpoint != nil && *info.P2PEndpoint != p.p2pEndpoint {
		p.p2pEndpoint = *info.P2PEndpoint
		dirty = true
	}
	if info.WebSite != nil && *info.WebSite != p.website {
		p.website = *info.WebSite
		dirty = true
	}
	if info.Node != nil && !info.Node.Equal(p.node) {
		p.node = common.AddressToPtr(info.Node)
		dirty = true
	}
	if dirty {
		p.setDirty()
	}
	return
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

func (p *PRepBaseState) InitCommissionInfo(ci *CommissionInfo) error {
	if ci == nil {
		return scoreresult.InvalidParameterError.New("InvalidCommissionInfo")
	}
	if p.ci != nil {
		return icmodule.DuplicateError.New("CommissionInfoAlreadySet")
	}
	p.ci = ci
	p.migrateVersion(PRepBaseVersion2)
	p.setDirty()
	return nil
}

func (p *PRepBaseState) SetCommissionRate(rate icmodule.Rate) error {
	if p.ci == nil {
		return icmodule.NotFoundError.New("CommissionInfoNotFound")
	}
	if rate == p.ci.Rate() {
		return nil
	}
	err := p.ci.SetRate(rate)
	if err == nil {
		p.setDirty()
	}
	return err
}

func (p *PRepBaseState) CommissionInfoExists() bool {
	return p.ci != nil
}

var emptyPRepBaseSnapshot = &PRepBaseSnapshot{
	PRepBaseData: PRepBaseData{
		irep: new(big.Int),
	},
}

func newPRepBaseWithTag(tag icobject.Tag) *PRepBaseSnapshot {
	return NewPRepBaseSnapshot(tag.Version())
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
	jso := make([]interface{}, size)
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
