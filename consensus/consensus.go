package consensus

import (
	"bytes"
	"errors"
	"log"
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

type blockPartSet struct {
	PartSet
	block module.Block
}

type consensus struct {
	hrs

	bm     module.BlockManager
	wallet module.Wallet
	dm     module.Membership
	mutex  sync.Mutex

	msgQueue []message

	lastBlock         module.Block
	validators        module.ValidatorList
	votes             module.VoteList
	hvs               heightVoteSet
	nextProposeTime   time.Time
	lockedRound       int32
	lockedBlockParts  *blockPartSet
	proposalPOLRound  int32
	currentBlockParts *blockPartSet

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
	cs.resetForNewRound(0)
}

func (cs *consensus) resetForNextHeight() {
	votes := cs.hvs.votesFor(cs.round, voteTypePrecommit).voteList()
	cs.resetForNewHeight(cs.currentBlockParts.block, votes)
}

func (cs *consensus) resetForNewRound(round int32) {
	cs.proposalPOLRound = -1
	cs.currentBlockParts = nil
	cs.round = round
	cs.step = stepPrepropose
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
	cs.currentBlockParts = &blockPartSet{
		PartSet: newPartSetFromID(&msg.proposal.BlockPartSetID),
	}

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
	added, votes := cs.hvs.add(index, msg)
	if !added {
		return false, nil
	}
	if !votes.hasOverTwoThirds() {
		return true, nil
	}
	if msg.Type == voteTypePrevote {
		return cs.handlePrevoteMessage(msg, votes)
	} else {
		return cs.handlePrecommitMessage(msg, votes)
	}
}

func (cs *consensus) handlePrevoteMessage(msg *voteMessage, prevotes *voteSet) (bool, error) {
	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()
	if ok {
		if cs.lockedRound < msg.Round && cs.lockedBlockParts != nil && !cs.lockedBlockParts.ID().Equal(partSetID) {
			cs.lockedRound = -1
			cs.lockedBlockParts = &blockPartSet{}
		}
		if cs.round == msg.Round && !cs.currentBlockParts.ID().Equal(partSetID) {
			cs.currentBlockParts = &blockPartSet{
				PartSet: newPartSetFromID(partSetID),
			}
		}
	}
	if cs.round > msg.Round && cs.step < stepPrevote && msg.Round == cs.proposalPOLRound {
		cs.enterPrevote()
	} else if cs.round == msg.Round && cs.step < stepPrevote {
		cs.enterPrevote()
	} else if cs.round == msg.Round && cs.step == stepPrevote {
		cs.enterPrevoteWait()
	} else if cs.round == msg.Round && cs.step == stepPrevoteWait {
		if ok {
			cs.enterPrecommit()
		}
	} else if cs.round < msg.Round {
		cs.resetForNewRound(msg.Round)
		if cs.step < stepPrevote {
			cs.enterPrevote()
		}
	}
	return true, nil
}

func (cs *consensus) handlePrecommitMessage(msg *voteMessage, precommits *voteSet) (bool, error) {
	if msg.Round < cs.round {
		return true, nil
	}

	if cs.round == msg.Round && cs.step < stepPrecommit {
		cs.enterPrecommit()
	} else if cs.round == msg.Round && cs.step == stepPrecommit {
		cs.enterPrecommitWait()
	} else if cs.round == msg.Round && cs.step == stepPrecommitWait {
		partSetID, ok := precommits.getOverTwoThirdsPartSetID()
		if partSetID != nil {
			cs.enterCommit(partSetID)
		} else if ok && partSetID == nil {
			cs.resetForNewRound(cs.round + 1)
		}
	} else if cs.round < msg.Round {
		cs.resetForNewRound(msg.Round)
		if cs.step < stepPrecommit {
			cs.enterPrecommit()
		}
	}
	return true, nil
}

func (cs *consensus) enqueueMessage(msg message) {
	cs.msgQueue = append(cs.msgQueue, msg)
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

func (cs *consensus) setStep(step step) {
	if cs.step >= step {
		log.Panicf("bad step transition (%v->%v)\n", cs.step, step)
	}
	cs.step = step
	log.Printf("consensus: setStep(%v)\n", cs.step)
}

func (cs *consensus) enterPropose() {
	cs.resetForNewStep()
	cs.setStep(stepPropose)

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
		if cs.lockedBlockParts != nil && cs.lockedBlockParts.IsComplete() {
			cs.sendProposal(cs.lockedBlockParts, cs.lockedRound)
			cs.currentBlockParts = cs.lockedBlockParts
			cs.dispatchQueuedMessage()
		} else {
			cs.bm.Propose(cs.lastBlock.ID(), cs.votes,
				func(blk module.Block, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.hrs != hrs {
						return
					}
					psb := newPartSetBuffer(1024 * 10)
					blk.MarshalHeader(psb)
					blk.MarshalBody(psb)
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.currentBlockParts = &blockPartSet{
						PartSet: bps,
						block:   blk,
					}
					cs.enterPrevote()
				},
			)
		}
	}
	cs.dispatchQueuedMessage()
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep()
	cs.setStep(stepPrevote)

	if cs.lockedBlockParts != nil {
		cs.sendVote(voteTypePrevote, cs.lockedBlockParts.ID())
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.IsComplete() {
		hrs := cs.hrs
		var err error
		cs.cancelBlockRequest, err = cs.bm.Import(
			cs.currentBlockParts.NewReader(),
			func(blk module.Block, err error) {
				cs.mutex.Lock()
				defer cs.mutex.Unlock()

				if cs.hrs != hrs {
					return
				}

				if err == nil {
					cs.currentBlockParts.block = blk
					cs.sendVote(voteTypePrevote, cs.currentBlockParts.ID())
				} else {
					cs.sendVote(voteTypePrevote, nil)
				}
			},
		)
		if err != nil {
			cs.sendVote(voteTypePrevote, nil)
		}
	} else {
		cs.sendVote(voteTypePrevote, nil)
	}

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	if prevotes.hasOverTwoThirds() {
		cs.enterPrevoteWait()
	}
}

func (cs *consensus) enterPrevoteWait() {
	cs.resetForNewStep()
	cs.setStep(stepPrevoteWait)

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	_, ok := prevotes.getOverTwoThirdsPartSetID()
	if ok {
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

func (cs *consensus) enterPrecommit() {
	cs.resetForNewStep()
	cs.setStep(stepPrecommit)

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()

	if !ok {
		cs.sendVote(voteTypePrecommit, nil)
	} else if partSetID == nil {
		cs.lockedRound = -1
		cs.lockedBlockParts = nil
		cs.sendVote(voteTypePrecommit, nil)
	} else if cs.lockedBlockParts != nil && cs.lockedBlockParts.ID().Equal(partSetID) {
		cs.lockedRound = cs.round
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts.ID())
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.IsComplete() {
		cs.lockedRound = cs.round
		cs.lockedBlockParts = cs.currentBlockParts
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts.ID())
	} else {
		// polka for a block we don't have
		if cs.currentBlockParts != nil && !cs.currentBlockParts.ID().Equal(partSetID) {
			cs.currentBlockParts = &blockPartSet{
				PartSet: newPartSetFromID(partSetID),
			}
		}
		cs.lockedRound = -1
		cs.lockedBlockParts = nil
		cs.sendVote(voteTypePrecommit, nil)
	}

	precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
	if precommits.hasOverTwoThirds() {
		cs.enterPrecommitWait()
	}
}

func (cs *consensus) enterPrecommitWait() {
	cs.resetForNewStep()
	cs.setStep(stepPrecommitWait)

	precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
	partSetID, ok := precommits.getOverTwoThirdsPartSetID()
	if ok && partSetID != nil {
		cs.enterCommit(partSetID)
	} else if ok && partSetID == nil {
		cs.resetForNewRound(cs.round + 1)
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(timeoutPrecommit, func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.resetForNewRound(cs.round + 1)
		})
	}
}

func (cs *consensus) tryCommit() {
}

func (cs *consensus) enterCommit(partSetID *PartSetID) {
	cs.resetForNewStep()
	cs.setStep(stepCommit)
	cs.nextProposeTime = time.Now().Add(timeoutCommit)

	if !cs.currentBlockParts.ID().Equal(partSetID) {
		cs.currentBlockParts = &blockPartSet{
			PartSet: newPartSetFromID(partSetID),
		}
	}
	if cs.currentBlockParts.IsComplete() {
		if cs.currentBlockParts.block == nil {
			hrs := cs.hrs
			cs.bm.Import(
				cs.currentBlockParts.NewReader(),
				func(blk module.Block, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.hrs != hrs {
						return
					}

					if err != nil {
						panic(err)
					}
					cs.currentBlockParts.block = blk
					cs.bm.Finalize(cs.currentBlockParts.block)
					cs.enterNewHeight()
				},
			)
		} else {
			cs.bm.Finalize(cs.currentBlockParts.block)
			cs.enterNewHeight()
		}
	}
}

func (cs *consensus) enterNewHeight() {
	cs.resetForNewStep()
	cs.setStep(stepNewHeight)

	now := time.Now()
	if cs.nextProposeTime.Before(now) {
		cs.resetForNextHeight()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

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

func (cs *consensus) sendVote(vt voteType, partSetID *PartSetID) {
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
		prevotes := cs.hvs.votesFor(cs.proposalPOLRound, voteTypePrevote)
		if id, _ := prevotes.getOverTwoThirdsPartSetID(); id != nil {
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
		newVoteList(nil),
	)
	if err != nil {
		return
	}
	lastFinalizedBlock := gblks[len(gblks)-1]
	cs.resetForNewHeight(lastFinalizedBlock, newVoteList(nil))
	cs.dm.RegistReactor("consensus", cs, protocols)
}
