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

package icstage

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	GlobalVersion1 int = iota
	GlobalVersion2
	GlobalVersion3
)

type Global interface {
	icobject.Impl
	GetV1() *GlobalV1
	GetV2() *GlobalV2
	GetV3() *GlobalV3
	GetIISSVersion() int
	GetStartHeight() int64
	GetOffsetLimit() int
	GetTermPeriod() int
	GetRevision() int
	GetElectedPRepCount() int
	GetBondRequirement() int
	String() string
}

func newGlobal(tag icobject.Tag) (Global, error) {
	switch tag.Version() {
	case GlobalVersion1:
		return newGlobalV1(), nil
	case GlobalVersion2:
		return newGlobalV2(), nil
	case GlobalVersion3:
		return newGlobalV3(), nil
	default:
		return nil, errors.CriticalFormatError.Errorf("InvalidGlobalVersion(%d)", tag.Version())
	}
}

type globalBase struct {
	icobject.NoDatabase
	iissVersion      int
	startHeight      int64
	offsetLimit      int
	revision         int
	electedPRepCount int
	bondRequirement  int
}

func (g *globalBase) GetIISSVersion() int {
	return g.iissVersion
}

func (g *globalBase) GetStartHeight() int64 {
	return g.startHeight
}

func (g *globalBase) GetOffsetLimit() int {
	return g.offsetLimit
}

func (g *globalBase) GetTermPeriod() int {
	return g.offsetLimit + 1
}

func (g *globalBase) GetRevision() int {
	return g.revision
}

func (g *globalBase) GetElectedPRepCount() int {
	return g.electedPRepCount
}

func (g *globalBase) GetBondRequirement() int {
	return g.bondRequirement
}

type GlobalV1 struct {
	globalBase
	irep          *big.Int
	rrep          *big.Int
	mainPRepCount int
}

func (g *GlobalV1) Version() int {
	return GlobalVersion1
}

func (g *GlobalV1) GetIRep() *big.Int {
	return g.irep
}

func (g *GlobalV1) GetRRep() *big.Int {
	return g.rrep
}

func (g *GlobalV1) GetMainRepCount() int {
	return g.mainPRepCount
}

func (g *GlobalV1) GetBondRequirement() int {
	return 0
}

func (g *GlobalV1) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.iissVersion,
		&g.startHeight,
		&g.offsetLimit,
		&g.revision,
		&g.irep,
		&g.rrep,
		&g.mainPRepCount,
		&g.electedPRepCount,
	)
	return err
}

func (g *GlobalV1) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.revision,
		g.irep,
		g.rrep,
		g.mainPRepCount,
		g.electedPRepCount,
	)
}

func (g *GlobalV1) String() string {
	return fmt.Sprintf("revision=%d iissVersion=%d startHeight=%d offsetLimit=%d irep=%s rrep=%s "+
		"mainPRepCount=%d electedPRepCount=%d",
		g.revision,
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.irep,
		g.rrep,
		g.mainPRepCount,
		g.electedPRepCount,
	)
}

func (g *GlobalV1) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "GlobalV1{revision=%d iissVersion=%d startHeight=%d offsetLimit=%d irep=%s "+
				"rrep=%s mainPRepCount=%d electedPRepCount=%d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.irep,
				g.rrep,
				g.mainPRepCount,
				g.electedPRepCount,
			)
		} else {
			fmt.Fprintf(f, "GlobalV1{%d %d %d %d %s %s %d %d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.irep,
				g.rrep,
				g.mainPRepCount,
				g.electedPRepCount,
			)
		}
	}
}

func (g *GlobalV1) Equal(impl icobject.Impl) bool {
	if g2, ok := impl.(*GlobalV1); ok {
		return g.iissVersion == g2.iissVersion &&
			g.startHeight == g2.startHeight &&
			g.offsetLimit == g2.offsetLimit &&
			g.revision == g2.revision &&
			g.irep.Cmp(g2.irep) == 0 &&
			g.rrep.Cmp(g2.rrep) == 0 &&
			g.mainPRepCount == g2.mainPRepCount &&
			g.electedPRepCount == g2.electedPRepCount
	} else {
		return false
	}
}

func (g *GlobalV1) GetV1() *GlobalV1 {
	return g
}

