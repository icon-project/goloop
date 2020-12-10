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
	"reflect"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type storageEntry interface {
	Bytes() []byte
	SetBytes([]byte) ([]byte, error)
	GetObject(t reflect.Type) (trie.Object, error)
	SetObject(o trie.Object) (trie.Object, error)
	Delete() (Value, error)
}

type valueImpl struct {
	entry storageEntry
}

func (e *valueImpl) Bytes() []byte {
	return e.entry.Bytes()
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

func (e *valueImpl) Object(t reflect.Type) trie.Object {
	if obj, err := e.entry.GetObject(t); err != nil {
		return nil
	} else {
		return obj
	}
}

type writableValueImpl struct {
	valueImpl
}

func (e *writableValueImpl) Delete() (Value, error) {
	return e.valueImpl.entry.Delete()
}

func (e *writableValueImpl) Set(v interface{}) error {
	if obj, ok := v.(trie.Object); ok {
		_, err := e.valueImpl.entry.SetObject(obj)
		return err
	} else {
		_, err := e.valueImpl.entry.SetBytes(ToBytes(v))
		return err
	}
}

func (e *writableValueImpl) SetBytes(bs []byte) ([]byte, error) {
	return e.valueImpl.entry.SetBytes(bs)
}

type bytesEntry []byte

func (e bytesEntry) Bytes() []byte {
	return []byte(e)
}

func (e bytesEntry) SetBytes([]byte) ([]byte, error) {
	panic("invalid usage")
}

func (e bytesEntry) Delete() (Value, error) {
	panic("invalid usage")
}

func (e bytesEntry) GetObject(t reflect.Type) (trie.Object, error) {
	if t == nil || t == trie.TypeBytesObject {
		if e == nil {
			return nil, nil
		} else {
			return trie.BytesObject(e), nil
		}
	} else {
		return nil, errors.UnsupportedError.Errorf("UnsupportedType(%s)", t)
	}
}

func (e bytesEntry) SetObject(o trie.Object) (trie.Object, error) {
	panic("Invalid usage")
}

func newValueFromBytes(bs []byte) Value {
	if bs == nil {
		return nil
	}
	return &valueImpl{bytesEntry(bs)}
}

type bytesStoreEntry struct {
	key   []byte
	store BytesStoreState
}

func (e *bytesStoreEntry) Delete() (Value, error) {
	if bs, err := e.store.DeleteValue(e.key); err != nil || bs == nil {
		return nil, err
	} else {
		return newValueFromBytes(bs), nil
	}
}

func (e *bytesStoreEntry) SetBytes(bs []byte) ([]byte, error) {
	return e.store.SetValue(e.key, bs)
}

func (e *bytesStoreEntry) Bytes() []byte {
	if bs, err := e.store.GetValue(e.key); err == nil && bs != nil {
		return bs
	} else {
		return nil
	}
}

func (e *bytesStoreEntry) GetObject(t reflect.Type) (trie.Object, error) {
	if t == nil || t == trie.TypeBytesObject {
		if bs, err := e.store.GetValue(e.key); err != nil {
			return nil, err
		} else {
			if bs == nil {
				return nil, nil
			} else {
				return trie.BytesObject(bs), nil
			}
		}
	} else {
		return nil, errors.UnsupportedError.Errorf("UnsupportedType(%s)", t)
	}
}

func (e *bytesStoreEntry) SetObject(o trie.Object) (trie.Object, error) {
	if bs, err := e.store.SetValue(e.key, o.Bytes()); err != nil {
		return nil, err
	} else {
		if bs == nil {
			return nil, nil
		} else {
			return trie.BytesObject(bs), nil
		}
	}
}

func newValueFromBytesStore(store BytesStoreState, kbytes []byte) WritableValue {
	return &writableValueImpl{valueImpl{
		&bytesStoreEntry{kbytes, store},
	}}
}

type objectStoreEntry struct {
	key   []byte
	store ObjectStoreState
}

func (e *objectStoreEntry) Bytes() []byte {
	if obj, err := e.store.GetValue(e.key, trie.TypeBytesObject); err != nil {
		return nil
	} else {
		if bs, ok := obj.(trie.BytesObject); ok {
			return bs
		} else {
			return nil
		}
	}
}

func (e *objectStoreEntry) SetBytes(bytes []byte) ([]byte, error) {
	_, err := e.store.SetValue(e.key, trie.BytesObject(bytes))
	return nil, err
}

func (e *objectStoreEntry) GetObject(t reflect.Type) (trie.Object, error) {
	return e.store.GetValue(e.key, t)
}

func (e *objectStoreEntry) SetObject(o trie.Object) (trie.Object, error) {
	return e.store.SetValue(e.key, o)
}

func (e *objectStoreEntry) Delete() (Value, error) {
	if ro, err := e.store.DeleteValue(e.key); err != nil || ro == nil {
		return nil, err
	} else {
		return newValueFromObject(ro), nil
	}
}

func newValueFromObjectStore(state ObjectStoreState, key []byte) WritableValue {
	return &writableValueImpl{valueImpl{&objectStoreEntry{key, state}}}
}

type objectEntry struct {
	object trie.Object
}

func (e *objectEntry) Bytes() []byte {
	if e.object == nil {
		return nil
	} else {
		return e.object.Bytes()
	}
}

func (e *objectEntry) SetBytes(bytes []byte) ([]byte, error) {
	panic("invalid usage")
}

func (e *objectEntry) GetObject(t reflect.Type) (trie.Object, error) {
	if e.object == nil {
		return nil, nil
	}
	if t == nil || reflect.TypeOf(e.object) == t {
		return e.object, nil
	} else {
		return nil, errors.InvalidStateError.Errorf("IncompatibleObject(type=%T)", e.object)
	}
}

func (e *objectEntry) SetObject(o trie.Object) (trie.Object, error) {
	panic("invalid usage")
}

func (e *objectEntry) Delete() (Value, error) {
	panic("invalid usage")
}

func newValueFromObject(obj trie.Object) Value {
	return &valueImpl{&objectEntry{obj}}
}
