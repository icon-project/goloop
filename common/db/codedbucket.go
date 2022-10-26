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

package db

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
)

type CodedBucket struct {
	dbBucket Bucket
	hasher   Hasher
	codec    codec.Codec
}

func NewCodedBucket(database Database, id BucketID, c codec.Codec) (*CodedBucket, error) {
	b := &CodedBucket{}
	dbb, err := database.GetBucket(id)
	if err != nil {
		return nil, err
	}
	b.dbBucket = dbb
	b.hasher = id.Hasher()
	if c == nil {
		c = codec.BC
	}
	b.codec = c
	return b, nil
}

func NewCodedBucketFromBucket(bk Bucket, hasher Hasher, c codec.Codec) *CodedBucket {
	b := &CodedBucket{}
	b.dbBucket = bk
	if hasher == nil {
		hasher = sha3Hasher{}
	}
	b.hasher = hasher
	if c == nil {
		c = codec.BC
	}
	b.codec = c
	return b
}

type Raw []byte

func (b *CodedBucket) _marshal(obj interface{}) ([]byte, error) {
	if bs, ok := obj.(Raw); ok {
		return bs, nil
	}
	buf := bytes.NewBuffer(nil)
	err := b.codec.Marshal(buf, obj)
	return buf.Bytes(), err
}

func (b *CodedBucket) Get(key interface{}, value interface{}) error {
	bs, err := b.GetBytes(key)
	if err != nil {
		return err
	}
	return b.codec.Unmarshal(bytes.NewBuffer(bs), value)
}

func (b *CodedBucket) GetBytes(key interface{}) ([]byte, error) {
	keyBS, err := b._marshal(key)
	if err != nil {
		return nil, err
	}
	bs, err := b.dbBucket.Get(keyBS)
	if bs == nil && err == nil {
		err = errors.NotFoundError.New("NotFound")
	}
	return bs, err
}

func (b *CodedBucket) Set(key interface{}, value interface{}) error {
	keyBS, err := b._marshal(key)
	if err != nil {
		return err
	}
	valueBS, err := b._marshal(value)
	if err != nil {
		return err
	}
	err = b.dbBucket.Set(keyBS, valueBS)
	if err != nil {
		err = errors.Wrap(err, "Fail to set KV DB")
	}
	return err
}

func (b *CodedBucket) Put(value interface{}) error {
	valueBS, err := b._marshal(value)
	if err != nil {
		return err
	}
	keyBS := b.hasher.Hash(valueBS)
	err = b.dbBucket.Set(keyBS, valueBS)
	if err != nil {
		err = errors.Wrap(err, "Fail to set KV DB")
	}
	return err
}
