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
	"github.com/icon-project/goloop/common/trie"
)

type bytesConverter struct{}

func (bytesConverter) ObjectToBytes(value trie.Object) []byte {
	return value.(*Object).BytesValue()
}

func (bytesConverter) BytesToObject(value []byte) trie.Object {
	return NewBytesObject(value)
}

type ObjectStoreState struct {
	trie.MutableForObject
	bytesConverter
}

func NewObjectStoreState(t trie.MutableForObject) *ObjectStoreState {
	return &ObjectStoreState{t, bytesConverter{}}
}

type ObjectStoreSnapshot struct {
	trie.ImmutableForObject
	bytesConverter
}

func (o *ObjectStoreSnapshot) Set(_ []byte, _ trie.Object) (trie.Object, error) {
	panic("invalid usage")
}

func (o *ObjectStoreSnapshot) Delete(_ []byte) (trie.Object, error) {
	panic("invalid usage")
}

func NewObjectStoreSnapshot(t trie.ImmutableForObject) *ObjectStoreSnapshot {
	return &ObjectStoreSnapshot{t, bytesConverter{}}
}
