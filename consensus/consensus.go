package consensus

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
)

var logger *log.Logger

var zeroAddress = common.NewAddress(make([]byte, common.AddressBytes))

const (
	timeoutPropose   = time.Second * 1
	timeoutPrevote   = time.Second * 1
	timeoutPrecommit = time.Second * 1
	timeoutCommit    = time.Second * 1
)

const configBlockPartSize = 1024 * 100

type hrs struct {
	height int64
	round  int32
	step   step
}

type blockPartSet struct {
	PartSet
	block    module.Block
	noImport bool
}

type consensus struct {
	hrs

	bm     module.BlockManager
	wallet module.Wallet
	dm     module.ProtocolHandler
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
	cs := &consensus{
		bm:     bm,
		wallet: c.Wallet(),
	}
	cs.dm, _ = nm.RegisterReactor("consensus", cs, protocols, 1)
	return cs
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

	msg, err := unmarshalMessage(sp, bs)
	if err != nil {
		logger.Printf("OnReceive: error=%v\n", err)
		return false, err
	}
	logger.Printf("OnReceive %+v\n", msg)
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

func (cs *consensus) OnError(err error, subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) {
	cs.mutex.Lock()
	logger.Printf("OnError\n")
	defer cs.mutex.Unlock()
}

func (cs *consensus) OnJoin(id module.PeerID) {
}
func (cs *consensus) OnLeave(id module.PeerID) {
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
		PartSet: newPartSetFromID(msg.proposal.BlockPartSetID),
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
	if cs.currentBlockParts.IsComplete() {
		block, err := cs.bm.NewBlockFromReader(cs.currentBlockParts.NewReader())
		if err != nil {
			panic(err)
		}
		cs.currentBlockParts.block = block
	}

	if cs.step == stepPropose && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
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
		cs.enterProposeForRound(msg.Round)
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
		logger.Println("handlePrecommit: <stepPrecommit")
		cs.enterPrecommit()
	} else if cs.round == msg.Round && cs.step == stepPrecommit {
		logger.Println("handlePrecommit: ==stepPrecommit")
		cs.enterPrecommitWait()
	} else if cs.round == msg.Round && cs.step == stepPrecommitWait {
		logger.Println("handlePrecommit: ==stepPrecommitWait")
		partSetID, ok := precommits.getOverTwoThirdsPartSetID()
		if partSetID != nil {
			cs.enterCommit(partSetID)
		} else if ok && partSetID == nil {
			cs.enterProposeForRound(cs.round + 1)
		}
	} else if cs.round < msg.Round {
		logger.Println("handlePrecommit future Round")
		cs.enterProposeForRound(msg.Round)
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
		logger.Panicf("bad step transition (%v->%v)\n", cs.step, step)
	}
	cs.step = step
	logger.Printf("setStep(H=%d,R=%d,S=%v)\n", cs.height, cs.round, cs.step)
}

