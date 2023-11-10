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
	"github.com/icon-project/goloop/common/codec"
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

type nilVerifyCtx struct {
}

var theNilVerifyCtx nilVerifyCtx

func (ctx nilVerifyCtx) ValidNID(nid uint32) bool {
	return true
}

func (ctx nilVerifyCtx) NID() int {
	return 0
}

func TestNewPrecommitMessage(t *testing.T) {
	w := wallet.New()
	vm := NewPrecommitMessage(
		w,
		1, 0, make([]byte, 32), nil, 0,
	)
	err := vm.Verify(theNilVerifyCtx)
	assert.NoError(t, err)
}

func FuzzNewProposalMessage(f *testing.F) {
	f.Add([]byte("\xef\x800"))
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := NewProposalMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify(theNilVerifyCtx)
		}
	})
}

func FuzzNewBlockPartMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newBlockPartMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify(theNilVerifyCtx)
		}
	})
}

func FuzzNewVoteMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newVoteMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify(theNilVerifyCtx)
		}
	})
}

func FuzzNewRoundStateMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newRoundStateMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify(theNilVerifyCtx)
		}
	})
}

func FuzzNewVoteListMessage(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		msg := newVoteListMessage()
		_, err := msgCodec.UnmarshalFromBytes(data, msg)
		if err == nil {
			msg.Verify(theNilVerifyCtx)
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
	_ = msg.Sign(wp.wallet)
	assert.NoError(msg.Verify(theNilVerifyCtx))
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
	_ = msg.Sign(w)
	assert.Error(msg.Verify(theNilVerifyCtx))
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
	_ = msg.Sign(w)
	assert.Error(msg.Verify(theNilVerifyCtx))
}

func TestProposal_EncodeAsV1IfPossible(t *testing.T) {
	msgV1 := proposalV1{
		_HR: _HR{
			1, 1,
		},
		BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
	}
	msgV2 := proposalV2{
		_HR: _HR{
			1, 1,
		},
		BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
	}
	assert.Equal(t,
		codec.MustMarshalToBytes(&msgV1), codec.MustMarshalToBytes(&msgV2),
	)
}

func TestProposal_SendV1ReceiveV2(t *testing.T) {
	msgV1 := proposalV1{
		_HR: _HR{
			1, 1,
		},
		BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
	}
	bsV1 := codec.MustMarshalToBytes(&msgV1)
	var msgV2 proposalV2
	codec.MustUnmarshalFromBytes(bsV1, &msgV2)
	assert.Equal(t,
		proposalV2{
			_HR: _HR{
				1, 1,
			},
			BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
		},
		msgV2,
	)
}

func TestProposal_SendV2ReceiveV1(t *testing.T) {
	msgV2 := proposalV2{
		_HR: _HR{
			1, 1,
		},
		BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
	}
	bsV2 := codec.MustMarshalToBytes(&msgV2)
	var msgV1 proposalV1
	codec.MustUnmarshalFromBytes(bsV2, &msgV1)
	assert.Equal(t,
		proposalV1{
			_HR: _HR{
				1, 1,
			},
			BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
		},
		msgV1,
	)
}

func TestProposal_SendV2ReceiveV2(t *testing.T) {
	msgV2 := proposalV2{
		_HR: _HR{
			1, 1,
		},
		BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
		NID:            1,
	}
	bsV2 := codec.MustMarshalToBytes(&msgV2)
	var msgV2Another proposalV2
	codec.MustUnmarshalFromBytes(bsV2, &msgV2Another)
	assert.Equal(t,
		proposalV2{
			_HR: _HR{
				1, 1,
			},
			BlockPartSetID: &PartSetID{1, []byte{0, 1, 2}},
			NID:            1,
		},
		msgV2Another,
	)
}

func TestProposalMessage_Encoding(t *testing.T) {
	msg1 := NewProposalMessageV1()
	msg1.Height = 1
	msg1.Round = 1
	msg1.BlockPartSetID = &PartSetID{1, []byte{0, 1, 2}}
	msg1.POLRound = 1
	bs := codec.MustMarshalToBytes(msg1)

	msg2 := NewProposalMessage()
	msg2.Height = 1
	msg2.Round = 1
	msg2.BlockPartSetID = &PartSetID{1, []byte{0, 1, 2}}
	msg2.POLRound = 1
	bs2 := codec.MustMarshalToBytes(msg2)
	assert.Equal(t, bs, bs2)
}

func TestProposalMessage_SendV1ReceiveV2(t *testing.T) {
	msg1 := NewProposalMessageV1()
	msg1.Height = 1
	msg1.Round = 1
	msg1.BlockPartSetID = &PartSetID{1, []byte{0, 1, 2}}
	msg1.POLRound = 1
	bsV1 := codec.MustMarshalToBytes(&msg1)
	var msg2 *ProposalMessage
	codec.MustUnmarshalFromBytes(bsV1, &msg2)
	assert.EqualValues(t, 1, msg2.Height)
	assert.EqualValues(t, 1, msg2.Round)
	assert.EqualValues(t, &PartSetID{1, []byte{0, 1, 2}}, msg2.BlockPartSetID)
	assert.EqualValues(t, 1, msg2.POLRound)
	assert.EqualValues(t, 0, msg2.NID)
}

func TestProposalMessage_SendV2ReceiveV1(t *testing.T) {
	msg1 := NewProposalMessage()
	msg1.Height = 1
	msg1.Round = 1
	msg1.BlockPartSetID = &PartSetID{1, []byte{0, 1, 2}}
	msg1.POLRound = 1
	bsV1 := codec.MustMarshalToBytes(&msg1)
	var msg2 *ProposalMessageV1
	codec.MustUnmarshalFromBytes(bsV1, &msg2)
	assert.EqualValues(t, 1, msg2.Height)
	assert.EqualValues(t, 1, msg2.Round)
	assert.EqualValues(t, &PartSetID{1, []byte{0, 1, 2}}, msg2.BlockPartSetID)
	assert.EqualValues(t, 1, msg2.POLRound)
}

func TestProposalMessage_SendV2ReceiveV2(t *testing.T) {
	msg1 := NewProposalMessage()
	msg1.Height = 1
	msg1.Round = 1
	msg1.BlockPartSetID = &PartSetID{1, []byte{0, 1, 2}}
	msg1.POLRound = 1
	bsV1 := codec.MustMarshalToBytes(&msg1)
	var msg2 *ProposalMessage
	codec.MustUnmarshalFromBytes(bsV1, &msg2)
	assert.EqualValues(t, 1, msg2.Height)
	assert.EqualValues(t, 1, msg2.Round)
	assert.EqualValues(t, &PartSetID{1, []byte{0, 1, 2}}, msg2.BlockPartSetID)
	assert.EqualValues(t, 1, msg2.POLRound)
	assert.EqualValues(t, 0, msg2.NID)
}
