package consensus

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

var (
	logger *log.Logger
	debug  *log.Logger
)

var zeroAddress = common.NewAddress(make([]byte, common.AddressBytes))

var csProtocols = []module.ProtocolInfo{protoProposal, protoBlockPart, protoVote}

const (
	timeoutPropose   = time.Second * 1
	timeoutPrevote   = time.Second * 1
	timeoutPrecommit = time.Second * 1
	timeoutCommit    = time.Second * 1
	timeoutNewRound  = time.Second * 1
)

const (
	configBlockPartSize               = 1024 * 100
	configCommitCacheCap              = 60
	configEnginePriority              = 2
	configSyncerPriority              = 3
	configRoundWALID                  = "round"
	configRoundWALDataSize            = 1024 * 500
	configLockWALID                   = "lock"
	configLockWALDataSize             = 1024 * 1024 * 5
	configCommitWALID                 = "commit"
	configCommitWALDataSize           = 1024 * 500
	configRoundTimeoutThresholdFactor = 2
)

type hrs struct {
	height int64
	round  int32
	step   step
}

type blockPartSet struct {
	PartSet
	block     module.Block // nil if partset is incomplete or invalid block
	validated bool
}

type commit struct {
	height       int64
	commitVotes  *commitVoteList
	votes        *voteList
	blockPartSet PartSet
}

type consensus struct {
	hrs

	nm        module.NetworkManager
	bm        module.BlockManager
	wallet    module.Wallet
	ph        module.ProtocolHandler
	mutex     sync.Mutex
	syncer    Syncer
	walDir    string
	roundWAL  *walMessageWriter
	lockWAL   *walMessageWriter
	commitWAL *walMessageWriter

	lastBlock          module.Block
	validators         module.ValidatorList
	votes              *commitVoteList
	hvs                heightVoteSet
	nextProposeTime    time.Time
	lockedRound        int32
	lockedBlockParts   *blockPartSet
	proposalPOLRound   int32
	currentBlockParts  *blockPartSet
	consumedNonunicast bool
	commitRound        int32

	timer              *time.Timer
	cancelBlockRequest func() bool

	// commit cache
	commitMRU       *list.List
	commitForHeight map[int64]*commit
}

func NewConsensus(c module.Chain, bm module.BlockManager, nm module.NetworkManager, walDir string) module.Consensus {
	cs := &consensus{
		nm:              nm,
		bm:              bm,
		wallet:          c.Wallet(),
		walDir:          walDir,
		commitMRU:       list.New(),
		commitForHeight: make(map[int64]*commit, configCommitCacheCap),
	}
	return cs
}

func (cs *consensus) resetForNewHeight(prevBlock module.Block, votes *commitVoteList) {
	cs.height = prevBlock.Height() + 1
	cs.lastBlock = prevBlock
	cs.validators = cs.lastBlock.NextValidators()
	cs.votes = votes
	cs.hvs.reset(cs.validators.Len())
	cs.lockedRound = -1
	cs.lockedBlockParts = nil
	cs.consumedNonunicast = false
	cs.commitRound = -1
	cs.resetForNewRound(0)
}

func (cs *consensus) resetForNewRound(round int32) {
	cs.proposalPOLRound = -1
	cs.currentBlockParts = nil
	cs.round = round
	cs.step = stepPrepropose
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

	msg, err := unmarshalMessage(sp.Uint16(), bs)
	if err != nil {
		logger.Printf("OnReceive: %+v\n", err)
		return false, err
	}
	debug.Printf("OnReceive: %+v\n", msg)
	if err = msg.verify(); err != nil {
		logger.Printf("OnReceive: %+v\n", err)
		return false, err
	}
	switch m := msg.(type) {
	case *proposalMessage:
		err = cs.ReceiveProposalMessage(m, false)
	case *blockPartMessage:
		_, err = cs.ReceiveBlockPartMessage(m, false)
	case *voteMessage:
		_, err = cs.ReceiveVoteMessage(m, false)
	default:
		logger.Printf("OnReceived: unexpected broadcast message %v", msg)
	}
	if err != nil {
		logger.Printf("OnReceive: %+v\n", err)
		return false, err
	}
	return true, nil
}

func (cs *consensus) OnError(err error, subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) {
	logger.Printf("OnError: %v\n", err)
}

func (cs *consensus) OnJoin(id module.PeerID) {
	logger.Printf("OnJoin: %v\n", id)
}

func (cs *consensus) OnLeave(id module.PeerID) {
	logger.Printf("OnLeave: %v\n", id)
}

