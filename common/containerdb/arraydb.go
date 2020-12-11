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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/scoreresult"
)

type ArrayDB struct {
	key   KeyBuilder
	size  WritableValue
	store StoreState
}

func NewArrayDB(source interface{}, key KeyBuilder) *ArrayDB {
	store := ToStoreState(source)
	return &ArrayDB{
		key:   key,
		size:  store.At(key.Build()),
		store: store,
	}
}

func (a *ArrayDB) Size() int {
	return int(a.size.Int64())
}

func (a *ArrayDB) Get(i int) Value {
	key := a.key.Append(i).Build()
	return a.store.GetValue(key)
}

func (a *ArrayDB) Set(i int, v interface{}) error {
	if i < 0 || i >= a.Size() {
		return scoreresult.ErrInvalidContainerAccess
	}
	key := a.key.Append(i).Build()
	return a.store.At(key).Set(v)
}

func (a *ArrayDB) Put(v interface{}) error {
	idx := a.Size()
	key := a.key.Append(idx).Build()
	if err := a.store.At(key).Set(v); err != nil {
		return err
	}
	return a.size.Set(idx + 1)
}

func (a *ArrayDB) Pop() Value {
	idx := a.Size()
	if idx == 0 {
		return nil
	}

	key := a.key.Append(idx - 1).Build()
	ov, err := a.store.At(key).Delete()
	if err != nil {
		log.Panicf("Fail to delete last element")
	}
	if idx > 1 {
		if err := a.size.Set(idx - 1); err != nil {
			log.Panicf("Fail to update size")
		}
	} else {
		if _, err := a.size.Delete(); err != nil {
			log.Panicf("Fail to delete size")
		}
	}
	return ov
}
