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

type BitSetFilter []byte

func MakeBitSetFilter(bytes int) BitSetFilter {
	return make([]byte, bytes)
}

func (f BitSetFilter) Set(i int64) {
	f[i/byteBits] |= 1 << (i % byteBits)
}

func (f BitSetFilter) Test(i int64) bool {
	return f[i/byteBits]&(1<<(i%byteBits)) != 0
}