func (cs *consensus) ReceiveProposalMessage(msg *proposalMessage, unicast bool) error {
	if msg.Height != cs.height || msg.Round != cs.round {
		return nil
	}
	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return errors.Errorf("bad proposer %v", msg.address())
	}
	if cs.getProposerIndex(cs.height, cs.round) != index {
		// TODO : add evict
		return errors.Errorf("bad validator proposer %v", msg.address())
	}

	// TODO receive multiple proposal
	if cs.currentBlockParts != nil {
		return nil
	}
	cs.proposalPOLRound = msg.proposal.POLRound
	cs.currentBlockParts = &blockPartSet{
		PartSet: newPartSetFromID(msg.proposal.BlockPartSetID),
	}

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.enterNewHeight()
	}
	return nil
}

func (cs *consensus) ReceiveBlockPartMessage(msg *blockPartMessage, unicast bool) (int, error) {
	if msg.Height != cs.height {
		return -1, nil
	}
	if cs.currentBlockParts == nil {
		return -1, nil
	}

	bp, err := newPart(msg.BlockPart)
	if err != nil {
		return -1, err
	}
	if cs.currentBlockParts.GetPart(bp.Index()) != nil {
		return -1, nil
	}
	if err := cs.currentBlockParts.AddPart(bp); err != nil {
		return -1, err
	}
	if cs.currentBlockParts.IsComplete() {
		block, err := cs.bm.NewBlockFromReader(cs.currentBlockParts.NewReader())
		if err != nil {
			logger.Printf("ReceivedBlockPartMessage: cannot create block: %+v\n", err)
		} else {
			cs.currentBlockParts.block = block
		}
	}

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
	return bp.Index(), nil
}

func (cs *consensus) ReceiveVoteMessage(msg *voteMessage, unicast bool) (int, error) {
	if msg.Height != cs.height {
		return -1, nil
	}
	index := cs.validators.IndexOf(msg.address())
	if index < 0 {
		return -1, errors.Errorf("bad voter %v", msg.address())
	}
	added, votes := cs.hvs.add(index, msg)
	if !added {
		return -1, nil
	}
	if !unicast {
		cs.consumedNonunicast = true
	}

	if !votes.hasOverTwoThirds() {
		return index, nil
	}
	if msg.Type == voteTypePrevote {
		return index, cs.handlePrevoteMessage(msg, votes)
	} else {
		return index, cs.handlePrecommitMessage(msg, votes)
	}
}

func (cs *consensus) handlePrevoteMessage(msg *voteMessage, prevotes *voteSet) error {
	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()
	if ok {
		if cs.lockedRound < msg.Round && cs.lockedBlockParts != nil && !cs.lockedBlockParts.ID().Equal(partSetID) {
			cs.lockedRound = -1
			cs.lockedBlockParts = nil
		}
		if cs.round == msg.Round && partSetID != nil && (cs.currentBlockParts == nil || !cs.currentBlockParts.ID().Equal(partSetID)) {
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
		cs.enterPrevote()
	}
	return nil
}

func (cs *consensus) handlePrecommitMessage(msg *voteMessage, precommits *voteSet) error {
	if msg.Round < cs.round && cs.step < stepCommit {
		if psid, _ := precommits.getOverTwoThirdsPartSetID(); psid != nil {
			cs.enterCommit(precommits, psid, msg.Round)
		}
	} else if cs.round == msg.Round && cs.step < stepPrecommit {
		cs.enterPrecommit()
	} else if cs.round == msg.Round && cs.step == stepPrecommit {
		cs.enterPrecommitWait()
	} else if cs.round == msg.Round && cs.step == stepPrecommitWait {
		partSetID, ok := precommits.getOverTwoThirdsPartSetID()
		if partSetID != nil {
			cs.enterCommit(precommits, partSetID, msg.Round)
		} else if ok && partSetID == nil {
			cs.enterNewRound()
		}
	} else if cs.round < msg.Round {
		cs.resetForNewRound(msg.Round)
		cs.enterPrecommit()
	}
	return nil
}

func (cs *consensus) notifySyncer() {
	if cs.syncer != nil {
		cs.syncer.OnEngineStepChange()
	}
}

func (cs *consensus) setStep(step step) {
	if cs.step >= step {
		logger.Panicf("bad step transition (%v->%v)\n", cs.step, step)
	}
	cs.step = step
	logger.Printf("setStep(%v.%v.%v)\n", cs.height, cs.round, cs.step)
}

func (cs *consensus) enterProposeForNextHeight() {
	votes := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit).commitVoteListForOverTwoThirds()
	cs.resetForNewHeight(cs.currentBlockParts.block, votes)
	cs.enterPropose()
}

func (cs *consensus) enterProposeForRound(round int32) {
	cs.resetForNewRound(round)
	cs.enterPropose()
}

