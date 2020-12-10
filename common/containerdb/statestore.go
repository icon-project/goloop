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

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/module"
)

type BytesStoreState interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) ([]byte, error)
	DeleteValue(key []byte) ([]byte, error)
}

type BytesStoreSnapshot interface {
	GetValue(key []byte) ([]byte, error)
}

type ObjectStoreState interface {
	GetValue(key []byte, t reflect.Type) (trie.Object, error)
	SetValue(key []byte, value trie.Object) (trie.Object, error)
	DeleteValue(key []byte) (trie.Object, error)
}

type ObjectStoreSnapshot interface {
	GetValue(key []byte, t reflect.Type) (trie.Object, error)
}

type ValueStoreState interface {
	GetValue(k []byte) Value
	At(k []byte) WritableValue
}

type Value interface {
	BigInt() *big.Int
	Int64() int64
	Address() module.Address
	Bytes() []byte
	String() string
	Bool() bool
	Object(t reflect.Type) trie.Object
}

type WritableValue interface {
	Value
	Delete() (Value, error)
	SetBytes(bs []byte) ([]byte, error)
	Set(interface{}) error
}

type bytesStoreStateForSnapshot struct {
	BytesStoreSnapshot
}

func (*bytesStoreStateForSnapshot) SetValue(key []byte, value []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("SetValue() on BytesStoreSnapshot")
}

func (*bytesStoreStateForSnapshot) DeleteValue(key []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("DeleteValue() on BytesStoreSnapshot")
}

func NewBytesStoreStateWithSnapshot(s BytesStoreSnapshot) BytesStoreState {
	if s == nil {
		return nil
	}
	return &bytesStoreStateForSnapshot{s}
}

type objectStoreStateForSnapshot struct {
	ObjectStoreSnapshot
}

func (o *objectStoreStateForSnapshot) SetValue(key []byte, value trie.Object) (trie.Object, error) {
	return nil, errors.InvalidStateError.New("SetValue() on ObjectStoreSnapshot")
}

func (o *objectStoreStateForSnapshot) DeleteValue(key []byte) (trie.Object, error) {
	return nil, errors.InvalidStateError.New("DeleteValue() on ObjectStoreSnapshot")
}

func NewObjectStoreStateWithSnapshot(s ObjectStoreSnapshot) ObjectStoreState {
	if s == nil {
		return nil
	}
	return &objectStoreStateForSnapshot{s}
}

type valueStoreStateForBytes struct {
	bss BytesStoreState
}

func (ss *valueStoreStateForBytes) GetValue(k []byte) Value {
	if bs, err := ss.bss.GetValue(k); err != nil {
		return nil
	} else {
		return newValueFromBytes(bs)
	}
}

func (ss *valueStoreStateForBytes) At(k []byte) WritableValue {
	return newValueFromBytesStore(ss.bss, k)
}

type valueStoreStateForObject struct {
	oss ObjectStoreState
}

func (ss *valueStoreStateForObject) GetValue(k []byte) Value {
	if obj, err := ss.oss.GetValue(k, nil); err != nil || obj == nil {
		return nil
	} else {
		return newValueFromObject(obj)
	}
}

func (ss *valueStoreStateForObject) At(k []byte) WritableValue {
	return newValueFromObjectStore(ss.oss, k)
}

type objectStoreStateForMutable struct {
	trie trie.MutableForObject
}

func (s *objectStoreStateForMutable) GetValue(key []byte, t reflect.Type) (trie.Object, error) {
	return s.trie.Get(key, t)
}

func (s *objectStoreStateForMutable) SetValue(key []byte, value trie.Object) (trie.Object, error) {
	return s.trie.Set(key, value)
}

func (s *objectStoreStateForMutable) DeleteValue(key []byte) (trie.Object, error) {
	return s.trie.Delete(key)
}

type objectStoreStateForImmutable struct {
	trie trie.ImmutableForObject
}

func (s *objectStoreStateForImmutable) GetValue(key []byte, t reflect.Type) (trie.Object, error) {
	return s.trie.Get(key, t)
}

func (s *objectStoreStateForImmutable) SetValue(key []byte, value trie.Object) (trie.Object, error) {
	return nil, errors.InvalidStateError.New("SetValue() on ImmutableForObject")
}

func (s *objectStoreStateForImmutable) DeleteValue(key []byte) (trie.Object, error) {
	return nil, errors.InvalidStateError.New("DeleteValue() on ImmutableForObject")
}

func NewObjectStoreStateWithImmutable(t trie.ImmutableForObject) ObjectStoreState {
	if t == nil {
		return nil
	}
	return &objectStoreStateForImmutable{t}
}

func NewStateStore(store interface{}) ValueStoreState {
	switch ss := store.(type) {
	case ValueStoreState:
		return ss
	case BytesStoreState:
		return &valueStoreStateForBytes{ss}
	case ObjectStoreState:
		return &valueStoreStateForObject{ss}
	case trie.MutableForObject:
		return &valueStoreStateForObject{&objectStoreStateForMutable{ss}}
	default:
		panic("invalid usage")
	}
}
