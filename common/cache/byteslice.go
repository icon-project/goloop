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

package cache

var pseudoNil = []byte{0}

type ByteSlice struct {
	val []byte
}

func (s ByteSlice) Get(fn func() []byte) []byte {
	if s.val == nil {
		s.val = fn()
		if s.val == nil {
			s.val = pseudoNil
		}
	}
	if len(s.val) == 1 && &s.val[0] == &pseudoNil[0] {
		return nil
	}
	return s.val
}
