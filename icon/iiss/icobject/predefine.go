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

package icobject

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/trie"
	"math/big"
)

const (
	TypeBigInt = TypeReserved
)

type ObjectBigInt struct {
	NoDatabase
	Value *big.Int
}

func (obi *ObjectBigInt) Version() int {
	return 0
}

func (obi *ObjectBigInt) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&obi.Value)
}

func (obi *ObjectBigInt) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(obi.Value)
}

func (obi *ObjectBigInt) Equal(o Impl) bool {
	if obi2, ok := o.(*ObjectBigInt); ok {
		return obi.Value.Cmp(obi2.Value) == 0
	} else {
		return false
	}
}

func (obi *ObjectBigInt) Clear() {
	obi.Value = new(big.Int)
}

func (obi *ObjectBigInt) IsEmpty() bool {
	return obi.Value == nil || obi.Value.Sign() == 0
}

func ToBigInt(obj trie.Object) *ObjectBigInt {
	if obj == nil {
		return nil
	}
	return obj.(*Object).Real().(*ObjectBigInt)
}

func NewObjectBigInt(tag Tag) *ObjectBigInt {
	return &ObjectBigInt{
		Value: new(big.Int),
	}
}
