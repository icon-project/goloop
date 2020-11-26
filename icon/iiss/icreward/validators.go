/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icreward

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Validators struct {
	icobject.NoDatabase
	Addresses []*common.Address
}

func (v *Validators) Version() int {
	return 0
}

func (v *Validators) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&v.Addresses)
	return err
}

func (v *Validators) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(v.Addresses)
}

func (v *Validators) Equal(o icobject.Impl) bool {
	if v2, ok := o.(*Validators); ok {
		if len(v.Addresses) != len(v2.Addresses) {
			return false
		}
		for i, a := range v.Addresses {
			if a.Equal(v2.Addresses[i]) == false {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

func (v *Validators) Clear() {
	v.Addresses = nil
}

func (v *Validators) IsEmpty() bool {
	return v.Addresses == nil || len(v.Addresses) == 0
}

func (v *Validators) Add(addr *common.Address) {
	if v.Addresses == nil {
		v.Addresses = make([]*common.Address, 0)
	}
	v.Addresses = append(v.Addresses, addr)
}

func newValidators(tag icobject.Tag) *Validators {
	return NewValidators()
}

func NewValidators() *Validators {
	return new(Validators)
}

