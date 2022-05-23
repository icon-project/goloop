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

package main

import (
	"time"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type commitVoteSet struct {
	TimestampValue int64
	hash           []byte
}

func (c *commitVoteSet) VerifyBlock(block module.BlockData, validators module.ValidatorList) ([]bool, error) {
	return nil, nil
}

func (c *commitVoteSet) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(c)
}

func (c *commitVoteSet) Hash() []byte {
	if c.hash == nil {
		c.hash = crypto.SHA3Sum256(c.Bytes())
	}
	return c.hash
}

func (c *commitVoteSet) Timestamp() int64 {
	return c.TimestampValue
}

func (c *commitVoteSet) VoteRound() int32 {
	return 0
}

func (c *commitVoteSet) BlockVoteSetBytes() []byte {
	return c.Bytes()
}

func (c *commitVoteSet) NTSDProofCount() int {
	return 0
}

func (c *commitVoteSet) NTSDProofAt(i int) []byte {
	return nil
}

func NewCommitVotes() *commitVoteSet {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	return &commitVoteSet{
		TimestampValue: now,
	}
}

func DecodeCommitVotes(bs []byte) module.CommitVoteSet {
	vs := new(commitVoteSet)
	if len(bs) > 0 {
		codec.BC.MustUnmarshalFromBytes(bs, vs)
	}
	return vs
}
