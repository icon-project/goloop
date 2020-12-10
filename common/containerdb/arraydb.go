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
	store StateStore
}

func NewArrayDB(store StateStore, key KeyBuilder) *ArrayDB {
	return &ArrayDB{
		key:   key,
		size:  NewValueFromStore(store, key.Build()),
		store: store,
	}
}

func (a *ArrayDB) Size() int {
	return int(a.size.Int64())
}

func (a *ArrayDB) Get(i int) Value {
	key := a.key.Append(i).Build()
	bs, err := a.store.GetValue(key)
	if err != nil || bs == nil {
		return nil
	}
	return NewValueFromBytes(bs)
}

func (a *ArrayDB) Set(i int, v interface{}) error {
	if i < 0 || i >= a.Size() {
		return scoreresult.ErrInvalidContainerAccess
	}
	key := a.key.Append(i).Build()
	return must(a.store.SetValue(key, ToBytes(v)))
}

func (a *ArrayDB) Put(v interface{}) error {
	idx := a.Size()
	key := a.key.Append(idx).Build()
	value := ToBytes(v)
	if err := must(a.store.SetValue(key, value)); err != nil {
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
	var bs []byte
	if ov, err := a.store.DeleteValue(key); err != nil {
		log.Panicf("Fail to delete last element")
	} else {
		bs = ov
	}
	if idx > 1 {
		if err := a.size.Set(idx - 1); err != nil {
			log.Panicf("Fail to update size")
		}
	} else {
		if err := a.size.Delete(); err != nil {
			log.Panicf("Fail to delete size")
		}
	}
	return NewValueFromBytes(bs)
}
