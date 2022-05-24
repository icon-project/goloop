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

package block

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
)

type ntsdProofList struct {
	dbase          db.Database
	ntsdProves     [][]byte
	_hashListBytes []byte
	_hashListHash  []byte
}

type ntsdProofHashListFormat struct {
	NtsdProofHashes [][]byte
}

func newNTSDProofList(
	dbase db.Database,
	ntsdProves [][]byte,
) *ntsdProofList {
	return &ntsdProofList{
		dbase:      dbase,
		ntsdProves: ntsdProves,
	}
}

func (pl *ntsdProofList) Len() int {
	return len(pl.ntsdProves)
}

func (pl *ntsdProofList) ProofAt(i int) ([]byte, error) {
	return pl.ntsdProves[i], nil
}

func (pl *ntsdProofList) Proves() ([][]byte, error) {
	return pl.ntsdProves, nil
}

func (pl *ntsdProofList) hashListBytes() []byte {
	if pl._hashListBytes == nil {
		if len(pl.ntsdProves) == 0 {
			pl._hashListBytes = make([]byte, 0)
		} else {
			format := ntsdProofHashListFormat{
				make([][]byte, 0, len(pl.ntsdProves)),
			}
			for _, proof := range pl.ntsdProves {
				format.NtsdProofHashes = append(format.NtsdProofHashes, crypto.SHA3Sum256(proof))
			}
			pl._hashListBytes = codec.MustMarshalToBytes(&format)
		}
	}
	if len(pl._hashListHash) == 0 {
		return nil
	}
	return pl._hashListBytes
}

func (pl *ntsdProofList) HashListHash() []byte {
	if pl._hashListHash == nil {
		if len(pl.hashListBytes()) == 0 {
			pl._hashListHash = make([]byte, 0)
		} else {
			pl._hashListHash = crypto.SHA3Sum256(pl.hashListBytes())
		}
	}
	if len(pl._hashListHash) == 0 {
		return nil
	}
	return pl._hashListHash
}

func (pl *ntsdProofList) Flush() error {
	cbk, err := db.NewCodedBucket(pl.dbase, db.BytesByHash, nil)
	if err != nil {
		return err
	}
	if pl.hashListBytes() == nil {
		return nil
	}
	err = cbk.Put(db.Raw(pl.hashListBytes()))
	if err != nil {
		return err
	}
	bk, err := pl.dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return err
	}
	for _, proof := range pl.ntsdProves {
		err = bk.Set(crypto.SHA3Sum256(proof), proof)
		if err != nil {
			return err
		}
	}
	return nil
}

type ntsdProofHashList struct {
	dbase            db.Database
	ntsdProofHashes  [][]byte
	hasAllNTSDProves bool
	ntsdProves       [][]byte
	hashListBytes    []byte
	_hashListHash    []byte
}

func newNTSDProofHashListFromHash(
	dbase db.Database,
	hash []byte,
) (*ntsdProofHashList, error) {
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	if hash == nil {
		return &ntsdProofHashList{
			dbase:            dbase,
			ntsdProofHashes:  nil,
			hasAllNTSDProves: true,
			ntsdProves:       nil,
			hashListBytes:    nil,
		}, nil
	}
	hashListBytes, err := bk.Get(hash)
	if err != nil {
		return nil, err
	}
	format := ntsdProofHashListFormat{}
	_, err = codec.UnmarshalFromBytes(hashListBytes, &format)
	if err != nil {
		return nil, err
	}
	return &ntsdProofHashList{
		dbase:           dbase,
		ntsdProofHashes: format.NtsdProofHashes,
		ntsdProves:      make([][]byte, len(format.NtsdProofHashes)),
		hashListBytes:   hashListBytes,
	}, nil
}

func (phl *ntsdProofHashList) Len() int {
	return len(phl.ntsdProofHashes)
}

func (phl *ntsdProofHashList) ProofAt(i int) ([]byte, error) {
	if phl.ntsdProves[i] == nil {
		bk, err := phl.dbase.GetBucket(db.BytesByHash)
		if err != nil {
			return nil, err
		}
		phl.ntsdProves[i], err = bk.Get(phl.ntsdProofHashes[i])
		if err != nil {
			return nil, err
		}
	}
	return phl.ntsdProves[i], nil
}

func (phl *ntsdProofHashList) Proves() ([][]byte, error) {
	if !phl.hasAllNTSDProves {
		for i := 0; i < len(phl.ntsdProves); i++ {
			_, err := phl.ProofAt(i)
			if err != nil {
				return nil, err
			}
		}
		phl.hasAllNTSDProves = true
	}
	return phl.ntsdProves, nil
}

func (phl *ntsdProofHashList) HashListHash() []byte {
	if phl.hashListBytes == nil {
		return nil
	}
	if phl._hashListHash == nil {
		phl._hashListHash = crypto.SHA3Sum256(phl.hashListBytes)
	}
	return phl._hashListHash
}

func (phl *ntsdProofHashList) Flush() error {
	return nil
}
