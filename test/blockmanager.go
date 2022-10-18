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

package test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/module"
)

func AssertBlock(
	t *testing.T, blk module.Block,
	version int, height int64, id []byte, prevID []byte,
) {
	assert.EqualValues(t, version, blk.Version())
	assert.EqualValues(t, height, blk.Height())
	assert.EqualValues(t, id, blk.ID())
	assert.EqualValues(t, prevID, blk.PrevID())
}

func AssertBlockInBM(
	t *testing.T, bm module.BlockManager,
	version int, height int64, id []byte, prevID []byte,
) {
	blk, err := bm.GetBlockByHeight(height)
	assert.NoError(t, err)
	AssertBlock(t, blk, version, height, id, prevID)

	blk, err = bm.GetBlock(id)
	assert.NoError(t, err)
	AssertBlock(t, blk, version, height, id, prevID)
}

func AssertLastBlock(
	t *testing.T, bm module.BlockManager,
	height int64, prevID []byte, version int,
) {
	blk, err := bm.GetLastBlock()
	assert.NoError(t, err)
	AssertBlock(t, blk, version, height, blk.ID(), prevID)
	AssertBlockInBM(t, bm, version, height, blk.ID(), prevID)
}

func GetLastBlock(t *testing.T, bm module.BlockManager) module.Block {
	blk, err := bm.GetLastBlock()
	assert.NoError(t, err)
	return blk
}

type cbResult struct {
	bc    module.BlockCandidate
	cbErr error
}

func ProposeBlock(
	bm module.BlockManager,
	prevID []byte, votes module.CommitVoteSet,
) (bc module.BlockCandidate, err error, cbError error) {
	ch := make(chan cbResult)
	_, err = bm.Propose(
		prevID, votes, func(bc module.BlockCandidate, err error) {
			ch <- cbResult{bc, err}
		},
	)
	if err != nil {
		return nil, err, nil
	}
	res := <-ch
	return res.bc, nil, res.cbErr
}

func ImportBlockByReader(
	t *testing.T, bm module.BlockManager,
	r io.Reader, flag int,
) (resBc module.BlockCandidate, err error, cbErr error) {
	ch := make(chan cbResult)
	_, err = bm.Import(
		r, flag, func(bc module.BlockCandidate, err error) {
			assert.NoError(t, err)
			ch <- cbResult{bc, err}
		},
	)
	if err != nil {
		return nil, err, nil
	}
	res := <-ch
	return res.bc, nil, res.cbErr
}

func ImportBlock(
	t *testing.T, bm module.BlockManager,
	bc module.BlockCandidate, flag int,
) (resBc module.BlockCandidate, err error, cbErr error) {
	ch := make(chan cbResult)
	_, err = bm.ImportBlock(
		bc, flag, func(bc module.BlockCandidate, err error) {
			assert.NoError(t, err)
			ch <- cbResult{bc, err}
		},
	)
	if err != nil {
		return nil, err, nil
	}
	res := <-ch
	return res.bc, nil, res.cbErr
}

func FinalizeBlock(
	t *testing.T, bm module.BlockManager, bc module.BlockCandidate,
) {
	err := bm.Finalize(bc)
	assert.NoError(t, err)
}
