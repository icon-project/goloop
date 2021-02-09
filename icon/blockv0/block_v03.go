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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

type BlockV03JSON struct {
	Hash             common.HexBytes    `json:"hash"`
	Version          string             `json:"version"`
	PrevHash         common.HexHash     `json:"prevHash"`
	TransactionsHash common.HexHash     `json:"transactionsHash"`
	StateHash        common.HexHash     `json:"stateHash"`
	ReceiptsHash     common.HexHash     `json:"receiptsHash"`
	RepsHash         common.HexHash     `json:"repsHash"`
	NextRepsHash     common.HexHash     `json:"nextRepsHash"`
	LeaderVotesHash  common.HexHash     `json:"leaderVotesHash"`
	PrevVotesHash    common.HexHash     `json:"prevVotesHash"`
	LogsBloom        txresult.LogsBloom `json:"logsBloom"`
	Timestamp        common.HexInt64    `json:"timestamp"`
	Transactions     []Transaction      `json:"transactions"`
	LeaderVotes      *LeaderVoteList    `json:"leaderVotes"`
	PrevVotes        *BlockVoteList     `json:"prevVotes"`
	Height           common.HexInt64    `json:"height"`
	Leader           common.Address     `json:"leader"`
	NextLeader       common.Address     `json:"nextLeader"`
	Signature        common.Signature   `json:"signature"`
}

type BlockV03 struct {
	json     *BlockV03JSON
	txs      module.TransactionList
	reps     *RepsList
	nextReps *RepsList
}

func (b *BlockV03) Version() string {
	return b.json.Version
}

func (b *BlockV03) ID() []byte {
	return b.json.Hash.Bytes()
}

func (b *BlockV03) Height() int64 {
	return b.json.Height.Value
}

func (b *BlockV03) PrevID() []byte {
	return b.json.PrevHash.Bytes()
}

func (b *BlockV03) Votes() *BlockVoteList {
	return b.json.PrevVotes
}

func (b *BlockV03) NextValidators() *RepsList {
	return b.nextReps
}

func (b *BlockV03) NormalTransactions() module.TransactionList {
	return b.txs
}

func (b *BlockV03) Timestamp() int64 {
	return b.json.Timestamp.Value
}

func (b *BlockV03) Proposer() module.Address {
	return &b.json.Leader
}

func (b *BlockV03) LogsBloom() module.LogsBloom {
	return &b.json.LogsBloom
}

func (b *BlockV03) ToJSON(rcpVersion module.JSONVersion) (interface{}, error) {
	return b.json, nil
}

func calcMerkleRootOfTransactions(txs []Transaction) []byte {
	items := make([]merkleItem, len(txs))
	for i, tx := range txs {
		items[i] = hashedItem(tx.ID())
	}
	return calcHashOfList(items)
}

func (b *BlockV03) calcHash() []byte {
	items := make([]merkleItem, 0, 13)
	items = append(items,
		hashedItem(b.json.PrevHash.Bytes()),
		hashedItem(b.json.TransactionsHash.Bytes()),
		hashedItem(b.json.ReceiptsHash.Bytes()),
		hashedItem(b.json.StateHash.Bytes()),
		hashedItem(b.json.RepsHash.Bytes()),
		hashedItem(b.json.NextRepsHash.Bytes()),
		hashedItem(b.json.LeaderVotesHash.Bytes()),
		hashedItem(b.json.PrevVotesHash.Bytes()),
		valueItem(b.json.LogsBloom.LogBytes()),
		valueItem(intconv.SizeToBytes(uint64(b.json.Height.Value))),
		valueItem(intconv.SizeToBytes(uint64(b.json.Timestamp.Value))),
		valueItem(b.json.Leader.ID()),
		valueItem(b.json.NextLeader.ID()),
	)
	return calcHashOfList(items)
}

func (b *BlockV03) Verify(prev Block) error {
	if err := b.json.LeaderVotes.Verify(b.reps); err != nil {
		return err
	}
	if err := b.json.PrevVotes.Verify(); err != nil {
		return err
	}
	for _, tx := range b.json.Transactions {
		if err := tx.Verify(); err != nil {
			return err
		}
	}
	txs := calcMerkleRootOfTransactions(b.json.Transactions)
	if !bytes.Equal(b.json.TransactionsHash.Bytes(), txs) {
		return errors.CriticalFormatError.Errorf(
			"InvalidTransactionHash(exp=%#x,calc=%#x)",
			b.json.TransactionsHash.Bytes(), txs)
	}
	if !bytes.Equal(b.json.LeaderVotesHash.Bytes(), b.json.LeaderVotes.Hash()) {
		return errors.CriticalFormatError.Errorf(
			"InvalidLeaderVotesHash(exp=%#x,calc=%#x)",
			b.json.LeaderVotesHash.Bytes(), b.json.LeaderVotes.Hash())
	}
	if !bytes.Equal(b.json.PrevVotesHash.Bytes(), b.json.PrevVotes.Hash()) {
		return errors.CriticalFormatError.Errorf(
			"InvalidPrevVotesHash(exp=%#x,calc=%#x)",
			b.json.PrevVotesHash.Bytes(), b.json.PrevVotes.Hash())
	}
	if hash := b.calcHash(); !bytes.Equal(b.json.Hash.Bytes(), hash) {
		return errors.CriticalFormatError.Errorf(
			"InvalidHashValue(exp=%#x,calc=%#x)", b.json.Hash.Bytes(), hash)
	}
	if prev != nil {
		switch pb := prev.(type) {
		case *BlockV03:
			voted := b.json.PrevVotes.Quorum()
			if !bytes.Equal(pb.ID(), voted) {
				return errors.InvalidStateError.Errorf(
					"InvalidConsensus(voted=%#x,id=%#x)", voted, pb.ID())
			}
			var leader module.Address = &pb.json.NextLeader
			// New term starts, so the next leader should be the first
			// one of next leaders. For remarking, it uses empty user address.
			if leader.String() == "hx0000000000000000000000000000000000000000" {
				leader = b.nextReps.Get(0)
			}
			if b.json.LeaderVotesHash.Bytes() != nil {
				if addr := b.json.LeaderVotes.Quorum(); addr == nil {
					return errors.InvalidStateError.New("NoValidLeader")
				} else {
					leader = addr
				}
			}
			if !b.json.Leader.Equal(leader) {
				return errors.InvalidStateError.Errorf(
					"InvalidLeader(exp=%s,real=%s)",
					leader,
					&b.json.Leader,
				)
			}
		default:
			return errors.InvalidStateError.Errorf("UnknownBlockVersion(%T)", prev)
		}
	}
	return nil
}

func ParseBlockV03(b []byte, lc Store) (Block, error) {
	jso := new(BlockV03JSON)
	if err := json.Unmarshal(b, jso); err != nil {
		return nil, err
	}
	txs := make([]module.Transaction, len(jso.Transactions))
	for i, tx := range jso.Transactions {
		txs[i] = tx.Transaction
	}
	var current, next *RepsList
	if jso.RepsHash != nil {
		if reps, err := lc.GetRepsByHash(jso.RepsHash); err != nil {
			return nil, err
		} else {
			current = reps
		}
	}
	if jso.NextRepsHash != nil {
		if reps, err := lc.GetRepsByHash(jso.NextRepsHash); err != nil {
			return nil, err
		} else {
			next = reps
		}
	} else {
		next = current
	}
	return &BlockV03{
		json:     jso,
		txs:      transaction.NewTransactionListV1FromSlice(txs),
		reps:     current,
		nextReps: next,
	}, nil
}
