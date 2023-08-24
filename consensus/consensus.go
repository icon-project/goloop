package consensus

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server/metric"
)

const (
	configRandomImportFail = false
)

var CsProtocols = []module.ProtocolInfo{
	ProtoProposal,
	ProtoBlockPart,
	ProtoVote,
	ProtoVoteList,
}

type LastVoteData struct {
	Height     int64
	VotesBytes []byte
}

const (
	timeoutPropose   = time.Second * 1
	timeoutPrevote   = time.Second * 1
	timeoutPrecommit = time.Second * 1
	timeoutNewRound  = time.Second * 1
)

const (
	ConfigEnginePriority = 2
	ConfigSyncerPriority = 3

	ConfigBlockPartSize               = 1024 * 100
	configCommitCacheCap              = 60
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

type consensus struct {
	hrs

	c           base.Chain
	log         log.Logger
	ph          module.ProtocolHandler
	mutex       common.Mutex
	syncer      Syncer
	walDir      string
	wm          WALManager
	roundWAL    *WalMessageWriter
	lockWAL     *WalMessageWriter
	commitWAL   *WalMessageWriter
	timestamper module.Timestamper
	nid         []byte
	bpp         fastsync.BlockProofProvider
	srcUID      []byte

	lastBlock          module.Block
	validators         module.ValidatorList
	prevValidators     addressIndexer
	members            module.MemberList
	minimizeBlockGen   bool
	roundLimit         int32
	sentPatch          bool
	lastVotes          VoteSet
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
	pcmForLastBlock    module.BTPProofContextMap
	nextPCM            module.BTPProofContextMap

	timer *time.Timer

	// commit cache
	commitCache *commitCache

	// prefetch buffer
	prefetchItems []fastsync.BlockResult

	// monitor
	metric *metric.ConsensusMetric

	lastVoteData *LastVoteData
}

func NewConsensus(
	c base.Chain,
	walDir string,
	timestamper module.Timestamper,
	bpp fastsync.BlockProofProvider,
) module.Consensus {
	cs := New(c, walDir, nil, timestamper, bpp, nil)
	cs.log.Debugf("NewConsensus\n")
	return cs
}

func New(
	c base.Chain,
	walDir string,
	wm WALManager,
	timestamper module.Timestamper,
	bpp fastsync.BlockProofProvider,
	lastVoteData *LastVoteData,
) *consensus {
	if wm == nil {
		wm = defaultWALManager
	}
	cs := &consensus{
		c:            c,
		walDir:       walDir,
		wm:           wm,
		commitCache:  newCommitCache(configCommitCacheCap),
		metric:       metric.NewConsensusMetric(c.MetricContext()),
		timestamper:  timestamper,
		nid:          codec.MustMarshalToBytes(c.NID()),
		bpp:          bpp,
		srcUID:       module.GetSourceNetworkUID(c),
		lastVoteData: lastVoteData,
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
		cs.c.NetworkManager().SetRole(cs.height, module.RoleValidator, peerIDs...)
	}
	nextMembers, err := cs.c.ServiceManager().GetMembers(cs.lastBlock.Result())
	if err != nil {
		cs.log.Warnf("cannot get members. error:%+v\n", err)
	} else {
		if cs.members == nil || !cs.members.Equal(nextMembers) {
			cs.members = nextMembers
			var peerIDs []module.PeerID
			if nextMembers != nil {
				for it := nextMembers.Iterator(); it.Has(); cs.log.Must(it.Next()) {
					addr, _ := it.Get()
					peerIDs = append(peerIDs, network.NewPeerIDFromAddress(addr))
				}
				cs.c.NetworkManager().SetRole(cs.height, module.RoleNormal, peerIDs...)
			}
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
	cs.pcmForLastBlock = cs.nextPCM
	nextPCM, err := cs.nextPCM.Update(prevBlock)
	cs.log.Must(err)
	cs.nextPCM = nextPCM
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
	if cs.step < stepPropose && step > stepPropose {
		now := time.Now()
		cs.nextProposeTime = now
		cs.c.Regulator().OnPropose(now)
	}
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

	msg, err := UnmarshalMessage(sp.Uint16(), bs)
	if err != nil {
		cs.log.Warnf("malformed consensus message: OnReceive(subprotocol:%v, from:%v): %+v\n", sp, common.HexPre(id.Bytes()), err)
		return false, err
	}
	cs.log.Debugf("OnReceive(msg:%v, from:%v)\n", msg, common.HexPre(id.Bytes()))
	if err = msg.Verify(); err != nil {
		cs.log.Warnf("consensus message verify failed: OnReceive(msg:%v, from:%v): %+v\n", msg, common.HexPre(id.Bytes()), err)
		return false, err
	}
	switch m := msg.(type) {
	case *ProposalMessage:
		err = cs.ReceiveProposalMessage(m, false)
	case *BlockPartMessage:
		_, err = cs.ReceiveBlockPartMessage(m, false)
	case *VoteMessage:
		_, err = cs.ReceiveVoteMessage(m, false)
	case *VoteListMessage:
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

func (cs *consensus) OnJoin(id module.PeerID) {
	cs.log.Debugf("OnJoin(peer:%v)\n", common.HexPre(id.Bytes()))
}

func (cs *consensus) OnLeave(id module.PeerID) {
	cs.log.Debugf("OnLeave(peer:%v)\n", common.HexPre(id.Bytes()))
}

func (cs *consensus) ReceiveProposalMessage(msg *ProposalMessage, unicast bool) error {
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
	cs.currentBlockParts.SetByPartSetID(msg.proposal.BlockPartSetID)

	if (cs.step == stepTransactionWait || cs.step == stepPropose) && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	}
	return nil
}

func (cs *consensus) ReceiveBlockPartMessage(msg *BlockPartMessage, unicast bool) (int, error) {
	if msg.Height != cs.height {
		return -1, nil
	}
	if cs.currentBlockParts.IsZero() || cs.currentBlockParts.IsComplete() {
		return -1, nil
	}

	bp, err := NewPart(msg.BlockPart)
	if err != nil {
		return -1, err
	}
	if cs.currentBlockParts.GetPart(bp.Index()) != nil {
		return -1, nil
	}
	added, err := cs.currentBlockParts.AddPart(bp, cs.c.BlockManager())
	if !added && err != nil {
		return -1, err
	}
	if added && err != nil {
		cs.log.Warnf("fail to create block. %+v", err)
	}

	if (cs.step == stepTransactionWait || cs.step == stepPropose) && cs.isProposalAndPOLPrevotesComplete() {
		cs.enterPrevote()
	} else if cs.step == stepCommit && cs.currentBlockParts.IsComplete() {
		cs.commitAndEnterNewHeight()
	}
	return bp.Index(), nil
}

func (cs *consensus) ReceiveVoteMessage(msg *VoteMessage, unicast bool) (int, error) {
	lastPC :=
		msg.Height == cs.height-1 &&
			cs.step <= stepTransactionWait &&
			msg.Type == VoteTypePrecommit
	if lastPC {
		if cs.prevValidators != nil {
			index := cs.prevValidators.IndexOf(msg.address())
			if index >= 0 {
				err := msg.VerifyNTSDProofParts(
					cs.pcmForLastBlock, cs.srcUID, index,
				)
				if err != nil {
					return -1, err
				}
				cs.lastVotes.Add(index, msg)
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
	err := msg.VerifyNTSDProofParts(cs.nextPCM, cs.srcUID, index)
	if err != nil {
		return -1, err
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
	if msg.Type == VoteTypePrevote {
		cs.handlePrevoteMessage(msg, votes)
	} else {
		cs.handlePrecommitMessage(msg, votes)
	}
	return index, nil
}

func (cs *consensus) ReceiveVoteListMessage(msg *VoteListMessage, unicast bool) error {
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

func (cs *consensus) handlePrevoteMessage(msg *VoteMessage, prevotes *voteSet) {
	if cs.step >= stepCommit {
		return
	}

	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()
	if ok {
		if cs.lockedRound < msg.Round && !cs.lockedBlockParts.IsZero() && !cs.lockedBlockParts.ID().Equal(partSetID) {
			cs.lockedRound = -1
			cs.lockedBlockParts.Zerofy()
		}
		if cs.round == msg.Round && partSetID != nil {
			cs.currentBlockParts.SetByPartSetID(partSetID)
		}
	}

	if cs.round > msg.Round && cs.step < stepPrevote && msg.Round == cs.proposalPOLRound && cs.isProposalAndPOLPrevotesComplete() {
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

func (cs *consensus) handlePrecommitMessage(msg *VoteMessage, precommits *voteSet) {
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
	for i := 0; i < len(cs.prefetchItems); {
		pi := cs.prefetchItems[i]
		if pi.Block().Height() < cs.height {
			last := len(cs.prefetchItems) - 1
			cs.prefetchItems[i] = cs.prefetchItems[last]
			cs.prefetchItems[last] = nil
			cs.prefetchItems = cs.prefetchItems[:last]
			pi.Consume()
			continue
		} else if pi.Block().Height() == cs.height {
			last := len(cs.prefetchItems) - 1
			cs.prefetchItems[i] = cs.prefetchItems[last]
			cs.prefetchItems[last] = nil
			cs.prefetchItems = cs.prefetchItems[:last]
			cs.processBlock(pi)
			return
		}
		i++
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
					if err == nil {
						cs.sentPatch = true
					}
				}
			}
			var err error
			cvl, err := cs.lastVotes.CommitVoteSet(cs.pcmForLastBlock)
			if err != nil {
				cs.log.Panicf("fail to make CommitVoteSet: %+v", err)
			}
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

					psb := NewPartSetBuffer(ConfigBlockPartSize)
					cs.log.Must(blk.MarshalHeader(psb))
					cs.log.Must(blk.MarshalBody(psb))
					bps := psb.PartSet()

					cs.sendProposal(bps, -1)
					cs.currentBlockParts.SetByPartSetAndValidatedBlock(bps, blk)
					cs.enterPrevote()
				},
			)
			if err != nil {
				cs.log.Warnf("propose error: %+v\n", err)
			}
		}
	} else {
		if cs.isProposalAndPOLPrevotesComplete() {
			cs.enterPrevote()
		}
	}
	cs.notifySyncer()
}

func (cs *consensus) proposalHasValidProposer() bool {
	if !cs.isProposalAndPOLPrevotesComplete() {
		return false
	}
	// if POLRound > 0, here we have polka for the round, which means at least
	// +1/3 honest validators checked proposer before they send block prevote
	if cs.proposalPOLRound == -1 {
		proposer := cs.currentBlockParts.block.Proposer()
		index := cs.validators.IndexOf(proposer)
		if index < 0 {
			return false
		}
		if cs.getProposerIndex(cs.height, cs.round) != index {
			return false
		}
	}
	return true
}

func (cs *consensus) enterPrevote() {
	cs.resetForNewStep(stepPrevote)

	if !cs.lockedBlockParts.IsZero() {
		cs.sendVote(VoteTypePrevote, &cs.lockedBlockParts)
	} else if cs.currentBlockParts.HasBlockData() {
		hrs := cs.hrs
		if cs.currentBlockParts.HasValidatedBlock() {
			cs.sendVote(VoteTypePrevote, &cs.currentBlockParts)
		} else if !cs.proposalHasValidProposer() {
			cs.sendVote(VoteTypePrevote, nil)
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
						cur := cs.currentBlockParts.block
						// do not set validated block if currentBlockParts
						// has different ID from blk.ID
						if cur != nil && bytes.Equal(cur.ID(), blk.ID()) {
							cs.currentBlockParts.SetByValidatedBlock(blk)
						}
						if cs.hrs.step <= stepPrevoteWait {
							cs.sendVote(VoteTypePrevote, &cs.currentBlockParts)
						}
					} else {
						cs.log.Warnf("import cb error: %+v\n", err)
						if cs.hrs.step <= stepPrevoteWait {
							cs.sendVote(VoteTypePrevote, nil)
						}
					}
				},
			)
			if err != nil {
				cs.log.Warnf("import error: %+v\n", err)
				cs.sendVote(VoteTypePrevote, nil)
				return
			}
			cs.cancelBlockRequest = canceler
		}
	} else {
		cs.sendVote(VoteTypePrevote, nil)
	}

	cs.notifySyncer()

	// we double-check vote count because we may not sendVote
	if cs.step == stepPrevote {
		prevotes := cs.hvs.votesFor(cs.round, VoteTypePrevote)
		if prevotes.hasOverTwoThirds() {
			cs.enterPrevoteWait()
		}
	}
}

func (cs *consensus) enterPrevoteWait() {
	cs.resetForNewStep(stepPrevoteWait)

	prevotes := cs.hvs.votesFor(cs.round, VoteTypePrevote)
	msg := newVoteListMessage()
	msg.VoteList = prevotes.voteList()
	if err := cs.roundWAL.WriteMessage(msg); err != nil {
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

	prevotes := cs.hvs.votesFor(cs.round, VoteTypePrevote)
	partSetID, ok := prevotes.getOverTwoThirdsPartSetID()

	if !ok {
		cs.log.Traceln("enterPrecommit: no +2/3 precommit")
		cs.sendVote(VoteTypePrecommit, nil)
	} else if partSetID == nil {
		cs.log.Traceln("enterPrecommit: nil +2/3 precommit")
		cs.lockedRound = -1
		cs.lockedBlockParts.Zerofy()
		cs.sendVote(VoteTypePrecommit, nil)
	} else if cs.lockedBlockParts.ID().Equal(partSetID) {
		cs.log.Traceln("enterPrecommit: update lock round")
		cs.lockedRound = cs.round
		cs.sendVote(VoteTypePrecommit, &cs.lockedBlockParts)
	} else if cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.HasBlockData() {
		cs.log.Traceln("enterPrecommit: update lock")
		cs.lockedRound = cs.round
		cs.lockedBlockParts.Assign(&cs.currentBlockParts)
		msg := newVoteListMessage()
		msg.VoteList = prevotes.voteList()
		if err := cs.lockWAL.WriteMessage(msg); err != nil {
			cs.log.Errorf("fail to write WAL: enterPrecommit: %+v\n", err)
		}
		for i := 0; i < cs.lockedBlockParts.Parts(); i++ {
			msg := newBlockPartMessage()
			msg.Height = cs.height
			msg.Index = uint16(i)
			msg.BlockPart = cs.lockedBlockParts.GetPart(i).Bytes()
			if err := cs.lockWAL.WriteMessage(msg); err != nil {
				cs.log.Errorf("fail to write WAL: enterPrecommit: %+v\n", err)
			}
		}
		if err := cs.lockWAL.Sync(); err != nil {
			cs.log.Errorf("fail to sync WAL: enterPrecommit: %+v\n", err)
		}
		cs.sendVote(VoteTypePrecommit, &cs.lockedBlockParts)
	} else if cs.currentBlockParts.ID().Equal(partSetID) && cs.currentBlockParts.IsComplete() {
		// polka for a block that we cannot create
		// we cannot advance without upgrading the node
		cs.log.Panicf("Cannot create block for polka block part set. Consider node upgrade. bpsID=%s", cs.currentBlockParts.ID())
	} else {
		// polka for a block we don't have.
		// send nil precommit because we cannot write locked block on the WAL.
		cs.log.Traceln("enterPrecommit: polka for the block we don't have")
		cs.currentBlockParts.SetByPartSetID(partSetID)
		cs.lockedRound = -1
		cs.lockedBlockParts.Zerofy()
		cs.sendVote(VoteTypePrecommit, nil)
	}

	cs.notifySyncer()

	// sendVote increases vote count. We check the count there. However,
	// we double-check the count because we may not send vote (e.g. not a
	// validator).
	// check current step since sendVote may have changed step
	if cs.step == stepPrecommit {
		precommits := cs.hvs.votesFor(cs.round, VoteTypePrecommit)
		if precommits.hasOverTwoThirds() {
			cs.enterPrecommitWait()
		}
	}
}

func (cs *consensus) enterPrecommitWait() {
	cs.resetForNewStep(stepPrecommitWait)

	precommits := cs.hvs.votesFor(cs.round, VoteTypePrecommit)
	msg := newVoteListMessage()
	msg.VoteList = precommits.voteList()
	if err := cs.roundWAL.WriteMessage(msg); err != nil {
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
	if !cs.currentBlockParts.HasValidatedBlock() {
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
				cs.currentBlockParts.SetByValidatedBlock(blk)
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
	if err := cs.commitWAL.WriteMessage(msg); err != nil {
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

	cs.currentBlockParts.SetByPartSetID(partSetID)

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
		callback, err := cs.c.BlockManager().WaitForTransaction(cs.lastBlock.ID(), func() {
			cs.mutex.Lock()
			defer cs.mutex.Unlock()

			if cs.hrs != hrs || !cs.started {
				return
			}

			cs.enterPropose()
		})
		cs.log.Must(err)
		if callback {
			cs.notifySyncer()
			return
		}
	}
	cs.notifySyncer()
	cs.enterPropose()
}

func (cs *consensus) enterNewHeight() {
	votes := cs.hvs.votesFor(cs.commitRound, VoteTypePrecommit)
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

func (cs *consensus) sendProposal(blockParts PartSet, polRound int32) {
	err := cs.doSendProposal(blockParts, polRound)
	if err != nil {
		cs.log.Debugf("%+v", err)
	}
}

func (cs *consensus) doSendProposal(blockParts PartSet, polRound int32) error {
	msg := NewProposalMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.BlockPartSetID = blockParts.ID()
	msg.POLRound = polRound
	err := msg.Sign(cs.c.Wallet())
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.WriteMessageBytes(msg.subprotocol(), msgBS); err != nil {
		cs.log.Errorf("fail to write WAL: sendProposal: %+v\n", err)
		return err
	}
	if err := cs.roundWAL.Sync(); err != nil {
		cs.log.Errorf("fail to sync WAL: sendProposal: %+v\n", err)
		return err
	}
	cs.log.Debugf("sendProposal %v\n", msg)
	err = cs.ph.Broadcast(ProtoProposal, msgBS, module.BroadcastAll)
	if err != nil {
		cs.log.Warnf("sendProposal: %+v\n", err)
		return err
	}

	if polRound >= 0 {
		prevotes := cs.hvs.votesFor(polRound, VoteTypePrevote)
		vl := prevotes.voteListForOverTwoThirds()
		vlmsg := newVoteListMessage()
		vlmsg.VoteList = vl
		cs.log.Debugf("sendVoteList %v\n", vlmsg)
		vlmsgBS, err := msgCodec.MarshalToBytes(vlmsg)
		if err != nil {
			return err
		}
		err = cs.ph.Multicast(ProtoVoteList, vlmsgBS, module.RoleValidator)
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
		err = cs.ph.Broadcast(ProtoBlockPart, bpmsgBS, module.BroadcastAll)
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
	} else if cs.currentBlockParts.HasBlockData() {
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

func (cs *consensus) sendVote(vt VoteType, blockParts *blockPartSet) {
	err := cs.doSendVote(vt, blockParts)
	if err != nil {
		cs.log.Debugf("%+v", err)
	}
}

func (cs *consensus) ntsVoteBaseAndDecisionProofParts(
	ntsHashEntries module.NTSHashEntryList,
) ([]ntsVoteBase, [][]byte, error) {
	ntsVoteBases := make([]ntsVoteBase, 0, ntsHashEntries.NTSHashEntryCount())
	ntsdProofParts := make([][]byte, 0, ntsHashEntries.NTSHashEntryCount())
	for i := 0; i < ntsHashEntries.NTSHashEntryCount(); i++ {
		ntsHashEntry := ntsHashEntries.NTSHashEntryAt(i)
		ntid := ntsHashEntry.NetworkTypeID
		pc, err := cs.nextPCM.ProofContextFor(ntid)
		if errors.Is(err, errors.ErrNotFound) {
			// do not vote for first NTS
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		ntsVoteBases = append(ntsVoteBases, ntsVoteBase(ntsHashEntry))
		ntsd := pc.NewDecision(
			cs.srcUID,
			ntsHashEntry.NetworkTypeID,
			cs.height,
			cs.round,
			ntsHashEntry.NetworkTypeSectionHash,
		)
		pp, err := pc.NewProofPart(ntsd.Hash(), cs.c)
		if err != nil {
			return nil, nil, err
		}
		ntsdProofParts = append(ntsdProofParts, pp.Bytes())
	}
	return ntsVoteBases, ntsdProofParts, nil
}

func (cs *consensus) doSendVote(vt VoteType, blockParts *blockPartSet) error {
	if cs.validators.IndexOf(cs.c.Wallet().Address()) < 0 {
		return nil
	}

	msg := newVoteMessage()
	msg.Height = cs.height
	msg.Round = cs.round
	msg.Type = vt

	if blockParts != nil {
		var ntsVoteBases []ntsVoteBase
		var ntsdProofParts [][]byte
		if vt == VoteTypePrecommit && blockParts != nil {
			ntsHashEntries, err := blockParts.block.NTSHashEntryList()
			if err != nil {
				return err
			}
			ntsVoteBases, ntsdProofParts, err = cs.ntsVoteBaseAndDecisionProofParts(ntsHashEntries)
			if err != nil {
				return err
			}
		}
		msg.SetRoundDecision(blockParts.block.ID(), blockParts.ID().WithAppData(uint16(len(ntsVoteBases))), ntsVoteBases)
		msg.NTSDProofParts = ntsdProofParts
	} else {
		msg.SetRoundDecision(cs.nid, nil, nil)
	}
	msg.Timestamp = cs.voteTimestamp()

	err := msg.Sign(cs.c.Wallet())
	if err != nil {
		return err
	}
	msgBS, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		return err
	}
	if err := cs.roundWAL.WriteMessageBytes(msg.subprotocol(), msgBS); err != nil {
		cs.log.Errorf("fail to write WAL: sendVote: %+v\n", err)
	}
	if err := cs.roundWAL.Sync(); err != nil {
		cs.log.Errorf("fail to sync WAL: sendVote: %+v\n", err)
	}
	cs.log.Debugf("sendVote %v\n", msg)
	if vt == VoteTypePrevote {
		err = cs.ph.Multicast(ProtoVote, msgBS, module.RoleValidator)
	} else {
		err = cs.ph.Broadcast(ProtoVote, msgBS, module.BroadcastAll)
	}
	if err != nil {
		cs.log.Warnf("sendVote: %+v\n", err)
	}
	_, err = cs.ReceiveVoteMessage(msg, true)
	return err
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
	if cs.validators == nil || cs.validators.Len() == 0 {
		return false
	}
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
		prevotes := cs.hvs.votesFor(cs.proposalPOLRound, VoteTypePrevote)
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
		msg, err := UnmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.Verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *ProposalMessage:
			if m.height() != cs.height {
				continue
			}
			if !m.address().Equal(cs.c.Wallet().Address()) {
				continue
			}
			cs.log.Tracef("WAL: my proposal %v\n", m)
			if round < m.round() || (round == m.round() && rstep <= stepPropose) {
				round = m.round()
				rstep = stepPropose
			}
		case *VoteMessage:
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
			if m.Type == VoteTypePrevote {
				mstep = stepPrevote
			} else {
				mstep = stepPrecommit
			}
			if round < m.round() || (round == m.round() && rstep <= mstep) {
				round = m.round()
				rstep = mstep
			}
		case *VoteListMessage:
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
			if vmsg.Type == VoteTypePrevote {
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
		msg, err := UnmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.Verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *VoteListMessage:
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
			prevotes := cs.hvs.votesFor(vmsg.Round, VoteTypePrevote)
			psid, ok := prevotes.getOverTwoThirdsPartSetID()
			if ok && psid != nil {
				cs.log.Tracef("WAL: POL R=%v psid=%v\n", vmsg.Round, psid)
				bpset = NewPartSetFromID(psid)
				bpsetLockRound = vmsg.Round
			}
			// update round/step
			var mstep step
			if vmsg.Type == VoteTypePrevote {
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
		case *BlockPartMessage:
			if m.Height != cs.height {
				continue
			}
			if bpset == nil {
				continue
			}
			bp, err := NewPart(m.BlockPart)
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
		cs.currentBlockParts.SetByPartSetAndBlock(lastBPSet, blk)
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
		msg, err := UnmarshalMessage(sp, bs[2:])
		if err != nil {
			return err
		}
		if err = msg.Verify(); err != nil {
			return err
		}
		switch m := msg.(type) {
		case *VoteListMessage:
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
				if vmsg.Type == VoteTypePrevote {
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

func (cs *consensus) applyLastVote(
	cvs module.CommitVoteSet,
	prevValidators addressIndexer,
) error {
	blk := cs.lastBlock
	cvl, ok := cvs.(*CommitVoteList)
	if !ok {
		if vs, ok := cvs.(VoteSet); ok {
			cs.lastVotes = vs
			return nil
		}
		return errors.ErrInvalidState
	}
	var prevBlk module.Block
	var err error
	var vl *VoteList
	if blk.Height() == 0 {
		vl, err = cvl.toVoteListWithBlock(blk, nil, cs.c.Database())
	} else {
		prevBlk, err = cs.c.BlockManager().GetBlockByHeight(blk.Height() - 1)
		if err != nil {
			return err
		}
		vl, err = cvl.toVoteListWithBlock(blk, prevBlk, cs.c.Database())
	}
	if err != nil {
		return err
	}
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

func (cs *consensus) applyLastVoteData(prevValidators addressIndexer) error {
	if cs.lastVoteData == nil {
		return nil
	}
	lastVoteData := cs.lastVoteData
	cs.lastVoteData = nil
	if cs.lastBlock.Height() == lastVoteData.Height {
		cvs := cs.c.CommitVoteSetDecoder()(lastVoteData.VotesBytes)
		return cs.applyLastVote(cvs, prevValidators)
	}
	return nil
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
	return cs.applyLastVote(cvs, prevValidators)
}

func StartConsensusWithLastVotes(cs module.Consensus, lastVoteData *LastVoteData) error {
	return cs.(*consensus).StartWithLastVote(lastVoteData)
}

func (cs *consensus) StartWithLastVote(lastVoteData *LastVoteData) error {
	cs.lastVoteData = lastVoteData
	return cs.Start()
}

func (cs *consensus) Start() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	lastBlock, err := cs.c.BlockManager().GetLastBlock()
	if err != nil {
		return err
	}
	var validators addressIndexer
	var pcMap module.BTPProofContextMap
	if lastBlock.Height() > 0 {
		prevBlock, err := cs.c.BlockManager().GetBlockByHeight(lastBlock.Height() - 1)
		if err != nil {
			return err
		}
		validators = prevBlock.NextValidators()
		pcMap, err = prevBlock.NextProofContextMap()
		if err != nil {
			return err
		}
	} else {
		validators = &emptyAddressIndexer{}
		pcMap = btp.ZeroProofContextMap
	}

	cs.ph, err = cs.c.NetworkManager().RegisterReactor("consensus", module.ProtoConsensus, cs, CsProtocols, ConfigEnginePriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		return err
	}

	cs.nextPCM = pcMap
	cs.resetForNewHeight(lastBlock, newVoteSet(0))
	cs.prevValidators = validators
	if err := cs.applyWAL(validators); err != nil {
		return err
	}
	if err := cs.applyGenesis(validators); err != nil {
		return err
	}
	if err := cs.applyLastVoteData(validators); err != nil {
		return err
	}

	ww, err := cs.wm.OpenForWrite(path.Join(cs.walDir, configRoundWALID), &WALConfig{
		FileLimit:  configRoundWALDataSize,
		TotalLimit: configRoundWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.roundWAL = &WalMessageWriter{ww}

	ww, err = cs.wm.OpenForWrite(path.Join(cs.walDir, configLockWALID), &WALConfig{
		FileLimit:  configLockWALDataSize,
		TotalLimit: configLockWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.lockWAL = &WalMessageWriter{ww}

	ww, err = cs.wm.OpenForWrite(path.Join(cs.walDir, configCommitWALID), &WALConfig{
		FileLimit:  configCommitWALDataSize,
		TotalLimit: configCommitWALDataSize * 3,
	})
	if err != nil {
		return err
	}
	cs.commitWAL = &WalMessageWriter{ww}

	cs.started = true
	cs.log.Infof("Start consensus wallet:%v", common.HexPre(cs.c.Wallet().Address().ID()))
	cs.syncer, err = newSyncer(cs, cs.log, cs.c.NetworkManager(), cs.c.BlockManager(), &cs.mutex, cs.c.Wallet().Address())
	if err != nil {
		return err
	}
	err = cs.syncer.Start()
	if err != nil {
		return err
	}
	if cs.step == stepNewHeight && cs.round == 0 {
		cs.enterTransactionWait()
	} else if cs.step == stepNewHeight && cs.round > 0 {
		cs.enterPropose()
	} else if cs.step == stepPropose {
		cs.enterPrevote()
	} else if cs.step == stepPrevote {
		prevotes := cs.hvs.votesFor(cs.round, VoteTypePrevote)
		if prevotes.hasOverTwoThirds() {
			cs.enterPrevoteWait()
		}
	} else if cs.step == stepPrecommit {
		precommits := cs.hvs.votesFor(cs.round, VoteTypePrecommit)
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

	err := cs.c.NetworkManager().UnregisterReactor(cs)
	if err != nil {
		cs.log.Warnf("%+v", err)
	}
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

func (cs *consensus) getVotesByHeight(height int64) (module.CommitVoteSet, error) {
	c, err := cs.getCommit(height)
	if err != nil {
		return nil, err
	}
	if c.commitVotes == nil {
		return nil, errors.NotFoundError.Errorf("not found vote height=%d", height)
	}
	return c.commitVotes, nil
}

func (cs *consensus) GetVotesByHeight(height int64) (module.CommitVoteSet, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	return cs.getVotesByHeight(height)
}

func (cs *consensus) getCommit(h int64) (*commit, error) {
	if h > cs.height || (h == cs.height && cs.step < stepCommit) {
		return nil, errors.NotFoundError.Errorf("not found commit height=%d", h)
	}

	c := cs.commitCache.GetByHeight(h)
	if c != nil {
		return c, nil
	}

	if h == cs.height && !cs.currentBlockParts.IsComplete() {
		pcs := cs.hvs.votesFor(cs.commitRound, VoteTypePrecommit)
		cvl, err := pcs.commitVoteListForOverTwoThirds(cs.nextPCM)
		if err != nil {
			return nil, err
		}
		return &commit{
			height:       h,
			commitVotes:  cvl,
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts.PartSet,
		}, nil
	}

	if h == cs.height {
		pcs := cs.hvs.votesFor(cs.commitRound, VoteTypePrecommit)
		cvl, err := pcs.commitVoteListForOverTwoThirds(cs.nextPCM)
		if err != nil {
			return nil, err
		}
		c = &commit{
			height:       h,
			commitVotes:  cvl,
			votes:        pcs.voteListForOverTwoThirds(),
			blockPartSet: cs.currentBlockParts.PartSet,
		}
	} else {
		b, err := cs.c.BlockManager().GetBlockByHeight(h)
		if err != nil {
			return nil, err
		}
		var cvs module.CommitVoteSet
		if h == cs.height-1 {
			cvs, err = cs.lastVotes.CommitVoteSet(cs.pcmForLastBlock)
			if err != nil {
				return nil, err
			}
		} else {
			nb, err := cs.c.BlockManager().GetBlockByHeight(h + 1)
			if err != nil {
				return nil, err
			}
			cvs = nb.Votes()
		}
		var bps PartSet
		var vl *VoteList
		if cvl, ok := cvs.(*CommitVoteList); ok {
			if h == 0 {
				vl, err = cvl.toVoteListWithBlock(b, nil, cs.c.Database())
				if err != nil {
					return nil, err
				}
			} else {
				prev, err := cs.c.BlockManager().GetBlockByHeight(h - 1)
				if err != nil {
					return nil, err
				}
				vl, err = cvl.toVoteListWithBlock(b, prev, cs.c.Database())
				if err != nil {
					return nil, err
				}
			}
			psb := NewPartSetBuffer(ConfigBlockPartSize)
			cs.log.Must(b.MarshalHeader(psb))
			cs.log.Must(b.MarshalBody(psb))
			bps = psb.PartSet()
		}
		c = &commit{
			height:       h,
			commitVotes:  cvs,
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
		return nil
	}
	return c.blockPartSet
}

func (cs *consensus) GetCommitPrecommits(h int64) *VoteList {
	c, err := cs.getCommit(h)
	if err != nil {
		return nil
	}
	return c.votes
}

func (cs *consensus) GetPrecommits(r int32) *VoteList {
	return cs.hvs.votesFor(r, VoteTypePrecommit).voteList()
}

func (cs *consensus) GetVotes(r int32, prevotesMask *BitArray, precommitsMask *BitArray) *VoteList {
	return cs.hvs.getVoteListForMask(r, prevotesMask, precommitsMask)
}

func (cs *consensus) GetRoundState() *peerRoundState {
	prs := &peerRoundState{}
	prs.Height = cs.height
	prs.Round = cs.round
	prs.PrevotesMask = cs.hvs.votesFor(cs.round, VoteTypePrevote).getMask()
	prs.PrecommitsMask = cs.hvs.votesFor(cs.round, VoteTypePrecommit).getMask()
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

func (cs *consensus) ReceiveBlockResult(br fastsync.BlockResult) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.ReceiveBlock(br)
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

	votes := cvl.(*CommitVoteList)
	vl, err := votes.toVoteListWithBlock(
		blk, cs.lastBlock, cs.c.Database(),
	)
	if err != nil {
		cs.log.Warnf("fail to convert to VoteList: %+v", err)
		br.Reject()
		return
	}
	for i := 0; i < vl.Len(); i++ {
		m := vl.Get(i)
		index := cs.validators.IndexOf(m.address())
		if index < 0 {
			cs.log.Warnf("processBlock: invalid signer in commit vote list signer=%x indexInVoteList=%d", m.address(), i)
			br.Reject()
			return
		}
		cs.hvs.add(index, m)
	}

	precommits := cs.hvs.votesFor(votes.Round, VoteTypePrecommit)
	id, ok := precommits.getOverTwoThirdsPartSetID()
	if !ok {
		cs.log.Warnf("processBlock: no +2/3 precommits made for block id=%x", blk.ID())
		br.Reject()
		return
	}
	psb := NewPartSetBuffer(ConfigBlockPartSize)
	log.Must(blk.MarshalHeader(psb))
	log.Must(blk.MarshalBody(psb))
	ps := psb.PartSet()
	if !ps.ID().Equal(id) {
		cs.log.Warnf("processBlock: invalid blockBPSID blockBPSID=%s commitBPSID=%s blockID=%x", ps.ID(), id, blk.ID())
		br.Reject()
		return
	}
	cs.currentBlockParts.SetByPartSetAndBlock(ps, blk)
	cs.syncing = false
	br.Consume()
	if cs.step < stepCommit {
		cs.enterCommit(precommits, id, votes.Round)
	} else {
		cs.commitAndEnterNewHeight()
	}
}

func (cs *consensus) GetBlockProof(height int64, opt int32) ([]byte, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.bpp != nil {
		proof, err := cs.bpp.GetBlockProof(height, opt)
		if err != nil {
			return nil, err
		}
		if proof != nil {
			return proof, nil
		}
	}

	cvs, err := cs.getVotesByHeight(height)
	if err != nil {
		return nil, err
	}
	return cvs.Bytes(), nil
}

type WalMessageWriter struct {
	WALWriter
}

func (w *WalMessageWriter) WriteMessage(msg Message) error {
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

func (w *WalMessageWriter) WriteMessageBytes(sp uint16, msg []byte) error {
	bs := make([]byte, 2+len(msg))
	binary.BigEndian.PutUint16(bs, sp)
	copy(bs[2:], msg)
	//cs.log.Tracef("write WAL: %x\n", bs)
	_, err := w.WriteBytes(bs)
	return err
}
