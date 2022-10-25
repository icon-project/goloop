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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func TestBlockV2_ToJSON(t *testing.T) {
	nd := test.NewNode(t)
	defer nd.Close()
	assert := assert.New(t)

	blk, err := nd.BM.GetBlock(make([]byte, crypto.HashLen))
	assert.Error(err)

	nd.ProposeFinalizeBlock(consensus.NewEmptyCommitVoteList())
	blk = nd.GetLastBlock()
	mp_, err := blk.ToJSON(module.JSONVersion3)
	mp := mp_.(map[string]interface{})
	assert.EqualValues(block.V2String, mp["version"])
	assert.EqualValues(hex.EncodeToString(blk.PrevID()), mp["prev_block_hash"])
	assert.EqualValues(hex.EncodeToString(blk.ID()), mp["block_hash"])
	assert.EqualValues(1, mp["height"])
}
