/*
 * Copyright 2021 ICON Foundation
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

package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var allPRepArrayPrefix = containerdb.ToKey(
	containerdb.HashBuilder,
	scoredb.ArrayDBPrefix,
	"active_prep",
)

// =====================================================

type AllPRepCache struct {
	arraydb *containerdb.ArrayDB
}

// Add adds a new active PRep to State
// Duplicated address check MUST BE done before adding
func (c *AllPRepCache) Add(owner module.Address) error {
	if owner == nil {
		return errors.Errorf("Invalid argument")
	}
	o := icobject.NewBytesObject(owner.Bytes())
	return c.arraydb.Put(o)
}

func (c *AllPRepCache) Size() int {
	return c.arraydb.Size()
}

func (c *AllPRepCache) Get(i int) module.Address {
	if i < 0 || i >= c.Size() {
		return nil
	}
	return c.arraydb.Get(i).Address()
}

func NewAllPRepCache(store containerdb.ObjectStoreState) *AllPRepCache {
	return &AllPRepCache{
		arraydb: containerdb.NewArrayDB(store, allPRepArrayPrefix),
	}
}
