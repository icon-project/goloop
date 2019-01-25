package consensus

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

var vlCodec = codec.MP

type commitVoteList struct {
	Round          int32
	BlockPartSetID *PartSetID
	Signatures     []common.Signature
}

func (vl *commitVoteList) Verify(block module.Block, validators module.ValidatorList) error {
	// TODO height should be 0
	if block.Height() == 0 {
		if len(vl.Signatures) == 0 {
			return nil
		} else {
			return errors.Errorf("voters for height 1\n")
		}
	}
	vset := make([]bool, validators.Len())
	msg := newVoteMessage()
	msg.Height = block.Height()
	msg.Round = vl.Round
	msg.Type = voteTypePrecommit
	msg.BlockID = block.ID()
	msg.BlockPartSetID = vl.BlockPartSetID
	for i, sig := range vl.Signatures {
		msg.setSignature(sig)
		index := validators.IndexOf(msg.address())
		if index < 0 {
			logger.Println(msg)
			return errors.Errorf("bad voter %x at index %d in vote list", msg.address(), i)
		}
		if vset[index] {
			logger.Printf("voteList: %v\n", vl)
			return errors.Errorf("vl.Verify: duplicated validator %v\n", msg.address())
		}
		vset[index] = true
	}
	twoThirds := validators.Len() * 2 / 3
	if len(vl.Signatures) > twoThirds {
		return nil
	}
	return errors.Errorf("votes(%d) <= 2/3 of validators(%d)", len(vl.Signatures), validators.Len())
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
		vl.Round, vl.BlockPartSetID, len(vl.Signatures))
}

func (vl *commitVoteList) voteList(h int64, bid []byte) *voteList {
	rvl := newVoteList()
	msg := newVoteMessage()
	msg.Height = h
	msg.Round = vl.Round
	msg.Type = voteTypePrecommit
	msg.BlockID = bid
	msg.BlockPartSetID = vl.BlockPartSetID
	for _, sig := range vl.Signatures {
		msg.setSignature(sig)
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
		vl.Signatures = make([]common.Signature, l)
		blockID := msgs[0].BlockID
		for i := 0; i < l; i++ {
			vl.Signatures[i] = msgs[i].Signature
			if !bytes.Equal(blockID, msgs[i].BlockID) {
				logger.Panicf("newVoteList: bad block id in messages <%x> <%x>", blockID, msgs[i].BlockID)
			}
		}
	}
	return vl
}

// NewCommitVoteSetFromBytes returns VoteList from serialized bytes
func NewCommitVoteSetFromBytes(bs []byte) module.CommitVoteSet {
	vl := &commitVoteList{}
	_, err := vlCodec.UnmarshalFromBytes(bs, vl)
	if err != nil {
		return nil
	}
	return vl
}
