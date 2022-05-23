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

package block

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

type blockV2HeaderFormat1 struct {
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte
	PrevID                 []byte
	VotesHash              []byte
	NextValidatorsHash     []byte
	PatchTransactionsHash  []byte
	NormalTransactionsHash []byte
	LogsBloom              []byte
	Result                 []byte
}

type blockV2BodyFormat1 struct {
	PatchTransactions  [][]byte
	NormalTransactions [][]byte
	Votes              []byte
}

type blockV2Format1 struct {
	blockV2HeaderFormat1
	blockV2BodyFormat1
}

func TestBlockV2_EncodeAsFormat1IfPossible(t *testing.T) {
	blockHeaderFormat1 := blockV2HeaderFormat1{
		Version:                module.BlockVersion2,
		Height:                 1,
		Timestamp:              10,
		Proposer:               []byte("proposer"),
		PrevID:                 []byte("prevID"),
		VotesHash:              []byte("voteHash"),
		NextValidatorsHash:     []byte("nextValidatorHash"),
		PatchTransactionsHash:  []byte("patchTransactionHash"),
		NormalTransactionsHash: []byte("normalTransactionHash"),
		LogsBloom:              []byte("logsBloom"),
		Result:                 []byte("result"),
	}
	blockHeaderFormat2 := blockV2HeaderFormat{
		Version:                module.BlockVersion2,
		Height:                 1,
		Timestamp:              10,
		Proposer:               []byte("proposer"),
		PrevID:                 []byte("prevID"),
		VotesHash:              []byte("voteHash"),
		NextValidatorsHash:     []byte("nextValidatorHash"),
		PatchTransactionsHash:  []byte("patchTransactionHash"),
		NormalTransactionsHash: []byte("normalTransactionHash"),
		LogsBloom:              []byte("logsBloom"),
		Result:                 []byte("result"),
		NSFilter:               nil,
		NTSDProofHashListHash:  nil,
	}
	assert.EqualValues(
		t,
		codec.MustMarshalToBytes(&blockHeaderFormat1),
		codec.MustMarshalToBytes(&blockHeaderFormat2),
	)
}
