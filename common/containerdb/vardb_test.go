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
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"log"
	"testing"
)

type TestStore struct {
	mutable trie.Mutable
}

func (s *TestStore) GetValue(k []byte) ([]byte, error) {
	v, err := s.mutable.Get(k)
	log.Printf("TestStore.GetValue(<%x>) -> <% x>, err=%+v", k, v, err)
	return v, err
}

func (s *TestStore) SetValue(k []byte, v []byte) ([]byte, error) {
	log.Printf("TestStore.SetValue(<%x>,<% x>)", k, v)
	return s.mutable.Set(k, v)
}

func (s *TestStore) DeleteValue(k []byte) ([]byte, error) {
	log.Printf("TestStore.DeleteValue(<%x>)", k)
	return s.mutable.Delete(k)
}

func TestNewVarDB(t *testing.T) {
	mdb := db.NewMapDB()
	tree := trie_manager.NewMutable(mdb, nil)
	db := NewVarDB(&TestStore{tree}, ToKey(HashBuilder, 1))
	db.Set(int(1))

	if v := int(db.Int64()); v != 1 {
		log.Printf("Returned Bytes <% x>", db.Bytes())
		t.Errorf("Fail to retrieved value (%v) is different", v)
		return
	}
	db.Delete()

	if v := db.Int64(); v != 0 {
		t.Errorf("Delete value should be zero v=%d", v)
	}
}
