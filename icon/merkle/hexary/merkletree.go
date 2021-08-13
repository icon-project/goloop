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

package hexary

import (
	"bytes"
	"math/bits"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

var ErrVerify = errors.NewBase(errors.IllegalArgumentError, "VerifyError")

type Prover interface {
	// Prove returns proof for a key.
	// if from >= 0, first `from` elements are omitted.
	// if from < 0, only difference between proof for key and proof for key-1.
	// for example, if full proof for 0x00FF is [A, B, C, D] and full proof for
	// 0x0100 is [A, B, E, F], Prove(0x0100, -1) returns ([E, F], nil).
	Prove(key int64, from int) ([][]byte, error)
}

type MerkleTree interface {
	Prover
	// Add verifies proof for given key and adds the proof in this accumulator.
	// Proof can be partial. If full proof has common prefix with proof of preceding
	// key (= key-1), the common prefix branches can be omitted.
	// Returns (hash, nil) if proof is correct,
	// (nil, wrapped ErrVerify) if proof is incorrect and
	// (nil, other error) if correctness cannot be checked due to some error.
	Add(key int64, hash []byte, proof [][]byte) (err error)
	Cap() int64
}

type merkleTree struct {
	bdb      *nodeDB
	level    int
	rootHash *node
	cap      int64
}

func LevelFromLen(len int64) int {
	if len == 0 {
		return 0
	}
	return (bits.Len64(uint64(len)-1) + 3) / 4
}

// NewMerkleTree creates a new hex-ary merkle tree.
// cacheCap is max number of branches in cache. Default value is used
// if -1.
func NewMerkleTree(
	bk db.Bucket,
	header *MerkleHeader,
	cacheCap int,
) (MerkleTree, error) {
	br, err := newNodeFromBytes(header.RootHash)
	if err != nil {
		return nil, err
	}
	return &merkleTree{
		bdb:      newCachedNodeDB(bk, cacheCap),
		level:    LevelFromLen(header.Leaves),
		rootHash: br,
		cap:      header.Leaves,
	}, nil
}

func (sa *merkleTree) Cap() int64 {
	return sa.cap
}

func (sa *merkleTree) minProofLenForKey(key int64) int {
	minLen := ((bits.TrailingZeros64(^uint64(key^(key-1))) + 3) / 4) - 1
	if minLen > sa.level {
		minLen = sa.level
	}
	return minLen
}

func (sa *merkleTree) Prove(key int64, from int) (proof [][]byte, err error) {
	res := make([][]byte, sa.level)
	br := sa.rootHash
	for i := 0; i < sa.level; i++ {
		k := (key >> ((sa.level - i) * 4)) & 0xf
		hash := br.Get(int(k))
		br, err = sa.bdb.Get(hash)
		if err != nil {
			return nil, err
		}
		res[i] = br.Bytes()
	}
	if from < 0 {
		from = sa.level - sa.minProofLenForKey(key)
		if from < 0 {
			from = 0
		}
	}
	return res[from:], nil
}

func (sa *merkleTree) Add(key int64, hash []byte, proof [][]byte) error {
	minLen := sa.minProofLenForKey(key)
	if len(proof) < minLen {
		return errors.Wrapf(
			ErrVerify, "too short proof (height=%d len=%d", key, len(proof),
		)
	}
	br := sa.rootHash
	omit := sa.level - len(proof)
	proofBr := make([]*node, len(proof))
	for i := 0; i < sa.level; i++ {
		k := (key >> ((sa.level - i) * 4)) & 0xf
		curHash := br.Get(int(k))
		var err error
		if i < omit {
			if br, err = sa.bdb.Get(curHash); err != nil {
				return err
			}
		} else {
			if br, err = newNodeFromBytes(proof[i-omit]); err != nil {
				return err
			}
			proofBr[i-omit] = br
			if !bytes.Equal(br.Hash(), curHash) {
				return errors.Wrapf(
					ErrVerify, "bad node hash index=%d", i-omit,
				)
			}
		}
	}
	if !bytes.Equal(br.Get(int(key&0xf)), hash) {
		return errors.Wrapf(ErrVerify, "bad final hash")
	}
	for _, p := range proofBr {
		if err := sa.bdb.Put(p); err != nil {
			return err
		}
	}
	return nil
}
