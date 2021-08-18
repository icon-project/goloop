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

package merkle

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/crypto"
)

type Item interface {
	Hash() []byte
}

type HashedItem []byte

func (i HashedItem) Hash() []byte {
	return i
}
func (i HashedItem) String() string {
	return fmt.Sprintf("hash(%#x)", []byte(i))
}

type ValueItem []byte

func (i ValueItem) Hash() []byte {
	if i == nil {
		return nil
	}
	if len(i) == crypto.HashLen {
		return i
	}
	return crypto.SHA3Sum256(i)
}
func (i ValueItem) String() string {
	return fmt.Sprintf("value(%#x)", []byte(i))
}

var nullHashBytes = make([]byte, crypto.HashLen)

func getHash(i Item) []byte {
	if i == nil {
		return nullHashBytes
	}
	if hv := i.Hash(); hv == nil {
		return nullHashBytes
	} else {
		return hv
	}
}

func mergeWithBytes(a, b Item) Item {
	bs := make([]byte, 0, crypto.HashLen*2)
	bs = append(bs, getHash(a)...)
	bs = append(bs, getHash(b)...)
	return HashedItem(crypto.SHA3Sum256(bs))
}

func hashToBytes(v []byte) []byte {
	if bytes.Equal(v, nullHashBytes) {
		return nil
	}
	return v
}

func CalcHashOfList(items []Item) []byte {
	if len(items) == 0 {
		return nil
	}
	merge := mergeWithBytes
	entries := make([]Item, len(items))
	copy(entries, items)
	for len(entries) > 1 {
		var idx int
		for idx = 0; idx < len(entries); idx += 2 {
			if idx+1 < len(entries) {
				entries[idx/2] = merge(entries[idx], entries[idx+1])
			} else {
				entries[idx/2] = entries[idx]
			}
		}
		entries = entries[0 : idx/2]
	}
	return hashToBytes(getHash(entries[0]))
}
