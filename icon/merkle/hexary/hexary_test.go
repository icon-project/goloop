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

package hexary_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/merkle/hexary"
)

func newMapBucket() db.Bucket {
	bk, _ := db.NewMapDB().GetBucket("")
	return bk
}

const (
	maxHash    = 16*16 + 1
	cacheMax   = 64
)

func TestBasics(t *testing.T) {
	hashes := make([][]byte, maxHash)
	perm := newMapBucket()
	ac, err := hexary.NewAccumulator(perm, newMapBucket(), "")
	assert.NoError(t, err)
	for i:=int64(0); i < maxHash; i++ {
		bs := codec.MustMarshalToBytes(&i)
		hashes[i] = crypto.SHA3Sum256(bs)
		assert.Equal(t, i, ac.Len())
		err = ac.Add(hashes[i])
		assert.NoError(t, err)
	}
	header, err := ac.Finalize()
	assert.NoError(t, err)

	prover, err := hexary.NewMerkleTree(perm, header, cacheMax)
	assert.NoError(t, err)
	builder, err := hexary.NewMerkleTree(newMapBucket(), header, cacheMax)
	assert.NoError(t, err)
	for i := int64(0); i < maxHash; i++ {
		proof, err := prover.Prove(i, -1)
		assert.NoError(t, err)

		// test modified hash
		err = builder.Add(i, hashes[(i+1) %maxHash], proof)
		assert.True(t, errors.Is(err, hexary.ErrVerify), "unexpected error or nil : %+v", err)

		l := len(proof)
		// test too short proof
		if l > 0 {
			err = builder.Add(i, hashes[i], proof[1:])
			assert.True(t, errors.Is(err, hexary.ErrVerify), "unexpected error or nil : %+v", err)
			err = builder.Add(i, hashes[i], proof[:l-1])
			assert.True(t, errors.Is(err, hexary.ErrVerify), "unexpected error or nil : %+v", err)
		}

		// test modified proof
		if l > 0 {
			proof[l-1][0] = ^proof[l-1][0]
			err = builder.Add(i, hashes[i], proof)
			assert.True(t, errors.Is(err, hexary.ErrVerify), "unexpected error or nil : %+v", err)
			proof[l-1][0] = ^proof[l-1][0]
		}

		// test correct proof
		err = builder.Add(i, hashes[i], proof)
		assert.NoError(t, err)
	}
}

func TestProofLen(t *testing.T) {
	proofLen := make([]int, maxHash)
	proofLen[0] = 3
	for i := 16; i < maxHash; i += 16 {
		proofLen[i] = 1
	}
	proofLen[256] = 2
	hashes := make([][]byte, maxHash)

	perm := newMapBucket()
	ac, err := hexary.NewAccumulator(perm, newMapBucket(), "")
	assert.NoError(t, err)
	for i:=int64(0); i < maxHash; i++ {
		bs := codec.MustMarshalToBytes(&i)
		hashes[i] = crypto.SHA3Sum256(bs)
		err = ac.Add(hashes[i])
		assert.NoError(t, err)
	}
	header, err := ac.Finalize()
	assert.NoError(t, err)

	prover, err := hexary.NewMerkleTree(perm, header, cacheMax)
	assert.NoError(t, err)
	for i := int64(0); i < maxHash; i++ {
		proof, err := prover.Prove(i, -1)
		assert.NoError(t, err)
		assert.Equal(t, proofLen[i], len(proof), "at %d", i)
	}
}

func TestRequiredLevel(t *testing.T) {
	assert.Equal(t, 0, hexary.LevelFromLen(0))
	assert.Equal(t, 0, hexary.LevelFromLen(1))
	assert.Equal(t, 1, hexary.LevelFromLen(16))
	assert.Equal(t, 2, hexary.LevelFromLen(17))
	assert.Equal(t, 2, hexary.LevelFromLen(256))
	assert.Equal(t, 3, hexary.LevelFromLen(257))
	assert.Equal(t, 3, hexary.LevelFromLen(4096))
	assert.Equal(t, 4, hexary.LevelFromLen(4097))
}
