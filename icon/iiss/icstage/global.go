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

type GlobalV1 struct {
	icobject.NoDatabase
	iissVersion      int
	startHeight      int64
	offsetLimit      int
	revision         int
	irep             *big.Int
	rrep             *big.Int
	mainPRepCount    int
	electedPRepCount int
}

func (g *GlobalV1) Version() int {
	return GlobalVersion1
}

func (g *GlobalV1) GetIISSVersion() int {
	return g.iissVersion
}

func (g *GlobalV1) GetStartHeight() int64 {
	return g.startHeight
}

func (g *GlobalV1) GetOffsetLimit() int {
	return g.offsetLimit
}

func (g *GlobalV1) GetTermPeriod() int {
	return g.offsetLimit + 1
}

func (g *GlobalV1) GetRevision() int {
	return g.revision
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

func (g *GlobalV1) GetElectedPRepCount() int {
	return g.electedPRepCount
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
		iissVersion:      iissVersion,
		startHeight:      startHeight,
		offsetLimit:      offsetLimit,
		revision:         revision,
		irep:             irep,
		rrep:             rrep,
		mainPRepCount:    mainPRepCount,
		electedPRepCount: electedPRepCount,
	}
}

type GlobalV2 struct {
	icobject.NoDatabase
	iissVersion      int
	startHeight      int64
	offsetLimit      int
	revision         int
	iglobal          *big.Int
	iprep            *big.Int
	ivoter           *big.Int
	icps             *big.Int
	irelay           *big.Int
	electedPRepCount int
	bondRequirement  int
}

func (g *GlobalV2) Version() int {
	return GlobalVersion2
}

func (g *GlobalV2) GetIISSVersion() int {
	return g.iissVersion
}

func (g *GlobalV2) GetStartHeight() int64 {
	return g.startHeight
}

func (g *GlobalV2) GetOffsetLimit() int {
	return g.offsetLimit
}

func (g *GlobalV2) GetTermPeriod() int {
	return g.offsetLimit + 1
}

func (g *GlobalV2) GetRevision() int {
	return g.revision
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

func (g *GlobalV2) GetElectedPRepCount() int {
	return g.electedPRepCount
}

func (g *GlobalV2) GetBondRequirement() int {
	return g.bondRequirement
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
		iissVersion:      iissVersion,
		startHeight:      startHeight,
		offsetLimit:      offsetLimit,
		revision:         revision,
		iglobal:          iglobal,
		iprep:            iprep,
		ivoter:           ivoter,
		icps:             icps,
		irelay:           irelay,
		electedPRepCount: electedPRepCount,
		bondRequirement:  bondRequirement,
	}
}

type GlobalV3 struct {
	GlobalV2
	minBond *big.Int
}

func (g *GlobalV3) Version() int {
	return GlobalVersion3
}

func (g *GlobalV3) GetIVoter() *big.Int {
	return new(big.Int)
}

func (g *GlobalV3) GetIWage() *big.Int {
	// use ivoter as an iwage
	return g.ivoter
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
		&g.iglobal,
		&g.iprep,
		&g.ivoter,
		&g.icps,
		&g.irelay,
		&g.electedPRepCount,
		&g.bondRequirement,
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
		g.iglobal,
		g.iprep,
		g.ivoter,
		g.icps,
		g.irelay,
		g.electedPRepCount,
		g.bondRequirement,
		g.minBond,
	)
}

func (g *GlobalV3) String() string {
	return fmt.Sprintf("revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%s "+
		"iprep=%s iwage=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d minBond=%d",
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
		g.minBond,
	)
}

func (g *GlobalV3) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "GlobalV3{revision=%d iissVersion=%d startHeight=%d offsetLimit=%d iglobal=%s "+
				"iprep=%s iwage=%d icps=%d irelay=%d electedPRepCount=%d bondRequirement=%d minBond=%d}",
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
				g.minBond,
			)
		} else {
			fmt.Fprintf(f, "GlobalV3{%d %d %d %d %s %s %d %d %d %d %d %d}",
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
			g.iglobal.Cmp(g2.iglobal) == 0 &&
			g.iprep.Cmp(g2.iprep) == 0 &&
			g.ivoter.Cmp(g2.ivoter) == 0 &&
			g.icps.Cmp(g2.icps) == 0 &&
			g.irelay.Cmp(g2.irelay) == 0 &&
			g.electedPRepCount == g2.electedPRepCount &&
			g.bondRequirement == g2.bondRequirement &&
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
		GlobalV2: GlobalV2{
			iglobal: new(big.Int),
			iprep:   new(big.Int),
			ivoter:  new(big.Int),
			icps:    new(big.Int),
			irelay:  new(big.Int),
		},
	}
}

func NewGlobalV3(
	iissVersion int,
	startHeight int64,
	offsetLimit int,
	revision int,
	iglobal *big.Int,
	iprep *big.Int,
	iwage *big.Int,
	icps *big.Int,
	irelay *big.Int,
	electedPRepCount int,
	bondRequirement int,
	minBond *big.Int,
) *GlobalV3 {
	return &GlobalV3{
		GlobalV2: GlobalV2{
			iissVersion:      iissVersion,
			startHeight:      startHeight,
			offsetLimit:      offsetLimit,
			revision:         revision,
			iglobal:          iglobal,
			iprep:            iprep,
			ivoter:           iwage,
			icps:             icps,
			irelay:           irelay,
			electedPRepCount: electedPRepCount,
			bondRequirement:  bondRequirement,
		},
		minBond: minBond,
	}
}
