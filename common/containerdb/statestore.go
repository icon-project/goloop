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
	"github.com/icon-project/goloop/module"
)

type StateStore interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) ([]byte, error)
	DeleteValue(key []byte) ([]byte, error)
}

type SnapshotStore interface {
	GetValue(key []byte) ([]byte, error)
}

type BytesStore interface {
	Bytes() []byte
	SetBytes([]byte) error
	Delete() error
}

type Value interface {
	BigInt() *big.Int
	Int64() int64
	Address() module.Address
	Bytes() []byte
	String() string
	Bool() bool
}

type WritableValue interface {
	Value
	Delete() error
	Set(interface{}) error
}

type snapshotStore struct {
	SnapshotStore
}

func (*snapshotStore) SetValue(key []byte, value []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("SetValue() on SnapshotStore")
}

func (*snapshotStore) DeleteValue(key []byte) ([]byte, error) {
	return nil, errors.InvalidStateError.New("DeleteValue() on SnapshotStore")
}

func NewStateStoreWith(s SnapshotStore) StateStore {
	if s == nil {
		return nil
	}
	return &snapshotStore{s}
}
