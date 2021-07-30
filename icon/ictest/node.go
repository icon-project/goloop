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

package ictest

import (
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
	"github.com/icon-project/goloop/test"
)

func NodeGenerateBlocksAndFinalizeMerkle(n* test.Node, blks int64) ([]byte, int64) {
	t := n.T
	temp, err := n.Chain.Database().GetBucket("temp")
	assert.NoError(t, err)
	mkl, err := n.Chain.Database().GetBucket(icdb.BlockMerkle)
	ac, err := hexary.NewAccumulator(mkl, temp, "")
	const height = 10
	// add genesis
	err = ac.Add(n.LastBlock.Hash())
	assert.NoError(t, err)
	for i:=1; i<height; i++ {
		n.ProposeFinalizeBlock((*blockv0.BlockVoteList)(nil))
		err = ac.Add(n.LastBlock.Hash())
		assert.NoError(t, err)
	}
	root, leaves, err := ac.Finalize("")
	assert.NoError(t, err)
	return root, leaves
}
