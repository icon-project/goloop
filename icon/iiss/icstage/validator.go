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

package icstage

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Validator struct {
	icobject.NoDatabase
	Address *common.Address
}

func (v *Validator) Version() int {
	return 0
}

func (v *Validator) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&v.Address)
}

func (v *Validator) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(v.Address)
}

func (v *Validator) Equal(o icobject.Impl) bool {
	if v2, ok := o.(*Validator); ok {
		return v.Address.Equal(v2.Address)
	} else {
		return false
	}
}

func (v *Validator) Clear() {
	v.Address = nil
}

func (v *Validator) IsEmpty() bool {
	return v.Address == nil
}

func newValidator(tag icobject.Tag) *Validator {
	return &Validator{}
}

