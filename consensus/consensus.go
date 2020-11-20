package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/internal/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server/metric"
)

const (
	configRandomImportFail = false
)

var csProtocols = []module.ProtocolInfo{
	protoProposal,
	protoBlockPart,
	protoVote,
	protoVoteList,
}

const (
	timeoutPropose   = time.Second * 1
	timeoutPrevote   = time.Second * 1
	timeoutPrecommit = time.Second * 1
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

func (hrs hrs) String() string {
	return fmt.Sprintf("{Height:%d Round:%d Step:%s}", hrs.height, hrs.round, hrs.step)
}

type blockPartSet struct {
	PartSet

	// nil if partset is incomplete
	block          module.BlockData
	validatedBlock module.BlockCandidate
}

func (bps *blockPartSet) Zerofy() {
	bps.PartSet = nil
	bps.block = nil
	if bps.validatedBlock != nil {
		bps.validatedBlock.Dispose()
		bps.validatedBlock = nil
	}
}

func (bps *blockPartSet) ID() *PartSetID {
	if bps.PartSet == nil {
		return nil
	}
	return bps.PartSet.ID()
}

func (bps *blockPartSet) IsZero() bool {
	return bps.PartSet == nil && bps.block == nil
}

func (bps *blockPartSet) IsComplete() bool {
	return bps.block != nil
}

func (bps *blockPartSet) Assign(oth *blockPartSet) {
	bps.PartSet = oth.PartSet
	bps.block = oth.block
	if bps.validatedBlock == oth.validatedBlock {
		return
	}
	if bps.validatedBlock != nil {
		bps.validatedBlock.Dispose()
	}
	if oth.validatedBlock == nil {
		bps.validatedBlock = nil
	} else {
		bps.validatedBlock = oth.validatedBlock.Dup()
	}
}

func (bps *blockPartSet) Set(ps PartSet, blk module.BlockData, bc module.BlockCandidate) {
	bps.PartSet = ps
	bps.block = blk
	if bps.validatedBlock == bc {
		return
	}
	if bps.validatedBlock != nil {
		bps.validatedBlock.Dispose()
	}
	bps.validatedBlock = bc
}

type consensus struct {
	hrs

	c           module.Chain
	log         log.Logger
	ph          module.ProtocolHandler
	mutex       common.Mutex
	syncer      Syncer
	walDir      string
	wm          WALManager
	roundWAL    *walMessageWriter
	lockWAL     *walMessageWriter
	commitWAL   *walMessageWriter
	timestamper module.Timestamper
	nid         []byte

	lastBlock          module.Block
	validators         module.ValidatorList
	prevValidators     addressIndexer
	members            module.MemberList
	minimizeBlockGen   bool
	roundLimit         int32
	sentPatch          bool
	lastVotes          *voteSet
	hvs                heightVoteSet
	nextProposeTime    time.Time
	lockedRound        int32
	lockedBlockParts   blockPartSet
	proposalPOLRound   int32
	currentBlockParts  blockPartSet
	consumedNonunicast bool
	commitRound        int32
	syncing            bool
	started            bool
	cancelBlockRequest module.Canceler

	timer *time.Timer

	// commit cache
	commitCache *commitCache

	// prefetch buffer
	prefetchItems []fastsync.BlockResult

	// monitor
	metric *metric.ConsensusMetric
}

func NewConsensus(c module.Chain, walDir string, timestamper module.Timestamper) module.Consensus {
	cs := newConsensus(c, walDir, defaultWALManager, timestamper)
	cs.log.Debugf("NewConsensus\n")
	return cs
}

func newConsensus(c module.Chain, walDir string, wm WALManager, timestamper module.Timestamper) *consensus {
	cs := &consensus{
		c:           c,
		walDir:      walDir,
		wm:          wm,
		commitCache: newCommitCache(configCommitCacheCap),
		metric:      metric.NewConsensusMetric(c.MetricContext()),
		timestamper: timestamper,
		nid:         codec.MustMarshalToBytes(c.NID()),
	}
	cs.log = c.Logger().WithFields(log.Fields{
		log.FieldKeyModule: "CS",
	})

	return cs
}

func (cs *consensus) _resetForNewHeight(prevBlock module.Block, votes *voteSet) {
	cs.height = prevBlock.Height() + 1
	cs.lastBlock = prevBlock
	cs.prevValidators = cs.validators
	if cs.validators == nil || !bytes.Equal(cs.validators.Hash(), cs.lastBlock.NextValidatorsHash()) {
		cs.validators = cs.lastBlock.NextValidators()
		peerIDs := make([]module.PeerID, cs.validators.Len())
		for i := 0; i < cs.validators.Len(); i++ {
			v, _ := cs.validators.Get(i)
			peerIDs[i] = network.NewPeerIDFromAddress(v.Address())
		}
		cs.c.NetworkManager().SetRole(cs.height, module.ROLE_VALIDATOR, peerIDs...)
	}
	nextMembers, err := cs.c.ServiceManager().GetMembers(cs.lastBlock.Result())
	if err != nil {
		cs.log.Warnf("cannot get members. error:%+v\n", err)
	} else {
		if cs.members == nil || !cs.members.Equal(nextMembers) {
			cs.members = nextMembers
			var peerIDs []module.PeerID
			for it := nextMembers.Iterator(); it.Has(); cs.log.Must(it.Next()) {
				addr, _ := it.Get()
				peerIDs = append(peerIDs, network.NewPeerIDFromAddress(addr))
			}
			cs.c.NetworkManager().SetRole(cs.height, module.ROLE_NORMAL, peerIDs...)
		}
	}
	cs.minimizeBlockGen = cs.c.ServiceManager().GetMinimizeBlockGen(cs.lastBlock.Result())
	cs.roundLimit = int32(cs.c.ServiceManager().GetRoundLimit(cs.lastBlock.Result(), cs.validators.Len()))
	cs.sentPatch = false
	cs.lastVotes = votes
	cs.hvs.reset(cs.validators.Len())
	cs.lockedRound = -1
	cs.lockedBlockParts.Zerofy()
	cs.consumedNonunicast = false
	cs.commitRound = -1
	cs.syncing = true
	cs.metric.OnHeight(cs.height)
}

func (cs *consensus) resetForNewHeight(prevBlock module.Block, votes *voteSet) {
	cs.endStep()
	cs._resetForNewHeight(prevBlock, votes)
	cs._resetForNewRound(0)
	cs.beginStep(stepNewHeight)
}

func (cs *consensus) _resetForNewRound(round int32) {
	cs.proposalPOLRound = -1
	cs.currentBlockParts.Zerofy()
	cs.round = round
	cs.hvs.removeLowerRoundExcept(cs.round-1, cs.lockedRound)
	cs.log.Infof("enter round Height:%d Round:%d\n", cs.height, cs.round)
	cs.metric.OnRound(cs.round)
	if cs.cancelBlockRequest != nil {
		cs.cancelBlockRequest.Cancel()
		cs.cancelBlockRequest = nil
	}
}

func (cs *consensus) resetForNewRound(round int32) {
	cs.endStep()
	cs._resetForNewRound(round)
	cs.beginStep(stepNewRound)
}

func (cs *consensus) resetForNewStep(step step) {
	cs.endStep()
	cs.beginStep(step)
}

func (cs *consensus) endStep() {
	if (cs.step == stepPropose || cs.step == stepCommit) && cs.cancelBlockRequest != nil {
		cs.cancelBlockRequest.Cancel()
		cs.cancelBlockRequest = nil
	}
	if cs.timer != nil {
		cs.timer.Stop()
		cs.timer = nil
	}
}

func isValidTransition(from step, to step) bool {
	switch to {
	case stepNewHeight:
		return from == stepNewHeight || from == stepCommit
	case stepNewRound:
		return true
	default:
		return from < to
	}
}

func (cs *consensus) beginStep(step step) {
	if !isValidTransition(cs.step, step) {
		cs.log.Panicf("bad step transition %v->%v\n", cs.step, step)
	}
	cs.step = step
	cs.log.Debugf("enterStep %v\n", cs.hrs)
}

func (cs *consensus) OnReceive(
	sp module.ProtocolInfo,
	bs []byte,
	id module.PeerID,
) (bool, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if !cs.started {
		return false, nil
	}

	msg, err := unmarshalMessage(sp.Uint16(), bs)
	if err != nil {
		cs.log.Warnf("malformed consensus message: OnReceive(subprotocol:%v, from:%v): %+v\n", sp, common.HexPre(id.Bytes()), err)
		return false, err
	}
	cs.log.Debugf("OnReceive(msg:%v, from:%v)\n", msg, common.HexPre(id.Bytes()))
	if err = msg.verify(); err != nil {
		cs.log.Warnf("consensus message verify failed: OnReceive(msg:%v, from:%v): %+v\n", msg, common.HexPre(id.Bytes()), err)
		return false, err
	}
	switch m := msg.(type) {
	case *proposalMessage:
		err = cs.ReceiveProposalMessage(m, false)
	case *blockPartMessage:
		_, err = cs.ReceiveBlockPartMessage(m, false)
	case *voteMessage:
		_, err = cs.ReceiveVoteMessage(m, false)
	case *voteListMessage:
		err = cs.ReceiveVoteListMessage(m, false)
	default:
		err = errors.Errorf("unexpected broadcast message %v", m)
	}
	if err != nil {
		cs.log.Warnf("OnReceive(msg:%v, from:%v): %+v\n", msg, common.HexPre(id.Bytes()), err)
		return false, err
	}
	return true, nil
}

func (cs *consensus) OnFailure(err error, pi module.ProtocolInfo, b []byte) {
	cs.log.Debugf("OnFailure(subprotocol:%v,  err:%+v)\n", pi, err)
}

func (cs *consensus) OnJoin(id module.PeerID) {
	cs.log.Debugf("OnJoin(peer:%v)\n", common.HexPre(id.Bytes()))
}

func (cs *consensus) OnLeave(id module.PeerID) {
	cs.log.Debugf("OnLeave(peer:%v)\n", common.HexPre(id.Bytes()))
}

func (cs *consensus) ReceiveProposalMessage(msg *proposalMessage, unicast bool) error {
	if msg.Height != cs.height || msg.Round != cs.round || cs.step >= stepCommit {
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
	if !cs.currentBlockParts.IsZero() {
		return nil
	}
	cs.proposalPOLRound = msg.proposal.POLRound
	cs.currentBlockParts.Set(newPartSetFromID(msg.proposal.BlockPartSetID), nil, nil)

	if (cs.step == stepTransactionWait || cs.step == stepPropose) && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	}
	return nil
}

func (cs *consensus) ReceiveBlockPartMessage(msg *blockPartMessage, unicast bool) (int, error) {
	if msg.Height != cs.height {
		return -1, nil
	}
	if cs.currentBlockParts.IsZero() || cs.currentBlockParts.IsComplete() {
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
	if cs.currentBlockParts.PartSet.IsComplete() {
		block, err := cs.c.BlockManager().NewBlockDataFromReader(cs.currentBlockParts.NewReader())
		if err != nil {
			cs.log.Warnf("failed to create block. %+v\n", err)
		} else {
			cs.currentBlockParts.block = block
		}
	}

	if (cs.step == stepTransactionWait || cs.step == stepPropose) && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
	return bp.Index(), nil
}

func (cs *consensus) ReceiveVoteMessage(msg *voteMessage, unicast bool) (int, error) {
	psid, ok := cs.lastVotes.getOverTwoThirdsPartSetID()
	lastPC := ok &&
		msg.Height == cs.height-1 &&
		cs.step <= stepTransactionWait &&
		msg.Type == voteTypePrecommit &&
		msg.Round == cs.lastVotes.getRound() &&
		msg.BlockPartSetID.Equal(psid)
	if lastPC {
		if cs.prevValidators != nil {
			index := cs.prevValidators.IndexOf(msg.address())
			if index >= 0 {
				cs.lastVotes.add(index, msg)
			}
		}
	}

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
		cs.handlePrevoteMessage(msg, votes)
	} else {
		cs.handlePrecommitMessage(msg, votes)
	}
	return index, nil
}

func (cs *consensus) ReceiveVoteListMessage(msg *voteListMessage, unicast bool) error {
	var err error
	for i := 0; i < msg.VoteList.Len(); i++ {
		vmsg := msg.VoteList.Get(i)
		if _, e := cs.ReceiveVoteMessage(vmsg, unicast); e != nil {
			cs.log.Warnf("bad vote in vote list. VoteMessage:%v Error:%+v\n", vmsg, e)
			err = errors.Errorf("bad vote in VoteList. LastError: %+v", e)
		}
	}
	return err
}

func (cs *consensus) handlePrevoteMessage(msg *voteMessage, prevotes *voteSet) {
	if cs.step >= stepCommit {
		return
	}

	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()
	if ok {
		if cs.lockedRound < msg.Round && !cs.lockedBlockParts.IsZero() && !cs.lockedBlockParts.ID().Equal(partSetID) {
			cs.lockedRound = -1
			cs.lockedBlockParts.Zerofy()
		}
		if cs.round == msg.Round && partSetID != nil && !cs.currentBlockParts.ID().Equal(partSetID) {
			cs.currentBlockParts.Set(newPartSetFromID(partSetID), nil, nil)
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
	} else if cs.round < msg.Round && cs.step < stepCommit {
		cs.resetForNewRound(msg.Round)
		cs.enterPrevote()
	}
}

func (cs *consensus) handlePrecommitMessage(msg *voteMessage, precommits *voteSet) {
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
	} else if cs.round < msg.Round && cs.step < stepCommit {
		cs.resetForNewRound(msg.Round)
		cs.enterPrecommit()
	}
}

func (cs *consensus) notifySyncer() {
	if cs.syncer != nil {
		cs.syncer.OnEngineStepChange()
	}
}

func (cs *consensus) processPrefetchItems() {
	for i := 0; i < len(cs.prefetchItems); i++ {
		pi := cs.prefetchItems[i]
		if pi.Block().Height() < cs.height {
			last := len(cs.prefetchItems) - 1
			cs.prefetchItems[i] = cs.prefetchItems[last]
			cs.prefetchItems[last] = nil
			cs.prefetchItems = cs.prefetchItems[:last]
			pi.Consume()
		} else if pi.Block().Height() == cs.height {
			last := len(cs.prefetchItems) - 1
			cs.prefetchItems[i] = cs.prefetchItems[last]
			cs.prefetchItems[last] = nil
			cs.prefetchItems = cs.prefetchItems[:last]
			cs.processBlock(pi)
			return
		}
	}
}

func (cs *consensus) enterPropose() {
	cs.resetForNewStep(stepPropose)

	now := time.Now()
	if int(cs.round) > cs.validators.Len()*configRoundTimeoutThresholdFactor {
		cs.nextProposeTime = now.Add(timeoutNewRound)
	} else {
		cs.nextProposeTime = now
	}
	cs.c.Regulator().OnPropose(now)

	hrs := cs.hrs
	cs.timer = time.AfterFunc(timeoutPropose, func() {
		cs.mutex.Lock()
		defer cs.mutex.Unlock()

		if cs.hrs != hrs || !cs.started {
			return
		}
		cs.enterPrevote()
	})

	if cs.isProposer() {
		if !cs.lockedBlockParts.IsZero() {
			cs.sendProposal(cs.lockedBlockParts.PartSet, cs.lockedRound)
			cs.currentBlockParts.Assign(&cs.lockedBlockParts)
		} else {
			if cs.height > 1 && cs.roundLimit > 0 && cs.round > cs.roundLimit && !cs.sentPatch {
				roundEvidences := cs.hvs.getRoundEvidences(cs.roundLimit, cs.nid)
				if roundEvidences != nil {
					err := cs.c.ServiceManager().SendPatch(newSkipPatch(roundEvidences))
					if err != nil {
						cs.sentPatch = true
					}
				}
			}
			var err error
			cvl := cs.lastVotes.commitVoteListForOverTwoThirds()
			cs.cancelBlockRequest, err = cs.c.BlockManager().Propose(cs.lastBlock.ID(), cvl,
				func(blk module.BlockCandidate, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.hrs != hrs || !cs.started {
						if blk != nil {
							blk.Dispose()
						}
						return
					}

					if err != nil {
						cs.log.Warnf("propose cb error: %+v\n", err)
						cs.enterPrevote()
						return
					}

					psb := newPartSetBuffer(configBlockPartSize)
					cs.log.Must(blk.MarshalHeader(psb))
					cs.log.Must(blk.MarshalBody(psb))
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.currentBlockParts.Set(bps, blk, blk)
					cs.enterPrevote()
				},
			)
			if err != nil {
				cs.log.Panicf("propose error: %+v\n", err)
			}
		}
	}
	cs.notifySyncer()
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep(stepPrevote)

	if !cs.lockedBlockParts.IsZero() {
		cs.sendVote(voteTypePrevote, &cs.lockedBlockParts)
	} else if cs.currentBlockParts.IsComplete() {
		hrs := cs.hrs
		if cs.currentBlockParts.validatedBlock != nil {
			cs.sendVote(voteTypePrevote, &cs.currentBlockParts)
		} else {
			var err error
			var canceler module.Canceler
			canceler, err = cs.c.BlockManager().ImportBlock(
				cs.currentBlockParts.block,
				0,
				func(blk module.BlockCandidate, err error) {
					cs.mutex.Lock()
					defer cs.mutex.Unlock()

					if cs.cancelBlockRequest == canceler {
						cs.cancelBlockRequest = nil
					}

					late := cs.hrs.height != hrs.height ||
						cs.hrs.round != hrs.round ||
						cs.hrs.step >= stepCommit ||
						!cs.started
					if late {
						if blk != nil {
							blk.Dispose()
						}
						return
					}

					if configRandomImportFail && rand.Int31n(3) > 0 {
						err = errors.New("bad luck")
						blk.Dispose()
					}

					if err == nil {
						cs.currentBlockParts.validatedBlock = blk
						if cs.hrs.step <= stepPrevoteWait {
							cs.sendVote(voteTypePrevote, &cs.currentBlockParts)
						}
					} else {
						cs.log.Warnf("import cb error: %+v\n", err)
						if cs.hrs.step <= stepPrevoteWait {
							cs.sendVote(voteTypePrevote, nil)
						}
					}
				},
			)
			if err != nil {
				cs.log.Warnf("import error: %+v\n", err)
				cs.sendVote(voteTypePrevote, nil)
				return
			}
			cs.cancelBlockRequest = canceler
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
	cs.resetForNewStep(stepPrevoteWait)

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	msg := newVoteListMessage()
	msg.VoteList = prevotes.voteList()
	if err := cs.roundWAL.writeMessage(msg); err != nil {
		cs.log.Errorf("fail to write WAL: %+v\n", err)
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

			if cs.hrs != hrs || !cs.started {
				return
			}
			cs.enterPrecommit()
		})
	}
}

func (cs *consensus) enterPrecommit() {
	cs.resetForNewStep(stepPrecommit)

	prevotes := cs.hvs.votesFor(cs.round, voteTypePrevote)
	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()

	if !ok {
		cs.log.Traceln("enterPrecommit: no +2/3 precommit")
		cs.sendVote(voteTypePrecommit, nil)
	} else if partSetID == nil {
		cs.log.Traceln("enterPrecommit: nil +2/3 precommit")
		cs.lockedRound = -1
		cs.lockedBlockParts.Zerofy()
		cs.sendVote(voteTypePrecommit, nil)
	} else if cs.lockedBlockParts.ID().Equal(partSetID) {
		cs.log.Traceln("enterPrecommit: update lock round")
		cs.lockedRound = cs.round
		cs.sendVote(voteTypePrecommit, &cs.lockedBlockParts)
	} else if cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.validatedBlock != nil {
		cs.log.Traceln("enterPrecommit: update lock")
		cs.lockedRound = cs.round
		cs.lockedBlockParts.Assign(&cs.currentBlockParts)
		msg := newVoteListMessage()
		msg.VoteList = prevotes.voteList()
		if err := cs.lockWAL.writeMessage(msg); err != nil {
			cs.log.Errorf("fail to write WAL: enterPrecommit: %+v\n", err)
		}
		for i := 0; i < cs.lockedBlockParts.Parts(); i++ {
			msg := newBlockPartMessage()
			msg.Height = cs.height
			msg.Index = uint16(i)
			msg.BlockPart = cs.lockedBlockParts.GetPart(i).Bytes()
			if err := cs.lockWAL.writeMessage(msg); err != nil {
				cs.log.Errorf("fail to write WAL: enterPrecommit: %+v\n", err)
			}
		}
		if err := cs.lockWAL.Sync(); err != nil {
			cs.log.Errorf("fail to sync WAL: enterPrecommit: %+v\n", err)
		}
		cs.sendVote(voteTypePrecommit, &cs.lockedBlockParts)
	} else {
		// polka for a block we don't have
		cs.log.Traceln("enterPrecommit: polka for we don't have")
		if !cs.currentBlockParts.ID().Equal(partSetID) {
			cs.currentBlockParts.Set(newPartSetFromID(partSetID), nil, nil)
		}
		cs.lockedRound = -1
		cs.lockedBlockParts.Zerofy()
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
	cs.resetForNewStep(stepPrecommitWait)

	precommits := cs.hvs.votesFor(cs.round, voteTypePrecommit)
	msg := newVoteListMessage()
	msg.VoteList = precommits.voteList()
	if err := cs.roundWAL.writeMessage(msg); err != nil {
		cs.log.Errorf("fail to write WAL: enterPrecommitWait: %+v\n", err)
	}

	cs.notifySyncer()

	partSetID, ok := precommits.getOverTwoThirdsPartSetID()
	if ok && partSetID != nil {
		cs.enterCommit(precommits, partSetID, cs.round)
	} else if ok && partSetID == nil {
		cs.enterNewRound()
	} else {
		cs.log.Traceln("enterPrecommitWait: start timer")
		hrs := cs.hrs
		cs.timer = time.AfterFunc(timeoutPrecommit, func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs || !cs.started {
				return
			}
			cs.enterNewRound()
		})
	}
}

func (cs *consensus) commitAndEnterNewHeight() {
	if cs.currentBlockParts.validatedBlock == nil {
		hrs := cs.hrs
		if cs.cancelBlockRequest != nil {
			cs.cancelBlockRequest.Cancel()
			cs.cancelBlockRequest = nil
		}
		_, err := cs.c.BlockManager().ImportBlock(
			cs.currentBlockParts.block,
			module.ImportByForce,
			func(blk module.BlockCandidate, err error) {
				cs.mutex.Lock()
				defer cs.mutex.Unlock()

				if cs.hrs != hrs || !cs.started {
					if blk != nil {
						blk.Dispose()
					}
					return
				}

				if err != nil {
					cs.log.Panicf("commitAndEnterNewHeight: %+v\n", err)
				}
				cs.currentBlockParts.validatedBlock = blk
				err = cs.c.BlockManager().Finalize(cs.currentBlockParts.validatedBlock)
				if err != nil {
					cs.log.Panicf("commitAndEnterNewHeight: %+v\n", err)
				}
				cs.enterNewHeight()
			},
		)
		if err != nil {
			cs.log.Panicf("commitAndEnterNewHeight: %+v\n", err)
		}
	} else {
		err := cs.c.BlockManager().Finalize(cs.currentBlockParts.validatedBlock)
		if err != nil {
			cs.log.Panicf("commitAndEnterNewHeight: %+v\n", err)
		}
		cs.enterNewHeight()
	}
}

func (cs *consensus) enterCommit(precommits *voteSet, partSetID *PartSetID, round int32) {
	cs.resetForNewStep(stepCommit)
	cs.commitRound = round

	msg := newVoteListMessage()
	msg.VoteList = precommits.voteList()
	if err := cs.commitWAL.writeMessage(msg); err != nil {
		cs.log.Errorf("fail to write WAL: enterCommit: %+v\n", err)
	}
	if err := cs.commitWAL.Sync(); err != nil {
		cs.log.Errorf("fail to sync WAL: cs.enterCommit: %+v\n", err)
	}

	cs.nextProposeTime = time.Now()
	if cs.consumedNonunicast || cs.validators.Len() == 1 {
		if cs.timestamper == nil {
			cs.nextProposeTime = cs.nextProposeTime.Add(cs.c.Regulator().CommitTimeout())
		}
	}

	if !cs.currentBlockParts.ID().Equal(partSetID) {
		cs.currentBlockParts.Set(newPartSetFromID(partSetID), nil, nil)
	}

	cs.notifySyncer()

	if cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
}

func (cs *consensus) enterNewRound() {
	cs.resetForNewRound(cs.round + 1)
	cs.notifySyncer()

	now := time.Now()
	if cs.nextProposeTime.After(now) {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs || !cs.started {
				return
			}
			cs.enterPropose()
		})
	} else {
		cs.enterPropose()
	}
}

