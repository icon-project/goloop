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
	"github.com/icon-project/goloop/module"
)

type valueImpl struct {
	BytesStore
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
	if bs := e.Bytes(); len(bs) != 0 {
		if len(bs) <= 8 {
			return intconv.BytesToInt64(bs)
		} else {
			return intconv.BytesToInt64(bs[len(bs)-8:])
		}
	}
	return 0
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
	if bs := e.Bytes(); len(bs) <= 1 {
		return intconv.BytesToInt64(bs) != 0
	} else {
		var value big.Int
		return intconv.BigIntSetBytes(&value, bs).Sign() != 0
	}
}

func (e *valueImpl) Set(v interface{}) error {
	bs := ToBytes(v)
	return e.SetBytes(bs)
}

type bytesEntry []byte

func (e bytesEntry) Bytes() []byte {
	return []byte(e)
}

func (e bytesEntry) SetBytes([]byte) error {
	return nil
}

func (e bytesEntry) Delete() error {
	return nil
}

func NewValueFromBytes(bs []byte) Value {
	if bs == nil {
		return nil
	}
	return &valueImpl{bytesEntry(bs)}
}

type storeEntry struct {
	key   []byte
	store StateStore
}

func (e *storeEntry) Delete() error {
	return must(e.store.DeleteValue(e.key))
}

func (e *storeEntry) SetBytes(bs []byte) error {
	return must(e.store.SetValue(e.key, bs))
}

func (e *storeEntry) Bytes() []byte {
	if bs, err := e.store.GetValue(e.key); err == nil && bs != nil {
		return bs
	} else {
		return nil
	}
}

func NewValueFromStore(store StateStore, kbytes []byte) WritableValue {
	return &valueImpl{&storeEntry{kbytes, store}}
}