func (g *GlobalV1) GetV2() *GlobalV2 {
	return nil
}

func (g *GlobalV1) GetV3() *GlobalV3 {
	return nil
}

func newGlobalV1() *GlobalV1 {
	return &GlobalV1{
		irep: new(big.Int),
		rrep: new(big.Int),
	}
}

func NewGlobalV1(
	iissVersion int,
	startHeight int64,
	offsetLimit int,
	revision int,
	irep *big.Int,
	rrep *big.Int,
	mainPRepCount int,
	electedPRepCount int,
) *GlobalV1 {
	return &GlobalV1{
		globalBase: globalBase{
			iissVersion:      iissVersion,
			startHeight:      startHeight,
			offsetLimit:      offsetLimit,
			revision:         revision,
			electedPRepCount: electedPRepCount,
		},
		irep:          irep,
		rrep:          rrep,
		mainPRepCount: mainPRepCount,
	}
}

type GlobalV2 struct {
	globalBase
	iglobal *big.Int
	iprep   *big.Int
	ivoter  *big.Int
	icps    *big.Int
	irelay  *big.Int
}

func (g *GlobalV2) Version() int {
	return GlobalVersion2
}

func (g *GlobalV2) GetIGlobal() *big.Int {
	return g.iglobal
}

func (g *GlobalV2) GetIPRep() *big.Int {
	return g.iprep
}

func (g *GlobalV2) GetIVoter() *big.Int {
	return g.ivoter
}

func (g *GlobalV2) GetICps() *big.Int {
	return g.icps
}

func (g *GlobalV2) GetIRelay() *big.Int {
	return g.irelay
}

func (g *GlobalV2) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.iissVersion,
		&g.startHeight,
		&g.offsetLimit,
		&g.revision,
		&g.iglobal,
		&g.iprep,
		&g.ivoter,
		&g.icps,
		&g.irelay,
		&g.electedPRepCount,
		&g.bondRequirement,
	)
	return err
}

func (g *GlobalV2) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.revision,
		g.iglobal,
		g.iprep,
		g.ivoter,
		g.icps,
		g.irelay,
		g.electedPRepCount,
		g.bondRequirement,
	)
}

func (g *GlobalV2) String() string {
	return fmt.Sprintf("revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%s "+
		"iprep=%s ivoter=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d",
		g.revision,
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.iglobal,
		g.iprep,
		g.ivoter,
		g.icps,
		g.irelay,
		g.electedPRepCount,
		g.bondRequirement,
	)
}

func (g *GlobalV2) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "GlobalV2{revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%s "+
				"iprep=%s ivoter=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.iglobal,
				g.iprep,
				g.ivoter,
				g.icps,
				g.irelay,
				g.electedPRepCount,
				g.bondRequirement,
			)
		} else {
			fmt.Fprintf(f, "GlobalV2{%d %d %d %d %s %s %d %d %d %d %d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.iglobal,
				g.iprep,
				g.ivoter,
				g.icps,
				g.irelay,
				g.electedPRepCount,
				g.bondRequirement,
			)
		}
	}
}

func (g *GlobalV2) Equal(impl icobject.Impl) bool {
	if g2, ok := impl.(*GlobalV2); ok {
		return g.iissVersion == g2.iissVersion &&
			g.startHeight == g2.startHeight &&
			g.offsetLimit == g2.offsetLimit &&
			g.revision == g2.revision &&
			g.iglobal.Cmp(g2.iglobal) == 0 &&
			g.iprep.Cmp(g2.iprep) == 0 &&
			g.ivoter.Cmp(g2.ivoter) == 0 &&
			g.icps.Cmp(g2.icps) == 0 &&
			g.irelay.Cmp(g2.irelay) == 0 &&
			g.electedPRepCount == g2.electedPRepCount &&
			g.bondRequirement == g2.bondRequirement
	} else {
		return false
	}
}

func (g *GlobalV2) GetV1() *GlobalV1 {
	return nil
}

func (g *GlobalV2) GetV2() *GlobalV2 {
	return g
}

func (g *GlobalV2) GetV3() *GlobalV3 {
	return nil
}

func newGlobalV2() *GlobalV2 {
	return &GlobalV2{
		iglobal: new(big.Int),
		iprep:   new(big.Int),
		ivoter:  new(big.Int),
		icps:    new(big.Int),
		irelay:  new(big.Int),
	}
}

