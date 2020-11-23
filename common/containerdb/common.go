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
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

func rlpCountBytesForSize(b int) int {
	var cnt = int(1)
	for b >>= 8; b > 0; cnt++ {
		b >>= 8
	}
	return cnt
}

func rlpEncodeBytes(b []byte) []byte {
	var blen = len(b)
	if blen == 1 && b[0] < 0x80 {
		return b
	}
	if blen <= 55 {
		buf := make([]byte, blen+1)
		buf[0] = byte(0x80 + blen)
		copy(buf[1:], b)
		return buf
	}
	tslen := rlpCountBytesForSize(blen)
	buf := make([]byte, 1+tslen+blen)
	buf[0] = byte(0x80 + 55 + tslen)
	for tsidx := tslen; tsidx > 0; tsidx-- {
		buf[tsidx] = byte(blen & 0xff)
		blen >>= 8
	}
	copy(buf[tslen+1:], b)
	return buf
}

func AppendKeys(key []byte, keys ...interface{}) []byte {
	list := make([][]byte, len(keys))
	size := len(key)
	for i, k := range keys {
		list[i] = rlpEncodeBytes(ToBytes(k))
		size += len(list[i])
	}
	kbytes := make([]byte, len(key), size)
	copy(kbytes, key)
	for _, k := range list {
		kbytes = append(kbytes, k...)
	}
	return kbytes
}

func AppendRawKeys(key []byte, keys ...interface{}) []byte {
	list := make([][]byte, len(keys))
	size := len(key)
	for i, k := range keys {
		list[i] = ToBytes(k)
		size += len(list[i])
	}
	kbytes := make([]byte, len(key), size)
	copy(kbytes, key)
	for _, k := range list {
		kbytes = append(kbytes, k...)
	}
	return kbytes
}

func ToBytes(v interface{}) []byte {
	switch obj := v.(type) {
	case Value:
		return obj.Bytes()
	case module.Address:
		return obj.Bytes()
	case bool:
		if obj {
			return []byte{1}
		} else {
			return []byte{0}
		}
	case int:
		return intconv.Int64ToBytes(int64(obj))
	case int16:
		return intconv.Int64ToBytes(int64(obj))
	case int32:
		return intconv.Int64ToBytes(int64(obj))
	case int64:
		return intconv.Int64ToBytes(obj)
	case *big.Int:
		return intconv.BigIntToBytes(obj)
	case *common.HexInt:
		return obj.Bytes()
	case string:
		return []byte(obj)
	case []byte:
		return obj
	case byte:
		return []byte{obj}
	default:
		log.Panicf("UnknownType(%T)", v)
		return []byte{}
	}
}

func must(b []byte, e error) error {
	return errors.WithCode(e, errors.CriticalIOError)
}
