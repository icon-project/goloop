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
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type AccountCache struct {
	dict     *containerdb.DictDB
	accounts map[string]*Account
}

func (c *AccountCache) Add(account *Account) {
	key := icutils.ToKey(account.address)
	c.accounts[key] = account
}

func (c *AccountCache) Remove(owner module.Address) error {
	account := c.Get(owner)
	if account == nil {
		return errors.Errorf("Account not found: %s", owner)
	}

	account.Clear()
	return nil
}

func (c *AccountCache) Get(owner module.Address) *Account {
	key := icutils.ToKey(owner)
	account := c.accounts[key]
	if account != nil {
		return account
	}

	o := c.dict.Get(owner)
	if o == nil {
		account = newAccount(owner)
		c.Add(account)
		c.accounts[key] = account
	} else {
		account = ToAccount(o.Object(), owner)
		c.accounts[key] = account
	}
	return account
}

func (c *AccountCache) Clear() {
	c.accounts = make(map[string]*Account)
}

func (c *AccountCache) Reset() {
	for _, account := range c.accounts {
		value := c.dict.Get(account.address)

		if value == nil {
			account.Clear()
		} else {
			account.Set(ToAccount(value.Object(), account.address))
		}
	}
}

func (c *AccountCache) GetSnapshot() {
	for _, account := range c.accounts {
		account.freeze()
		key := account.address

		if account.IsEmpty() {
			if err := c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete Account key %x, err+%+v", key, err)
			}
		} else {
			o := icobject.New(TypeAccount, account)
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set snapshot for %x, err+%+v", key, err)
			}
		}
	}
}

func newAccountCache(store containerdb.ObjectStoreState) *AccountCache {
	return &AccountCache{
		accounts: make(map[string]*Account),
		dict:     containerdb.NewDictDB(store, 1, AccountDictPrefix),
	}
}
