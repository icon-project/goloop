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

package icconsensus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/ictest"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/test"
)

func TestConsensus_WithAccumulator(t *testing.T) {
	gen := test.NewFixture(t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1)
	defer gen.Close()

	temp, err := gen.Chain.Database().GetBucket("temp")
	assert.NoError(t, err)
	mkl, err := gen.Chain.Database().GetBucket(icdb.BlockMerkle)
	ac, err := hexary.NewAccumulator(mkl, temp, "")
	const height = 10
	// add genesis
	err = ac.Add(gen.LastBlock.Hash())
	assert.NoError(t, err)
	for i:=1; i<height; i++ {
		gen.ProposeFinalizeBlock((*blockv0.BlockVoteList)(nil))
		err = ac.Add(gen.LastBlock.Hash())
		assert.NoError(t, err)
	}
	root, leaves, err := ac.Finalize("")
	assert.NoError(t, err)

	gen = test.NewFixture(
		t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1,
		ictest.UseMerkle(root, leaves), ictest.UseDB(gen.Chain.Database()),
	)
	defer gen.Close()
	for i:=0; i<height; i++ {
		_, err = gen.BM.GetBlockByHeight(1)
		assert.NoError(t, err)
	}

	f := test.NewFixture(
		t, ictest.UseBMForBlockV1, ictest.UseCSForBlockV1,
		ictest.UseMerkle(root, leaves),
	)
	defer f.Close()

	err = gen.CS.Start()
	assert.NoError(t, err)

	err = f.CS.Start()
	assert.NoError(t, err)

	f.NM.Connect(gen.NM)

	for f.CS.GetStatus().Height < height {
		time.Sleep(50 * time.Millisecond)
	}
	blk, err := f.BM.GetLastBlock()
	assert.NoError(t, err)
	assert.EqualValues(t, height-1, blk.Height())
}
