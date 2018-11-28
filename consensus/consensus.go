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
	round  int32
	step   step
}

type consensus struct {
	hrs

	bm     module.BlockManager
	wallet module.Wallet
	dm     module.Membership
	mutex  sync.Mutex

	msgQueue []message

	lastBlock        module.Block
	validators       module.ValidatorList
	votes            module.VoteList
	hvs              heightVoteSet
	nextProposeTime  time.Time
	lockedRound      int32
	lockedBlockParts PartSet
	lockedBlock      module.Block

	proposalPOLRound  int32
	currentBlockParts PartSet
	currentBlock      module.Block

	timer              *time.Timer
	cancelBlockRequest func() bool
}

func NewConsensus(c module.Chain, bm module.BlockManager, nm module.NetworkManager) module.Consensus {
	return &consensus{
		bm:     bm,
		wallet: c.Wallet(),
		dm:     nm.GetMembership(""),
	}
}

func (cs *consensus) resetForNewHeight(prevBlock module.Block, votes module.VoteList) {
	cs.height = prevBlock.Height() + 1
	cs.lastBlock = prevBlock
	cs.validators = cs.lastBlock.NextValidators()
	cs.votes = votes
	cs.hvs.reset(cs.validators.Len())
	cs.lockedRound = -1
	cs.lockedBlockParts = nil
	cs.lockedBlock = nil
	cs.resetForNewRound(0)
}

func (cs *consensus) resetForNextHeight() {
	votes := cs.hvs.votesFor(cs.round, voteTypePrecommit).voteList()
	cs.resetForNewHeight(cs.currentBlock, votes)
}

func (cs *consensus) resetForNewRound(round int32) {
	cs.proposalPOLRound = -1
	cs.currentBlockParts = nil
	cs.currentBlock = nil
	cs.round = round
	cs.enterPropose()
}

func (cs *consensus) resetForNewStep() {
	if cs.cancelBlockRequest != nil {
		cs.cancelBlockRequest()
		cs.cancelBlockRequest = nil
	}
	if cs.timer != nil {
		cs.timer.Stop()
		cs.timer = nil
	}
}

func (cs *consensus) enqueueMessage(msg message) {
	cs.msgQueue = append(cs.msgQueue, msg)
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
		cs.enqueueMessage(msg)
		return true, nil
	}
	return msg.dispatch(cs)
}

func (cs *consensus) OnError() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
}

func (cs *consensus) receiveProposal(msg *proposalMessage) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}
	if msg.Round > cs.round {
		cs.enqueueMessage(msg)
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
	if cs.currentBlockParts != nil {
		return false, nil
	}
	cs.proposalPOLRound = msg.proposal.POLRound
	cs.currentBlockParts = newPartSetFromID(&msg.proposal.BlockPartsHeader)

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.enterNewHeight()
	}
	return true, nil
}

func (cs *consensus) receiveBlockPart(msg *blockPartMessage) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}
	if msg.Round > cs.round || (msg.Round == cs.round && cs.currentBlockParts == nil) {
		cs.enqueueMessage(msg)
		return true, nil
	}

	bp, err := newPart(msg.BlockPart)
	if err != nil {
		return false, err
	}
	if err := cs.currentBlockParts.AddPart(bp); err != nil {
		return false, err
	}

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.enterNewHeight()
	}
	return true, nil
}

func (cs *consensus) receiveVote(msg *voteMessage) (bool, error) {
	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return false, nil
	}
	if added := cs.hvs.add(index, msg); !added {
		return false, nil
	}
	return false, nil
}

/*
func (cs *consensus) receiveVote(msg *voteMessage) (bool, error) {
	if msg.Type == voteTypePrevote {
		return cs.receivePrevote(msg)
	} else {
		return cs.receivePrecommit(msg)
	}
}
*/

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
		if msg.Round != cs.proposalPOLRound {
			return true, nil
		}
		prevotes := cs.hvs._votes[msg.Round][voteTypePrevote]
		bid, _ := prevotes.getOverTwoThirdsBlockID()
		if bid != nil {
			cs.enterPrevote()
		}
		return true, nil
	}

	prevotes := cs.hvs._votes[msg.Round][voteTypePrevote]
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

	precommits := cs.hvs._votes[msg.Round][voteTypePrecommit]
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
	cs.resetForNewRound(cs.round + 1)
}

func (cs *consensus) dispatchQueuedMessage() {
	msgQueue := cs.msgQueue
	cs.msgQueue = nil
	for _, msg := range msgQueue {
		if msg.height() == cs.height && msg.round() == cs.round {
			msg.dispatch(cs)
		} else {
			cs.msgQueue = append(cs.msgQueue, msg)
		}
	}
}

func (cs *consensus) enterPropose() {
	cs.resetForNewStep()
	cs.step = stepPropose

	hrs := cs.hrs
	cs.timer = time.AfterFunc(timeoutPropose, func() {
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		if cs.hrs != hrs {
			return
		}
		// cannot send proposal
		cs.enterPrevote()
	})

	if cs.isProposer() {
		if cs.lockedBlockParts.IsComplete() {
			cs.sendProposal(cs.lockedBlockParts, cs.lockedRound)
			cs.dispatchQueuedMessage()
		} else {
			cs.bm.Propose(cs.lastBlock.ID(), cs.votes,
				func(blk module.Block, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.hrs != hrs {
						return
					}
					psb := newPartSetBuffer()
					blk.MarshalHeader(psb)
					blk.MarshalBody(psb)
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.enterPrevote()
				},
			)
		}
	}
	cs.dispatchQueuedMessage()
}

