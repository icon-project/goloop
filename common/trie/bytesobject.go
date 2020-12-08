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

package trie

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
)

type BytesObject []byte

var TypeBytesObject = reflect.TypeOf(BytesObject(nil))

func (o BytesObject) Bytes() []byte {
	return o
}

func (o BytesObject) Reset(db db.Database, k []byte) error {
	log.Panicln("Bytes object can't RESET!!")
	return nil
}

func (o BytesObject) Flush() error {
	// Nothing to do because it comes from database itself.
	return nil
}

func (o BytesObject) String() string {
	return fmt.Sprintf("[%x]", []byte(o))
}

func (o BytesObject) Equal(n Object) bool {
	if bo, ok := n.(BytesObject); n != nil && !ok {
		return false
	} else {
		return bytes.Equal(bo, o)
	}
}

func (o BytesObject) Resolve(builder merkle.Builder) error {
	return nil
}

func (o BytesObject) ClearCache() {
	// nothing to do, because it doesn't have belonging objects.
}
