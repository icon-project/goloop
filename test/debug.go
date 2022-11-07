/*
 * Copyright 2022 ICON Foundation
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

package test

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	cerrors "github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
)

func bytesToSize(bs []byte) (int, error) {
	if value := int(intconv.BytesToSize(bs)); value < 0 {
		return 0, cerrors.Wrapf(codec.ErrInvalidFormat, "InvalidFormat(size=%d)", value)
	} else {
		return value, nil
	}
}

func DumpRLP(indent string, data []byte) string {
	buf := bytes.NewBuffer(nil)
	p := 0
	stack := []int{}
	limit := len(data)
	read := func(n int) ([]byte, bool) {
		if n < 0 {
			fmt.Fprintf(buf, "%sinvalid size(offset=%d,size=%d)\n", indent, p, n)
			return nil, false
		}
		if n <= limit-p {
			f, t := p, p+n
			p += n
			return data[f:t], true
		}
		fmt.Fprintf(buf, "%sno data(offset=%d,size=%d,limit=%d)\n", indent, p, n, limit)
		return nil, false
	}
	readSize := func(n int) (int, bool) {
		bs, ok := read(n)
		if !ok {
			return 0, false
		}
		if value, err := bytesToSize(bs); err != nil {
			return 0, false
		} else {
			return value, true
		}
	}
	pop := func() bool {
		if len(stack) < 1 {
			return false
		}
		limit = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		indent = indent[:len(indent)-2]
		fmt.Fprintf(buf, "%s]\n", indent)
		return true
	}
	push := func(n int) bool {
		if n < 0 {
			fmt.Fprintf(buf, "%sinvalid size(offset=%d,size=%d)\n", indent, p, n)
			return false
		}
		if n > limit-p {
			fmt.Fprintf(buf, "%sno data(offset=%d,size=%d,limit=%d)\n", indent, p, n, limit)
			return false
		}
		fmt.Fprintf(buf, "%slist(%#x:%d) [\n", indent, n, n)
		stack = append(stack, limit)
		limit = p + n
		indent += "  "
		return true
	}
loop:
	for {
		for p == limit {
			if ok := pop(); !ok {
				break loop
			}
		}
		switch q := data[p]; {
		case q < 0x80:
			bs, ok := read(1)
			if !ok {
				break loop
			}
			fmt.Fprintf(buf, "%sbytes(0x%x:%d) : %x\n", indent, 1, 1, bs)
		case q <= 0xb7:
			l := int(q - 0x80)
			p += 1
			bs, ok := read(l)
			if !ok {
				break loop
			}
			fmt.Fprintf(buf, "%sbytes(0x%x:%d) : %x\n", indent, l, l, bs)
		case q <= 0xbf:
			ll := int(q - 0xb7)
			p += 1
			l, ok := readSize(ll)
			if !ok {
				break loop
			}
			bs, ok := read(l)
			if !ok {
				break loop
			}
			fmt.Fprintf(buf, "%sbytes(0x%x:%d) : %x\n", indent, l, l, bs)
		case q <= 0xf7:
			l := int(q - 0xc0)
			p += 1
			if ok := push(l); !ok {
				break loop
			}
		default:
			ll := int(q - 0xf7)
			p += 1
			l, ok := readSize(ll)
			if !ok {
				break loop
			}
			if ll == 1 && l == 0 {
				fmt.Fprintf(buf, "%snull(0x2:2)\n", indent)
				break
			}
			if ok := push(l); !ok {
				break loop
			}
		}
	}
	return buf.String()
}
