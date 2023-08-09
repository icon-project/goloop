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
	"encoding/json"
	"strconv"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
)

type LeaderVoteSharable struct {
	BlockHeight common.HexInt64  `json:"blockHeight"`
	OldLeader   common.Address   `json:"oldLeader"`
	NewLeader   common.Address   `json:"newLeader"`
	Round_      *int             `json:"round_,omitempty"`
	Round       *common.HexInt64 `json:"round,omitempty"`
}

func (s *LeaderVoteSharable) Equal(s2 *LeaderVoteSharable) bool {
	return s.BlockHeight == s2.BlockHeight &&
		intPtrEqual(s.Round_, s2.Round_) &&
		hexInt64PtrEqual(s.Round, s2.Round) &&
		common.AddressEqual(&s.OldLeader, &s2.OldLeader) &&
		common.AddressEqual(&s.NewLeader, &s2.NewLeader)
}

type LeaderVoteIndividual struct {
	Rep         common.Address  `json:"rep"`
	Timestamp   common.HexInt64 `json:"timestamp"`
	Signature   []byte          `json:"signature"`
}

type LeaderVote struct {
	LeaderVoteSharable
	LeaderVoteIndividual
	hash []byte
}

func (v *LeaderVote) calcHash() {
	hash := sha3.New256()

	hash.Write([]byte("icx_vote"))

	hash.Write([]byte(".blockHeight."))
	hash.Write([]byte(v.BlockHeight.String()))

	hash.Write([]byte(".newLeader."))
	hash.Write([]byte(v.NewLeader.String()))

	hash.Write([]byte(".oldLeader."))
	hash.Write([]byte(v.OldLeader.String()))

	hash.Write([]byte(".rep."))
	hash.Write([]byte(v.Rep.String()))

	if v.Round != nil {
		hash.Write([]byte(".round."))
		hash.Write([]byte(v.Round.String()))
	}

	if v.Round_ != nil {
		hash.Write([]byte(".round_."))
		hash.Write([]byte(strconv.Itoa(*v.Round_)))
	}

	hash.Write([]byte(".timestamp."))
	hash.Write([]byte(v.Timestamp.String()))

	v.hash = hash.Sum(nil)
}

func (v *LeaderVote) GetRound() int {
	if v.Round_ != nil {
		return *v.Round_
	}
	if v.Round != nil {
		return int(v.Round.Value)
	}
	return -1
}

func (v *LeaderVote) Hash() []byte {
	if v.hash == nil {
		v.calcHash()
	}
	return v.hash
}

func (v *LeaderVote) Verify() error {
	if v == nil {
		return nil
	}
	hash := v.Hash()
	sig, err := crypto.ParseSignature(v.Signature)
	if err != nil {
		return err
	}
	pk, err := sig.RecoverPublicKey(hash)
	if err != nil {
		return err
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if !addr.Equal(&v.Rep) {
		return errors.InvalidStateError.Errorf("SignatureInvalid(exp=%s,calc=%s)",
			&v.Rep, addr)
	}
	return nil
}

type LeaderVoteList struct {
	votes []*LeaderVote
	root  []byte
	hash  []byte
	bytes []byte
}

func (s *LeaderVoteList) Copy() *LeaderVoteList {
	if s == nil {
		return nil
	}
	jsn, err := s.MarshalJSON()
	log.Must(err)
	var res LeaderVoteList
	err = res.UnmarshalJSON(jsn)
	log.Must(err)
	return &res
}

func (s *LeaderVoteList) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.votes)
}

func (s *LeaderVoteList) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.votes)
}

func (s *LeaderVoteList) Root() []byte {
	if s == nil {
		return nil
	}
	if s.root == nil {
		s.calcRoot()
	}
	return s.root
}

func (s *LeaderVoteList) calcRoot() {
	items := make([]merkle.Item, len(s.votes))
	for i, v := range s.votes {
		if v != nil {
			items[i] = v
		}
	}
	s.root = merkle.CalcHashOfList(items)
}

func (s *LeaderVoteList) isVotedOverHalf(leader module.Address) bool {
	quorum := len(s.votes) / 2
	return s.checkVoted(quorum, leader)
}

