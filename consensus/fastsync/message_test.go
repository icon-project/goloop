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

package fastsync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
)

func TestBlockRequest_EncodeAsV1IfPossible(t *testing.T) {
	msgV1 := BlockRequestV1{
		RequestID: 1,
		Height:    1,
	}
	msgV2 := BlockRequestV2{
		RequestID:   1,
		Height:      1,
		ProofOption: 0,
	}
	assert.Equal(t,
		codec.MustMarshalToBytes(&msgV1),
		codec.MustMarshalToBytes(&msgV2),
	)
}

func TestBlockRequest_SendV1ReceiveV2(t *testing.T) {
	msgV1 := BlockRequestV1{
		RequestID: 1,
		Height:    1,
	}
	bsV1 := codec.MustMarshalToBytes(&msgV1)
	var msgV2 BlockRequestV2
	codec.MustUnmarshalFromBytes(bsV1, &msgV2)
	assert.Equal(t,
		BlockRequestV2{
			RequestID:   1,
			Height:      1,
			ProofOption: 0,
		},
		msgV2,
	)
}

func TestBlockRequest_SendV2ReceiveV1(t *testing.T) {
	msgV2 := BlockRequestV2{
		RequestID:   1,
		Height:      1,
		ProofOption: 1,
	}
	bsV2 := codec.MustMarshalToBytes(&msgV2)
	var msgV1 BlockRequestV1
	codec.MustUnmarshalFromBytes(bsV2, &msgV1)
	assert.Equal(t,
		BlockRequestV1{
			RequestID: 1,
			Height:    1,
		},
		msgV1,
	)
}

func TestBlockRequest_SendV2ReceiveV2(t *testing.T) {
	msgV2 := BlockRequestV2{
		RequestID:   1,
		Height:      1,
		ProofOption: 1,
	}
	bsV2 := codec.MustMarshalToBytes(&msgV2)
	var msgV2Another BlockRequestV2
	codec.MustUnmarshalFromBytes(bsV2, &msgV2Another)
	assert.Equal(t,
		BlockRequestV2{
			RequestID:   1,
			Height:      1,
			ProofOption: 1,
		},
		msgV2Another,
	)
}

func FuzzBlockRequest(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var msg BlockRequest
		codec.UnmarshalFromBytes(data, &msg)
	})
}

func FuzzBlockMetadata(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var msg BlockMetadata
		codec.UnmarshalFromBytes(data, &msg)
	})
}

func FuzzBlockData(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		var msg BlockData
		codec.UnmarshalFromBytes(data, &msg)
	})
}
