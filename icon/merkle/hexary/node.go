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

package hexary

import (
	"github.com/icon-project/goloop/common/cache"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const (
	hashLen      = 32
	maxChildren  = 16
	maxNodeBytes = hashLen * maxChildren

	defaultCacheCap = 32
)

func validateNodeBytes(bytes []byte) error {
	if len(bytes)%hashLen != 0 || len(bytes) > maxNodeBytes {
		return errors.InvalidStateError.Errorf("bad node bytes length %d", bytes)
	}
	return nil
}

type node struct {
	bytes []byte
	_hash []byte
}

func newNode() *node {
	return &node{
		bytes: make([]byte, 0, maxNodeBytes),
		_hash: nil,
	}
}

func newNodeFromBytes(bytes []byte) (*node, error) {
	if err := validateNodeBytes(bytes); err != nil {
		return nil, err
	}
	return &node{
		bytes: bytes,
		_hash: nil,
	}, nil
}

func (b *node) Len() int {
	return len(b.bytes) / hashLen
}

func (b *node) SetLen(l int) {
	b.bytes = b.bytes[:l*hashLen]
	b._hash = nil
}

func (b *node) Full() bool {
	return len(b.bytes) == maxNodeBytes
}

func (b *node) Empty() bool {
	return len(b.bytes) == 0
}

func (b *node) Get(i int) []byte {
	if i >= maxChildren {
		log.Panicf("bad index %d for node", i)
	}
	if i < b.Len() {
		return b.bytes[i*hashLen : (i+1)*hashLen]
	}
	return nil
}

func (b *node) GetCopy(i int) []byte {
	return append([]byte(nil), b.Get(i)...)
}

func (b *node) Bytes() []byte {
	return b.bytes
}

func (b *node) Add(hash []byte) bool {
	if len(hash) != hashLen {
		log.Panicf("bad hash len")
	}
	if b.Full() {
		log.Panicf("add on full node")
	}
	b.bytes = append(b.bytes, hash...)
	b._hash = nil
	return b.Full()
}

func (b *node) RemoveBack() {
	if b.Empty() {
		log.Panicf("unadding on zero length node")
	}
	b.bytes = b.bytes[:len(b.bytes)-hashLen]
	b._hash = nil
}

func (b *node) Hash() []byte {
	if b.Empty() {
		return nil
	}
	if b._hash == nil {
		b._hash = crypto.SHA3Sum256(b.bytes)
	}
	return b._hash
}

func (b *node) Clear() {
	b.bytes = b.bytes[:0]
	b._hash = nil
}

func (b *node) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(b.bytes)
}

func (b *node) RLPDecodeSelf(d codec.Decoder) error {
	bytes := make([]byte, 0, maxNodeBytes)
	if err := d.Decode(&bytes); err != nil {
		return err
	}
	if err := validateNodeBytes(bytes); err != nil {
		return err
	}
	b.bytes = bytes
	b._hash = nil
	return nil
}

type nodeDB struct {
	bk        db.Bucket
	nodeCache *cache.SimpleCache
}

func newCachedNodeDB(bk db.Bucket, cap int) *nodeDB {
	if cap <= 0 {
		cap = defaultCacheCap
	}
	return &nodeDB{
		bk:        bk,
		nodeCache: cache.NewSimpleCache(cap),
	}
}

func (b *nodeDB) Put(br *node) error {
	if b.nodeCache.Get(br.Hash()) != nil {
		return nil
	}
	if err := b.bk.Set(br.Hash(), br.bytes); err != nil {
		return err
	}
	b.nodeCache.Put(br.Hash(), br)
	return nil
}

func (b *nodeDB) Get(key []byte) (*node, error) {
	r := b.nodeCache.Get(key)
	if r != nil {
		return r.(*node), nil
	}
	bs, err := db.DoGet(b.bk, key)
	if err != nil {
		return nil, err
	}
	br, err := newNodeFromBytes(bs)
	if err != nil {
		return nil, err
	}
	b.nodeCache.Put(key, br)
	return br, err
}
