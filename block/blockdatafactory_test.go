/*
 * Copyright 2022 ICON Foundation
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

package block_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/test"
)

func TestBlockDataFactory_Basics(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk := nd.LastBlock

	bdf, err := block.NewBlockDataFactory(nd.Chain, nil)
	assert.NoError(err)
	buf := bytes.NewBuffer(nil)
	err = blk.Marshal(buf)
	assert.NoError(err)
	bd, err := bdf.NewBlockDataFromReader(buf)
	assert.NoError(err)
	assert.EqualValues(blk.ID(), bd.ID())
}

func FuzzBlockDataFactory_NewBlockDataFromReader(f *testing.F) {
	nd := test.NewNode(f)
	defer nd.Close()
	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())

	bdf, err := block.NewBlockDataFactory(nd.Chain, nil)
	assert.NoError(f, err)
	testcases := [][]byte{
		[]byte("\xf5\x02000000\x80\x8000000000000000000000000000000000000000000000\xde\xc0\xc00000000000000000000000000000"),
		[]byte("\xd0\x02000000\x80\x800000000\xe9\xc0\xc0000000000000000000000000000000000000000"),
	}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, bs []byte) {
		bd, err := bdf.NewBlockDataFromReader(bytes.NewReader(bs))
		if err != nil {
			return
		}
		bd.Marshal(io.Discard)
	})
}
