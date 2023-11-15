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
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
)

func newAccumulator(t *testing.T, dbase db.Database) hexary.Accumulator {
	if dbase == nil {
		dbase = db.NewMapDB()
	}
	tbk, err := dbase.GetBucket(icdb.BlockMerkle)
	assert.NoError(t, err)
	ibk, err := dbase.GetBucket("i")
	assert.NoError(t, err)
	hac, err := hexary.NewAccumulator(tbk, ibk, "")
	assert.NoError(t, err)
	return hac
}

func TestAccumulator_ZeroValue(t *testing.T) {
	hac := newAccumulator(t, nil)
	hd := hac.GetMerkleHeader()
	assert.EqualValues(t, &hexary.MerkleHeader{}, hd)
	hd, err := hac.Finalize()
	assert.NoError(t, err)
	assert.EqualValues(t, &hexary.MerkleHeader{}, hd)
}

const (
	hashLen = 32
	children = 16
	nodeLen = hashLen * children
)

func merkle(in []byte) []byte {
	var out []byte
	for i, l := 0, len(in); i < l; i += nodeLen {
		end := i + nodeLen
		if end > l {
			end = l
		}
		out = append(out, crypto.SHA3Sum256(in[i:end])...)
	}
	return out
}

func merkleUpTo(n int) []byte {
	if n == 0 {
		return nil
	}
	var bs []byte
	for i:=0; i<n; i++ {
		hash := crypto.SHA3Sum256(codec.MustMarshalToBytes(i))
		bs = append(bs, hash...)
	}
	for len(bs) != hashLen {
		bs = merkle(bs)
	}
	return bs
}

func TestAccumulator_GetMerkleHeader(t *testing.T) {
	hac := newAccumulator(t, nil)
	for i:=0; i<0x102; i++ {
		err := hac.Add(crypto.SHA3Sum256(codec.MustMarshalToBytes(i)))
		assert.NoError(t, err)
		hd := hac.GetMerkleHeader()
		assert.Equal(t, merkleUpTo(i+1), hd.RootHash, "at %d", i)
		assert.EqualValues(t, i+1, hd.Leaves)
	}
}

func TestAccumulator_Persistence(t *testing.T) {
	tdb := db.NewMapDB()
	hac := newAccumulator(t, tdb)
	for i:=0; i<0x102; i++ {
		err := hac.Add(crypto.SHA3Sum256(codec.MustMarshalToBytes(i)))
		assert.NoError(t, err)
		hd := hac.GetMerkleHeader()
		assert.Equal(t, merkleUpTo(i+1), hd.RootHash, "at %d", i)
		assert.EqualValues(t, i+1, hd.Leaves)

		hac2 := newAccumulator(t, tdb)
		hd2 := hac2.GetMerkleHeader()
		assert.Equal(t, hd, hd2)
	}
}

func TestAccumulator_Finalize(t *testing.T) {
	hac := newAccumulator(t, nil)
	for i:=0; i<0x102; i++ {
		err := hac.Add(crypto.SHA3Sum256(codec.MustMarshalToBytes(i)))
		assert.NoError(t, err)
		hd := hac.GetMerkleHeader()
		assert.Equal(t, merkleUpTo(i+1), hd.RootHash, "at %d", i)
		assert.EqualValues(t, i+1, hd.Leaves)
		hd2, err := hac.Finalize()
		assert.NoError(t, err)
		assert.Equal(t, hd, hd2)
		hd3 := hac.GetMerkleHeader()
		assert.Equal(t, hd, hd3)
	}
}

func accumulateUpTo(t *testing.T, hac hexary.Accumulator, l int64) {
	for i := hac.Len(); i < l; i++ {
		err := hac.Add(crypto.SHA3Sum256(codec.MustMarshalToBytes(i)))
		assert.NoError(t, err)
	}
	assert.EqualValues(t, l, hac.Len())
}

func TestAccumulator_SetLen(t *testing.T) {
	const max = 0x102
	headerForLen := make([]*hexary.MerkleHeader, max+1)
	hac := newAccumulator(t, nil)
	headerForLen[0] = hac.GetMerkleHeader()
	for i:=0; i<max; i++ {
		err := hac.Add(crypto.SHA3Sum256(codec.MustMarshalToBytes(i)))
		assert.NoError(t, err)
		headerForLen[i+1] = hac.GetMerkleHeader()
	}

	tdb := db.NewMapDB()
	for i:=int64(0); i<max; i++ {
		hac := newAccumulator(t, tdb)
		err := hac.SetLen(0)
		assert.NoError(t, err)
		accumulateUpTo(t, hac, i)
		for j:=i; j>=0; j-- {
			err := hac.SetLen(j)
			assert.NoError(t, err, "at i=%d j=%d", i, j)
			hd := hac.GetMerkleHeader()
			assert.EqualValues(t, headerForLen[j], hd, "at i=%d j=%d", i, j)
			assert.Equal(t, j, hd.Leaves)
		}
	}
}

func testSetLen(t *testing.T, old, new int64) {
	hac := newAccumulator(t, nil)
	accumulateUpTo(t, hac, new)
	hd := hac.GetMerkleHeader()
	accumulateUpTo(t, hac, old)
	err := hac.SetLen(new)
	assert.NoError(t, err)
	assert.Equal(t, hd, hac.GetMerkleHeader())
}

func TestAccumulator_SetLen2(t *testing.T) {
	testSetLen(t, 19459, 16980)
	testSetLen(t, 16, 16)
	testSetLen(t, 1000, 255)
}

func TestAccumulator_SetLen3(t *testing.T) {
	/*
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	const from = 1024
	for step:=1; step <= from; step++ {
		for i:=from; i>=0; i -= step {
			testSetLen(t, from, int64(i))
		}
	}
	 */
}
