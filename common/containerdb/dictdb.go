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
	"github.com/icon-project/goloop/service/scoreresult"
)

type DictDB struct {
	key   KeyBuilder
	store StoreState
	depth int
}

func NewDictDB(source interface{}, depth int, key KeyBuilder) *DictDB {
	store := ToStoreState(source)
	return &DictDB{
		key:   key,
		store: store,
		depth: depth,
	}
}

func (d *DictDB) GetDB(keys ...interface{}) *DictDB {
	if len(keys) >= d.depth {
		return nil
	}

	return &DictDB{
		key:   d.key.Append(keys...),
		store: d.store,
		depth: d.depth - len(keys),
	}
}

func (d *DictDB) Get(keys ...interface{}) Value {
	if len(keys) != d.depth {
		return nil
	}
	return d.store.GetValue(d.key.Append(keys...).Build())
}

func (d *DictDB) Set(params ...interface{}) error {
	if len(params) != d.depth+1 {
		return scoreresult.ErrInvalidContainerAccess
	}

	key := d.key.Append(params[:len(params)-1]...).Build()
	return d.store.At(key).Set(params[len(params)-1])
}

func (d *DictDB) Delete(kv ...interface{}) error {
	if len(kv) != d.depth {
		return scoreresult.ErrInvalidContainerAccess
	}
	_, err := d.store.At(d.key.Append(kv...).Build()).Delete()
	return err
}
