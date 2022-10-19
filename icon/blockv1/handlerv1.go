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
	"io"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/module"
)

type handler struct {
	chain base.Chain
}

func NewHandler(
	chain base.Chain,
) base.BlockHandler {
	return &handler{
		chain,
	}
}

func (b *handler) Version() int {
	return module.BlockVersion1
}

func (b *handler) NewBlock(
	height int64, ts int64, proposer module.Address, prev module.Block,
	logsBloom module.LogsBloom, result []byte,
	patchTransactions module.TransactionList,
	normalTransactions module.TransactionList,
	nextValidators module.ValidatorList, blockVote module.CommitVoteSet,
	bs module.BTPSection,
) base.Block {
	// called for genesis in product
	// called for propose only in test
	if nextValidators == nil || nextValidators.Len() == 0 {
		return NewBlockV11(
			height, ts, proposer, prev, logsBloom, result, patchTransactions,
			normalTransactions, nextValidators,
		)
	}
	return NewBlockV13(
		height, ts, proposer, prev, logsBloom, result, patchTransactions,
		normalTransactions, nextValidators, blockVote,
	)
}

func (b *handler) NewBlockFromHeaderReader(
	r io.Reader,
) (base.Block, error) {
	return NewBlockFromHeaderReader(b.chain.Database(), r)
}

func (b *handler) NewBlockDataFromReader(
	r io.Reader,
) (base.BlockData, error) {
	return NewBlockFromReader(b.chain.Database(), r)
}

func (b *handler) GetBlock(id []byte) (base.Block, error) {
	dbase := b.chain.Database()
	hash, err := db.DoGetWithBucketID(dbase, icdb.IDToHash, id)
	if errors.NotFoundError.Equals(err) {
		return nil, errors.WithStack(errors.ErrUnsupported)
	} else if err != nil {
		return nil, err
	}

	headerBytes, err := db.DoGetWithBucketID(dbase, db.BytesByHash, hash)
	if err != nil {
		return nil, err
	}
	return b.NewBlockFromHeaderReader(bytes.NewReader(headerBytes))
}
