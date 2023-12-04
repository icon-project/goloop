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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const (
	defaultAccumulatorKey = "accumulator"
)

type Accumulator interface {
	Add(hash []byte) error

	// Len returns number of added hashes.
	Len() int64

	// SetLen sets length and rewinds an accumulator. Error if l is larger
	// than Len() of the accumulator.
	SetLen(l int64) error

	GetMerkleHeader() *MerkleHeader

	// Finalize finalizes node data
	Finalize() (header *MerkleHeader, err error)
}

type accumulatorData struct {
	Len   int64
	Roots []*node
}

type accumulator struct {
	data               accumulatorData
	treeBucket         db.Bucket
	accumulatorBucket  *db.CodedBucket
	accumulatorDataKey []byte
}

func (ba *accumulator) GetMerkleHeader() *MerkleHeader {
	var carry []byte
	for i, r := range ba.data.Roots {
		var restore bool
		if carry != nil {
			r.Add(carry)
			restore = true
		}
		if i == len(ba.data.Roots)-1 && r.Len() == 1 {
			carry = r.GetCopy(0)
		} else {
			carry = r.Hash()
		}
		if restore {
			r.RemoveBack()
		}
	}
	return &MerkleHeader{carry, ba.data.Len }
}

func powerOf16(n uint64) bool {
	for n > 0xf {
		if n&0xf != 0 {
			return false
		}
		n = n >> 4
	}
	return n == 1
}

func (ba *accumulator) SetLen(l int64) error {
	if l > ba.data.Len {
		return errors.IllegalArgumentError.Errorf(
			"l(=%d) > Len(=%d)", l, ba.data.Len,
		)
	}
	if l == 0 {
		ba.data = accumulatorData{ 0, nil }
		return nil
	}
	if l == ba.data.Len {
		return nil
	}
	hd, err := ba.Finalize()
	if err != nil {
		return err
	}
	mt, err := NewMerkleTree(ba.treeBucket, hd, 0)
	if err != nil {
		return err
	}
	proof, err := mt.Prove(l-1, 0)
	if err != nil {
		return err
	}
	lvl := LevelFromLen(l)
	if powerOf16(uint64(l)) {
		lvl++
	}
	if len(proof) < lvl {
		if len(proof) + 1 != lvl {
			log.Panicf("invalid proof length %d for SetLen(%d)", len(proof), l)
		}
		tmp := append([][]byte(nil), hd.RootHash)
		proof = append(tmp, proof...)
	} else {
		over := len(proof) - lvl
		proof = proof[over:]
	}
	roots := make([]*node, lvl)
	d := l
	for i := range roots {
		copied := append([]byte(nil), proof[len(proof)-1-i]...)
		roots[i], err = newNodeFromBytes(copied)
		if err != nil {
			return err
		}
		roots[i].SetLen(int(d%16))
		d = d/16
	}
	ba.data = accumulatorData{ l, roots }
	return ba.accumulatorBucket.Set(db.Raw(ba.accumulatorDataKey), &ba.data)
}

func (ba *accumulator) add(i int, hash []byte) error {
	if i >= len(ba.data.Roots) {
		ba.data.Roots = append(ba.data.Roots, newNode())
	}
	rb := ba.data.Roots[i]
	rb.Add(hash)
	if rb.Full() {
		if err := ba.treeBucket.Set(rb.Hash(), rb.Bytes()); err != nil {
			return err
		}
		hash := rb.Hash()
		rb.Clear()
		if err := ba.add(i+1, hash); err != nil {
			return err
		}
	}
	return nil
}

func (ba *accumulator) Add(hash []byte) error {
	if err := ba.add(0, hash); err != nil {
		return err
	}
	ba.data.Len++
	return ba.accumulatorBucket.Set(db.Raw(ba.accumulatorDataKey), &ba.data)
}

func (ba *accumulator) Len() int64 {
	return ba.data.Len
}

func (ba *accumulator) Finalize() (header *MerkleHeader, err error) {
	var carry []byte
	for i, r := range ba.data.Roots {
		var restore bool
		if carry != nil {
			r.Add(carry)
			restore = true
		}
		if i == len(ba.data.Roots)-1 && r.Len() == 1 {
			carry = r.GetCopy(0)
		} else {
			hash := r.Hash()
			if hash != nil {
				if err = ba.treeBucket.Set(hash, r.Bytes()); err != nil {
					return nil, err
				}
			}
			carry = hash
		}
		if restore {
			r.RemoveBack()
		}
	}
	return &MerkleHeader{ carry, ba.data.Len }, nil
}

// NewAccumulator creates a new accumulator. Merkle node is written in tree
// bucket, accumulator is written on accumulator data key in accumulator bucket.
func NewAccumulator(
	treeBucket db.Bucket,
	accumulatorBucket db.Bucket,
	accumulatorDataKey string,
) (Accumulator, error) {
	if len(accumulatorDataKey) == 0 {
		accumulatorDataKey = defaultAccumulatorKey
	}
	ba := &accumulator{
		treeBucket:         treeBucket,
		accumulatorBucket:  db.NewCodedBucketFromBucket(accumulatorBucket, nil, nil),
		accumulatorDataKey: []byte(accumulatorDataKey),
	}
	err := ba.accumulatorBucket.Get(db.Raw(accumulatorDataKey), &ba.data)
	if err != nil && !errors.NotFoundError.Equals(err) {
		return nil, err
	}
	return ba, nil
}
