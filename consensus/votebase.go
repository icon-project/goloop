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

package consensus

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type VoteType byte

const (
	VoteTypePrevote VoteType = iota
	VoteTypePrecommit
	numberOfVoteTypes
)

func (vt VoteType) String() string {
	switch vt {
	case VoteTypePrevote:
		return "PreVote"
	case VoteTypePrecommit:
		return "PreCommit"
	default:
		return "Unknown"
	}
}

type blockVoteBase struct {
	_HR
	Type                          VoteType
	BlockID                       []byte
	BlockPartSetIDAndNTSVoteCount *PartSetIDAndAppData
}

type ntsVoteBase module.NTSHashEntryFormat

func (ntsVote ntsVoteBase) String() string {
	return fmt.Sprintf(
		"NTID=%d NTSHash=%x",
		ntsVote.NetworkTypeID,
		ntsVote.NetworkTypeSectionHash,
	)
}

type voteBase struct {
	blockVoteBase
	NTSVoteBases   []ntsVoteBase
	decisionDigest []byte
}

type roundDecision struct {
	BlockID                  []byte
	BlockPartSetIDAndAppData *PartSetIDAndAppData
	NTSVoteBases             []ntsVoteBase
}

func (v *voteBase) SetRoundDecision(bid []byte, bpsIDAndNTSVoteCount *PartSetIDAndAppData, ntsVoteBases []ntsVoteBase) {
	v.BlockID = bid
	v.BlockPartSetIDAndNTSVoteCount = bpsIDAndNTSVoteCount
	v.NTSVoteBases = ntsVoteBases
	v.decisionDigest = nil
}

// RoundDecisionDigest returns digest for a vote. The digest values of two
// votes are different if their BlockID, BlockPartSetIDAndAppData, len(NTSVotes),
// NTSVotes[i].NetworkTypeID or NTSVotes[i].NetworkTypeSectionHash is
// different where 0 <= i < len(NTSVotes).
func (v *voteBase) RoundDecisionDigest() []byte {
	if v.decisionDigest == nil {
		format := roundDecision{
			BlockID:                  v.BlockID,
			BlockPartSetIDAndAppData: v.BlockPartSetIDAndNTSVoteCount,
			NTSVoteBases:             v.NTSVoteBases,
		}
		// Sometimes we make zero length array for NTSVoteBases. Normalize for
		// consistent hash value.
		if len(format.NTSVoteBases) == 0 {
			format.NTSVoteBases = nil
		}
		v.decisionDigest = crypto.SHA3Sum256(codec.MustMarshalToBytes(&format))
	}
	return v.decisionDigest
}

func (v *voteBase) Equal(v2 *voteBase) bool {
	return v.Height == v2.Height &&
		v.Round == v2.Round &&
		v.Type == v2.Type &&
		bytes.Equal(v.RoundDecisionDigest(), v2.RoundDecisionDigest())
}

func (v voteBase) String() string {
	if len(v.NTSVoteBases) == 0 {
		return fmt.Sprintf(
			"{%s H:%d R:%d BID:%v BPSID:%v}",
			v.Type,
			v.Height,
			v.Round,
			common.HexPre(v.BlockID),
			v.BlockPartSetIDAndNTSVoteCount,
		)
	}
	return fmt.Sprintf(
		"{%s H:%d R:%d BID:%v BPSID:%v NTSVoteBases:%v}",
		v.Type,
		v.Height,
		v.Round,
		common.HexPre(v.BlockID),
		v.BlockPartSetIDAndNTSVoteCount,
		v.NTSVoteBases,
	)
}

func (v *voteBase) NID() (uint32, error) {
	if v.BlockPartSetIDAndNTSVoteCount == nil {
		var nid int32
		_, err := codec.UnmarshalFromBytes(v.BlockID, &nid)
		if err != nil {
			return 0, err
		}
		return uint32(nid), nil
	}
	nid, _ := destructPSIDAppData(v.BlockPartSetIDAndNTSVoteCount.AppData())
	return nid, nil
}
