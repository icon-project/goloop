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
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Global struct {
	icobject.NoDatabase
	StartBlockHeight int64
	OffsetLimit      int
}

func (g *Global) Version() int {
	return 0
}

func (g *Global) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&g.StartBlockHeight,
		&g.OffsetLimit,
	)
	return err
}

func (g *Global) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		g.StartBlockHeight,
		g.OffsetLimit,
	)
}

func (g *Global) Equal(o icobject.Impl) bool {
	if g2, ok := o.(*Global); ok {
		return  g.StartBlockHeight == g2.StartBlockHeight && g.OffsetLimit == g2.OffsetLimit
	} else {
		return false
	}
}

func (g *Global) Clear() {
	g.StartBlockHeight = 0
	g.OffsetLimit = 0
}

func (g *Global) IsEmpty() bool {
	return g.StartBlockHeight == 0 && g.OffsetLimit == 0
}

func newGlobal(tag icobject.Tag) *Global {
	return new(Global)
}