func NewGlobalV2(
	iissVersion int,
	startHeight int64,
	offsetLimit int,
	revision int,
	iglobal *big.Int,
	iprep *big.Int,
	ivoter *big.Int,
	icps *big.Int,
	irelay *big.Int,
	electedPRepCount int,
	bondRequirement int,
) *GlobalV2 {
	return &GlobalV2{
		globalBase: globalBase{
			iissVersion:      iissVersion,
			startHeight:      startHeight,
			offsetLimit:      offsetLimit,
			revision:         revision,
			electedPRepCount: electedPRepCount,
			bondRequirement:  bondRequirement,
		},
		iglobal: iglobal,
		iprep:   iprep,
		ivoter:  ivoter,
		icps:    icps,
		irelay:  irelay,
	}
}

type GlobalV3 struct {
	globalBase
	rFund   *rewardFund
	minBond *big.Int
}

func (g *GlobalV3) Version() int {
	return GlobalVersion3
}

func (g *GlobalV3) GetRewardFundByKey(key rFundKey) *big.Int {
	return g.rFund.Get(key)
}

func (g *GlobalV3) GetIGlobal() *big.Int {
	return g.rFund.Get(keyIglobal)
}

func (g *GlobalV3) GetIprep() *big.Int {
	return g.rFund.Get(keyIprep)
}

func (g *GlobalV3) GetIcps() *big.Int {
	return g.rFund.Get(keyIcps)
}

func (g *GlobalV3) GetIrelay() *big.Int {
	return g.rFund.Get(keyIrelay)
}

func (g *GlobalV3) GetIWage() *big.Int {
	return g.rFund.Get(keyIwage)
}

func (g *GlobalV3) GetMinBond() *big.Int {
	return g.minBond
}

func (g *GlobalV3) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.iissVersion,
		&g.startHeight,
		&g.offsetLimit,
		&g.revision,
		&g.electedPRepCount,
		&g.bondRequirement,
		&g.rFund,
		&g.minBond,
	)
	return err
}

func (g *GlobalV3) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.revision,
		g.electedPRepCount,
		g.bondRequirement,
		g.rFund,
		g.minBond,
	)
}

func (g *GlobalV3) String() string {
	return fmt.Sprintf("revision=%d iissVersion=%d startHeight=%d offsetLimit=%d electedPRepCount=%d "+
		"bondRequirement=%d rFund=%s minBond=%d",
		g.revision,
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.electedPRepCount,
		g.bondRequirement,
		g.rFund,
		g.minBond,
	)
}

func (g *GlobalV3) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "GlobalV3{revision=%d iissVersion=%d startHeight=%d offsetLimit=%d "+
				"electedPRepCount=%d bondRequirement=%d rFund=%+v minBond=%d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.electedPRepCount,
				g.bondRequirement,
				g.rFund,
				g.minBond,
			)
		} else {
			fmt.Fprintf(f, "GlobalV3{%d %d %d %d %d %d %v %d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.electedPRepCount,
				g.bondRequirement,
				g.rFund,
				g.minBond,
			)
		}
	}
}

func (g *GlobalV3) Equal(impl icobject.Impl) bool {
	if g2, ok := impl.(*GlobalV3); ok {
		return g.iissVersion == g2.iissVersion &&
			g.startHeight == g2.startHeight &&
			g.offsetLimit == g2.offsetLimit &&
			g.revision == g2.revision &&
			g.electedPRepCount == g2.electedPRepCount &&
			g.bondRequirement == g2.bondRequirement &&
			g.rFund.Equal(g2.rFund) &&
			g.minBond.Cmp(g2.minBond) == 0
	} else {
		return false
	}
}

func (g *GlobalV3) GetV1() *GlobalV1 {
	return nil
}

func (g *GlobalV3) GetV2() *GlobalV2 {
	return nil
}

func (g *GlobalV3) GetV3() *GlobalV3 {
	return g
}

func newGlobalV3() *GlobalV3 {
	g := &GlobalV3{
		rFund: newRewardFund(),
	}
	g.rFund.Set(keyIglobal, new(big.Int))
	g.rFund.Set(keyIprep, new(big.Int))
	g.rFund.Set(keyIwage, new(big.Int))
	g.rFund.Set(keyIcps, new(big.Int))
	g.rFund.Set(keyIrelay, new(big.Int))
	return g
}

