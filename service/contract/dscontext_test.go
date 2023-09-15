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

package contract

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
)

func TestDSContextHistory_Basic(t *testing.T) {
	var history DSContextHistory

	// ensure there is no registered hash
	value := history.Get(100)
	assert.Nil(t, value)
	_, ok := history.FirstHeight()
	assert.False(t, ok)

	// register hash of height 100
	ok, err := history.Push(100, []byte("100"))
	assert.NoError(t, err)
	assert.True(t, ok)

	// check results
	value = history.Get(100)
	assert.Equal(t, []byte("100"), value)
	value = history.Get(101)
	assert.Equal(t, []byte("100"), value)
	value = history.Get(99)
	assert.Nil(t, value)
	fh, ok := history.FirstHeight()
	assert.True(t, ok)
	assert.EqualValues(t, 100, fh)

	// expecting error on pushing invalids
	ok, err = history.Push(100, []byte("100"))
	assert.Error(t, err)
	assert.False(t, ok)
	ok, err = history.Push(99, []byte("100"))
	assert.Error(t, err)
	assert.False(t, ok)

	// register hash of height 200
	ok, err = history.Push(200, []byte("200"))
	assert.NoError(t, err)
	assert.True(t, ok)

	// check results
	value = history.Get(101)
	assert.Equal(t, []byte("100"), value)
	value = history.Get(200)
	assert.Equal(t, []byte("200"), value)

	// skip same data
	oldSize := len(history)
	ok, err = history.Push(300, []byte("200"))
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, oldSize, len(history))
}

func TestDSContextHistory_Bytes(t *testing.T) {
	var h1 DSContextHistory
	ok, err := h1.Push(128, []byte("128"))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = h1.Push(130, []byte("130"))
	assert.NoError(t, err)
	assert.True(t, ok)

	bs := h1.Bytes()

	h2, err := DSContextHistoryFromBytes(bs)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)

	var h3 DSContextHistory
	assert.Nil(t, h3.Bytes())
	h4, err := DSContextHistoryFromBytes(nil)
	assert.NoError(t, err)
	assert.Nil(t, h4)

	h4, err = DSContextHistoryFromBytes([]byte{0x12, 0x44})
	assert.Error(t, err)
	assert.Nil(t, h4)
}

func TestDSContextHistoryLimit(t *testing.T) {
	var h1 DSContextHistory
	var base int64 = 458390
	makeHash := func(i int) []byte {
		return crypto.SHA3Sum256([]byte(fmt.Sprintf("hash%d", i)))
	}
	for i := 1 ; i<=DSContextHistoryLimit*2 ; i++ {
		ok, err := h1.Push(base+int64(i), makeHash(i))
		assert.NoError(t, err)
		assert.True(t, ok)
	}
	for i := 1 ; i<=DSContextHistoryLimit ; i++ {
		v := h1.Get(base+int64(i))
		assert.NotEqual(t, makeHash(i), v)
	}
	for i := DSContextHistoryLimit+1 ; i<=DSContextHistoryLimit*2 ; i++ {
		v := h1.Get(base+int64(i))
		assert.Equal(t, makeHash(i), v)
	}
	fh, ok := h1.FirstHeight()
	assert.True(t, ok)
	assert.Equal(t, base+DSContextHistoryLimit+1, fh)
	assert.Less(t, len(h1.Bytes()), 1024-36)
}
