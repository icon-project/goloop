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
)

type Global interface {
	icobject.Impl
	GetV1() *GlobalV1
	GetV2() *GlobalV2
	GetIISSVersion() int
	GetStartHeight() int64
	GetOffsetLimit() int
	GetTermPeriod() int
	GetRevision() int
	GetElectedPRepCount() int
	GetBondRequirement() int
	String() string
}

func NewGlobal(version int) (Global, error) {
	switch version {
	case GlobalVersion1:
		return newGlobalV1(), nil
	case GlobalVersion2:
		return newGlobalV2(), nil
	default:
		return nil, errors.CriticalFormatError.Errorf("InvalidGlobalVersion(%d)", version)
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
	return fmt.Sprintf("Revision: %d, IISSVersion: %d, StartHeight: %d, OffsetLimit: %d, Irep: %s, Rrep: %s, "+
		"MainPRepCount: %d, ElectedPRepCount: %d",
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
		g.electedPRepCount,
		g.bondRequirement,
	)
}

func (g *GlobalV2) String() string {
	return fmt.Sprintf("Revision: %d, IISSVersion: %d, StartHeight: %d, OffsetLimit: %d, Iglobal: %s, "+
		"Iprep: %s, Ivoter: %d, ElectedPRepCount: %d, BondRequirement: %d",
		g.revision,
		g.iissVersion,
		g.startHeight,
		g.offsetLimit,
		g.iglobal,
		g.iprep,
		g.ivoter,
		g.electedPRepCount,
		g.bondRequirement,
	)
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

func newGlobalV2() *GlobalV2 {
	return &GlobalV2{
		iglobal: new(big.Int),
		iprep:   new(big.Int),
		ivoter:  new(big.Int),
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
		electedPRepCount: electedPRepCount,
		bondRequirement:  bondRequirement,
	}
}