func (cs *consensus) enterTransactionWait() {
	cs.resetForNewStep(stepTransactionWait)

	waitTx := cs.minimizeBlockGen
	if len(cs.lastBlock.NormalTransactions().Hash()) > 0 {
		waitTx = false
	}

	if waitTx {
		hrs := cs.hrs
		callback := cs.c.BlockManager().WaitForTransaction(cs.lastBlock.ID(), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs || !cs.started {
				return
			}

			cs.enterPropose()
		})
		if callback {
			cs.notifySyncer()
			return
		}
	}
	cs.notifySyncer()
	cs.enterPropose()
}

func (cs *consensus) enterNewHeight() {
	votes := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit)
	cs.resetForNewHeight(cs.currentBlockParts.validatedBlock, votes)
	cs.notifySyncer()

	now := time.Now()
	if cs.nextProposeTime.After(now) {
		hrs := cs.hrs
		cs.timer = time.AfterFunc(cs.nextProposeTime.Sub(now), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs || !cs.started {
				return
			}
			cs.processPrefetchItems()
			if cs.step <= stepTransactionWait {
				cs.enterTransactionWait()
			}
		})
	} else {
		cs.processPrefetchItems()
		if cs.step <= stepTransactionWait {
			cs.enterTransactionWait()
		}
	}
}

