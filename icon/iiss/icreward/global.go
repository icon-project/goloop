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

package icreward

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Global struct {
	icobject.NoDatabase
	Irep          *big.Int
	Rrep          *big.Int
	MainPRepCount *big.Int
	PRepCount     *big.Int
}

func (g *Global) Version() int {
	return 0
}

func (g *Global) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&g.Irep, &g.Rrep, &g.MainPRepCount, &g.PRepCount)
	return err
}

func (g *Global) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(g.Irep, g.Rrep, g.MainPRepCount, g.PRepCount)
}

func (g *Global) Equal(o icobject.Impl) bool {
	if g2, ok := o.(*Global); ok {
		return g.Irep.Cmp(g2.Irep) == 0 &&
			g2.Rrep.Cmp(g2.Rrep) == 0 &&
			g2.MainPRepCount.Cmp(g2.MainPRepCount) == 0 &&
			g2.PRepCount.Cmp(g2.PRepCount) == 0
	} else {
		return false
	}
}

func (g *Global) Clear() {
	g.Irep = new(big.Int)
	g.Rrep = new(big.Int)
	g.MainPRepCount = new(big.Int)
	g.PRepCount = new(big.Int)
}

func (g *Global) IsEmpty() bool {
	return (g.Irep == nil || g.Irep.Sign() == 0) &&
		(g.Rrep == nil || g.Rrep.Sign() == 0)
}

func newGlobal(tag icobject.Tag) *Global {
	return NewGlobal()
}

func NewGlobal() *Global {
	return &Global{
		Irep:          new(big.Int),
		Rrep:          new(big.Int),
		MainPRepCount: new(big.Int),
		PRepCount:     new(big.Int),
	}
}