func (cs *consensus) enterPrevoteForRound(round int32) {
	cs.resetForNewRound(round)
	if cs.step < stepPrevote {
		cs.enterPrevote()
	}
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep()
	cs.step = stepPrevote

	if cs.lockedBlockParts != nil && cs.lockedRound < cs.proposalPOLRound {
		cs.lockedBlockParts = nil
		cs.lockedRound = -1
	}

	if cs.lockedBlockParts != nil {
		//		cs.sendVote(voteTypePrevote, cs.lockedBlockID)
	} else if cs.currentBlockParts.IsComplete() {
		hrs := cs.hrs
		var err error
		cs.cancelBlockRequest, err = cs.bm.Import(cs.currentBlockParts.NewReader(), func(blk module.Block, err error) {
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

	prevotes := cs.hvs._votes[cs.round][voteTypePrevote]
	if prevotes.hasOverTwoThirds() {
		cs.enterPrevoteWait()
	}
}

func (cs *consensus) enterPrevoteWait() {
	cs.resetForNewStep()
	cs.step = stepPrevoteWait

	prevotes := cs.hvs._votes[cs.round][voteTypePrevote]
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

func (cs *consensus) enterPrecommitForRound(round int32) {
	cs.resetForNewRound(round)
	cs.enterPrecommit()
}

func (cs *consensus) enterPrecommit() {
	cs.resetForNewStep()
	cs.step = stepPrecommit

	prevotes := cs.hvs._votes[cs.round][voteTypePrevote]
	bid, overTwoThirds := prevotes.getOverTwoThirdsBlockID()
	if overTwoThirds && bid != nil {
		// TODO check blockID
		if cs.currentBlockParts.IsComplete() {
			//			cs.lockedBlockID = bid
			cs.lockedRound = cs.round
			cs.lockedBlockParts = cs.currentBlockParts
			cs.sendVote(voteTypePrecommit, bid)
		} else {
			// TODO clear proposal if blockID doesn't match
			//			cs.lockedBlockID = nil
			cs.lockedRound = -1
			cs.sendVote(voteTypePrecommit, nil)
		}
	} else if overTwoThirds && bid == nil {
		//		cs.lockedBlockID = nil
		cs.lockedRound = -1
		cs.sendVote(voteTypePrecommit, nil)
	} else {
		cs.sendVote(voteTypePrecommit, nil)
	}

	precommits := cs.hvs._votes[cs.round][voteTypePrecommit]
	if precommits.hasOverTwoThirds() {
		cs.enterPrecommitWait()
	}
}

func (cs *consensus) enterPrecommitWait() {
	cs.resetForNewStep()
	cs.step = stepPrecommitWait

	precommits := cs.hvs._votes[cs.round][voteTypePrecommit]
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
	cs.resetForNewStep()
	cs.step = stepCommit
	cs.nextProposeTime = time.Now().Add(timeoutCommit)
	if cs.currentBlockParts.IsComplete() {
		// TODO import and finalize
		if cs.currentBlock == nil {
		}
		cs.enterNewHeight()
	}
}

func (cs *consensus) enterNewHeight() {
	now := time.Now()
	if cs.nextProposeTime.Before(now) {
		cs.resetForNextHeight()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			if cs.hrs != hrs {
				return
			}
			cs.resetForNextHeight()
		})
	}
}

func (cs *consensus) sendProposal(blockParts PartSet, polRound int32) {
	// TODO sign, send proposal and blockParts, receive
}

func (cs *consensus) sendVote(vt voteType, bid []byte) {
	// TODO sign, send, receive
}

func getProposerIndex(
	validators module.ValidatorList,
	height int64,
	round int32,
) int {
	return int((height + int64(round)) % int64(validators.Len()))
}

func (cs *consensus) getProposerIndex(height int64, round int32) int {
	return getProposerIndex(cs.validators, height, round)
}

func (cs *consensus) isProposerFor(height int64, round int32) bool {
	pindex := getProposerIndex(cs.validators, height, round)
	v, _ := cs.validators.Get(pindex)
	if v == nil {
		return false
	}
	return bytes.Equal(v.PublicKey(), cs.wallet.PublicKey())
}

func (cs *consensus) isProposer() bool {
	return cs.isProposerFor(cs.height, cs.round)
}

func (cs *consensus) isProposalAndPOLPrevotesComplete() bool {
	if !cs.currentBlockParts.IsComplete() {
		return false
	}
	if cs.proposalPOLRound > 0 {
		prevotes := cs.hvs._votes[cs.proposalPOLRound][voteTypePrevote]
		if bid, _ := prevotes.getOverTwoThirdsBlockID(); bid != nil {
			return true
		}
		return false
	}
	return true
}

func (cs *consensus) Start() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	gblks, err := cs.bm.FinalizeGenesisBlocks(
		zeroAddress,
		time.Time{},
		nil,
	)
	if err != nil {
		return
	}
	lastFinalizedBlock := gblks[len(gblks)-1]
	cs.resetForNewHeight(lastFinalizedBlock, newVoteList(nil))
	cs.dm.RegistReactor("consensus", cs, protocols)
}
