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
	"bufio"
	"bytes"
	"io"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/module"
)

type handler struct {
	chain module.Chain
	v2Handler block.Handler
}

func NewHandler(
	chain module.Chain,
	v2Handler block.Handler,
) block.Handler {
	return &handler{
		chain,
		v2Handler,
	}
}

func (b *handler) NewBlock(
	ctx block.HandlerContext,
	height int64, ts int64, proposer module.Address, prevID []byte,
	logsBloom module.LogsBloom, result []byte,
	patchTransactions module.TransactionList,
	normalTransactions module.TransactionList,
	nextValidators module.ValidatorList, blockVote module.CommitVoteSet,
) module.Block {
	if height != 0 || prevID != nil {
		log.Panicf("Not a genesis. Cannot propose v1 block")
	}
	blkV1 := &blockV11{
		Block: Block{
			height:             height,
			timestamp:          ts,
			proposer:           proposer,
			prevHash:           prevID,
			logsBloom:          logsBloom,
			result:             result,
			// use zero value for signature
			prevID:             prevID,
			versionV0:          "0.1a",
			patchTransactions:  patchTransactions,
			normalTransactions: normalTransactions,
		},
	}
	blkV1.blockDetail = blkV1
	return &blkV1.Block
}

func (b *handler) NewBlockFromHeaderReader(
	ctx block.HandlerContext,
	r io.Reader,
) (module.Block, error) {
	return NewBlockFromHeaderReader(b.chain.Database(), r)
}

func (b *handler) GetBlock(ctx block.HandlerContext, id []byte) (module.Block, error) {
	dbase := b.chain.Database()
	hash, err := db.DoGetWithBucketID(dbase, icdb.IDToHash, id)
	if errors.NotFoundError.Equals(err) {
		return b.v2Handler.GetBlock(ctx, id)
	} else if err != nil {
		return nil, err
	}

	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return nil, err
	}
	return b.NewBlockFromHeaderReader(ctx, bytes.NewReader(headerBytes))
}

func (b *handler) NewBlockDataFromReader(
	ctx block.HandlerContext,
	r io.Reader,
) (module.BlockData, error) {
	br := bufio.NewReader(r)
	version, err := block.PeekVersion(br)
	if err != nil {
		return nil, err
	}
	if version != module.BlockVersion1 {
		return b.v2Handler.NewBlockDataFromReader(ctx, br)
	}
	return NewBlockFromReader(b.chain.Database(), br)
}

func (b* handler) GetBlockByHeight(
	ctx block.HandlerContext,
	height int64,
) (module.Block, error) {
	dbase := b.chain.Database()
	headerHashByHeight, err := db.NewCodedBucket(
		dbase,
		db.BlockHeaderHashByHeight,
		nil,
	)
	if err != nil {
		return nil, err
	}
	hash, err := headerHashByHeight.GetBytes(height)
	if err != nil {
		return nil, err
	}

	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return nil, err
	}
	if headerBytes == nil {
		return nil, errors.InvalidStateError.Errorf("nil header")
	}

	r := bytes.NewReader(headerBytes)
	br := bufio.NewReader(r)
	version, err := block.PeekVersion(br)
	if err != nil {
		return nil, err
	}
	_, _ = r.Seek(0, io.SeekStart)
	if version != module.BlockVersion1 {
		return b.v2Handler.NewBlockFromHeaderReader(ctx, r)
	}
	return b.NewBlockFromHeaderReader(ctx, r)
}

func (b *handler) FinalizeHeader(
	ctx block.HandlerContext,
	blk module.Block,
) error {
	if blk.Version() != module.BlockVersion1 {
		return b.v2Handler.FinalizeHeader(ctx, blk)
	}
	blkV1 := blk.(*Block)
	return blkV1.WriteTo(b.chain.Database())
}

func (b *handler) GetVoters(
	ctx block.HandlerContext,
	height int64,
) (module.ValidatorList, error) {
	blk, err := ctx.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	if blk.Version() != module.BlockVersion1 {
		return b.v2Handler.GetVoters(ctx, height)
	}
	return blk.(*Block).NextValidators(), nil
}
