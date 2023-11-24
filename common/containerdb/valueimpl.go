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

package containerdb

import (
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type valueImpl struct {
	store ValueSnapshot
}

func (e *valueImpl) Bytes() []byte {
	return e.store.Bytes()
}

func (e *valueImpl) Object() trie.Object {
	return e.store.Object()
}

func (e *valueImpl) BigInt() *big.Int {
	if bs := e.Bytes(); bs != nil {
		value := new(big.Int)
		return intconv.BigIntSetBytes(value, bs)
	} else {
		return nil
	}
}

func (e *valueImpl) Int64() int64 {
	return intconv.BytesToInt64(e.Bytes())
}

func (e *valueImpl) Uint64() uint64 {
	return intconv.BytesToUint64(e.Bytes())
}

func (e *valueImpl) Address() module.Address {
	if bs := e.Bytes(); bs != nil {
		if addr, err := common.NewAddress(bs); err == nil {
			return addr
		}
	}
	return nil
}

func (e *valueImpl) String() string {
	if bs := e.Bytes(); bs != nil {
		return string(bs)
	} else {
		return ""
	}
}

func (e *valueImpl) Bool() bool {
	if bs := e.Bytes(); len(bs) == 0 {
		return false
	} else if len(bs) == 1 {
		return bs[0] != 0
	} else {
		var value big.Int
		return intconv.BigIntSetBytes(&value, bs).Sign() != 0
	}
}

func NewValue(vs ValueSnapshot) Value {
	return &valueImpl{vs}
}

type writableValueImpl struct {
	valueImpl
}

func (e *writableValueImpl) Set(v interface{}) error {
	if obj, ok := v.(trie.Object); ok {
		return e.store.(ValueState).SetObject(obj)
	} else {
		bs := ToBytes(v)
		return e.store.(ValueState).SetBytes(bs)
	}
}

func (e *writableValueImpl) Delete() (Value, error) {
	if vs, err := e.store.(ValueState).Delete(); err != nil {
		return nil, err
	} else {
		return &valueImpl{vs}, nil
	}
}

func NewWritableValue(vs ValueState) WritableValue {
	return &writableValueImpl{valueImpl{vs}}
}

func BigIntSafe(v Value) *big.Int {
	if v != nil {
		return intconv.BigIntSafe(v.BigInt())
	} else {
		return intconv.BigIntZero
	}
}

func Int64Safe(v Value) int64 {
	if v != nil {
		return v.Int64()
	} else {
		return 0
	}
}
