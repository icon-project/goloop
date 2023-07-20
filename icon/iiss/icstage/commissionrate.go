/*
 * Copyright 2023 ICON Foundation
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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type CommissionRate struct {
	icobject.NoDatabase
	value icmodule.Rate
}

func (cr *CommissionRate) Version() int {
	return 0
}

func (cr *CommissionRate) Value() icmodule.Rate {
	return cr.value
}

func (cr *CommissionRate) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&cr.value)
}

func (cr *CommissionRate) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(cr.value)
}

func (cr *CommissionRate) Equal(o icobject.Impl) bool {
	if ee2, ok := o.(*CommissionRate); ok {
		return cr.value == ee2.value
	} else {
		return false
	}
}

func (cr *CommissionRate) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "CommissionRate{value=%s}", cr.value)
		} else {
			fmt.Fprintf(f, "CommissionRate{%s}", cr.value)
		}
	}
}

func newCommissionRate(_ icobject.Tag) *CommissionRate {
	return new(CommissionRate)
}

func NewCommissionRate(value icmodule.Rate) *CommissionRate {
	return &CommissionRate{value: value}
}