func (cs *consensus) enterPropose() {
	cs.resetForNewStep()
	cs.setStep(stepPropose)

	if int(cs.round) > cs.validators.Len()*configRoundTimeoutThresholdFactor {
		cs.nextProposeTime = time.Now().Add(timeoutNewRound)
	} else {
		cs.nextProposeTime = time.Now()
	}

	hrs := cs.hrs
	cs.timer = time.AfterFunc(timeoutPropose, func() {
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		if cs.hrs != hrs {
			return
		}
		cs.enterPrevote()
	})

	if cs.isProposer() {
		if cs.lockedBlockParts != nil && cs.lockedBlockParts.IsComplete() {
			cs.sendProposal(cs.lockedBlockParts, cs.lockedRound)
			cs.currentBlockParts = cs.lockedBlockParts
		} else {
			var err error
			cs.cancelBlockRequest, err = cs.bm.Propose(cs.lastBlock.ID(), cs.votes,
				func(blk module.Block, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.hrs != hrs {
						return
					}

					if err != nil {
						logger.Panicf("propose cb: %+v\n", err)
					}

					psb := newPartSetBuffer(configBlockPartSize)
					blk.MarshalHeader(psb)
					blk.MarshalBody(psb)
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.currentBlockParts = &blockPartSet{
						PartSet:   bps,
						block:     blk,
						validated: true,
					}
					cs.enterPrevote()
				},
			)
			if err != nil {
				logger.Panicf("enterPropose: %+v\n", err)
			}
		}
	}
	cs.notifySyncer()
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep()
	cs.setStep(stepPrevote)

	if cs.lockedBlockParts != nil {
		cs.sendVote(voteTypePrevote, cs.lockedBlockParts)
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.IsComplete() {
		hrs := cs.hrs
		if cs.currentBlockParts.validated {
			cs.sendVote(voteTypePrevote, cs.currentBlockParts)
		} else {
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
						cs.currentBlockParts.validated = true
						cs.sendVote(voteTypePrevote, cs.currentBlockParts)
					} else {
						logger.Printf("enterPrevote: import cb error: %+v\n", err)
						cs.sendVote(voteTypePrevote, nil)
					}
				},
			)
			if err != nil {
				logger.Printf("enterPrevote: import error: %+v\n", err)
				cs.sendVote(voteTypePrevote, nil)
				return
			}
		}
	} else {
		cs.sendVote(voteTypePrevote, nil)
	}

	cs.notifySyncer()

	// send vote may change step
	// TODO simplify
	if cs.step == stepPrevote {
		prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
		if prevotes.hasOverTwoThirds() {
			cs.enterPrevoteWait()
		}
	}
}

func (cs *consensus) enterPrevoteWait() {
	cs.resetForNewStep()
	cs.setStep(stepPrevoteWait)

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	msg := newVoteListMessage()
	msg.VoteList = prevotes.voteList()
	if err := cs.roundWAL.writeMessage(msg); err != nil {
		logger.Printf("enterPrevoteWait: %+v\n", err)
	}

	cs.notifySyncer()

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
		debug.Println("enterPrecommit: no +2/3 prevote")
		cs.sendVote(voteTypePrecommit, nil)
	} else if partSetID == nil {
		debug.Println("enterPrecommit: nil +2/3 prevote")
		cs.lockedRound = -1
		cs.lockedBlockParts = nil
		cs.sendVote(voteTypePrecommit, nil)
	} else if cs.lockedBlockParts != nil && cs.lockedBlockParts.ID().Equal(partSetID) {
		debug.Println("enterPrecommit: update lock round")
		cs.lockedRound = cs.round
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts)
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.validated {
		debug.Println("enterPrecommit: update lock")
		cs.lockedRound = cs.round
		cs.lockedBlockParts = cs.currentBlockParts
		msg := newVoteListMessage()
		msg.VoteList = prevotes.voteList()
		if err := cs.lockWAL.writeMessage(msg); err != nil {
			logger.Printf("enterPrecommit: %+v\n", err)
		}
		for i := 0; i < cs.lockedBlockParts.Parts(); i++ {
			msg := newBlockPartMessage()
			msg.Height = cs.height
			msg.Index = uint16(i)
			msg.BlockPart = cs.lockedBlockParts.GetPart(i).Bytes()
			if err := cs.lockWAL.writeMessage(msg); err != nil {
				logger.Printf("enterPrecommit: %+v\n", err)
			}
		}
		if err := cs.lockWAL.Sync(); err != nil {
			logger.Printf("cs.enterPrecommit: %+v\n", err)
		}
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts)
	} else {
		// polka for a block we don't have
		debug.Println("enterPrecommit: polka for we don't have")
		if cs.currentBlockParts == nil || !cs.currentBlockParts.ID().Equal(partSetID) {
			cs.currentBlockParts = &blockPartSet{
				PartSet: newPartSetFromID(partSetID),
			}
		}
		cs.lockedRound = -1
		cs.lockedBlockParts = nil
		cs.sendVote(voteTypePrecommit, nil)
	}

	cs.notifySyncer()

	// sendVote may change step
	if cs.step == stepPrecommit {
		precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
		if precommits.hasOverTwoThirds() {
			cs.enterPrecommitWait()
		}
	}
}