func (cs *consensus) sendProposal(blockParts PartSet, polRound int32) error {
	msg := newProposalMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.BlockPartSetID = blockParts.ID()
	msg.POLRound = polRound
	err := msg.sign(cs.c.Wallet())
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.writeMessageBytes(msg.subprotocol(), msgBS); err != nil {
		cs.log.Errorf("fail to write WAL: sendProposal: %+v\n", err)
		return err
	}
	if err := cs.roundWAL.Sync(); err != nil {
		cs.log.Errorf("fail to sync WAL: sendProposal: %+v\n", err)
		return err
	}
	cs.log.Debugf("sendProposal %v\n", msg)
	err = cs.ph.Broadcast(protoProposal, msgBS, module.BROADCAST_ALL)
	if err != nil {
		cs.log.Warnf("sendProposal: %+v\n", err)
		return err
	}

	if polRound >= 0 {
		prevotes := cs.hvs.votesFor(polRound, voteTypePrevote)
		vl := prevotes.voteListForOverTwoThirds()
		vlmsg := newVoteListMessage()
		vlmsg.VoteList = vl
		cs.log.Debugf("sendVoteList %v\n", vlmsg)
		vlmsgBS, err := msgCodec.MarshalToBytes(vlmsg)
		if err != nil {
			return err
		}
		err = cs.ph.Multicast(protoVoteList, vlmsgBS, module.ROLE_VALIDATOR)
		if err != nil {
			cs.log.Warnf("sendVoteList: %+v\n", err)
			return err
		}
	}

	bpmsg := newBlockPartMessage()
	bpmsg.Height = cs.height
	for i := 0; i < blockParts.Parts(); i++ {
		bpmsg.BlockPart = blockParts.GetPart(i).Bytes()
		bpmsg.Index = uint16(i)
		bpmsg.Nonce = cs.round
		bpmsgBS, err := msgCodec.MarshalToBytes(bpmsg)
		if err != nil {
			return err
		}
		cs.log.Debugf("sendBlockPart %v\n", bpmsg)
		err = cs.ph.Broadcast(protoBlockPart, bpmsgBS, module.BROADCAST_ALL)
		if err != nil {
			cs.log.Warnf("sendBlockPart: %+v\n", err)
			return err
		}
	}

	return nil
}

