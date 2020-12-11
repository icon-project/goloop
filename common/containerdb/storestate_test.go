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
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
)

const (
	TypeBytes = iota
	TypeObject
)

type CustomObject struct {
	Type  int
	Value []byte
}

func (o *CustomObject) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(o)
}

func (o *CustomObject) BytesValue() []byte {
	if o.Type == TypeBytes {
		return o.Value
	} else {
		return nil
	}
}

func (o *CustomObject) Reset(s db.Database, k []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(k, o)
	return err
}

func (o *CustomObject) Flush() error {
	// do nothing
	return nil
}

func (o *CustomObject) Equal(object trie.Object) bool {
	if o2, ok := object.(*CustomObject); ok {
		return o.Type == o2.Type && bytes.Equal(o.Value, o2.Value)
	}
	return false
}

func (o *CustomObject) Resolve(builder merkle.Builder) error {
	return nil
}

func (o *CustomObject) ClearCache() {
	// do nothing
}

type customStoreState struct {
	trie.MutableForObject
}

func (c *customStoreState) ObjectToBytes(value trie.Object) []byte {
	return value.(*CustomObject).BytesValue()
}

func (c *customStoreState) BytesToObject(value []byte) trie.Object {
	return &CustomObject{TypeBytes, value}
}

func TestStoreStateForObject_VarDB(t *testing.T) {
	database := db.NewMapDB()
	tree := trie_manager.NewMutableForObject(database, nil, reflect.TypeOf((*CustomObject)(nil)))
	ss := &customStoreState{tree}
	array := NewArrayDB(ss, ToKey(RLPBuilder, []byte{0x02}, "objects"))
	obj1 := &CustomObject{TypeObject, []byte("Hello")}
	obj2 := &CustomObject{TypeObject, []byte("World")}
	array.Put(obj1)
	array.Put(obj2)

	snapshot := tree.GetSnapshot()
	snapshot.Flush()

	tree2 := trie_manager.NewMutableForObject(database, snapshot.Hash(), reflect.TypeOf((*CustomObject)(nil)))
	ss2 := &customStoreState{tree2}
	array2 := NewArrayDB(ss2, ToKey(RLPBuilder, []byte{0x02}, "objects"))

	assert.Equal(t, 2, array2.Size())
	assert.Equal(t, obj1, array2.Get(0).Object())
	assert.Equal(t, obj2, array2.Get(1).Object())
}