func (cs *consensus) enterPrecommitWait() {
	cs.resetForNewStep()
	cs.setStep(stepPrecommitWait)

	precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
	msg := newVoteListMessage()
	msg.VoteList = precommits.voteList()
	if err := cs.roundWAL.writeMessage(msg); err != nil {
		logger.Printf("enterPrecommitWait: %+v\n", err)
	}

	cs.notifySyncer()

	partSetID, ok := precommits.getOverTwoThirdsPartSetID()
	if ok && partSetID != nil {
		cs.enterCommit(precommits, partSetID, cs.round)
	} else if ok && partSetID == nil {
		cs.enterNewRound()
	} else {
		debug.Println("enterPrecommitWait: start timer")
		hrs := cs.hrs
		cs.timer = time.AfterFunc(timeoutPrecommit, func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterProposeForRound(cs.round + 1)
		})
	}
}

func (cs *consensus) commitAndEnterNewHeight() {
	if !cs.currentBlockParts.validated {
		hrs := cs.hrs
		_, err := cs.bm.Import(
			cs.currentBlockParts.NewReader(),
			func(blk module.Block, err error) {
				cs.mutex.Lock()
				defer cs.mutex.Unlock()

				if cs.hrs != hrs {
					logger.Panicf("commitAndEnterNewHeight: hrs mismatch cs.hrs=%v hrs=%v\n", cs.hrs, hrs)
				}

				if err != nil {
					logger.Panicf("commitAndEnterNewHeight: %+v\n", err)
				}
				cs.currentBlockParts.validated = true
				err = cs.bm.Finalize(cs.currentBlockParts.block)
				if err != nil {
					logger.Panicf("commitAndEnterNewHeight: %+v\n", err)
				}
				cs.enterNewHeight()
			},
		)
		if err != nil {
			logger.Panicf("commitAndEnterNewHeight: %+v\n", err)
		}
	} else {
		err := cs.bm.Finalize(cs.currentBlockParts.block)
		if err != nil {
			logger.Panicf("commitAndEnterNewHeight: %+v\n", err)
		}
		cs.enterNewHeight()
	}
}

func (cs *consensus) enterCommit(precommits *voteSet, partSetID *PartSetID, round int32) {
	cs.resetForNewStep()
	cs.setStep(stepCommit)
	cs.commitRound = round

	msg := newVoteListMessage()
	msg.VoteList = precommits.voteList()
	if err := cs.commitWAL.writeMessage(msg); err != nil {
		logger.Printf("enterCommit: %+v\n", err)
	}
	if err := cs.commitWAL.Sync(); err != nil {
		logger.Printf("cs.enterCommit: %+v\n", err)
	}

	if cs.consumedNonunicast {
		cs.nextProposeTime = time.Now().Add(timeoutCommit)
	} else {
		cs.nextProposeTime = time.Now()
	}

	if cs.currentBlockParts == nil || !cs.currentBlockParts.ID().Equal(partSetID) {
		cs.currentBlockParts = &blockPartSet{
			PartSet: newPartSetFromID(partSetID),
		}
	}

	cs.notifySyncer()

	if cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
}

func (cs *consensus) enterNewRound() {
	cs.resetForNewStep()
	cs.setStep(stepNewRound)
	cs.notifySyncer()

	now := time.Now()
	if cs.nextProposeTime.After(now) {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterProposeForRound(cs.round + 1)
		})
	} else {
		cs.enterProposeForRound(cs.round + 1)
	}
}

func (cs *consensus) enterNewHeight() {
	cs.resetForNewStep()
	cs.setStep(stepNewHeight)
	cs.notifySyncer()

	now := time.Now()
	if cs.nextProposeTime.After(now) {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterProposeForNextHeight()
		})
	} else {
		cs.enterProposeForNextHeight()
	}
}

