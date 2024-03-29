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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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
	GetBondRequirement() icmodule.Rate
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
	bondRequirement  icmodule.Rate
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

func (g *globalBase) GetBondRequirement() icmodule.Rate {
	return g.bondRequirement
}

// GlobalV1 global struct for icstate.IISSVersion2
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

// GlobalV2 global struct for icstate.IISSVersion3
type GlobalV2 struct {
	globalBase
	iglobal *big.Int
	iprep   icmodule.Rate
	ivoter  icmodule.Rate
	icps    icmodule.Rate
	irelay  icmodule.Rate
}

func (g *GlobalV2) Version() int {
	return GlobalVersion2
}

func (g *GlobalV2) GetIGlobal() *big.Int {
	return g.iglobal
}

func (g *GlobalV2) GetIPRep() icmodule.Rate {
	return g.iprep
}

func (g *GlobalV2) GetIVoter() icmodule.Rate {
	return g.ivoter
}

func (g *GlobalV2) GetICps() icmodule.Rate {
	return g.icps
}

func (g *GlobalV2) GetIRelay() icmodule.Rate {
	return g.irelay
}

func (g *GlobalV2) RLPDecodeFields(decoder codec.Decoder) error {
	var iprep, ivoter, icps, irelay, br int64 // unit: percent
	_, err := decoder.DecodeMulti(
		&g.iissVersion,
		&g.startHeight,
		&g.offsetLimit,
		&g.revision,
		&g.iglobal,
		&iprep,
		&ivoter,
		&icps,
		&irelay,
		&g.electedPRepCount,
		&br,
	)
	if err == nil {
		g.iprep = icmodule.ToRate(iprep)
		g.ivoter = icmodule.ToRate(ivoter)
		g.icps = icmodule.ToRate(icps)
		g.irelay = icmodule.ToRate(irelay)
		g.bondRequirement = icmodule.ToRate(br)
	}
	return err
}

func (g *GlobalV2) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.revision,
		g.iglobal,
		g.iprep.Percent(),
		g.ivoter.Percent(),
		g.icps.Percent(),
		g.irelay.Percent(),
		g.electedPRepCount,
		g.bondRequirement.Percent(),
	)
}

func (g *GlobalV2) String() string {
	return fmt.Sprintf("revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%d "+
		"iprep=%d ivoter=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d",
		g.revision,
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.iglobal,
		g.iprep.Percent(),
		g.ivoter.Percent(),
		g.icps.Percent(),
		g.irelay.Percent(),
		g.electedPRepCount,
		g.bondRequirement.Percent(),
	)
}

func (g *GlobalV2) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "GlobalV2{revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%d "+
				"iprep=%d ivoter=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.iglobal,
				g.iprep.Percent(),
				g.ivoter.Percent(),
				g.icps.Percent(),
				g.irelay.Percent(),
				g.electedPRepCount,
				g.bondRequirement.Percent(),
			)
		} else {
			fmt.Fprintf(f, "GlobalV2{%d %d %d %d %d %d %d %d %d %d %d}",
				g.revision,
				g.iissVersion,
				g.startHeight,
				g.offsetLimit,
				g.iglobal,
				g.iprep.Percent(),
				g.ivoter.Percent(),
				g.icps.Percent(),
				g.irelay.Percent(),
				g.electedPRepCount,
				g.bondRequirement.Percent(),
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
			g.iprep == g2.iprep &&
			g.ivoter == g2.ivoter &&
			g.icps == g2.icps &&
			g.irelay == g2.irelay &&
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
	}
}

func NewGlobalV2(
	iissVersion int,
	startHeight int64,
	offsetLimit int,
	revision int,
	iglobal *big.Int,
	iprep,
	ivoter,
	icps,
	irelay icmodule.Rate,
	electedPRepCount int,
	bondRequirement icmodule.Rate,
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

// GlobalV3 global struct for icstate.IISSVersion4
type GlobalV3 struct {
	globalBase
	rFund   *icstate.RewardFund
	minBond *big.Int
}

func (g *GlobalV3) Version() int {
	return GlobalVersion3
}

func (g *GlobalV3) GetRewardFundRateByKey(key icstate.RFundKey) icmodule.Rate {
	return g.rFund.GetAllocationByKey(key)
}

func (g *GlobalV3) GetIGlobal() *big.Int {
	return g.rFund.IGlobal()
}

func (g *GlobalV3) GetIPRep() icmodule.Rate {
	return g.rFund.GetAllocationByKey(icstate.KeyIprep)
}

func (g *GlobalV3) GetICps() icmodule.Rate {
	return g.rFund.GetAllocationByKey(icstate.KeyIcps)
}

func (g *GlobalV3) GetIRelay() icmodule.Rate {
	return g.rFund.GetAllocationByKey(icstate.KeyIrelay)
}

func (g *GlobalV3) GetIWage() icmodule.Rate {
	return g.rFund.GetAllocationByKey(icstate.KeyIwage)
}

func (g *GlobalV3) GetRewardFundAmountByKey(key icstate.RFundKey) *big.Int {
	return g.rFund.GetAmount(key)
}

func (g *GlobalV3) MinBond() *big.Int {
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
	return &GlobalV3{
		rFund: icstate.NewRewardFund(icstate.RFVersion2),
	}
}

func NewGlobalV3(
	iissVersion int, startHeight int64, revision int, offsetLimit, electedPRepCount int,
	bondRequirement icmodule.Rate, rFund *icstate.RewardFund, minBond *big.Int,
) *GlobalV3 {
	if minBond == nil {
		minBond = icmodule.BigIntZero
	}
	g := &GlobalV3{
		globalBase: globalBase{
			iissVersion:      iissVersion,
			startHeight:      startHeight,
			offsetLimit:      offsetLimit,
			revision:         revision,
			electedPRepCount: electedPRepCount,
			bondRequirement:  bondRequirement,
		},
		rFund:   rFund,
		minBond: minBond,
	}
	return g
}
