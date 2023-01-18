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
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type IScore struct {
	icobject.NoDatabase
	value *big.Int
}

func (is *IScore) Version() int {
	return 0
}

func (is *IScore) Value() *big.Int {
	return is.value
}

func (is *IScore) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&is.value)
}

func (is *IScore) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(is.value)
}

func (is *IScore) Equal(o icobject.Impl) bool {
	if is2, ok := o.(*IScore); ok {
		return is.value.Cmp(is2.value) == 0
	} else {
		return false
	}
}

func (is *IScore) Clear() {
	is.value = new(big.Int)
}

func (is *IScore) IsEmpty() bool {
	return is.value == nil || is.value.Sign() == 0
}

func (is *IScore) Added(amount *big.Int) *IScore {
	n := new(IScore)
	if is == nil {
		n.value = amount
	} else {
		n.value = new(big.Int).Add(is.value, amount)
	}
	return n
}

func (is *IScore) Subtracted(amount *big.Int) *IScore {
	n := new(IScore)
	if is == nil {
		n.value = new(big.Int).Neg(amount)
	} else {
		n.value = new(big.Int).Sub(is.value, amount)
	}
	return n
}

func (is *IScore) Clone() *IScore {
	if is == nil {
		return nil
	}
	return NewIScore(is.value)
}

func (is *IScore) Format(f fmt.State, c rune) {
	if is == nil {
		fmt.Fprintf(f, "nil")
		return
	}

	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "IScore{value=%d}", is.value)
		} else {
			fmt.Fprintf(f, "IScore{%d}", is.value)
		}
	case 's':
		fmt.Fprintf(f, "value=%d", is.value)
	}
}

func newIScore(_ icobject.Tag) *IScore {
	return new(IScore)
}

func NewIScore(value *big.Int) *IScore {
	return &IScore{
		value: value,
	}
}
