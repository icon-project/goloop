/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package transaction

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
)

type transactionData[T any] interface {
	serialize(buf *bytes.Buffer)
	*T
}

type wrapper[T any, PT transactionData[T]] struct {
	data  T
	id    []byte
	hash  []byte
	bytes []byte
}

func (w *wrapper[T,PT]) Bytes() []byte {
	if w.bytes == nil {
		w.bytes = codec.BC.MustMarshalToBytes(&w.data)
	}
	return w.bytes
}

func (w *wrapper[T,PT]) ID() []byte {
	if w.id == nil {
		buf := bytes.NewBuffer(nil)
		PT(&w.data).serialize(buf)
		w.id = crypto.SHA3Sum256(buf.Bytes())
	}
	return w.id
}

func (w *wrapper[T,PT]) Hash() []byte {
	if w.hash == nil {
		w.hash = crypto.SHA3Sum256(w.Bytes())
	}
	return w.hash
}
