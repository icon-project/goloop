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

	"github.com/icon-project/goloop/btp/ntm"
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

type walletProvider struct {
	wallet module.Wallet
}

func (wp *walletProvider) WalletFor(dsa string) module.BaseWallet {
	switch dsa {
	case "ecdsa/secp256k1":
		return wp.wallet
	}
	return nil
}

func TestVoteMessage_VerifyOK(t *testing.T) {
	assert := assert.New(t)
	wp := &walletProvider{wallet.New()}
	psb := NewPartSetBuffer(10)
	_, _ = psb.Write(make([]byte, 10))
	ps := psb.PartSet()
	msg := newVoteMessage()
	msg.Height = 1
	msg.Round = 0
	msg.Type = VoteTypePrecommit
	msg.BlockID = []byte("abc")
	msg.BlockPartSetIDAndNTSVoteCount = ps.ID().WithAppData(0)
	msg.NTSVoteBases = nil
	msg.Timestamp = 10
	msg.NTSDProofParts = nil
	_ = msg.sign(wp.wallet)
	assert.NoError(msg.Verify())
}

func TestVoteMessage_VerifyMismatchBetweenAppDataAndNTSDProofPartsLen(t *testing.T) {
	assert := assert.New(t)
	wp := &walletProvider{wallet.New()}
	w := wp.wallet
	psb := NewPartSetBuffer(10)
	_, _ = psb.Write(make([]byte, 10))
	ps := psb.PartSet()
	msg := newVoteMessage()
	msg.Height = 1
	msg.Round = 0
	msg.Type = VoteTypePrecommit
	msg.BlockID = []byte("abc")
	msg.BlockPartSetIDAndNTSVoteCount = ps.ID().WithAppData(0)
	msg.NTSVoteBases = []ntsVoteBase{
		{1, []byte("abc")},
	}
	msg.Timestamp = 10
	msg.NTSDProofParts = make([][]byte, 1)
	pc, _ := ntm.ForUID("eth").NewProofContext([][]byte{w.PublicKey()})
	pp, _ := pc.NewProofPart([]byte("abc"), wp)
	msg.NTSDProofParts[0] = pp.Bytes()
	_ = msg.sign(w)
	assert.Error(msg.Verify())
}

func TestVoteMessage_VerifyMismatchBetweenNTSVoteBasesAndNTSDProofParts(t *testing.T) {
	assert := assert.New(t)
	wp := &walletProvider{wallet.New()}
	w := wp.wallet
	psb := NewPartSetBuffer(10)
	_, _ = psb.Write(make([]byte, 10))
	ps := psb.PartSet()
	msg := newVoteMessage()
	msg.Height = 1
	msg.Round = 0
	msg.Type = VoteTypePrecommit
	msg.BlockID = []byte("abc")
	msg.BlockPartSetIDAndNTSVoteCount = ps.ID().WithAppData(1)
	msg.NTSVoteBases = []ntsVoteBase{}
	msg.Timestamp = 10
	msg.NTSDProofParts = make([][]byte, 1)
	pc, _ := ntm.ForUID("eth").NewProofContext([][]byte{w.PublicKey()})
	pp, _ := pc.NewProofPart([]byte("abc"), wp)
	msg.NTSDProofParts[0] = pp.Bytes()
	_ = msg.sign(w)
	assert.Error(msg.Verify())
}