func (s *LeaderVoteList) isVotedOverTwoThirds(leader module.Address) bool {
	quorum := len(s.votes) * 2 / 3
	return s.checkVoted(quorum, leader)
}

func (s *LeaderVoteList) checkVoted(quorum int, leader module.Address) bool {
	if len(s.votes) == 0 {
		return false
	}
	var vEmpty int
	var vCount int
	voted := make(map[string]int)
	for _, vote := range s.votes {
		if vote == nil {
			continue
		}
		if vote.NewLeader.Equal(emtpyAddress) {
			vEmpty += 1
		} else if vote.NewLeader.Equal(leader) {
			vCount += 1
		} else {
			voted[vote.NewLeader.String()] += 1
		}
	}
	if vEmpty + vCount <= quorum {
		return false
	}
	for _, cnt := range voted {
		if cnt > vCount {
			return false
		}
	}
	return true
}

func (s *LeaderVoteList) Verify(reps *RepsList) error {
	if s == nil || len(s.votes) == 0 {
		return nil
	}
	round := -1
	for i, v := range s.votes {
		if v == nil {
			continue
		}
		if err := v.Verify(); err != nil {
			return err
		}
		rep := reps.Get(i)
		if !v.Rep.Equal(rep) {
			return errors.InvalidStateError.Errorf(
				"InvalidVote(idx=%d,exp=%s,real=%s)",
				i,
				rep.String(),
				v.Rep.String(),
			)
		}
		if round == -1 {
			round = v.GetRound()
		} else if round != v.GetRound() {
			return errors.InvalidStateError.Errorf(
				"InvalidVoteRound(idx=%d,exp=%d,real=%d)",
				i,
				round,
				v.GetRound(),
			)
		}
	}
	return nil
}

type compactLeaderVoteList struct {
	Sharable []LeaderVoteSharable
	Entries []*compactLeaderVoteEntry
}

type compactLeaderVoteEntry struct {
	SharableIndex int16
	LeaderVoteIndividual
}

func (s *LeaderVoteList) compactFormat() *compactLeaderVoteList {
	var sharable []LeaderVoteSharable
	entries := make([]*compactLeaderVoteEntry, len(s.votes))
	for i, v := range s.votes {
		if v==nil {
			entries[i] = nil
		} else {
			index := len(sharable)
			for j, _ := range sharable {
				if sharable[j].Equal(&v.LeaderVoteSharable) {
					index = j
					break
				}
			}
			if index==len(sharable) {
				sharable = append(sharable, v.LeaderVoteSharable)
			}
			entries[i] = &compactLeaderVoteEntry{
				int16(index),
				v.LeaderVoteIndividual,
			}
		}
	}
	return &compactLeaderVoteList{sharable, entries}
}

func (s *LeaderVoteList) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(s.compactFormat())
}

func (s *LeaderVoteList) RLPDecodeSelf(d codec.Decoder) error {
	var cbvl compactLeaderVoteList
	err := d.Decode(&cbvl)
	if err != nil {
		return err
	}
	for i, sh := range cbvl.Sharable {
		if sh.Round == nil && sh.Round_ == nil {
			return errors.Errorf("LeaderVote with no round height=%d sharable index=%d", sh.BlockHeight, i)
		}
	}
	s.votes = make([]*LeaderVote, len(cbvl.Entries))
	for i, e := range cbvl.Entries {
		if e==nil {
			s.votes[i] = nil
		} else {
			if e.SharableIndex < 0 || int(e.SharableIndex) >= len(cbvl.Sharable) {
				return errors.Errorf("invalid sharable index len(Sharable)=%d index=%d", len(cbvl.Sharable), e.SharableIndex)
			}
			s.votes[i] = &LeaderVote{
				cbvl.Sharable[e.SharableIndex],
				e.LeaderVoteIndividual,
				nil,
			}
		}
	}
	s.root = nil
	return nil
}

func (s* LeaderVoteList) Hash() []byte {
	if s == nil {
		return nil
	}
	if s.hash == nil {
		s.hash = crypto.SHA3Sum256(s.Bytes())
	}
	return s.hash
}

func (s *LeaderVoteList) Bytes() []byte {
	if s.bytes == nil {
		s.bytes = codec.BC.MustMarshalToBytes(s)
	}
	return s.bytes
}