func (cs *consensus) sendProposal(blockParts PartSet, polRound int32) error {
	msg := newProposalMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.BlockPartSetID = blockParts.ID()
	msg.POLRound = polRound
	err := msg.sign(cs.wallet)
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.writeMessageBytes(msg.subprotocol(), msgBS); err != nil {
		logger.Printf("cs.sendProposal: %+v\n", err)
	}
	if err := cs.roundWAL.Sync(); err != nil {
		logger.Printf("cs.sendProposal: %+v\n", err)
	}
	logger.Printf("sendProposal = %+v\n", msg)
	err = cs.ph.Broadcast(protoProposal, msgBS, module.BROADCAST_ALL)
	if err != nil {
		logger.Printf("cs.sendProposal: %+v\n", err)
	}

	bpmsg := newBlockPartMessage()
	bpmsg.Height = cs.height
	for i := 0; i < blockParts.Parts(); i++ {
		bpmsg.BlockPart = blockParts.GetPart(i).Bytes()
		bpmsg.Index = uint16(i)
		bpmsgBS, err := msgCodec.MarshalToBytes(bpmsg)
		if err != nil {
			return err
		}
		logger.Printf("sendBlockPart = %+v\n", bpmsg)
		err = cs.ph.Broadcast(protoBlockPart, bpmsgBS, module.BROADCAST_ALL)
		if err != nil {
			logger.Printf("cs.sendProposal: %+v\n", err)
		}
	}

	return nil
}

func (cs *consensus) sendVote(vt voteType, blockParts *blockPartSet) error {
	if cs.validators.IndexOf(cs.wallet.Address()) < 0 {
		return nil
	}

	msg := newVoteMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.Type = vt

	if blockParts != nil {
		msg.BlockID = blockParts.block.ID()
		msg.BlockPartSetID = blockParts.ID()
	} else {
		msg.BlockID = nil
		msg.BlockPartSetID = nil
	}
	err := msg.sign(cs.wallet)
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.writeMessageBytes(msg.subprotocol(), msgBS); err != nil {
		logger.Printf("cs.sendVote: %+v\n", err)
	}
	if err := cs.roundWAL.Sync(); err != nil {
		logger.Printf("cs.sendVote: %+v\n", err)
	}
	logger.Printf("sendVote = %+v \n", msg)
	if vt == voteTypePrevote {
		err = cs.ph.Multicast(protoVote, msgBS, module.ROLE_VALIDATOR)
	} else {
		err = cs.ph.Broadcast(protoVote, msgBS, module.BROADCAST_ALL)
	}
	if err != nil {
		logger.Printf("cs.sendVote: %+v\n", err)
	}
	cs.ReceiveVoteMessage(msg, true)
	return nil
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
	wPubKey, err := crypto.ParsePublicKey(cs.wallet.PublicKey())
	if err != nil {
		panic(err)
	}
	waddr := common.NewAccountAddressFromPublicKey(wPubKey)
	return v.Address().Equal(waddr)
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

func (cs *consensus) applyRoundWAL() error {
	wr, err := OpenWALForRead(path.Join(cs.walDir, configRoundWALID))
	if err != nil {
		return err
	}
	round := int32(0)
	rstep := stepPrepropose
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			err = wr.Close()
			if err != nil {
				return err
			}
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			logger.Printf("applyRoundWAL: %+v\n", err)
			err := wr.Repair()
			if err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}
		if len(bs) < 2 {
			return errors.Errorf("too short wal message len=%v", len(bs))
		}
		sp := binary.BigEndian.Uint16(bs[0:2])
		msg, err := unmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *proposalMessage:
			if m.height() != cs.height {
				continue
			}
			debug.Printf("WAL: my proposal %v\n", m)
			if m.round() < round || (m.round() == round && rstep <= stepPropose) {
				round = m.round()
				rstep = stepPropose
			}
		case *voteMessage:
			if m.height() != cs.height {
				continue
			}
			debug.Printf("WAL: my vote %v\n", m)
			index := cs.validators.IndexOf(m.address())
			if index < 0 {
				continue
			}
			_, _ = cs.hvs.add(index, m)
			var mstep step
			if m.Type == voteTypePrevote {
				mstep = stepPrevote
			} else {
				mstep = stepPrecommit
			}
			if m.round() < round || (m.round() == round && rstep <= mstep) {
				round = m.round()
				rstep = mstep
			}
		case *voteListMessage:
			for i := 0; i < m.VoteList.Len(); i++ {
				vmsg := m.VoteList.Get(i)
				if vmsg.height() != cs.height {
					continue
				}
				debug.Printf("WAL: round vote %v\n", vmsg)
				index := cs.validators.IndexOf(vmsg.address())
				if index < 0 {
					continue
				}
				_, _ = cs.hvs.add(index, vmsg)
			}
			vmsg := m.VoteList.Get(0)
			if vmsg.Height != cs.height {
				continue
			}
			var mstep step
			if vmsg.Type == voteTypePrevote {
				mstep = stepPrevote
			} else {
				mstep = stepPrecommit
			}
			if round < vmsg.Round || (round == vmsg.Round && rstep < mstep) {
				votes := cs.hvs.votesFor(vmsg.Round, vmsg.Type)
				if votes.hasOverTwoThirds() {
					round = vmsg.Round
					rstep = mstep
				}
			}
		}
	}
	cs.round = round
	cs.step = rstep
	return nil
}

