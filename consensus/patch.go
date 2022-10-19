package consensus

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type skipPatch struct {
	VoteList voteList
}

func (s *skipPatch) Type() string {
	return module.PatchTypeSkipTransaction
}

func (s *skipPatch) Data() []byte {
	return codec.MustMarshalToBytes(s)
}

func (s *skipPatch) Height() int64 {
	if s.VoteList.Len() == 0 {
		return -1
	}
	return s.VoteList.Get(0).Height - 1
}

func (s *skipPatch) Verify(vl module.ValidatorList, roundLimit int64, nid int) error {
	vset := make([]bool, vl.Len())
	nidBytes := codec.MustMarshalToBytes(nid)
	l := s.VoteList.Len()
	if l == 0 {
		return errors.Errorf("votes(%d) <= 2/3 of validators(%d)", l, vl.Len())
	}
	round := s.VoteList.Get(0).Round
	if round < int32(roundLimit) {
		return errors.Errorf("bad round %d roundLimit %d", round, roundLimit)
	}
	for i := 0; i < l; i++ {
		msg := s.VoteList.Get(i)
		index := vl.IndexOf(msg.address())
		if index < 0 {
			return errors.Errorf("bad voter %v at index %d in vote list", msg.address(), i)
		}
		if vset[index] {
			return errors.Errorf("duplicated validator %v", msg.address())
		}
		if msg.BlockPartSetIDAndNTSVoteCount != nil {
			return errors.Errorf("BPSID is not nil for validator %v", msg.address())
		}
		if !bytes.Equal(msg.BlockID, nidBytes) {
			return errors.Errorf("bad nid %x for validator %v", msg.BlockID, msg.address())
		}
		if msg.Round != round {
			return errors.Errorf("different round %d %d in vote list", round, msg.Round)
		}
		vset[index] = true
	}
	f := vl.Len() / 3
	if l > f {
		return nil
	}
	return errors.Errorf("votes(%d) <= 1/3 of validators(%d)", l, vl.Len())
}

func newSkipPatch(vl *voteList) *skipPatch {
	return &skipPatch{VoteList: *vl}
}

func DecodePatch(t string, bs []byte) (module.Patch, error) {
	var err error
	var patch module.Patch
	switch t {
	case module.PatchTypeSkipTransaction:
		patch = &skipPatch{}
		_, err = codec.UnmarshalFromBytes(bs, patch)
	default:
		err = errors.ErrUnsupported
	}
	if err != nil {
		return nil, err
	}
	return patch, nil
}
