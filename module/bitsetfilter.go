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

package module

const (
	byteBits = 8
)

type BitSetFilter struct {
	s []byte
}

func MakeBitSetFilter(capInBytes int) BitSetFilter {
	return BitSetFilter{make([]byte, 0, capInBytes)}
}

func BitSetFilterFromBytes(s []byte, capInBytes int) BitSetFilter {
	f := MakeBitSetFilter(capInBytes)
	f.s = append(f.s, s...)
	return f
}

func (f *BitSetFilter) indexAndOffset(idx int64) (int, int) {
	return (int(idx) / byteBits) % cap(f.s), int(idx) % byteBits
}

func (f *BitSetFilter) Set(idx int64) {
	i, o := f.indexAndOffset(idx)
	if i >= len(f.s) {
		// increase len
		f.s = f.s[:i+1]
	}
	f.s[i] |= 1 << o
}

func (f BitSetFilter) Test(idx int64) bool {
	if cap(f.s) == 0 {
		return false
	}
	i, o := f.indexAndOffset(idx)
	return (f.s[i] & (1 << o)) != 0
}

func (f BitSetFilter) Bytes() []byte {
	if len(f.s) == 0 {
		return nil
	}
	return f.s
}