func (cs *consensus) applyLockWAL() error {
	wr, err := OpenWALForRead(path.Join(cs.walDir, configLockWALID))
	if err != nil {
		return err
	}
	var bpset PartSet
	var bpsetLockRound int32
	var lastBPSet PartSet
	var lastBPSetLockRound int32
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			err = wr.Close()
			if err != nil {
				return err
			}
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			logger.Printf("applyLockWAL: %+v\n", err)
			err := wr.Repair()
			if err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}
		if len(bs) < 2 {
			return errors.Errorf("too short wal message len=%v", len(bs))
		}
		sp := binary.BigEndian.Uint16(bs[0:2])
		msg, err := unmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *voteListMessage:
			if m.VoteList.Len() == 0 {
				continue
			}
			for i := 0; i < m.VoteList.Len(); i++ {
				vmsg := m.VoteList.Get(i)
				if vmsg.height() != cs.height {
					continue
				}
				debug.Printf("WAL: round vote %v\n", vmsg)
				index := cs.validators.IndexOf(vmsg.address())
				if index < 0 {
					continue
				}
				_, _ = cs.hvs.add(index, vmsg)
			}
			vmsg := m.VoteList.Get(0)
			if vmsg.Height != cs.height {
				continue
			}
			prevotes := cs.hvs.votesFor(vmsg.Round, voteTypePrevote)
			psid, ok := prevotes.getOverTwoThirdsPartSetID()
			if ok && psid != nil {
				debug.Printf("WAL: POL R=%v psid=%v\n", vmsg.Round, psid)
				bpset = newPartSetFromID(psid)
				bpsetLockRound = vmsg.Round
			}
			// update round/step
			var mstep step
			if vmsg.Type == voteTypePrevote {
				mstep = stepPrevote
			} else {
				mstep = stepPrecommit
			}
			if cs.round < vmsg.Round || (cs.round == vmsg.Round && cs.step < mstep) {
				votes := cs.hvs.votesFor(vmsg.Round, vmsg.Type)
				if votes.hasOverTwoThirds() {
					cs.round = vmsg.Round
					cs.step = mstep
				}
			}
		case *blockPartMessage:
			if m.Height != cs.height {
				continue
			}
			if bpset == nil {
				continue
			}
			bp, err := newPart(m.BlockPart)
			if err != nil {
				return err
			}
			err = bpset.AddPart(bp)
			debug.Printf("WAL: blockPart %v\n", m)
			if err == nil && bpset.IsComplete() {
				lastBPSet = bpset
				lastBPSetLockRound = bpsetLockRound
				debug.Printf("WAL: blockPart complete\n")
			}
		}
	}
	if lastBPSet != nil {
		blk, err := cs.bm.NewBlockFromReader(lastBPSet.NewReader())
		if err != nil {
			return err
		}
		cs.currentBlockParts = &blockPartSet{
			PartSet:   lastBPSet,
			block:     blk,
			validated: false,
		}
		cs.lockedBlockParts = cs.currentBlockParts
		cs.lockedRound = lastBPSetLockRound
	}
	return nil
}

