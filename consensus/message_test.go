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

package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

func NewPrecommitMessage(
	w module.Wallet,
	height int64, round int32, id []byte, partSetID *PartSetID, ts int64,
) *VoteMessage {
	return NewVoteMessage(
		w, VoteTypePrecommit, height, round, id, partSetID, ts, nil, nil, 0,
	)
}

func TestNewPrecommitMessage(t *testing.T) {
	w := wallet.New()
	vm := NewPrecommitMessage(
		w,
		1, 0, nil, nil, 0,
	)
	err := vm.Verify()
	assert.NoError(t, err)
}

func FuzzNewProposalMessage(f *testing.F) {
	f.Add([]byte("\xef\x800"))
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := NewProposalMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify()
		}
	})
}

func FuzzNewBlockPartMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newBlockPartMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify()
		}
	})
}

func FuzzNewVoteMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newVoteMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify()
		}
	})
}

func FuzzNewRoundStateMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newRoundStateMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify()
		}
	})
}

func FuzzNewVoteListMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newVoteListMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify()
		}
	})
}
