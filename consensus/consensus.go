package consensus

import (
	"bytes"
	"container/list"
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
)

const (
	configBlockPartSize  = 1024 * 100
	configCommitCacheCap = 60
	configEnginePriority = 2
	configSyncerPriority = 3
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
	votes        *roundVoteList
	blockPartSet PartSet
}

type consensus struct {
	hrs

	nm     module.NetworkManager
	bm     module.BlockManager
	wallet module.Wallet
	ph     module.ProtocolHandler
	mutex  sync.Mutex
	syncer Syncer

	lastBlock          module.Block
	validators         module.ValidatorList
	votes              *voteList
	hvs                heightVoteSet
	nextProposeTime    time.Time
	lockedRound        int32
	lockedBlockParts   *blockPartSet
	proposalPOLRound   int32
	currentBlockParts  *blockPartSet
	consumedNonunicast bool

	timer              *time.Timer
	cancelBlockRequest func() bool

	// commit cache
	commitMRU       *list.List
	commitForHeight map[int64]*commit
}

func NewConsensus(c module.Chain, bm module.BlockManager, nm module.NetworkManager) module.Consensus {
	cs := &consensus{
		nm:              nm,
		bm:              bm,
		wallet:          c.Wallet(),
		commitMRU:       list.New(),
		commitForHeight: make(map[int64]*commit, configCommitCacheCap),
	}
	return cs
}

func (cs *consensus) resetForNewHeight(prevBlock module.Block, votes *voteList) {
	cs.height = prevBlock.Height() + 1
	cs.lastBlock = prevBlock
	cs.validators = cs.lastBlock.NextValidators()
	cs.votes = votes
	cs.hvs.reset(cs.validators.Len())
	cs.lockedRound = -1
	cs.lockedBlockParts = nil
	cs.consumedNonunicast = false
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
		cs.enterProposeForRound(msg.Round)
		if cs.step < stepPrevote {
			cs.enterPrevote()
		}
	}
	return nil
}

func (cs *consensus) handlePrecommitMessage(msg *voteMessage, precommits *voteSet) error {
	if msg.Round < cs.round {
		return nil
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
			cs.enterProposeForRound(cs.round + 1)
		}
	} else if cs.round < msg.Round {
		cs.enterProposeForRound(msg.Round)
		if cs.step < stepPrecommit {
			cs.enterPrecommit()
		}
	}
	return nil
}

