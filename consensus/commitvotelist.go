package consensus

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

var vlCodec = codec.BC

type commitVoteItem struct {
	Timestamp int64
	Signature common.Signature
}

type commitVoteList struct {
	Round          int32
	BlockPartSetID *PartSetID
	Items          []commitVoteItem
}

func (vl *commitVoteList) Verify(block module.BlockData, validators module.ValidatorList) error {
	if block.Height() == 0 {
		if len(vl.Items) == 0 {
			return nil
		} else {
			return errors.Errorf("voters for height 0\n")
		}
	}
	vset := make([]bool, validators.Len())
	msg := newVoteMessage()
	msg.Height = block.Height()
	msg.Round = vl.Round
	msg.Type = voteTypePrecommit
	msg.BlockID = block.ID()
	msg.BlockPartSetID = vl.BlockPartSetID
	for i, item := range vl.Items {
		msg.Timestamp = item.Timestamp
		msg.setSignature(item.Signature)
		index := validators.IndexOf(msg.address())
		if index < 0 {
			return errors.Errorf("bad voter %x at index %d in vote list", msg.address(), i)
		}
		if vset[index] {
			return errors.Errorf("vl.Verify: duplicated validator %v\n", msg.address())
		}
		vset[index] = true
	}
	twoThirds := validators.Len() * 2 / 3
	if len(vl.Items) > twoThirds {
		return nil
	}
	return errors.Errorf("votes(%d) <= 2/3 of validators(%d)", len(vl.Items), validators.Len())
}

func (vl *commitVoteList) Bytes() []byte {
	bs, err := vlCodec.MarshalToBytes(vl)
	if err != nil {
		return nil
	}
	return bs
}

func (vl *commitVoteList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

func (vl *commitVoteList) String() string {
	return fmt.Sprintf("VoteList(R=%d,ID=%v,len(Signs)=%d)",
		vl.Round, vl.BlockPartSetID, len(vl.Items))
}

func (vl *commitVoteList) Timestamp() int64 {
	l := len(vl.Items)
	if l == 0 {
		return 0
	}
	ts := make([]int64, l)
	for i := range ts {
		ts[i] = vl.Items[i].Timestamp
	}
	sort.Slice(ts, func(i, j int) bool {
		return ts[i] < ts[j]
	})
	if l%2 == 1 {
		return ts[l/2]
	}
	return (ts[l/2-1] + ts[l/2]) / 2
}

func (vl *commitVoteList) voteList(h int64, bid []byte) *voteList {
	rvl := newVoteList()
	msg := newVoteMessage()
	msg.Height = h
	msg.Round = vl.Round
	msg.Type = voteTypePrecommit
	msg.BlockID = bid
	msg.BlockPartSetID = vl.BlockPartSetID
	for _, item := range vl.Items {
		msg.Timestamp = item.Timestamp
		msg.setSignature(item.Signature)
		rvl.AddVote(msg)
	}
	return rvl
}

func newCommitVoteList(msgs []*voteMessage) *commitVoteList {
	vl := &commitVoteList{}
	l := len(msgs)
	if l > 0 {
		vl.Round = msgs[0].Round
		vl.BlockPartSetID = msgs[0].BlockPartSetID
		vl.Items = make([]commitVoteItem, l)
		blockID := msgs[0].BlockID
		for i := 0; i < l; i++ {
			vl.Items[i] = commitVoteItem{
				msgs[i].Timestamp,
				msgs[i].Signature,
			}
			if !bytes.Equal(blockID, msgs[i].BlockID) {
				log.Panicf("newVoteList: bad block id in messages commonBID:%s msgBID:%s", common.HexPre(blockID), common.HexPre(msgs[i].BlockID))
			}
		}
	}
	return vl
}

// NewCommitVoteSetFromBytes returns VoteList from serialized bytes
func NewCommitVoteSetFromBytes(bs []byte) module.CommitVoteSet {
	vl := &commitVoteList{}
	if bs == nil {
		return vl
	}
	_, err := vlCodec.UnmarshalFromBytes(bs, vl)
	if err != nil {
		return nil
	}
	return vl
}