func (cs *consensus) applyCommitWAL(prevValidators module.ValidatorList) error {
	wr, err := OpenWALForRead(path.Join(cs.walDir, configCommitWALID))
	if err != nil {
		return nil
	}
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			err = wr.Close()
			if err != nil {
				return err
			}
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			logger.Printf("applyCommitWAL: %+v\n", err)
			err := wr.Repair()
			if err != nil {
				return err
			}
			break
		} else if err != nil {
			return err
		}
		if len(bs) < 2 {
			return errors.Errorf("too short wal message len=%v", len(bs))
		}
		sp := binary.BigEndian.Uint16(bs[0:2])
		msg, err := unmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *voteListMessage:
			if m.VoteList.Len() == 0 {
				continue
			}
			if m.VoteList.Get(0).height() == cs.height-1 {
				vs := newVoteSet(prevValidators.Len())
				for i := 0; i < m.VoteList.Len(); i++ {
					msg := m.VoteList.Get(i)
					debug.Printf("WAL: round vote %v\n", msg)
					index := prevValidators.IndexOf(msg.address())
					if index < 0 {
						return errors.Errorf("bad voter %v", msg.address())
					}
					_ = vs.add(index, msg)
				}
				psid, ok := vs.getOverTwoThirdsPartSetID()
				if ok && psid != nil {
					cs.votes = vs.commitVoteListForOverTwoThirds()
				}
			} else if m.VoteList.Get(0).height() == cs.height {
				for i := 0; i < m.VoteList.Len(); i++ {
					vmsg := m.VoteList.Get(i)
					debug.Printf("WAL: round vote %v\n", vmsg)
					index := cs.validators.IndexOf(vmsg.address())
					if index < 0 {
						continue
					}
					_, _ = cs.hvs.add(index, vmsg)
				}
				// update round/step
				vmsg := m.VoteList.Get(0)
				if vmsg.Height != cs.height {
					continue
				}
				var mstep step
				if vmsg.Type == voteTypePrevote {
					mstep = stepPrevote
				} else {
					mstep = stepPrecommit
				}
				if cs.round < vmsg.Round || (cs.round == vmsg.Round && cs.step < mstep) {
					votes := cs.hvs.votesFor(vmsg.Round, vmsg.Type)
					if votes.hasOverTwoThirds() {
						cs.round = vmsg.Round
						cs.step = mstep
					}
				}
			}
		}
	}
	return nil
}

func (cs *consensus) applyWAL(prevValidators module.ValidatorList) error {
	if err := cs.applyRoundWAL(); err != nil && !IsNotExist(err) {
		return err
	}
	if err := cs.applyLockWAL(); err != nil && !IsNotExist(err) {
		return err
	}
	if err := cs.applyCommitWAL(prevValidators); err != nil && !IsNotExist(err) {
		return err
	}
	return nil
}

