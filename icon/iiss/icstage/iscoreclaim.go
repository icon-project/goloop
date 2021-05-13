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
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type IScoreClaim struct {
	icobject.NoDatabase
	value *big.Int
}

func (ic *IScoreClaim) Version() int {
	return 0
}

func (ic *IScoreClaim) Value() *big.Int {
	return ic.value
}

func (ic *IScoreClaim) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&ic.value)
}

func (ic *IScoreClaim) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(ic.value)
}

func (ic *IScoreClaim) Equal(impl icobject.Impl) bool {
	if ic2, ok := impl.(*IScoreClaim); ok {
		return ic.value.Cmp(ic2.value) == 0
	} else {
		return false
	}
}

func (ic *IScoreClaim) Added(amount *big.Int) *IScoreClaim {
	n := new(IScoreClaim)
	if ic == nil {
		n.value = amount
	} else {
		n.value = new(big.Int).Add(ic.value, amount)
	}
	return n
}

func (ic *IScoreClaim) String() string {
	return fmt.Sprintf("value=%d", ic.value)
}

func (ic *IScoreClaim) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "IScoreClaim{value=%d}", ic.value)
		} else {
			fmt.Fprintf(f, "IScoreClaim{%d}", ic.value)
		}
	case 's':
		fmt.Fprint(f, ic.String())
	}
}

func newIScoreClaim(_ icobject.Tag) *IScoreClaim {
	return new(IScoreClaim)
}

func NewIScoreClaim(value *big.Int) *IScoreClaim {
	return &IScoreClaim{
		value: value,
	}
}
