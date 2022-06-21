package block

import (
	"bytes"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

// verifyNewBlock verifies new block b. prev must be the last finalized block.
// the function does not verify calculation result.
func (m *manager) verifyNewBlock(b module.BlockData, prev module.Block) (module.ConsensusInfo, error) {
	var prevResult []byte
	if prev != nil {
		prevResult = prev.Result()
	}
	if b.Version() != m.sm.GetNextBlockVersion(prevResult) {
		return nil, errors.Errorf("bad block version=%d exp=%d", b.Version(), m.sm.GetNextBlockVersion(prevResult))
	}
	if b.Height() != prev.Height()+1 {
		return nil, errors.New("bad height")
	}
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return nil, errors.New("bad prev ID")
	}
	csi, prevVoters, err := m.verifyProofForLastBlock(prev, b.Votes())
	if err != nil {
		return nil, err
	}
	if err := b.(base.BlockVersionSpec).VerifyTimestamp(prev, prevVoters); err != nil {
		return nil, err
	}
	return csi, nil
}

// verifyProofForLastBlock returns consensusInfo, prevVoters and nil error if
// succeeds. b must be the last finalized block.
func (m *manager) verifyProofForLastBlock(
	b module.Block,
	votes module.CommitVoteSet,
) (module.ConsensusInfo, module.ValidatorList, error) {
	validators, err := b.(base.BlockVersionSpec).GetVoters(m.handlerContext)
	if err != nil {
		return nil, nil, errors.InvalidStateError.Wrapf(err, "fail to get validators")
	}
	voted, err := votes.VerifyBlock(b, validators)
	if err != nil {
		return nil, nil, err
	}
	bd, err := b.BTPDigest()
	if err != nil {
		return nil, nil, errors.InvalidStateError.Wrapf(err, "fail to get digest id=%x", b.ID())
	}
	err = m.pcmForLastBlock.Verify(
		m.srcUID, b.Height(), votes.VoteRound(), bd, votes,
	)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to verify block id=%x", b.ID())
	}

	return common.NewConsensusInfo(b.Proposer(), validators, voted), validators, nil
}
