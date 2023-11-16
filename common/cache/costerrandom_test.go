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
 */

package cache

import (
	"testing"
)

type stackRandSource []uint64

func Reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func sourceOf(nextRands ...uint64) *stackRandSource {
	Reverse(nextRands)
	ret := new(stackRandSource)
	*ret = nextRands
	return ret
}

func (src *stackRandSource) Int63() int64 {
	if len(*src) == 0 {
		return 0
	}
	var r uint64
	*src, r = (*src)[:len(*src)-1], (*src)[len(*src)-1]
	return int64(r)
}

func (src *stackRandSource) Seed(seed int64) {
}

func (src *stackRandSource) Uint64() uint64 {
	if len(*src) == 0 {
		return 0
	}
	var r uint64
	*src, r = (*src)[:len(*src)-1], (*src)[len(*src)-1]
	return r
}

func TestCosterRandom_Basics(t *testing.T) {
	c := NewCosterRandom[string, value](10, sourceOf(1))

	assertGetEqual(t, value(0), false, c, "k0")

	c.Put("k0", 3)
	assertGetEqual(t, value(3), true, c, "k0")

	c.Put("k1", 4)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(4), true, c, "k1")

	c.Put("k2", 5)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(0), false, c, "k1")
	assertGetEqual(t, value(5), true, c, "k2")
}

func TestCosterRandom_RejectTooHeavyValue(t *testing.T) {
	c := NewCosterRandom[string, value](10, sourceOf())
	c.Put("k0", 11)
	assertGetEqual(t, value(0), false, c, "k0")

	c.Put("k0", 10)
	assertGetEqual(t, value(10), true, c, "k0")
}

func TestCosterRandom_Update(t *testing.T) {
	c := NewCosterRandom[string, value](10, sourceOf(1, 1))

	c.Put("k0", 3)
	c.Put("k1", 3)
	c.Put("k2", 3)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(3), true, c, "k1")
	assertGetEqual(t, value(3), true, c, "k2")

	c.Put("k2", 4)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(3), true, c, "k1")
	assertGetEqual(t, value(4), true, c, "k2")

	// 1(=k1) is selected, thus remove last item
	c.Put("k1", 4)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(4), true, c, "k1")
	assertGetEqual(t, value(0), false, c, "k2")

	// 1(=k1) is selected
	c.Put("k2", 4)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(0), false, c, "k1")
	assertGetEqual(t, value(4), true, c, "k2")

	c.Put("k2", 5)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(0), false, c, "k1")
	assertGetEqual(t, value(5), true, c, "k2")

	c.Put("k0", 11)
	assertGetEqual(t, value(3), true, c, "k0")
	assertGetEqual(t, value(0), false, c, "k1")
	assertGetEqual(t, value(5), true, c, "k2")
}

func TestMakeCosterRandom(t *testing.T) {
	c := MakeCosterRandom[string, value](10, nil)
	c.Put("k0", 3)
	assertGetEqual(t, value(3), true, &c, "k0")
}

func TestNewCosterRandom(t *testing.T) {
	c := NewCosterRandom[string, value](10, nil)
	c.Put("k0", 3)
	assertGetEqual(t, value(3), true, c, "k0")
}
