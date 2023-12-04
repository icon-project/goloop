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

package icon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/icon/lcimporter"
)

func TestPlatform_BlockV1Proof(t *testing.T) {
	base, err := os.MkdirTemp("", "platform*")
	assert.NoError(t, err)
	defer func(t *testing.T) {
		assert.NoError(t, os.RemoveAll(base))
	}(t)

	plt, err := NewPlatform(base, 1)
	assert.NoError(t, err)

	w := wallet.New()

	storage := plt.(lcimporter.BlockV1ProofStorage)
	root := crypto.SHA3Sum256([]byte("test_data"))
	height := int64(1234)
	votes := blockv0.NewBlockVoteList(blockv0.NewBlockVote(w, height, 0, root, 0))
	err = storage.SetBlockV1Proof(root, height, votes)
	assert.NoError(t, err)

	mh2, votes2, err := storage.GetBlockV1Proof()
	assert.NoError(t, err)
	assert.Equal(t, votes.Root(), votes2.Root())
	assert.Equal(t, votes.Hash(), votes2.Hash())
	assert.Equal(t, root, mh2.RootHash)
	assert.Equal(t, height, mh2.Leaves)
}
