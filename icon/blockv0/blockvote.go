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

package blockv0

import (
	"bytes"
	"encoding/json"
	"strconv"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/merkle"
)

type BlockVoteJSONSharable struct {
	BlockHeight common.HexInt64  `json:"blockHeight"`
	Round_      *int             `json:"round_,omitempty"`
	Round       *common.HexInt64 `json:"round,omitempty"`
	BlockHash   common.HexHash   `json:"blockHash"`
}

func (s *BlockVoteJSONSharable) Equal(s2 *BlockVoteJSONSharable) bool {
	return s.BlockHeight == s2.BlockHeight &&
		(s.Round_ == s2.Round_ || (s.Round_ != nil && s2.Round_ != nil && *s.Round_ == *s2.Round_)) &&
		(s.Round == s2.Round || (s.Round != nil && s2.Round != nil && *s.Round == *s2.Round)) &&
		bytes.Equal(s.BlockHash, s2.BlockHash)
}

type BlockVoteJSONIndividual struct {
	Rep       common.Address   `json:"rep"`
	Timestamp common.HexInt64  `json:"timestamp"`
	Signature common.Signature `json:"signature"`
}

type BlockVoteJSON struct {
	BlockVoteJSONSharable
	BlockVoteJSONIndividual
}

type BlockVote struct {
	json BlockVoteJSON
	hash []byte
}

func (v *BlockVote) calcHash() {
	hash := sha3.New256()

	hash.Write([]byte("icx_vote"))

	hash.Write([]byte(".blockHash."))
	hash.Write([]byte(v.json.BlockHash.String()))

	hash.Write([]byte(".blockHeight."))
	hash.Write([]byte(v.json.BlockHeight.String()))

	hash.Write([]byte(".rep."))
	hash.Write([]byte(v.json.Rep.String()))

	if v.json.Round != nil {
		hash.Write([]byte(".round."))
		hash.Write([]byte(v.json.Round.String()))
	}

	if v.json.Round_ != nil {
		hash.Write([]byte(".round_."))
		hash.Write([]byte(strconv.Itoa(*v.json.Round_)))
	}

	hash.Write([]byte(".timestamp."))
	hash.Write([]byte(v.json.Timestamp.String()))

	v.hash = hash.Sum(nil)
}

func (v *BlockVote) Hash() []byte {
	if v.hash == nil {
		v.calcHash()
	}
	return v.hash
}

func (v *BlockVote) Round() int {
	if v.json.Round_ != nil {
		return *v.json.Round_
	} else {
		return int(v.json.Round.Value)
	}
}

func (v *BlockVote) Verify() error {
	if v == nil {
		return nil
	}
	hash := v.Hash()
	pk, err := v.json.Signature.RecoverPublicKey(hash)
	if err != nil {
		return errors.WithStack(err)
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if !addr.Equal(&v.json.Rep) {
		return errors.InvalidStateError.Errorf("SignatureInvalid(exp=%s,calc=%s)",
			&v.json.Rep, addr)
	}
	return nil
}

func (v *BlockVote) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &v.json)
}

type BlockVoteList struct {
	votes []*BlockVote
	root  []byte
}

func (s *BlockVoteList) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.votes)
}

func (s *BlockVoteList) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.votes)
}

func (s *BlockVoteList) Root() []byte {
	if s.root == nil {
		s.calcRoot()
	}
	return s.root
}

func (s *BlockVoteList) calcRoot() {
	items := make([]merkle.Item, len(s.votes))
	for i, v := range s.votes {
		if v != nil {
			items[i] = v
		}
	}
	s.root = merkle.CalcHashOfList(items)
}

func (s *BlockVoteList) Verify(reps *RepsList) error {
	if s == nil || len(s.votes) == 0 {
		return nil
	}
	for i, v := range s.votes {
		if v == nil {
			continue
		}
		if err := v.Verify(); err != nil {
			return err
		}
		if reps != nil {
			rep := reps.Get(i)
			if !v.json.Rep.Equal(rep) {
				return errors.InvalidStateError.Errorf(
					"InvalidVote(idx=%d,exp=%s,real=%s)",
					i,
					rep.String(),
					v.json.Rep.String(),
				)
			}
		}
	}
	return nil
}

func (s *BlockVoteList) Quorum() []byte {
	n := len(s.votes)
	q := 2 * n / 3
	counter := make(map[string]int)
	for _, v := range s.votes {
		if v == nil {
			continue
		}
		id := v.json.BlockHash.String()
		if cnt, ok := counter[id]; ok {
			counter[id] = cnt + 1
			if cnt+1 > q {
				return v.json.BlockHash
			}
		} else {
			counter[id] = 1
			if 1 > q {
				return v.json.BlockHash
			}
		}
	}
	return nil
}

func (s *BlockVoteList) CheckVoters(reps *RepsList) error {
	if s == nil || len(s.votes) == 0 {
		return nil
	}
	count := reps.Size()
	for i := 0; i < count; i++ {
		vote := s.votes[i]
		if vote != nil {
			if !vote.json.Rep.Equal(reps.Get(i)) {
				return errors.InvalidStateError.Errorf(
					"VoterMismatch(exp=%s,real=%s)",
					reps.Get(i).String(),
					vote.json.Rep.String(),
				)
			}
		}
	}
	return nil
}

type compactBlockVoteList struct {
	Sharable []BlockVoteJSONSharable
	Entries  []*compactBlockVoteEntry
}

type compactBlockVoteEntry struct {
	SharableIndex int16
	BlockVoteJSONIndividual
}

func (s *BlockVoteList) compactFormat() *compactBlockVoteList {
	var sharable []BlockVoteJSONSharable
	entries := make([]*compactBlockVoteEntry, len(s.votes))
	for i, v := range s.votes {
		if v==nil {
			entries[i] = nil
		} else {
			index := len(sharable)
			for j, _ := range sharable {
				if sharable[j].Equal(&v.json.BlockVoteJSONSharable) {
					index = j
					break
				}
			}
			if index==len(sharable) {
				sharable = append(sharable, v.json.BlockVoteJSONSharable)
			}
			entries[i] = &compactBlockVoteEntry{
				int16(index),
				v.json.BlockVoteJSONIndividual,
			}
		}
	}
	return &compactBlockVoteList{sharable, entries}
}

func (s *BlockVoteList) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(s.compactFormat())
}

func (s *BlockVoteList) RLPDecodeSelf(d codec.Decoder) error {
	var cbvl compactBlockVoteList
	err := d.Decode(&cbvl)
	if err != nil {
		return err
	}
	s.votes = make([]*BlockVote, len(cbvl.Entries))
	for i, e := range cbvl.Entries {
		if e==nil {
			s.votes[i] = nil
		} else {
			s.votes[i] = &BlockVote{
				BlockVoteJSON{
					cbvl.Sharable[e.SharableIndex],
					e.BlockVoteJSONIndividual,
				},
				nil,
			}
		}
	}
	s.root = nil
	return nil
}

func (s* BlockVoteList) Hash() []byte {
	bs := codec.BC.MustMarshalToBytes(s)
	return crypto.SHA3Sum256(bs)
}
