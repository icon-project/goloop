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

package scoredb

import "github.com/icon-project/goloop/common/containerdb"

const (
	ArrayDBPrefix byte = 0x00
	DictDBPrefix  byte = 0x01
	VarDBPrefix   byte = 0x02
)

func NewArrayDB(store containerdb.BytesStoreState, keys ...interface{}) *containerdb.ArrayDB {
	key := containerdb.ToKey(containerdb.HashBuilder, ArrayDBPrefix).Append(keys...)
	return containerdb.NewArrayDB(store, key)
}

func NewDictDB(store containerdb.BytesStoreState, name string, depth int, keys ...interface{}) *containerdb.DictDB {
	key := containerdb.ToKey(containerdb.HashBuilder, DictDBPrefix, name).Append(keys...)
	return containerdb.NewDictDB(store, depth, key)
}

func NewVarDB(store containerdb.BytesStoreState, keys ...interface{}) *containerdb.VarDB {
	key := containerdb.ToKey(containerdb.HashBuilder, VarDBPrefix).Append(keys...)
	return containerdb.NewVarDB(store, key)
}

func NewStateStoreWith(s containerdb.BytesStoreSnapshot) containerdb.BytesStoreState {
	return containerdb.NewBytesStoreStateWithSnapshot(s)
}

func ToKey(t byte, keys ...interface{}) []byte {
	return containerdb.AppendKeys([]byte{t}, keys...)
}

func AppendKeys(b []byte, keys ...interface{}) []byte {
	return containerdb.AppendKeys(b, keys...)
}