func (cs *consensus) enterProposeForNextHeight() {
	votes := cs.hvs.votesFor(cs.round, voteTypePrecommit).voteListForOverTwoThirds()
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
						panic(err)
					}
					logger.Printf("enterPropose: onPropose: OK")

					psb := newPartSetBuffer(configBlockPartSize)
					blk.MarshalHeader(psb)
					blk.MarshalBody(psb)
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.currentBlockParts = &blockPartSet{
						PartSet:  bps,
						block:    blk,
						noImport: true,
					}
					cs.enterPrevote()
				},
			)
			if err != nil {
				logger.Panicf("enterPropose: %+v\n", err)
			}
		}
	}
	// dispatch twice to handle the case blockPart is queued before proposal
	cs.dispatchQueuedMessage()
	cs.dispatchQueuedMessage()
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep()
	cs.setStep(stepPrevote)

	if cs.lockedBlockParts != nil {
		cs.sendVote(voteTypePrevote, cs.lockedBlockParts)
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.IsComplete() {
		hrs := cs.hrs
		if cs.currentBlockParts.block != nil {
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
						logger.Println("prevote: onImport: OK")
						cs.currentBlockParts.noImport = true
						cs.sendVote(voteTypePrevote, cs.currentBlockParts)
					} else {
						logger.Println("prevote: onImport: ", err)
						cs.sendVote(voteTypePrevote, nil)
					}
				},
			)
			if err != nil {
				logger.Println("prevote: ", err)
				cs.sendVote(voteTypePrevote, nil)
				return
			}
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
		logger.Println("enterPrecommit: !ok")
		cs.sendVote(voteTypePrecommit, nil)
	} else if partSetID == nil {
		logger.Println("enterPrecommit: partSetID==nil")
		cs.lockedRound = -1
		cs.lockedBlockParts = nil
		cs.sendVote(voteTypePrecommit, nil)
	} else if cs.lockedBlockParts != nil && cs.lockedBlockParts.ID().Equal(partSetID) {
		logger.Println("enterPrecommit: updateLockRound")
		cs.lockedRound = cs.round
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts)
	} else if cs.currentBlockParts != nil && cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.IsComplete() {
		logger.Println("enterPrecommit: updateLock")
		cs.lockedRound = cs.round
		cs.lockedBlockParts = cs.currentBlockParts
		cs.sendVote(voteTypePrecommit, cs.lockedBlockParts)
	} else {
		// polka for a block we don't have
		logger.Println("enterPrecommit: polka for we don't have")
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
		logger.Println("enterPrecommitWait: ok && partSetID!=nil")
		cs.enterCommit(partSetID)
	} else if ok && partSetID == nil {
		logger.Println("enterPrecommitWait: ok && partSetID==nil")
		cs.enterProposeForRound(cs.round + 1)
	} else {
		logger.Println("enterPrecommitWait: else")
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
	if !cs.currentBlockParts.noImport {
		hrs := cs.hrs
		_, err := cs.bm.Import(
			cs.currentBlockParts.NewReader(),
			func(blk module.Block, err error) {
				cs.mutex.Lock()
				defer cs.mutex.Unlock()

				if cs.hrs != hrs {
					logger.Panicf("commitAndEnterNewHeight: bad cs.hrs=%v\n", cs.hrs)
				}

				if err != nil {
					logger.Panicf("commitAndEnterNewHeight: %v\n", err)
				}
				cs.currentBlockParts.noImport = true
				err = cs.bm.Finalize(cs.currentBlockParts.block)
				if err != nil {
					logger.Panicf("commitAndEnterNewHeight: %v\n", err)
				}
				cs.enterNewHeight()
			},
		)
		if err != nil {
			logger.Panicf("commitAndEnterNewHeight: %v\n", err)
		}
	} else {
		err := cs.bm.Finalize(cs.currentBlockParts.block)
		if err != nil {
			logger.Panicf("commitAndEnterNewHeight: %v\n", err)
		}
		cs.enterNewHeight()
	}
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
		cs.commitAndEnterNewHeight()
	}

	// dispatch to handle queued blockPart
	cs.dispatchQueuedMessage()
}

func (cs *consensus) enterNewHeight() {
	cs.resetForNewStep()
	cs.setStep(stepNewHeight)

	now := time.Now()
	if cs.nextProposeTime.Before(now) {
		cs.enterProposeForNextHeight()
	} else {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs {
				return
			}
			cs.enterProposeForNextHeight()
		})
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
		return errors.Errorf("sendVote : %v", err)
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return errors.Errorf("sendVote : %v", err)
	}
	logger.Printf("sendProposal = %+v \n", msg)
	cs.dm.Broadcast(protoProposal, msgBS, module.BROADCAST_ALL)

	bpmsg := newBlockPartMessage()
	bpmsg.Height = cs.height
	bpmsg.Round = cs.round
	for i := 0; i < blockParts.Parts(); i++ {
		bpmsg.BlockPart = blockParts.GetPart(i).Bytes()
		bpmsgBS, err := msgCodec.MarshalToBytes(bpmsg)
		if err != nil {
			return errors.Errorf("sendVote : %v", err)
		}
		logger.Printf("sendBlockPart = %+v \n", bpmsg)
		cs.dm.Broadcast(protoBlockPart, bpmsgBS, module.BROADCAST_ALL)
	}

	return nil
}

func (cs *consensus) sendVote(vt voteType, blockParts *blockPartSet) error {
	if cs.validators.IndexOf(cs.wallet.Address()) < 0 {
		return nil
	}

	defer func() {
		go func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			cs.dispatchQueuedMessage()
		}()
	}()
	msg := newVoteMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.Type = vt

	if blockParts != nil {
		// TODO fixme
		if blockParts.block != nil {
			msg.BlockID = blockParts.block.ID()
		} else {
			msg.BlockID = nil
		}
		msg.BlockPartSetID = blockParts.ID()
	} else {
		msg.BlockID = nil
		msg.BlockPartSetID = nil
	}
	err := msg.sign(cs.wallet)
	if err != nil {
		return errors.Errorf("sendVote : %v", err)
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return errors.Errorf("sendVote : %v", err)
	}
	logger.Printf("sendVote = %+v \n", msg)
	cs.dm.Broadcast(protoVote, msgBS, module.BROADCAST_ALL)
	cs.enqueueMessage(msg)
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
	return bytes.Equal(v.Address().Bytes(), waddr.Bytes())
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

	prefix := fmt.Sprintf("%x|CS|", cs.wallet.Address().Bytes()[1:3])
	logger = log.New(os.Stderr, prefix, log.Lshortfile|log.Lmicroseconds)

	logger.Printf("Consensus start wallet=%s", cs.wallet.Address())

	go func() {
		time.Sleep(time.Second * 3)

		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		cs.resetForNewHeight(lastFinalizedBlock, newVoteList(nil))
		cs.enterPropose()
	}()
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