func (cs *consensus) voteTimestamp() int64 {
	var timestamp int64
	blockIota := int64(cs.c.Regulator().MinCommitTimeout() / time.Microsecond)
	if !cs.lockedBlockParts.IsZero() {
		timestamp = cs.lockedBlockParts.block.Timestamp() + blockIota
	} else if cs.currentBlockParts.IsComplete() {
		timestamp = cs.currentBlockParts.block.Timestamp() + blockIota
	}
	now := common.UnixMicroFromTime(time.Now())
	if now > timestamp {
		timestamp = now
	}
	if cs.timestamper != nil {
		timestamp = cs.timestamper.GetVoteTimestamp(cs.height, timestamp)
	}
	return timestamp
}

func (cs *consensus) sendVote(vt voteType, blockParts *blockPartSet) error {
	if cs.validators.IndexOf(cs.c.Wallet().Address()) < 0 {
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
		msg.BlockID = cs.nid
		msg.BlockPartSetID = nil
	}
	msg.Timestamp = cs.voteTimestamp()

	err := msg.sign(cs.c.Wallet())
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.writeMessageBytes(msg.subprotocol(), msgBS); err != nil {
		cs.log.Errorf("fail to write WAL: sendVote: %+v\n", err)
	}
	if err := cs.roundWAL.Sync(); err != nil {
		cs.log.Errorf("fail to sync WAL: sendVote: %+v\n", err)
	}
	cs.log.Debugf("sendVote %v\n", msg)
	if vt == voteTypePrevote {
		err = cs.ph.Multicast(protoVote, msgBS, module.ROLE_VALIDATOR)
	} else {
		err = cs.ph.Broadcast(protoVote, msgBS, module.BROADCAST_ALL)
	}
	if err != nil {
		cs.log.Warnf("sendVote: %+v\n", err)
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
	return v.Address().Equal(cs.c.Wallet().Address())
}

func (cs *consensus) isProposer() bool {
	return cs.isProposerFor(cs.height, cs.round)
}

func (cs *consensus) isProposalAndPOLPrevotesComplete() bool {
	if !cs.currentBlockParts.IsComplete() {
		return false
	}
	if cs.proposalPOLRound >= 0 {
		prevotes := cs.hvs.votesFor(cs.proposalPOLRound, voteTypePrevote)
		if id, _ := prevotes.getOverTwoThirdsPartSetID(); id != nil {
			return true
		}
		return false
	}
	return true
}

func (cs *consensus) applyRoundWAL() error {
	wr, err := cs.wm.OpenForRead(path.Join(cs.walDir, configRoundWALID))
	if err != nil {
		return err
	}
	defer func() {
		cs.log.Must(wr.Close())
	}()
	round := int32(0)
	rstep := stepNewHeight
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			cs.log.Warnf("applyRoundWAL: %+v\n", err)
			err := wr.CloseAndRepair()
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
			if !m.address().Equal(cs.c.Wallet().Address()) {
				continue
			}
			cs.log.Tracef("WAL: my proposal %v\n", m)
			if m.round() < round || (m.round() == round && rstep <= stepPropose) {
				round = m.round()
				rstep = stepPropose
			}
		case *voteMessage:
			if m.height() != cs.height {
				continue
			}
			if !m.address().Equal(cs.c.Wallet().Address()) {
				continue
			}
			cs.log.Tracef("WAL: my vote %v\n", m)
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
				cs.log.Tracef("WAL: round vote %v\n", vmsg)
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
	wr, err := cs.wm.OpenForRead(path.Join(cs.walDir, configLockWALID))
	if err != nil {
		return err
	}
	defer func() {
		cs.log.Must(wr.Close())
	}()
	var bpset PartSet
	var bpsetLockRound int32
	var lastBPSet PartSet
	var lastBPSetLockRound int32
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			cs.log.Warnf("applyLockWAL: %+v\n", err)
			err := wr.CloseAndRepair()
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
				cs.log.Tracef("WAL: round vote %v\n", vmsg)
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
				cs.log.Tracef("WAL: POL R=%v psid=%v\n", vmsg.Round, psid)
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
			cs.log.Tracef("WAL: blockPart %v\n", m)
			if err == nil && bpset.IsComplete() {
				lastBPSet = bpset
				lastBPSetLockRound = bpsetLockRound
				cs.log.Tracef("WAL: blockPart complete\n")
			}
		}
	}
	if lastBPSet != nil {
		blk, err := cs.c.BlockManager().NewBlockDataFromReader(lastBPSet.NewReader())
		if err != nil {
			return err
		}
		cs.currentBlockParts.Set(lastBPSet, blk, nil)
		cs.lockedBlockParts.Assign(&cs.currentBlockParts)
		cs.lockedRound = lastBPSetLockRound
	}
	return nil
}