func NewGlobalV3(
	iissVersion int,
	startHeight int64,
	offsetLimit int,
	revision int,
	electedPRepCount int,
	bondRequirement int,
	iglobal *big.Int,
	iprep *big.Int,
	iwage *big.Int,
	icps *big.Int,
	irelay *big.Int,
	minBond *big.Int,
) *GlobalV3 {
	g := &GlobalV3{
		globalBase: globalBase{
			iissVersion:      iissVersion,
			startHeight:      startHeight,
			offsetLimit:      offsetLimit,
			revision:         revision,
			electedPRepCount: electedPRepCount,
			bondRequirement:  bondRequirement,
		},
		rFund:   newRewardFund(),
		minBond: minBond,
	}
	g.rFund.Set(keyIglobal, iglobal)
	g.rFund.Set(keyIprep, iprep)
	g.rFund.Set(keyIwage, iwage)
	g.rFund.Set(keyIcps, icps)
	g.rFund.Set(keyIrelay, irelay)
	return g
}

type rFundKey string

const (
	keyIglobal rFundKey = "iglobal"
	keyIprep            = "iprep"
	keyIwage            = "iwage"
	keyIcps             = "icps"
	keyIrelay           = "irelay"
)

var rFundKeys = []rFundKey{keyIglobal, keyIprep, keyIwage, keyIcps, keyIrelay}

type rElem struct {
	key   rFundKey
	value *big.Int
}

func (r *rElem) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(r.key, r.value)
}

func (r *rElem) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(&r.key, &r.value)
}

type rewardFund struct {
	value map[rFundKey]*big.Int
}

func (r *rewardFund) Get(key rFundKey) *big.Int {
	if v, ok := r.value[key]; ok {
		return v
	} else {
		return nil
	}
}

func (r *rewardFund) Set(key rFundKey, value *big.Int) {
	r.value[key] = value
}

func (r *rewardFund) toSlice() []*rElem {
	elem := make([]*rElem, 0)
	for k, v := range r.value {
		elem = append(elem, &rElem{k, v})
	}
	sort.Slice(elem, func(i, j int) bool {
		return elem[i].key < elem[j].key
	})
	return elem
}

func (r *rewardFund) Equal(r2 *rewardFund) bool {
	if len(r.value) != len(r2.value) {
		return false
	}
	for _, k := range rFundKeys {
		v1, ok1 := r.value[k]
		v2, ok2 := r2.value[k]
		if ok1 != ok2 {
			return false
		}
		if ok1 == true {
			if v1.Cmp(v2) != 0 {
				return false
			}
		}
	}
	return true
}

func (r *rewardFund) RLPEncodeSelf(encoder codec.Encoder) error {
	elem := r.toSlice()
	return encoder.Encode(elem)
}

func (r *rewardFund) RLPDecodeSelf(d codec.Decoder) error {
	elem := make([]*rElem, 0)
	if err := d.Decode(&elem); err != nil {
		return err
	}
	r.value = make(map[rFundKey]*big.Int)
	for _, e := range elem {
		r.value[e.key] = e.value
	}
	return nil
}

func (r *rewardFund) string(withName bool) string {
	ret := ""
	for _, k := range rFundKeys {
		if v, ok := r.value[k]; ok {
			if len(ret) == 0 {
				if withName {
					ret = fmt.Sprintf("%s=%d", k, v)
				} else {
					ret = fmt.Sprintf("%d", v)
				}
			} else {
				if withName {
					ret = fmt.Sprintf("%s %s=%d", ret, k, v)
				} else {
					ret = fmt.Sprintf("%s %d", ret, v)
				}
			}
		}
	}
	if withName {
		ret = fmt.Sprintf("rewardFund{%s}", ret)
	} else {
		ret = fmt.Sprintf("{%s}", ret)
	}
	return ret
}

func (r *rewardFund) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%s", r.string(true))
		} else {
			fmt.Fprintf(f, "%s", r.string(false))
		}
	case 's':
		fmt.Fprintf(f, "%s", r.string(true))
	}
}

func newRewardFund() *rewardFund {
	return &rewardFund{
		value: make(map[rFundKey]*big.Int),
	}
}
