/*
 * Copyright 2021 ICON Foundation
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

package blockv0_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
)

func TestBlockVote_NewBlockVote(t *testing.T) {
	w := wallet.New()
	bv := blockv0.NewBlockVote(w, 1, 0, []byte{0}, 0)
	err := bv.Verify()
	assert.NoError(t, err)
}

func FuzzNewBlockVotesFromBytes(f *testing.F) {
	cases := [][]byte{
		[]byte("\xc6\xc0\xc2\xc0\xc000"),
		[]byte("\xc4\xc0\xc1\xc00"),
		[]byte("\xce\xc0\xc4\u0081\xc0\xc000000000"),
	}
	for _, c := range cases {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, bs []byte) {
		bvl, err := blockv0.NewBlockVotesFromBytes(bs)
		if errors.Is(err, codec.ErrPanicInCustom) {
			t.Errorf("panic in custom %+v", err)
		}
		if err == nil {
			_, err := bvl.MarshalJSON()
			assert.NoError(t, err)
		}
	})
}
