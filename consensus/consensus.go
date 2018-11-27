package consensus

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

var zeroAddress = common.NewAddress(make([]byte, common.AddressBytes))

const (
	timeoutPropose   = time.Second * 3
	timeoutPrevote   = time.Second * 3
	timeoutPrecommit = time.Second * 3
	timeoutCommit    = time.Second * 3
)

type hrs struct {
	height int64
	round  int
	step   step
}

type consensus struct {
	hrs

	bm         module.BlockManager
	wallet     module.Wallet
	membership module.Membership
	mutex      sync.Mutex

	validators      module.ValidatorList
	lastBlock       module.Block
	hvs             heightVoteSet
	lockedRound     int
	lockedBlockID   []byte
	nextProposeTime time.Time

	proposal         *proposal
	blockParts       *blockPartSet
	currentBlock     module.Block
	receiveLaterList []message

	timer              *time.Timer
	cancelBlockRequest func() bool
}

func NewConsensus(manager module.BlockManager) module.Consensus {
	return &consensus{
		bm: manager,
	}
}

func (cs *consensus) resetHeight() {
}

func (cs *consensus) resetRound() {
	cs.proposal = nil
	cs.blockParts = nil
	cs.currentBlock = nil
}

func (cs *consensus) resetStep() {
	if cs.cancelBlockRequest != nil {
		cs.cancelBlockRequest()
		cs.cancelBlockRequest = nil
	}
	if cs.timer != nil {
		cs.timer.Stop()
		cs.timer = nil
	}
}

func (cs *consensus) receiveLater(msg message) {
	cs.receiveLaterList = append(cs.receiveLaterList, msg)
}

func (cs *consensus) OnReceive(
	sp module.ProtocolInfo,
	bs []byte,
	id module.PeerID,
) (bool, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	msg, err := unmarshalMessage(sp, bs)
	if err != nil {
		return false, err
	}
	if err := msg.verify(); err != nil {
		return false, err
	}
	if msg.height() < cs.height {
		return true, nil
	}
	if msg.height() > cs.height {
		cs.receiveLater(msg)
		return true, nil
	}
	return msg.dispatch(cs)
}

func (cs *consensus) receiveProposal(msg *proposalMessage) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}
	if msg.Round > cs.round {
		cs.receiveLater(msg)
		return true, nil
	}
	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return false, errors.New("bad proposer")
	}
	if cs.getProposerIndex(cs.height, cs.round) != index {
		return false, errors.New("bad proposer")
	}

	// TODO receive multiple proposal
	if cs.proposal != nil {
		return false, nil
	}

	cs.proposal = &msg.proposal

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.blockParts.isComplete() {
		cs.enterNewHeight()
	}
	return true, nil
}

func (cs *consensus) receiveBlockPart(msg *blockPartMessage) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}
	if msg.Round > cs.round {
		cs.receiveLater(msg)
		return true, nil
	}

	added, err := cs.blockParts.add(msg.Index, msg.Proof)
	if err != nil {
		return false, err
	}
	if !added {
		return false, nil
	}

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.blockParts.isComplete() {
		cs.enterNewHeight()
	}
	return true, nil
}

func (cs *consensus) receiveVote(msg *voteMessage) (bool, error) {
	if msg.Type == voteTypePrevote {
		return cs.receivePrevote(msg)
	} else {
		return cs.receivePrecommit(msg)
	}
}

func (cs *consensus) receivePrevote(msg *voteMessage) (bool, error) {
	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return false, nil
	}
	if added := cs.hvs.add(index, msg); !added {
		return false, nil
	}

	if msg.Round < cs.round {
		if cs.step >= stepPrevote {
			return true, nil
		}
		if !cs.isProposalAndPOLPrevotesComplete() {
			return false, nil
		}
		if msg.Round != cs.proposal.POLRound {
			return true, nil
		}
		prevotes := cs.hvs.votes[msg.Round][voteTypePrevote]
		bid, _ := prevotes.getOverTwoThirdsBlockID()
		if bid != nil {
			cs.enterPrevote()
		}
		return true, nil
	}

	prevotes := cs.hvs.votes[msg.Round][voteTypePrevote]
	if !prevotes.hasOverTwoThirds() {
		return true, nil
	}
	if cs.round < msg.Round || cs.step < stepPrevote {
		cs.enterPrevoteForRound(msg.Round)
		return true, nil
	}
	if cs.step == stepPrevote {
		cs.enterPrevoteWait()
		return true, nil
	}
	if cs.step == stepPrevoteWait {
		_, overTwoThirds := prevotes.getOverTwoThirdsBlockID()
		if !overTwoThirds {
			return true, nil
		}
		cs.enterPrecommit()
	}
	return true, nil
}

func (cs *consensus) receivePrecommit(msg *voteMessage) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}

	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return false, nil
	}
	if added := cs.hvs.add(index, msg); !added {
		return false, nil
	}

	precommits := cs.hvs.votes[msg.Round][voteTypePrecommit]
	if !precommits.hasOverTwoThirds() {
		return true, nil
	}
	if cs.round < msg.Round || cs.step < stepPrecommit {
		cs.enterPrecommitForRound(msg.Round)
		return true, nil
	}
	if cs.step == stepPrecommit {
		cs.enterPrecommitWait()
		return true, nil
	}
	if cs.step == stepPrecommitWait {
		bid, overTwoThirds := precommits.getOverTwoThirdsBlockID()
		if !overTwoThirds {
			return true, nil
		}
		if bid == nil {
			cs.enterProposeForNextRound()
		} else {
			cs.enterCommit(bid)
		}
	}
	return true, nil
}

