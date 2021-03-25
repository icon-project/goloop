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

package blockv1

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const blockV1String = "1.0"

type headerFormat struct {
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte // v0 proposer (PeerID)
	PrevHash               []byte
	VotesHash              []byte // v0 block vote hash
	NextValidatorsHash     []byte // hash of v0 validators
	PatchTransactionsHash  []byte
	NormalTransactionsHash []byte
	LogsBloom              []byte
	Result                 []byte

	OriginalVersion   string
	PrevID            []byte
	Signature         common.Signature
	OriginalStateHash []byte
	RepsHash          []byte
	LeaderVotesHash   []byte
	NextLeader        []byte

	// MerkleTreeRootHash (v0.1)
	// BlockHash (v0.1)
	// ReceiptsHash (v0.3)
	// NextRepsHash (v0.3)
}

type bodyFormat struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	Votes              []byte
	LeaderVotes        []byte
}

type format struct {
	headerFormat
	bodyFormat
}

type Block struct {
	height             int64
	timestamp          int64
	proposer           module.Address
	prevHash           []byte
	nextValidatorsHash []byte
	logsBloom          module.LogsBloom
	result             []byte
	prevID             []byte

	signature   common.Signature
	v0StateHash []byte
	repsHash    []byte
	nextLeader  module.Address

	_nextValidators    module.ValidatorList
	patchTransactions  module.TransactionList
	normalTransactions module.TransactionList
	votes              module.CommitVoteSet
	_id                []byte
	leaderVotes        module.CommitVoteSet
}

func (b *Block) V0MerkleTreeRootHash() []byte {
	return nil
}

func (b *Block) V0BlockHash() []byte {
	return nil
}

func (b *Block) V0ReceiptsHash() []byte {
	return nil
}

func (b *Block) V0NextLeader() module.Address {
	return nil
}

func (b *Block) Version() int {
	return module.BlockVersion1
}

func (b *Block) ID() []byte {
	if b._id == nil {
		bs := codec.BC.MustMarshalToBytes(b._headerFormat())
		b._id = crypto.SHA3Sum256(bs)
	}
	return b._id
}

func (b *Block) Height() int64 {
	return b.height
}

func (b *Block) PrevID() []byte {
	return b.prevID
}

func (b *Block) Votes() module.CommitVoteSet {
	return b.votes
}

func (b *Block) NextValidatorsHash() []byte {
	return b.nextValidatorsHash
}

func (b *Block) NextValidators() module.ValidatorList {
	return b._nextValidators
}

func (b *Block) NormalTransactions() module.TransactionList {
	return b.normalTransactions
}

func (b *Block) PatchTransactions() module.TransactionList {
	return b.patchTransactions
}

func (b *Block) Timestamp() int64 {
	return b.timestamp
}

func (b *Block) Proposer() module.Address {
	return b.proposer
}

func (b *Block) LogsBloom() module.LogsBloom {
	return b.logsBloom
}

func (b *Block) Result() []byte {
	return b.result
}

func (b *Block) MarshalHeader(w io.Writer) error {
	return codec.BC.Marshal(w, b._headerFormat())
}

func (b *Block) MarshalBody(w io.Writer) error {
	bf, err := b._bodyFormat()
	if err != nil {
		return err
	}
	return codec.BC.Marshal(w, bf)
}

func (b *Block) Marshal(w io.Writer) error {
	if err := b.MarshalHeader(w); err != nil {
		return err
	}
	return b.MarshalBody(w)
}

func (b *Block) _headerFormat() *headerFormat {
	var proposerBS []byte
	if b.proposer != nil {
		proposerBS = b.proposer.Bytes()
	}
	return &headerFormat{
		Version:                b.Version(),
		Height:                 b.height,
		Timestamp:              b.timestamp,
		Proposer:               proposerBS,
		PrevHash:               b.prevHash,
		VotesHash:              b.votes.Hash(),
		NextValidatorsHash:     b.nextValidatorsHash,
		PatchTransactionsHash:  b.patchTransactions.Hash(),
		NormalTransactionsHash: b.normalTransactions.Hash(),
		LogsBloom:              b.logsBloom.CompressedBytes(),
		Result:                 b.result,

		PrevID:          b.prevID,
		Signature:       b.signature,
		RepsHash:        b.repsHash,
		LeaderVotesHash: b.leaderVotes.Hash(),
		NextLeader:      b.nextLeader.Bytes(),
	}
}

func (b *Block) ToJSON(version module.JSONVersion) (interface{}, error) {
	res := make(map[string]interface{})
	res["version"] = blockV1String
	res["prev_block_hash"] = hex.EncodeToString(b.PrevID())
	res["merkle_tree_root_hash"] = hex.EncodeToString(b.NormalTransactions().Hash())
	res["time_stamp"] = b.Timestamp()
	res["confirmed_transaction_list"] = b.NormalTransactions()
	res["block_hash"] = hex.EncodeToString(b.ID())
	res["height"] = b.Height()
	if b.Proposer() != nil {
		res["peer_id"] = fmt.Sprintf("hx%x", b.Proposer().ID())
	} else {
		res["peer_id"] = ""
	}
	res["signature"] = ""
	return res, nil
}

func bssFromTransactionList(l module.TransactionList) ([][]byte, error) {
	var res [][]byte
	for it := l.Iterator(); it.Has(); log.Must(it.Next()) {
		tr, _, err := it.Get()
		if err != nil {
			return nil, err
		}
		bs := tr.Bytes()
		res = append(res, bs)
	}
	return res, nil
}

func (b *Block) _bodyFormat() (*bodyFormat, error) {
	ptBss, err := bssFromTransactionList(b.patchTransactions)
	if err != nil {
		return nil, err
	}
	ntBss, err := bssFromTransactionList(b.normalTransactions)
	if err != nil {
		return nil, err
	}
	return &bodyFormat{
		PatchTransactions:  ptBss,
		NormalTransactions: ntBss,
		Votes:              b.votes.Bytes(),
		LeaderVotes:        b.leaderVotes.Bytes(),
	}, nil
}

func (b *Block) NewBlock(vl module.ValidatorList) module.Block {
	if !bytes.Equal(b.nextValidatorsHash, vl.Hash()) {
		return nil
	}
	blk := *b
	blk._nextValidators = vl
	return &blk
}