func (cs *consensus) applyCommitWAL(prevValidators addressIndexer) error {
	wr, err := cs.wm.OpenForRead(path.Join(cs.walDir, configCommitWALID))
	if err != nil {
		return nil
	}
	defer func() {
		cs.log.Must(wr.Close())
	}()
	for {
		bs, err := wr.ReadBytes()
		if IsEOF(err) {
			break
		} else if IsCorruptedWAL(err) || IsUnexpectedEOF(err) {
			cs.log.Warnf("applyCommitWAL: %+v\n", err)
			err := wr.CloseAndRepair()
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
					cs.log.Tracef("WAL: round vote %v\n", msg)
					index := prevValidators.IndexOf(msg.address())
					if index < 0 {
						return errors.Errorf("bad voter %v", msg.address())
					}
					_ = vs.add(index, msg)
				}
				psid, ok := vs.getOverTwoThirdsPartSetID()
				if ok && psid != nil {
					cs.lastVotes = vs.voteSetForOverTwoThird()
				}
			} else if m.VoteList.Get(0).height() == cs.height {
				for i := 0; i < m.VoteList.Len(); i++ {
					vmsg := m.VoteList.Get(i)
					cs.log.Tracef("WAL: round vote %v\n", vmsg)
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

func (cs *consensus) applyWAL(prevValidators addressIndexer) error {
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

type addressIndexer interface {
	IndexOf(module.Address) int
	Len() int
}

type emptyAddressIndexer struct {
}

func (vl *emptyAddressIndexer) IndexOf(module.Address) int {
	return -1
}

func (vl *emptyAddressIndexer) Len() int {
	return 0
}

func (cs *consensus) applyGenesis(prevValidators addressIndexer) error {
	// apply genesis commit vote set in the same way as commit WAL
	blk, cvs, err := cs.c.BlockManager().GetGenesisData()
	if err != nil {
		return err
	}
	if blk == nil {
		return nil
	}
	if blk.Height() != cs.lastBlock.Height() {
		return nil
	}
	cvl, ok := cvs.(*commitVoteList)
	if !ok {
		return errors.ErrInvalidState
	}
	vl := cvl.voteList(blk.Height(), blk.ID())
	vs := newVoteSet(prevValidators.Len())
	for i := 0; i < vl.Len(); i++ {
		msg := vl.Get(i)
		cs.log.Tracef("Genesis: round vote %v\n", msg)
		index := prevValidators.IndexOf(msg.address())
		if index < 0 {
			return errors.Errorf("bad voter %v", msg.address())
		}
		_ = vs.add(index, msg)
	}
	psid, ok := vs.getOverTwoThirdsPartSetID()
	if ok && psid != nil {
		cs.lastVotes = vs.voteSetForOverTwoThird()
	}
	return nil
}

func (cs *consensus) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	lastBlock, err := cs.c.BlockManager().GetLastBlock()
	if err != nil {
		return err
	}
	var validators addressIndexer
	if lastBlock.Height() > 0 {
		prevBlock, err := cs.c.BlockManager().GetBlockByHeight(lastBlock.Height() - 1)
		if err != nil {
			return err
		}
		validators = prevBlock.NextValidators()
	} else {
		validators = &emptyAddressIndexer{}
	}

	cs.ph, err = cs.c.NetworkManager().RegisterReactor("consensus", module.ProtoConsensus, cs, csProtocols, configEnginePriority)
	if err != nil {
		return err
	}

	cs.resetForNewHeight(lastBlock, newVoteSet(0))
	cs.prevValidators = validators
	if err := cs.applyWAL(validators); err != nil {
		return err
	}
	if err := cs.applyGenesis(validators); err != nil {
		return err
	}

	ww, err := cs.wm.OpenForWrite(path.Join(cs.walDir, configRoundWALID), &WALConfig{
		FileLimit:  configRoundWALDataSize,
		TotalLimit: configRoundWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.roundWAL = &walMessageWriter{ww}

	ww, err = cs.wm.OpenForWrite(path.Join(cs.walDir, configLockWALID), &WALConfig{
		FileLimit:  configLockWALDataSize,
		TotalLimit: configLockWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.lockWAL = &walMessageWriter{ww}

	ww, err = cs.wm.OpenForWrite(path.Join(cs.walDir, configCommitWALID), &WALConfig{
		FileLimit:  configCommitWALDataSize,
		TotalLimit: configCommitWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.commitWAL = &walMessageWriter{ww}

	cs.started = true
	cs.log.Infof("Start consensus wallet:%v", common.HexPre(cs.c.Wallet().Address().ID()))
	cs.syncer = newSyncer(cs, cs.log, cs.c.NetworkManager(), cs.c.BlockManager(), &cs.mutex, cs.c.Wallet().Address())
	cs.syncer.Start()
	if cs.step == stepNewHeight && cs.round == 0 {
		cs.enterTransactionWait()
	} else if cs.step == stepNewHeight && cs.round > 0 {
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

func (cs *consensus) Term() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.started = false

	cs.c.NetworkManager().UnregisterReactor(cs)
	if cs.syncer != nil {
		cs.syncer.Stop()
	}

	if cs.timer != nil {
		cs.timer.Stop()
	}
	if cs.cancelBlockRequest != nil {
		cs.cancelBlockRequest.Cancel()
		cs.cancelBlockRequest = nil
	}
	if cs.roundWAL != nil {
		cs.log.Must(cs.roundWAL.Close())
	}
	if cs.lockWAL != nil {
		cs.log.Must(cs.lockWAL.Close())
	}
	if cs.commitWAL != nil {
		cs.log.Must(cs.commitWAL.Close())
	}

	if cs.log != nil {
		cs.log.Infof("Term consensus.\n")
	}
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
	if c.commitVotes == nil {
		return nil, errors.ErrNotFound
	}
	return c.commitVotes, nil
}

func (cs *consensus) getCommit(h int64) (*commit, error) {
	if h > cs.height || (h == cs.height && cs.step < stepCommit) {
		return nil, errors.ErrNotFound
	}

	c := cs.commitCache.GetByHeight(h)
	if c != nil {
		return c, nil
	}

	if h == cs.height && !cs.currentBlockParts.PartSet.IsComplete() {
		pcs := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit)
		return &commit{
			height:       h,
			commitVotes:  pcs.commitVoteListForOverTwoThirds(),
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts.PartSet,
		}, nil
	}

	if h == cs.height {
		pcs := cs.hvs.votesFor(cs.commitRound, voteTypePrecommit)
		c = &commit{
			height:       h,
			commitVotes:  pcs.commitVoteListForOverTwoThirds(),
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts.PartSet,
		}
	} else {
		b, err := cs.c.BlockManager().GetBlockByHeight(h)
		if err != nil {
			return nil, err
		}
		var cvl *commitVoteList
		if h == cs.height-1 {
			cvl = cs.lastVotes.commitVoteListForOverTwoThirds()
		} else {
			nb, err := cs.c.BlockManager().GetBlockByHeight(h + 1)
			if err != nil {
				return nil, err
			}
			cvl = nb.Votes().(*commitVoteList)
		}
		vl := cvl.voteList(h, b.ID())
		psb := newPartSetBuffer(configBlockPartSize)
		cs.log.Must(b.MarshalHeader(psb))
		cs.log.Must(b.MarshalBody(psb))
		bps := psb.PartSet()
		c = &commit{
			height:       h,
			commitVotes:  cvl,
			votes:        vl,
			blockPartSet: bps,
		}
	}
	cs.commitCache.Put(c)
	return c, nil
}

func (cs *consensus) GetCommitBlockParts(h int64) PartSet {
	c, err := cs.getCommit(h)
	if err != nil {
		cs.log.Panicf("cs.GetCommitBlockParts: %+v\n", err)
	}
	return c.blockPartSet
}

func (cs *consensus) GetCommitPrecommits(h int64) *voteList {
	c, err := cs.getCommit(h)
	if err != nil {
		cs.log.Panicf("cs.GetCommitPrecommits: %+v\n", err)
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
	prs.Sync = cs.syncing
	bp := cs.currentBlockParts
	// TODO optimize
	if !bp.IsZero() && cs.step >= stepCommit {
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

func (cs *consensus) ReceiveBlock(br fastsync.BlockResult) {
	blk := br.Block()
	cs.log.Debugf("ReceiveBlock Height:%d\n", blk.Height())

	if cs.height < blk.Height() {
		cs.prefetchItems = append(cs.prefetchItems, br)
		return
	}

	if cs.height > blk.Height() ||
		cs.height == blk.Height() && cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		br.Consume()
		return
	}

	cs.processBlock(br)
}

func (cs *consensus) processBlock(br fastsync.BlockResult) {
	if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.log.Panicf("shall not happen\n")
	}

	blk := br.Block()
	cs.log.Debugf("processBlock Height:%d\n", blk.Height())

	cvl := NewCommitVoteSetFromBytes(br.Votes())
	if cvl == nil {
		br.Reject()
		return
	}

	votes := cvl.(*commitVoteList)
	vl := votes.voteList(blk.Height(), blk.ID())
	for i := 0; i < vl.Len(); i++ {
		m := vl.Get(i)
		index := cs.validators.IndexOf(m.address())
		if index < 0 {
			br.Reject()
			return
		}
		cs.hvs.add(index, m)
	}

	precommits := cs.hvs.votesFor(votes.Round, voteTypePrecommit)
	id, ok := precommits.getOverTwoThirdsPartSetID()
	if !ok {
		br.Reject()
		return
	}
	bps := newPartSetFromID(id)
	var validatedBlock module.BlockCandidate
	if cs.currentBlockParts.ID().Equal(id) {
		validatedBlock = cs.currentBlockParts.validatedBlock
	}
	cs.currentBlockParts.Set(bps, blk, validatedBlock)
	cs.syncing = false
	br.Consume()
	if cs.step < stepCommit {
		cs.enterCommit(precommits, id, votes.Round)
	} else {
		cs.commitAndEnterNewHeight()
	}
}

type walMessageWriter struct {
	WALWriter
}

func (w *walMessageWriter) writeMessage(msg message) error {
	bs := make([]byte, 2, 32)
	binary.BigEndian.PutUint16(bs, msg.subprotocol())
	writer := bytes.NewBuffer(bs)
	if err := msgCodec.Marshal(writer, msg); err != nil {
		return err
	}
	//cs.log.Tracef("write WAL: %+v\n", msg)
	_, err := w.WriteBytes(writer.Bytes())
	return err
}

func (w *walMessageWriter) writeMessageBytes(sp uint16, msg []byte) error {
	bs := make([]byte, 2+len(msg))
	binary.BigEndian.PutUint16(bs, sp)
	copy(bs[2:], msg)
	//cs.log.Tracef("write WAL: %x\n", bs)
	_, err := w.WriteBytes(bs)
	return err
}