func (cs *consensus) setStep(step step) {
	if cs.step >= step {
		logger.Panicf("bad step transition (%v->%v)\n", cs.step, step)
	}
	cs.step = step
	logger.Printf("setStep(%v.%v.%v)\n", cs.height, cs.round, cs.step)
	cs.syncer.OnEngineStepChange()
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
	partSetID, ok := precommits.getOverTwoThirdsPartSetID()
	if ok && partSetID != nil {
		cs.enterCommit(partSetID)
	} else if ok && partSetID == nil {
		cs.enterProposeForRound(cs.round + 1)
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

func (cs *consensus) enterCommit(partSetID *PartSetID) {
	cs.resetForNewStep()
	cs.setStep(stepCommit)
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
	if cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
}

func (cs *consensus) enterNewHeight() {
	cs.resetForNewStep()
	cs.setStep(stepNewHeight)

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
	logger.Printf("sendProposal = %+v\n", msg)
	cs.ph.Broadcast(protoProposal, msgBS, module.BROADCAST_ALL)

	bpmsg := newBlockPartMessage()
	bpmsg.Height = cs.height
	for i := 0; i < blockParts.Parts(); i++ {
		bpmsg.BlockPart = blockParts.GetPart(i).Bytes()
		bpmsgBS, err := msgCodec.MarshalToBytes(bpmsg)
		if err != nil {
			return err
		}
		logger.Printf("sendBlockPart = %+v\n", bpmsg)
		cs.ph.Broadcast(protoBlockPart, bpmsgBS, module.BROADCAST_ALL)
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
	logger.Printf("sendVote = %+v \n", msg)
	if vt == voteTypePrevote {
		cs.ph.Multicast(protoVote, msgBS, module.ROLE_VALIDATOR)
	} else {
		cs.ph.Broadcast(protoVote, msgBS, module.BROADCAST_ALL)
	}
	cs.ReceiveVoteMessage(msg, false)
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
	debug = log.New(debugWriter, prefix, log.Lshortfile|log.Lmicroseconds)

	logger.Printf("Consensus start wallet=%s", cs.wallet.Address())
	cs.ph, _ = cs.nm.RegisterReactor("consensus", cs, csProtocols, configEnginePriority)

	cs.resetForNewHeight(lastFinalizedBlock, newVoteList(nil))
	cs.syncer = newSyncer(cs, cs.nm, &cs.mutex, cs.wallet.Address())
	cs.syncer.Start()
	cs.enterPropose()
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

func (cs *consensus) getCommit(h int64) *commit {
	if h > cs.height || (h == cs.height && cs.step < stepCommit) {
		logger.Panicf("cs.getCommit: invalid param h=%v cs.height=%v cs.step=%v\n", h, cs.height, cs.step)
	}

	c := cs.commitForHeight[h]
	if c != nil {
		return c
	}

	if h == cs.height && !cs.currentBlockParts.IsComplete() {
		return &commit{
			height:       h,
			votes:        cs.hvs.votesFor(cs.round, voteTypePrecommit).voteList(),
			blockPartSet: cs.currentBlockParts,
		}
	}

	if cs.commitMRU.Len() == configCommitCacheCap {
		c := cs.commitMRU.Remove(cs.commitMRU.Back()).(*commit)
		delete(cs.commitForHeight, c.height)
	}

	if h == cs.height {
		c = &commit{
			height:       h,
			votes:        cs.hvs.votesFor(cs.round, voteTypePrecommit).voteList(),
			blockPartSet: cs.currentBlockParts,
		}
	} else {
		b, err := cs.bm.GetBlockByHeight(h)
		if err != nil {
			logger.Panicf("cs.getCommit: %+v\n", err)
		}
		var vl *roundVoteList
		if h == cs.height-1 {
			vl = cs.votes.roundVoteList(h, b.ID())
		} else {
			nb, err := cs.bm.GetBlockByHeight(h + 1)
			if err != nil {
				logger.Panicf("cs.getCommit: %+v\n", err)
			}
			vl = nb.Votes().(*voteList).roundVoteList(h, b.ID())
		}
		psb := newPartSetBuffer(configBlockPartSize)
		b.MarshalHeader(psb)
		b.MarshalBody(psb)
		bps := psb.PartSet()
		c = &commit{
			height:       h,
			votes:        vl,
			blockPartSet: bps,
		}
	}
	cs.commitMRU.PushBack(c)
	cs.commitForHeight[c.height] = c
	return c
}

func (cs *consensus) GetCommitBlockParts(h int64) PartSet {
	c := cs.getCommit(h)
	return c.blockPartSet
}

func (cs *consensus) GetCommitPrecommits(h int64) *roundVoteList {
	c := cs.getCommit(h)
	return c.votes
}

func (cs *consensus) GetPrecommits(r int32) *roundVoteList {
	return cs.hvs.votesFor(r, voteTypePrecommit).voteList()
}

func (cs *consensus) GetVotes(r int32, prevotesMask *bitArray, precommitsMask *bitArray) *roundVoteList {
	return cs.hvs.getVoteListForMask(r, prevotesMask, precommitsMask)
}

func (cs *consensus) GetRoundState() *peerRoundState {
	prs := &peerRoundState{}
	prs.Height = cs.height
	prs.Round = cs.round
	prs.PrevotesMask = cs.hvs.votesFor(cs.round, voteTypePrevote).getMask()
	prs.PrecommitsMask = cs.hvs.votesFor(cs.round, voteTypePrecommit).getMask()
	bp := cs.currentBlockParts
	if bp != nil {
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
