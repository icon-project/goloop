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

package block

import (
	"bytes"
	"io"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

type blockV2Handler struct {
	chain base.Chain
	sm    ServiceManager
}

func NewBlockV2Handler(chain base.Chain) base.BlockHandler {
	return &blockV2Handler{
		chain: chain,
		sm: chain.ServiceManager(),
	}
}

func (b *blockV2Handler) Version() int {
	return module.BlockVersion2
}

func (b *blockV2Handler) bucketFor(id db.BucketID) (*db.CodedBucket, error) {
	return db.NewCodedBucket(b.chain.Database(), id, nil)
}

func (b *blockV2Handler) commitVoteSetFromHash(hash []byte) module.CommitVoteSet {
	bk, err := b.bucketFor(db.BytesByHash)
	if err != nil {
		return nil
	}
	bs, err := bk.GetBytes(db.Raw(hash))
	if err != nil {
		return nil
	}
	dec := b.chain.CommitVoteSetDecoder()
	return dec(bs)
}

func (b *blockV2Handler) NewBlock(
	height int64, ts int64, proposer module.Address, prev module.Block,
	logsBloom module.LogsBloom, result []byte,
	patchTransactions module.TransactionList,
	normalTransactions module.TransactionList,
	nextValidators module.ValidatorList, votes module.CommitVoteSet,
) base.Block {
	var prevID []byte
	if prev != nil {
		prevID = prev.ID()
	}
	return &blockV2{
		blockV2Immut: blockV2Immut{
			height:             height,
			timestamp:          ts,
			proposer:           proposer,
			prevID:             prevID,
			logsBloom:          logsBloom,
			result:             result,
			patchTransactions:  patchTransactions,
			normalTransactions: normalTransactions,
			nextValidatorsHash: nextValidators.Hash(),
			_nextValidators:    nextValidators,
			votes:              votes,
		},
	}
}

func (b *blockV2Handler) NewBlockFromHeaderReader(r io.Reader) (base.Block, error) {
	var header blockV2HeaderFormat
	err := v2Codec.Unmarshal(r, &header)
	if err != nil {
		return nil, err
	}
	sm := b.sm
	patches := sm.TransactionListFromHash(header.PatchTransactionsHash)
	if patches == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", header.PatchTransactionsHash)
	}
	normalTxs := sm.TransactionListFromHash(header.NormalTransactionsHash)
	if normalTxs == nil {
		return nil, errors.Errorf("TransactionListFromHash(%x) failed", header.NormalTransactionsHash)
	}
	nextValidators := sm.ValidatorListFromHash(header.NextValidatorsHash)
	if nextValidators == nil {
		return nil, errors.Errorf("ValidatorListFromHas(%x)", header.NextValidatorsHash)
	}
	votes := b.commitVoteSetFromHash(header.VotesHash)
	if votes == nil {
		return nil, errors.Errorf("commitVoteSetFromHash(%x) failed", header.VotesHash)
	}
	proposer, err := newProposer(header.Proposer)
	if err != nil {
		return nil, err
	}
	return &blockV2{
		blockV2Immut: blockV2Immut{
			height:             header.Height,
			timestamp:          header.Timestamp,
			proposer:           proposer,
			prevID:             header.PrevID,
			logsBloom:          txresult.NewLogsBloomFromCompressed(header.LogsBloom),
			result:             header.Result,
			patchTransactions:  patches,
			normalTransactions: normalTxs,
			nextValidatorsHash: nextValidators.Hash(),
			_nextValidators:    nextValidators,
			votes:              votes,
		},
	}, nil
}

func newTransactionListFromBSS(
	sm ServiceManager, bss [][]byte, version int,
) (module.TransactionList, error) {
	ts := make([]module.Transaction, len(bss))
	for i, bs := range bss {
		if tx, err := sm.TransactionFromBytes(bs, version); err != nil {
			return nil, err
		} else {
			ts[i] = tx
		}
	}
	return sm.TransactionListFromSlice(ts, version), nil
}

func (b *blockV2Handler) NewBlockDataFromReader(r io.Reader) (base.BlockData, error) {
	sm := b.sm
	var blockFormat blockV2Format
	err := v2Codec.Unmarshal(r, &blockFormat.blockV2HeaderFormat)
	if err != nil {
		return nil, err
	}
	err = v2Codec.Unmarshal(r, &blockFormat.blockV2BodyFormat)
	if err != nil {
		return nil, err
	}
	patches, err := newTransactionListFromBSS(
		sm,
		blockFormat.PatchTransactions,
		module.BlockVersion2,
	)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(patches.Hash(), blockFormat.PatchTransactionsHash) {
		return nil, errors.New("bad patch transactions hash")
	}
	normalTxs, err := newTransactionListFromBSS(
		sm,
		blockFormat.NormalTransactions,
		module.BlockVersion2,
	)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(normalTxs.Hash(), blockFormat.NormalTransactionsHash) {
		return nil, errors.New("bad normal transactions hash")
	}
	// nextValidators may be nil
	nextValidators := sm.ValidatorListFromHash(blockFormat.NextValidatorsHash)
	votes := b.chain.CommitVoteSetDecoder()(blockFormat.Votes)
	if !bytes.Equal(votes.Hash(), blockFormat.VotesHash) {
		return nil, errors.New("bad vote list hash")
	}
	proposer, err := newProposer(blockFormat.Proposer)
	if err != nil {
		return nil, err
	}
	return &blockV2{
		blockV2Immut: blockV2Immut{
			height:             blockFormat.Height,
			timestamp:          blockFormat.Timestamp,
			proposer:           proposer,
			prevID:             blockFormat.PrevID,
			logsBloom:          txresult.NewLogsBloomFromCompressed(blockFormat.LogsBloom),
			result:             blockFormat.Result,
			patchTransactions:  patches,
			normalTransactions: normalTxs,
			nextValidatorsHash: blockFormat.NextValidatorsHash,
			_nextValidators:    nextValidators,
			votes:              votes,
		},
	}, nil
}

func (b *blockV2Handler) GetBlock(id []byte) (base.Block, error) {
	dbase := b.chain.Database()
	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, id)
	if errors.NotFoundError.Equals(err) {
		return nil, errors.WithStack(errors.ErrUnsupported)
	} else if err != nil {
		return nil, err
	}
	return b.NewBlockFromHeaderReader(bytes.NewReader(headerBytes))
}
