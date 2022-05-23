package block

import (
	"bytes"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

func (m *manager) verifyBlock(b module.BlockData, prev module.Block) (module.ConsensusInfo, error) {
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
	proves, err := b.NTSDProofList().Proves()
	if err != nil {
		return nil, err
	}
	csi, prevVoters, err := m.verifyProof(prev, b.Votes(), proves)
	if err != nil {
		return nil, err
	}
	if err := b.(base.BlockVersionSpec).VerifyTimestamp(prev, prevVoters); err != nil {
		return nil, err
	}
	return csi, nil
}

// verifyProof returns consensusInfo, prevVoters and nil error if succeeds.
func (m *manager) verifyProof(
	b module.Block,
	votes module.CommitVoteSet,
	ntsdProves [][]byte,
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
		m.srcUID, b.Height(), votes.VoteRound(), bd, ntsdProves,
	)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to verify block id=%x", b.ID())
	}

	return common.NewConsensusInfo(b.Proposer(), validators, voted), validators, nil
}
