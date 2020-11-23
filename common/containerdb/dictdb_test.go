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

func TestNewDictDB(t *testing.T) {
	mdb := db.NewMapDB()
	tree := trie_manager.NewMutable(mdb, nil)
	store := &TestStore{tree}
	dict := NewDictDB(store, 2, ToKey(HashBuilder, "mapdb"))
	dict2 := dict.GetDB(1)

	dict2.Set(1, 1)
	if v := dict.Get(1, 1).Int64(); v != 1 {
		t.Errorf("Stored value=%d is different from 1", v)
		return
	}

	if err := dict.Set(1, 2, "Test"); err != nil {
		t.Errorf("Fail to DictDB[1][2] = Test")
		return
	}

	if err := dict.Set(1, make([]byte, 1024), 3); err != nil {
		t.Errorf("Fail to DictDB[1][long_bytes] = 3")
		return
	}

	if v := dict2.Get(2).String(); v != "Test" {
		t.Errorf("Returned string=%s is different from Test", v)
		return
	}

	if err := dict.Set(1, "Failed"); err == nil {
		t.Errorf("It should fail on DictDB[1] = 'Failed'")
		return
	}

	if dict3 := dict.GetDB(1, 2); dict3 != nil {
		t.Errorf("It should return nil for value depth")
	}

	if dict3 := dict.GetDB(1, 2, 3); dict3 != nil {
		t.Errorf("It should return nil with invalid key")
	}

	if v := dict2.Get(2, 3); v != nil {
		t.Errorf("it should return nil with invalid key")
	}

	if err := dict.Delete(1, 2); err != nil {
		t.Errorf("Fail to delete key")
	}

	if err := dict.Delete(1, 2, 3); err == nil {
		t.Errorf("It should fail to delete with illegal key")
	}

	if v := dict2.Get(2); v != nil {
		t.Errorf("it should return nil with invalid key")
	}
}
