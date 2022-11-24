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

type RawBytesStoreState interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) ([]byte, error)
	Delete(key []byte) ([]byte, error)
}

type RawBytesStoreSnapshot interface {
	Get(key []byte) ([]byte, error)
}

type ObjectStoreState interface {
	Get(key []byte) (trie.Object, error)
	Set(key []byte, obj trie.Object) (trie.Object, error)
	Delete(key []byte) (trie.Object, error)
	ObjectToBytes(value trie.Object) []byte
	BytesToObject(value []byte) trie.Object
}

type ObjectStoreSnapshot interface {
	Get(key []byte) (trie.Object, error)
	ObjectToBytes(value trie.Object) []byte
}

type StoreState interface {
	GetValue(key []byte) Value
	At(key []byte) WritableValue
}

type ValueStore interface {
	GetValue(key []byte) ValueSnapshot
	At(key []byte) ValueState
}

type ValueSnapshot interface {
	Bytes() []byte
	Object() trie.Object
}

type ValueState interface {
	ValueSnapshot
	SetBytes(value []byte) error
	SetObject(value trie.Object) error
	Delete() (ValueSnapshot, error)
}

type BytesStore interface {
	Bytes() []byte
	SetBytes([]byte) error
	Delete() error
}

type Value interface {
	BigInt() *big.Int
	Int64() int64
	Uint64() uint64
	Address() module.Address
	Bytes() []byte
	String() string
	Bool() bool
	Object() trie.Object
}

type WritableValue interface {
	Value
	Delete() (Value, error)
	Set(interface{}) error
}

type snapshotStore struct {
	BytesStoreSnapshot
}

func (*snapshotStore) SetValue(key []byte, value []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("SetValue() on BytesStoreSnapshot")
}

func (*snapshotStore) DeleteValue(key []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("DeleteValue() on BytesStoreSnapshot")
}

func NewBytesStoreStateWithSnapshot(s BytesStoreSnapshot) BytesStoreState {
	if s == nil {
		return nil
	}
	return &snapshotStore{s}
}

type bytesEntry []byte

func (e bytesEntry) Object() trie.Object {
	return nil
}

func (e bytesEntry) Bytes() []byte {
	return []byte(e)
}

func NewValueSnapshotFromBytes(bs []byte) ValueSnapshot {
	if bs == nil {
		return nil
	}
	return bytesEntry(bs)
}

type bytesStoreEntry struct {
	key   []byte
	store BytesStoreState
}

func (e *bytesStoreEntry) Object() trie.Object {
	panic("invalid usage")
}

func (e *bytesStoreEntry) SetObject(value trie.Object) error {
	panic("invalid usage")
}

func (e *bytesStoreEntry) Delete() (ValueSnapshot, error) {
	if bs, err := e.store.DeleteValue(e.key); err != nil {
		return nil, err
	} else {
		return bytesEntry(bs), nil
	}
}

func (e *bytesStoreEntry) SetBytes(bs []byte) error {
	_, err := e.store.SetValue(e.key, bs)
	return err
}

func (e *bytesStoreEntry) Bytes() []byte {
	if bs, err := e.store.GetValue(e.key); err == nil && bs != nil {
		return bs
	} else {
		return nil
	}
}

func newValueStateFromStore(store BytesStoreState, kbytes []byte) ValueState {
	return &bytesStoreEntry{kbytes, store}
}

type storeStateForBytes struct {
	store BytesStoreState
}

func (s *storeStateForBytes) GetValue(key []byte) Value {
	if bs, err := s.store.GetValue(key); err != nil || bs == nil {
		return nil
	} else {
		return NewValue(NewValueSnapshotFromBytes(bs))
	}
}

func (s *storeStateForBytes) At(key []byte) WritableValue {
	return NewWritableValue(newValueStateFromStore(s.store, key))
}

type valueSnapshotForObject struct {
	store  ObjectStoreState
	object trie.Object
}

func (v *valueSnapshotForObject) Bytes() []byte {
	return v.store.ObjectToBytes(v.object)
}

func (v *valueSnapshotForObject) Object() trie.Object {
	return v.object
}

type valueStateForObject struct {
	store ObjectStoreState
	key   []byte
}

func (v *valueStateForObject) Bytes() []byte {
	if obj, err := v.store.Get(v.key); err != nil || obj == nil {
		return nil
	} else {
		return v.store.ObjectToBytes(obj)
	}
}

func (v *valueStateForObject) Object() trie.Object {
	if obj, err := v.store.Get(v.key); err != nil || obj == nil {
		return nil
	} else {
		return obj
	}
}

func (v *valueStateForObject) SetBytes(value []byte) error {
	_, err := v.store.Set(v.key, v.store.BytesToObject(value))
	return err
}

func (v *valueStateForObject) SetObject(value trie.Object) error {
	_, err := v.store.Set(v.key, value)
	return err
}

func (v *valueStateForObject) Delete() (ValueSnapshot, error) {
	if obj, err := v.store.Delete(v.key); err != nil || obj == nil {
		return nil, err
	} else {
		return &valueSnapshotForObject{v.store, obj}, nil
	}
}

type storeStateForObject struct {
	store ObjectStoreState
}

func (o *storeStateForObject) GetValue(key []byte) Value {
	if obj, err := o.store.Get(key); err != nil || obj == nil {
		return nil
	} else {
		return NewValue(&valueSnapshotForObject{o.store, obj})
	}
}

func (o *storeStateForObject) At(key []byte) WritableValue {
	return NewWritableValue(&valueStateForObject{o.store, key})
}

func ToStoreState(store interface{}) StoreState {
	switch ss := store.(type) {
	case StoreState:
		return ss
	case BytesStoreState:
		return &storeStateForBytes{ss}
	case ObjectStoreState:
		return &storeStateForObject{ss}
	default:
		panic("invalid usage")
	}
}

type bytesStoreStateForRaw struct {
	rbs RawBytesStoreState
}

func (s bytesStoreStateForRaw) GetValue(key []byte) ([]byte, error) {
	return s.rbs.Get(key)
}

func (s bytesStoreStateForRaw) SetValue(key []byte, value []byte) ([]byte, error) {
	return s.rbs.Set(key, value)
}

func (s bytesStoreStateForRaw) DeleteValue(key []byte) ([]byte, error) {
	return s.rbs.Delete(key)
}

func NewBytesStoreStateFromRaw(s RawBytesStoreState) BytesStoreState {
	return bytesStoreStateForRaw{s}
}

type bytesStoreSnapshotForRaw struct {
	rbs RawBytesStoreSnapshot
}

func (b bytesStoreSnapshotForRaw) GetValue(key []byte) ([]byte, error) {
	return b.rbs.Get(key)
}

func NewBytesStoreSnapshotFromRaw(s RawBytesStoreSnapshot) BytesStoreSnapshot {
	return bytesStoreSnapshotForRaw{s}
}

type emptyBytesStoreSnapshot struct{}

func (s emptyBytesStoreSnapshot) GetValue(k []byte) ([]byte, error) {
	return nil, nil
}

var EmptyBytesStoreSnapshot BytesStoreSnapshot = emptyBytesStoreSnapshot{}
var EmptyBytesStoreState BytesStoreState = &snapshotStore{EmptyBytesStoreSnapshot}
