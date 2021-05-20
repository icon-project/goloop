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

package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type AccountCache struct {
	dict     *containerdb.DictDB
	accounts map[string]*AccountState
}

func (c *AccountCache) Get(owner module.Address, createIfNotExist bool) *AccountState {
	key := icutils.ToKey(owner)
	account, ok := c.accounts[key]
	if ok {
		return account
	}

	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			account = newAccountStateWithSnapshot(nil)
		} else {
			return nil
		}
	} else {
		account = newAccountStateWithSnapshot(ToAccount(o.Object()))
	}
	c.accounts[key] = account
	return account
}

func (c *AccountCache) Clear() {
	c.Flush()
	c.accounts = make(map[string]*AccountState)
}

func (c *AccountCache) GetSnapshot(owner module.Address) *AccountSnapshot {
	key := icutils.ToKey(owner)
	account, ok := c.accounts[key]
	if ok {
		return account.GetSnapshot()
	}
	o := c.dict.Get(owner)
	if o == nil {
		return nil
	} else {
		return ToAccount(o.Object())
	}
}

func (c *AccountCache) Reset() {
	for key, account := range c.accounts {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			panic(errors.Errorf("Address convert error"))
		}
		value := c.dict.Get(addr)
		if value == nil {
			account.Reset(emptyAccountSnapshot)
		} else {
			account.Reset(ToAccount(value.Object()))
		}
	}
}

func (c *AccountCache) Flush() {
	for k, account := range c.accounts {
		key, err := common.BytesToAddress([]byte(k))
		if err != nil {
			panic(errors.Errorf("AccountCache is broken: %s", k))
		}

		ass := account.GetSnapshot()
		if ass.IsEmpty() {
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete Account key %x, err+%+v", key, err)
			}
		} else {
			o := icobject.New(TypeAccount, ass)
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set state for %x, err+%+v", key, err)
			}
		}
	}
}

func newAccountCache(store containerdb.ObjectStoreState) *AccountCache {
	return &AccountCache{
		accounts: make(map[string]*AccountState),
		dict:     containerdb.NewDictDB(store, 1, AccountDictPrefix),
	}
}