func (cs *consensus) enterProposeForNextRound() {
	cs.round++
	cs.resetRound()
	cs.enterPropose()
}

func (cs *consensus) enterPropose() {
}

func (cs *consensus) enterPrevoteForRound(round int) {
	cs.round = round
	cs.resetRound()
	cs.enterPrevote()
}

func (cs *consensus) enterPrevote() {
	cs.resetStep()
	cs.step = stepPrevote

	if cs.lockedBlockID != nil && cs.proposal != nil &&
		cs.lockedRound < cs.proposal.POLRound {
		cs.lockedBlockID = nil
		cs.lockedRound = -1
	}

	if cs.lockedBlockID != nil {
		cs.sendVote(voteTypePrevote, cs.lockedBlockID)
	} else if cs.blockParts.isComplete() {
		hrs := cs.hrs
		var err error
		cs.cancelBlockRequest, err = cs.bm.Import(cs.blockParts.newReader(), func(blk module.Block, err error) {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}

			if err == nil {
				cs.currentBlock = blk
				cs.sendVote(voteTypePrevote, blk.ID())
			} else {
				cs.sendVote(voteTypePrevote, nil)
			}
		})
		if err != nil {
			cs.sendVote(voteTypePrevote, nil)
		}
	} else {
		cs.sendVote(voteTypePrevote, nil)
	}

	prevotes := cs.hvs.votes[cs.round][voteTypePrevote]
	if prevotes.hasOverTwoThirds() {
		cs.enterPrevoteWait()
	}
}

func (cs *consensus) enterPrevoteWait() {
	cs.resetStep()
	cs.step = stepPrevoteWait

	prevotes := cs.hvs.votes[cs.round][voteTypePrevote]
	_, overTwoThirds := prevotes.getOverTwoThirdsBlockID()
	if overTwoThirds {
		cs.enterPrecommit()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(timeoutPrevote, func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterPrecommit()
		})
	}
}

func (cs *consensus) enterPrecommitForRound(round int) {
	cs.round = round
	cs.resetRound()
	cs.enterPrecommit()
}

func (cs *consensus) enterPrecommit() {
	cs.resetStep()
	cs.step = stepPrecommit

	prevotes := cs.hvs.votes[cs.round][voteTypePrevote]
	bid, overTwoThirds := prevotes.getOverTwoThirdsBlockID()
	if overTwoThirds && bid != nil {
		cs.lockedBlockID = bid
		cs.lockedRound = cs.round
		cs.sendVote(voteTypePrecommit, bid)
	} else if overTwoThirds && bid == nil {
		cs.lockedBlockID = nil
		cs.lockedRound = -1
		cs.sendVote(voteTypePrecommit, nil)
	} else {
		cs.sendVote(voteTypePrecommit, nil)
	}

	precommits := cs.hvs.votes[cs.round][voteTypePrecommit]
	if precommits.hasOverTwoThirds() {
		cs.enterPrecommitWait()
	}
}

func (cs *consensus) enterPrecommitWait() {
	cs.resetStep()
	cs.step = stepPrecommitWait

	precommits := cs.hvs.votes[cs.round][voteTypePrecommit]
	bid, overTwoThrids := precommits.getOverTwoThirdsBlockID()
	if overTwoThrids && bid != nil {
		cs.enterCommit(bid)
	} else if overTwoThrids && bid == nil {
		cs.enterProposeForNextRound()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(timeoutPrecommit, func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterProposeForNextRound()
		})
	}
}

func (cs *consensus) enterCommit(bid []byte) {
	cs.resetStep()
	cs.step = stepCommit
	cs.nextProposeTime = time.Now().Add(timeoutCommit)
	if cs.blockParts.isComplete() {
		// TODO import and finalize
		if cs.currentBlock == nil {
		}
		cs.enterNewHeight()
	}
}

func (cs *consensus) enterNewHeight() {
	now := time.Now()
	if cs.nextProposeTime.Before(now) {
		cs.resetHeight()
		cs.height++
		cs.resetRound()
		cs.round = 0
		cs.enterPropose()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			if cs.hrs != hrs {
				return
			}
			cs.resetHeight()
			cs.height++
			cs.resetRound()
			cs.round = 0
			cs.enterPropose()
		})
	}
}

func (cs *consensus) sendVote(vt voteType, bid []byte) {
	// TODO
}

func getProposerIndex(
	validators module.ValidatorList,
	height int64,
	round int,
) int {
	return int((height + int64(round)) % int64(validators.Len()))
}

func (cs *consensus) getProposerIndex(height int64, round int) int {
	return getProposerIndex(cs.validators, height, round)
}

func (cs *consensus) isProposer(height int64, round int) bool {
	pindex := getProposerIndex(cs.validators, height, round)
	v, _ := cs.validators.Get(pindex)
	if v == nil {
		return false
	}
	return bytes.Equal(v.PublicKey(), cs.wallet.PublicKey())
}

func (cs *consensus) isProposalAndPOLPrevotesComplete() bool {
	// TODO
	return false
}

func (cs *consensus) Start() {
	gblks, err := cs.bm.FinalizeGenesisBlocks(
		zeroAddress,
		time.Time{},
		nil,
	)
	if err != nil {
		return
	}
	// last finalized block
	lfb := gblks[len(gblks)-1]
	cs.height = lfb.Height()
	cs.round = 0
}
