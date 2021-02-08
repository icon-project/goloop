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

package icobject

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/merkle"
)

type BytesImpl []byte

func (b BytesImpl) Version() int {
	return 0
}

func (b BytesImpl) RLPDecodeFields(decoder codec.Decoder) error {
	return errors.InvalidStateError.New("InvalidUsage")
}

func (b BytesImpl) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(b)
}

func (b BytesImpl) Reset(dbase db.Database) error {
	return nil
}

func (b BytesImpl) Resolve(builder merkle.Builder) error {
	return nil
}

func (b BytesImpl) ClearCache() {
	// do nothing
}

func (b BytesImpl) Flush() error {
	return nil
}

func (b BytesImpl) Equal(o Impl) bool {
	return bytes.Equal(b, o.(BytesImpl))
}

func NewBytesObject(bs []byte) *Object {
	return New(TypeBytes, BytesImpl(bs))
}
