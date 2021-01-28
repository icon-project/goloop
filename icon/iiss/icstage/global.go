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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"math/big"
)

const (
	GlobalVersion1 int = iota
	GlobalVersion2
)

type GlobalImpl interface {
	Version() int
	GetIISSVersion() int
	GetOffsetLimit() int
	GetElectedPRepCount() int
	GetBondRequirement() int
	RLPDecodeFields(decoder codec.Decoder) error
	RLPEncodeFields(encoder codec.Encoder) error
	Equal(impl GlobalImpl) bool
}

type Global struct {
	icobject.NoDatabase
	GlobalImpl
}

func (g *Global) Version() int {
	return g.GlobalImpl.Version()
}

func (g *Global) GetV1() *GlobalV1 {
	global, ok := g.GlobalImpl.(*GlobalV1)
	if ok {
		return global
	} else {
		return nil
	}
}

func (g *Global) GetV2() *GlobalV2 {
	global, ok := g.GlobalImpl.(*GlobalV2)
	if ok {
		return global
	} else {
		return nil
	}
}

func newGlobal(tag icobject.Tag) *Global {
	g := new(Global)
	switch tag.Version() {
	case GlobalVersion1:
		g.GlobalImpl = newGlobalV1()
	case GlobalVersion2:
		g.GlobalImpl = newGlobalV2()
	}
	return g
}

func (g *Global) RLPDecodeFields(decoder codec.Decoder) error {
	d, err := decoder.DecodeList()
	if err != nil {
		return err
	}
	var version int
	if err = d.Decode(&version); err != nil {
		return err
	}
	switch version {
	case GlobalVersion1:
		g.GlobalImpl = new(GlobalV1)
	case GlobalVersion2:
		g.GlobalImpl = new(GlobalV2)
	default:
		return errors.CriticalFormatError.Errorf(
			"InvalidGlobalVersion(version=%d)", version)
	}
	return g.GlobalImpl.RLPDecodeFields(d)
}

func (g *Global) RLPEncodeFields(encoder codec.Encoder) error {
	e, err := encoder.EncodeList()
	if err != nil {
		return err
	}
	if err := e.Encode(g.GlobalImpl.Version()); err != nil {
		return err
	}
	return g.GlobalImpl.RLPEncodeFields(e)
}

func (g *Global) Equal(o icobject.Impl) bool {
	if g2, ok := o.(*Global); ok {
		if g.Version() != g2.Version() {
			return false
		}
		return g.Equal(g2)
	} else {
		return false
	}
}

type GlobalV1 struct {
	IISSVersion      int
	OffsetLimit      int
	Irep             *big.Int
	Rrep             *big.Int
	MainPRepCount    int
	ElectedPRepCount int
}

func (g *GlobalV1) Version() int {
	return GlobalVersion1
}

func (g *GlobalV1) GetIISSVersion() int {
	return g.IISSVersion
}

func (g *GlobalV1) GetOffsetLimit() int {
	return g.OffsetLimit
}

func (g *GlobalV1) GetElectedPRepCount() int {
	return g.ElectedPRepCount
}

func (g *GlobalV1) GetBondRequirement() int {
	return 0
}

func (g *GlobalV1) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.IISSVersion,
		&g.OffsetLimit,
		&g.Irep,
		&g.Rrep,
		&g.MainPRepCount,
		&g.ElectedPRepCount,
	)
	return err
}

func (g *GlobalV1) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.IISSVersion,
		g.OffsetLimit,
		g.Irep,
		g.Rrep,
		g.MainPRepCount,
		g.ElectedPRepCount,
	)
}

func (g *GlobalV1) Equal(impl GlobalImpl) bool {
	if g2, ok := impl.(*GlobalV1); ok {
		return g.IISSVersion == g2.IISSVersion &&
			g.OffsetLimit == g2.OffsetLimit &&
			g.Irep.Cmp(g2.Irep) == 0 &&
			g.Rrep.Cmp(g2.Rrep) == 0 &&
			g.MainPRepCount == g2.MainPRepCount &&
			g.ElectedPRepCount == g2.ElectedPRepCount
	} else {
		return false
	}
}

func (g *GlobalV1) Clear() {
	g.IISSVersion = 0
	g.OffsetLimit = 0
	g.Irep.SetInt64(0)
	g.Rrep.SetInt64(0)
	g.MainPRepCount = 0
	g.ElectedPRepCount = 0
}

func (g *GlobalV1) IsEmpty() bool {
	return g.OffsetLimit == 0 &&
		g.Irep.Sign() == 0 &&
		g.Rrep.Sign() == 0 &&
		g.MainPRepCount == 0 &&
		g.ElectedPRepCount == 0
}

func newGlobalV1() *GlobalV1 {
	return &GlobalV1{
		Irep: new(big.Int),
		Rrep: new(big.Int),
	}
}

type GlobalV2 struct {
	IISSVersion      int
	OffsetLimit      int
	Iglobal          *big.Int
	Iprep            *big.Int
	Ivoter           *big.Int
	ElectedPRepCount int
	BondRequirement  int
}

func (g *GlobalV2) Version() int {
	return GlobalVersion2
}

func (g *GlobalV2) GetIISSVersion() int {
	return g.IISSVersion
}

func (g *GlobalV2) GetOffsetLimit() int {
	return g.OffsetLimit
}

func (g *GlobalV2) GetElectedPRepCount() int {
	return g.ElectedPRepCount
}

func (g *GlobalV2) GetBondRequirement() int {
	return g.BondRequirement
}

func (g *GlobalV2) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.IISSVersion,
		&g.OffsetLimit,
		&g.Iglobal,
		&g.Iprep,
		&g.Ivoter,
		&g.ElectedPRepCount,
		&g.BondRequirement,
	)
	return err
}

func (g *GlobalV2) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.IISSVersion,
		g.OffsetLimit,
		g.Iglobal,
		g.Iprep,
		g.Ivoter,
		g.ElectedPRepCount,
		g.BondRequirement,
	)
}

func (g *GlobalV2) Equal(impl GlobalImpl) bool {
	if g2, ok := impl.(*GlobalV2); ok {
		return g.IISSVersion == g2.IISSVersion &&
			g.OffsetLimit == g2.OffsetLimit &&
			g.Iglobal.Cmp(g2.Iglobal) == 0 &&
			g.Iprep.Cmp(g2.Iprep) == 0 &&
			g.Ivoter.Cmp(g2.Ivoter) == 0 &&
			g.ElectedPRepCount == g2.ElectedPRepCount &&
			g.BondRequirement == g2.BondRequirement
	} else {
		return false
	}
}

func (g *GlobalV2) Clear() {
	g.IISSVersion = 0
	g.OffsetLimit = 0
	g.Iglobal.SetInt64(0)
	g.Iprep.SetInt64(0)
	g.Ivoter.SetInt64(0)
	g.ElectedPRepCount = 0
	g.BondRequirement = 0
}

func (g *GlobalV2) IsEmpty() bool {
	return g.OffsetLimit == 0 &&
		g.Iglobal.Sign() == 0 &&
		g.Iprep.Sign() == 0 &&
		g.Ivoter.Sign() == 0 &&
		g.ElectedPRepCount == 0 &&
		g.BondRequirement == 0
}

func newGlobalV2() *GlobalV2 {
	return &GlobalV2{
		Iglobal: new(big.Int),
		Iprep:   new(big.Int),
		Ivoter:  new(big.Int),
	}
}
