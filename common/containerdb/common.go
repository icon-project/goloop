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
	"io"
	"math"
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

func rlpReadSize(b []byte, slen int) (int, error) {
	if slen > len(b) {
		return 0, errors.IllegalArgumentError.Errorf(
			"NotEnoughBytes(exp=%d,real=%d)", slen, len(b))
	}
	var s uint64
	switch slen {
	case 1:
		s = uint64(b[0])
	case 2:
		s = uint64(b[0])<<8 | uint64(b[1])
	case 3:
		s = uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		s = uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		s = uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 | uint64(b[4])
	case 6:
		s = uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
	case 7:
		s = uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 | uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		s = uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	if s < 56 || b[0] == 0 || s > math.MaxInt {
		return 0, errors.IllegalArgumentError.New("InvalidSizeField")
	}
	return int(s), nil
}

func rlpParseBytes(bs []byte) ([]byte, []byte, error) {
	if len(bs) == 0 {
		return nil, nil, io.EOF
	}
	tag := bs[0]
	data := bs[1:]
	switch {
	case tag < 0x80:
		return []byte{tag}, data, nil
	case tag < 0xB8:
		size := int(tag - 0x80)
		if len(data) < size {
			return nil, nil, errors.IllegalArgumentError.Errorf(
				"NotEnoughBytes(exp=%d,real=%d)", size, len(data))
		}
		return data[:size], data[size:], nil
	case tag < 0xC0:
		ts := int(tag - 0xb7)
		size, err := rlpReadSize(data, ts)
		if err != nil {
			return nil, nil, err
		}
		data = data[ts:]
		if len(data) < size {
			return nil, nil, errors.IllegalArgumentError.Errorf(
				"NotEnoughBytes(exp=%d,real=%d)", size, len(data))
		}
		return data[:size], data[size:], nil
	default:
		return nil, nil, errors.IllegalArgumentError.New(
			"InvalidType(exp=bytes,real=list)")
	}
}

func SplitKeys(key []byte) ([][]byte, error) {
	var keys [][]byte
	for len(key) > 0 {
		if part, remain, err := rlpParseBytes(key); err != nil {
			return nil, err
		} else {
			keys = append(keys, part)
			key = remain
		}
	}
	return keys, nil
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
