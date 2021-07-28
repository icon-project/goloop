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

package consensus_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/test"
)

func TestConsensus_FastSyncServer(t *testing.T) {
	f := test.NewFixture(t)
	defer f.Close()
	err := f.CS.Start()
	assert.NoError(t, err)
	blk, err := f.BM.GetLastBlock()
	assert.NoError(t, err)
	buf := bytes.NewBuffer(nil)
	err = blk.Marshal(buf)
	assert.NoError(t, err)
	_, h1 := f.NM.NewPeerFor(module.ProtoFastSync)
	h1.Unicast(
		fastsync.ProtoBlockRequest,
		&fastsync.BlockRequest {
			RequestID: 0,
			Height: 0,
			ProofOption: 0,
		},
		nil,
	)
	h1.ReceiveUnicast(
		fastsync.ProtoBlockMetadata,
		&fastsync.BlockMetadata{
			RequestID: 0,
			BlockLength: int32(buf.Len()),
			Proof: consensus.NewEmptyCommitVoteList().Bytes(),
		},
	)
}
