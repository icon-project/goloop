/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package consensus

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

func matchNID(nid1, nid2 uint32) bool {
	if nid1 == 0 || nid2 == 0 {
		return true
	}
	return nid1 == nid2
}

type dsVote struct {
	msg *VoteMessage
}

func (v *dsVote) Type() string {
	return module.DSTVote
}

func (v *dsVote) Height() int64 {
	return v.msg.Height
}

func (v *dsVote) Bytes() []byte {
	return msgCodec.MustMarshalToBytes(v.msg)
}

func (v *dsVote) Signer() []byte {
	return v.msg.address().ID()
}

func (v *dsVote) ValidateNetwork(nid int) bool {
	return true
}

func (v *dsVote) IsConflictWith(other module.DoubleSignData) bool {
	v2, ok := other.(*dsVote)
	if !ok {
		return false
	}
	if v == nil || v2 == nil {
		return false
	}
	nid1, _ := v.msg.NID()
	nid2, _ := v.msg.NID()
	if !matchNID(nid1, nid2) {
		return false
	}
	if (v2.msg.Type != v.msg.Type) ||
		v2.msg.Height != v.msg.Height ||
		v2.msg.Round != v.msg.Round ||
		!bytes.Equal(v2.Signer(), v.Signer()) {
		return false
	}

	return !bytes.Equal(v.msg.hash(), v2.msg.hash())
}

func newDoubleSignDataWithVoteMessage(msg *VoteMessage) (module.DoubleSignData, error) {
	if err := msg.verify(); err != nil {
		return nil, err
	}
	return &dsVote{
		msg: msg,
	}, nil
}

type dsProposal struct {
	msg *ProposalMessage
}

func (d *dsProposal) Type() string {
	return module.DSTProposal
}

func (d *dsProposal) Height() int64 {
	return d.msg.Height
}

func (d *dsProposal) Signer() []byte {
	return d.msg.address().ID()
}

func (v *dsProposal) ValidateNetwork(nid int) bool {
	return true
}

func (d *dsProposal) Bytes() []byte {
	return codec.MustMarshalToBytes(d.msg)
}

func (d *dsProposal) IsConflictWith(other module.DoubleSignData) bool {
	d2, ok := other.(*dsProposal)
	if !ok {
		return false
	}
	if d == nil || d2 == nil {
		return false
	}
	if !matchNID(d.msg.NID, d2.msg.NID) {
		return false
	}
	if d.msg.Height != d2.msg.Height ||
		d.msg.Round != d2.msg.Round ||
		!bytes.Equal(d2.Signer(), d.Signer()) {
		return false
	}
	return !bytes.Equal(d.msg.hash(), d2.msg.hash())
}

func newDoubleSignDataWithProposalMessage(msg *ProposalMessage) (module.DoubleSignData, error) {
	if err := msg.verify(); err != nil {
		return nil, err
	}
	return &dsProposal{
		msg: msg,
	}, nil
}

func DecodeDoubleSignData(t string, d []byte) (module.DoubleSignData, error) {
	switch t {
	case module.DSTVote:
		msg := newVoteMessage()
		_, err := msgCodec.UnmarshalFromBytes(d, msg)
		if err != nil {
			return nil, errors.IllegalArgumentError.Wrapf(err, "InvalidVoteMessage")
		}
		if ds, err := newDoubleSignDataWithVoteMessage(msg); err != nil {
			return nil, errors.IllegalArgumentError.Wrapf(err, "InvalidVoteMessage")
		} else {
			return ds, nil
		}
	case module.DSTProposal:
		msg := NewProposalMessage()
		_, err := msgCodec.UnmarshalFromBytes(d, msg)
		if err != nil {
			return nil, errors.IllegalArgumentError.Wrapf(err, "InvalidVoteMessage")
		}
		if ds, err := newDoubleSignDataWithProposalMessage(msg); err != nil {
			return nil, errors.IllegalArgumentError.Wrapf(err, "InvalidVoteMessage")
		} else {
			return ds, nil
		}
	default:
		return nil, errors.IllegalArgumentError.Errorf("InvalidDoubleSignDataType(type=%s)", t)
	}
}
