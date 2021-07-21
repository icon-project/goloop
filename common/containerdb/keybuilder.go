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
	"github.com/icon-project/goloop/common/crypto"
)

type KeyBuilder interface {
	Append(keys ...interface{}) KeyBuilder
	Build() []byte
}

type hashKeyBuilder []byte

func (b hashKeyBuilder) Append(keys ...interface{}) KeyBuilder {
	return hashKeyBuilder(AppendKeys(b, keys...))
}

func (b hashKeyBuilder) Build() []byte {
	return crypto.SHA3Sum256(b)
}

type prefixedHashKeyBuilder struct {
	rawPrefix  []byte
	hashPrefix []byte
}

func (b *prefixedHashKeyBuilder) Append(keys ...interface{}) KeyBuilder {
	return &prefixedHashKeyBuilder{
		rawPrefix:  b.rawPrefix,
		hashPrefix: AppendKeys(b.hashPrefix, keys...),
	}
}

func (b *prefixedHashKeyBuilder) Build() []byte {
	return AppendKeys(b.rawPrefix, crypto.SHA3Sum256(b.hashPrefix))
}

type rlpKeyBuilder []byte

func (b rlpKeyBuilder) Append(keys ...interface{}) KeyBuilder {
	return rlpKeyBuilder(AppendKeys(b, keys...))
}

func (b rlpKeyBuilder) Build() []byte {
	return b
}

type rawKeyBuilder []byte

func (b rawKeyBuilder) Append(keys ...interface{}) KeyBuilder {
	return rawKeyBuilder(AppendRawKeys(b, keys...))
}

func (b rawKeyBuilder) Build() []byte {
	return b
}

type KeyBuilderType int

const (
	HashBuilder KeyBuilderType = iota
	PrefixedHashBuilder
	RLPBuilder
	RawBuilder
)

func NewHashKey(prefix []byte, keys ...interface{}) KeyBuilder {
	return hashKeyBuilder(AppendKeys(prefix, keys...))
}

func ToKey(builderType KeyBuilderType, keys ...interface{}) KeyBuilder {
	switch builderType {
	case HashBuilder:
		return hashKeyBuilder(AppendKeys([]byte{}, keys...))
	case PrefixedHashBuilder:
		return &prefixedHashKeyBuilder{
			rawPrefix:  ToBytes(keys[0]),
			hashPrefix: AppendKeys([]byte{}, keys[1:]...),
		}
	case RLPBuilder:
		return rlpKeyBuilder(AppendKeys([]byte{}, keys...))
	case RawBuilder:
		return rawKeyBuilder(AppendRawKeys([]byte{}, keys...))
	default:
		panic("Unsupported")
	}
}
