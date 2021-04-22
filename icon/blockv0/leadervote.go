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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
)

type LeaderVote struct {
	Rep         common.Address  `json:"rep"`
	Timestamp   common.HexInt64 `json:"timestamp"`
	BlockHeight common.HexInt64 `json:"blockHeight"`
	OldLeader   common.Address  `json:"oldLeader"`
	NewLeader   common.Address  `json:"newLeader"`
	Round       int             `json:"round_"`
	Signature   []byte          `json:"signature"`

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

	hash.Write([]byte(".round_."))
	hash.Write([]byte(strconv.Itoa(v.Round)))

	hash.Write([]byte(".timestamp."))
	hash.Write([]byte(v.Timestamp.String()))

	v.hash = hash.Sum(nil)
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
}

func (s *LeaderVoteList) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.votes)
}

func (s *LeaderVoteList) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.votes)
}

func (s *LeaderVoteList) Root() []byte {
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

func (s *LeaderVoteList) Quorum() module.Address {
	if len(s.votes) == 0 {
		return nil
	}
	votes := make(map[string]int)
	quorum := len(s.votes) * 2 / 3
	for _, vote := range s.votes {
		if vote == nil {
			continue
		}
		cnt, _ := votes[vote.NewLeader.String()]
		cnt += 1
		if cnt > quorum {
			return &vote.NewLeader
		}
		votes[vote.NewLeader.String()] = cnt
	}
	return nil
}

func (s *LeaderVoteList) Verify(reps *RepsList) error {
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
		rep := reps.Get(i)
		if !v.Rep.Equal(rep) {
			return errors.InvalidStateError.Errorf(
				"InvalidVote(idx=%d,exp=%s,real=%s)",
				i,
				rep.String(),
				v.Rep.String(),
			)
		}
	}
	// TODO check votes for the next
	return nil
}
