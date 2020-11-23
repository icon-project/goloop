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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"testing"
)

func TestNewArrayDB(t *testing.T) {
	mdb := db.NewMapDB()
	tree := trie_manager.NewMutable(mdb, nil)
	store := &TestStore{tree}

	arraydb := NewArrayDB(store, ToKey(HashBuilder, "Test"))
	arraydb.Put("Value1")
	arraydb.Put("Value2")
	arraydb.Set(1, "Value3")

	if err := arraydb.Set(2, "Value4"); err == nil {
		t.Errorf("It should fail on Set(2,Value4)")
		return
	}

	if s := arraydb.Size(); s != 2 {
		t.Errorf("Size must be 2, but s=%d", s)
		return
	}

	if v := arraydb.Get(2); v != nil {
		t.Errorf("Index out of range should return nil")
	}

	if v := arraydb.Get(-2); v != nil {
		t.Errorf("Index out of range should return nil")
	}

	if v := arraydb.Get(0).String(); v != "Value1" {
		t.Errorf("Fail to verify array exp=%s value=%s", "Value1", v)
		return
	}

	if v := arraydb.Pop().String(); v != "Value3" {
		t.Errorf("Poped value=%s is different from Value3", v)
		return
	}
	if v := arraydb.Pop(); v == nil {
		t.Errorf("Poped value must not be nil")
		return
	}
	if v := arraydb.Pop(); v != nil {
		t.Errorf("Poping on empty array should return nil")
		return
	}
}