func (cs *consensus) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	prefix := fmt.Sprintf("%x|CS|", cs.wallet.Address().Bytes()[1:3])
	logger = log.New(os.Stderr, prefix, log.Lshortfile|log.Lmicroseconds)
	debug = log.New(debugWriter, prefix, log.Lshortfile|log.Lmicroseconds)

	var lastBlock module.Block
	var prevBlock module.Block
	lastBlock, err := cs.bm.GetLastBlock()
	if err == nil {
		prevBlock, err = cs.bm.GetBlockByHeight(lastBlock.Height() - 1)
		if err != nil {
			return err
		}
	} else if err == common.ErrNotFound {
		gblks, err := cs.bm.FinalizeGenesisBlocks(
			zeroAddress,
			time.Time{},
			newCommitVoteList(nil),
		)
		if err != nil {
			return err
		}
		lastBlock = gblks[len(gblks)-1]
		prevBlock = gblks[len(gblks)-2]
	} else {
		return err
	}

	cs.ph, err = cs.nm.RegisterReactor("consensus", cs, csProtocols, configEnginePriority)
	if err != nil {
		return err
	}

	cs.resetForNewHeight(lastBlock, newCommitVoteList(nil))
	if err := cs.applyWAL(prevBlock.NextValidators()); err != nil {
		return err
	}

	ww, err := OpenWALForWrite(path.Join(cs.walDir, configRoundWALID), &WALConfig{
		FileLimit:  configRoundWALDataSize,
		TotalLimit: configRoundWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.roundWAL = &walMessageWriter{ww}

	ww, err = OpenWALForWrite(path.Join(cs.walDir, configLockWALID), &WALConfig{
		FileLimit:  configLockWALDataSize,
		TotalLimit: configLockWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.lockWAL = &walMessageWriter{ww}

	ww, err = OpenWALForWrite(path.Join(cs.walDir, configCommitWALID), &WALConfig{
		FileLimit:  configCommitWALDataSize,
		TotalLimit: configCommitWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.commitWAL = &walMessageWriter{ww}

	logger.Printf("Consensus start wallet=%s", cs.wallet.Address())
	cs.syncer = newSyncer(cs, cs.nm, &cs.mutex, cs.wallet.Address())
	cs.syncer.Start()
	if cs.step == stepPrepropose {
		cs.enterPropose()
	} else if cs.step == stepPropose {
		cs.enterPrevote()
	} else if cs.step == stepPrevote {
		prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
		if prevotes.hasOverTwoThirds() {
			cs.enterPrevoteWait()
		}
	} else if cs.step == stepPrecommit {
		precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
		if precommits.hasOverTwoThirds() {
			cs.enterPrecommitWait()
		}
	}
	return nil
}

func (cs *consensus) GetStatus() *module.ConsensusStatus {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	res := &module.ConsensusStatus{
		Height: cs.height,
		Round:  cs.round,
	}
	if cs.validators != nil {
		res.Proposer = cs.isProposer()
	}
	return res
}

func (cs *consensus) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	c, err := cs.getCommit(height)
	if err != nil {
		return nil, err
	}
	return c.commitVotes, nil
}

func (cs *consensus) getCommit(h int64) (*commit, error) {
	if h > cs.height || (h == cs.height && cs.step < stepCommit) {
		return nil, errors.Errorf("cs.getCommit: invalid param h=%v cs.height=%v cs.step=%v\n", h, cs.height, cs.step)
	}

	c := cs.commitForHeight[h]
	if c != nil {
		return c, nil
	}

	if h == cs.height && !cs.currentBlockParts.IsComplete() {
		pcs := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit)
		return &commit{
			height:       h,
			commitVotes:  pcs.commitVoteListForOverTwoThirds(),
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts,
		}, nil
	}

	if cs.commitMRU.Len() == configCommitCacheCap {
		c := cs.commitMRU.Remove(cs.commitMRU.Back()).(*commit)
		delete(cs.commitForHeight, c.height)
	}

	if h == cs.height {
		pcs := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit)
		c = &commit{
			height:       h,
			commitVotes:  pcs.commitVoteListForOverTwoThirds(),
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts,
		}
	} else {
		b, err := cs.bm.GetBlockByHeight(h)
		if err != nil {
			return nil, err
		}
		var cvl *commitVoteList
		if h == cs.height-1 {
			cvl = cs.votes
		} else {
			nb, err := cs.bm.GetBlockByHeight(h + 1)
			if err != nil {
				return nil, err
			}
			cvl = nb.Votes().(*commitVoteList)
		}
		vl := cvl.voteList(h, b.ID())
		psb := newPartSetBuffer(configBlockPartSize)
		b.MarshalHeader(psb)
		b.MarshalBody(psb)
		bps := psb.PartSet()
		c = &commit{
			height:       h,
			commitVotes:  cvl,
			votes:        vl,
			blockPartSet: bps,
		}
	}
	cs.commitMRU.PushBack(c)
	cs.commitForHeight[c.height] = c
	return c, nil
}

func (cs *consensus) GetCommitBlockParts(h int64) PartSet {
	c, err := cs.getCommit(h)
	if err != nil {
		logger.Panicf("cs.GetCommitBlockParts: %+v\n", err)
	}
	return c.blockPartSet
}

func (cs *consensus) GetCommitPrecommits(h int64) *voteList {
	c, err := cs.getCommit(h)
	if err != nil {
		logger.Panicf("cs.GetCommitPrecommits: %+v\n", err)
	}
	return c.votes
}

func (cs *consensus) GetPrecommits(r int32) *voteList {
	return cs.hvs.votesFor(r, voteTypePrecommit).voteList()
}

func (cs *consensus) GetVotes(r int32, prevotesMask *bitArray, precommitsMask *bitArray) *voteList {
	return cs.hvs.getVoteListForMask(r, prevotesMask, precommitsMask)
}

func (cs *consensus) GetRoundState() *peerRoundState {
	prs := &peerRoundState{}
	prs.Height = cs.height
	prs.Round = cs.round
	prs.PrevotesMask = cs.hvs.votesFor(cs.round, voteTypePrevote).getMask()
	prs.PrecommitsMask = cs.hvs.votesFor(cs.round, voteTypePrecommit).getMask()
	bp := cs.currentBlockParts
	// TODO optimize
	if bp != nil && cs.step >= stepCommit {
		prs.BlockPartsMask = cs.currentBlockParts.GetMask()
	}
	return prs
}

func (cs *consensus) Height() int64 {
	return cs.height
}

func (cs *consensus) Round() int32 {
	return cs.round
}

func (cs *consensus) Step() step {
	return cs.step
}

type walMessageWriter struct {
	WALWriter
}

func (w *walMessageWriter) writeMessage(msg message) error {
	bs := make([]byte, 2, 32)
	binary.BigEndian.PutUint16(bs, msg.subprotocol())
	writer := bytes.NewBuffer(bs)
	if err := codec.Marshal(writer, msg); err != nil {
		return err
	}
	//debug.Printf("write WAL: %+v\n", msg)
	_, err := w.WriteBytes(writer.Bytes())
	return err
}

func (w *walMessageWriter) writeMessageBytes(sp uint16, msg []byte) error {
	bs := make([]byte, 2+len(msg))
	binary.BigEndian.PutUint16(bs, sp)
	copy(bs[2:], msg)
	//debug.Printf("write WAL: %x\n", bs)
	_, err := w.WriteBytes(bs)
	return err
}
