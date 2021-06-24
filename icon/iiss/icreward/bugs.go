/*
 * Copyright 2021 ICON Foundation
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

type BugDisabledPRep struct {
	icobject.NoDatabase
	amount *big.Int
}

func (is *BugDisabledPRep) Version() int {
	return 0
}

func (is *BugDisabledPRep) Value() *big.Int {
	return is.amount
}

func (is *BugDisabledPRep) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&is.amount)
}

func (is *BugDisabledPRep) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(is.amount)
}

func (is *BugDisabledPRep) Equal(o icobject.Impl) bool {
	if is2, ok := o.(*BugDisabledPRep); ok {
		return is.amount.Cmp(is2.amount) == 0
	} else {
		return false
	}
}

func (is *BugDisabledPRep) Format(f fmt.State, c rune) {
	if is == nil {
		fmt.Fprintf(f, "nil")
		return
	}

	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "BugDisabledPRep{amount=%d}", is.amount)
		} else {
			fmt.Fprintf(f, "BugDisabledPRep{%d}", is.amount)
		}
	case 's':
		fmt.Fprintf(f, "amount=%d", is.amount)
	}
}

func newBugDisabledPRep(_ icobject.Tag) *BugDisabledPRep {
	return new(BugDisabledPRep)
}

func NewBugDisabledPRep(value *big.Int) *BugDisabledPRep {
	return &BugDisabledPRep{
		amount: value,
	}
}
