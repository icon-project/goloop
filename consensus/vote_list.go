package consensus

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

var vlCodec = codec.MP

type voteList struct {
	Round      int32
	Signatures []common.Signature
}

func (vl *voteList) Verify(block module.Block) error {
	var msg voteMessage
	msg.Height = block.Height()
	msg.Round = vl.Round
	msg.Type = voteTypePrecommit
	msg.BlockID = block.ID()
	validators := block.NextValidators()
	for i, sig := range vl.Signatures {
		msg.Signature = sig
		index := validators.IndexOf(msg.address())
		if index < 0 {
			return errors.Errorf("bad voter at index %d in vote list", i)
		}
	}
	twoThirds := validators.Len() * 2 / 3
	if len(vl.Signatures) > twoThirds {
		return nil
	}
	return errors.Errorf("votes(%d) <= 2/3 of validators(%d)", len(vl.Signatures), validators.Len())
}

func (vl *voteList) Bytes() []byte {
	bs, err := vlCodec.MarshalToBytes(vl)
	if err != nil {
		return nil
	}
	return bs
}

func (vl *voteList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

func newVoteList(msgs []*voteMessage) *voteList {
	vl := &voteList{}
	l := len(msgs)
	if l > 0 {
		vl.Round = msgs[0].Round
		vl.Signatures = make([]common.Signature, l)
		for i := 0; i < l; i++ {
			vl.Signatures[i] = msgs[i].Signature
		}
	}
	return vl
}

// NewVoteListFromBytes returns VoteList from serialized bytes
func NewVoteListFromBytes(bs []byte) module.VoteList {
	var vl *voteList
	_, err := vlCodec.UnmarshalFromBytes(bs, &vl)
	if err != nil {
		return nil
	}
	return vl
}
