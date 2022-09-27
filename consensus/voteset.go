package consensus

import (
	"bytes"

	"github.com/icon-project/goloop/module"
)

type VoteSet interface {
	CommitVoteSet() module.CommitVoteSet
	Add(idx int, vote interface{}) bool
}

type counter struct {
	partsID *PartSetID
	count   int
}

type voteSet struct {
	msgs     []*voteMessage
	maxIndex int
	mask     *bitArray
	round    int32

	counters []counter
	count    int
}

// return true if added
func (vs *voteSet) add(index int, v *voteMessage) bool {
	omsg := vs.msgs[index]
	if omsg != nil {
		if omsg.vote.Equal(&v.vote) {
			return false
		}
		psid, ok := vs.getOverTwoThirdsPartSetID()
		if ok && psid != nil && psid.Equal(omsg.BlockPartSetID) {
			return false
		}
		for i, c := range vs.counters {
			if c.partsID.Equal(omsg.BlockPartSetID) {
				vs.counters[i].count--
				if vs.counters[i].count == 0 {
					last := len(vs.counters) - 1
					vs.counters[i] = vs.counters[last]
					vs.counters = vs.counters[:last]
				}
				break
			}
		}
		vs.count--
	}

	vs.msgs[index] = v
	found := false
	for i, c := range vs.counters {
		if c.partsID.Equal(v.BlockPartSetID) {
			vs.counters[i].count++
			found = true
			break
		}
	}
	if !found {
		vs.counters = append(vs.counters, counter{v.BlockPartSetID, 1})
	}
	vs.count++
	vs.maxIndex = -1
	vs.mask.Set(index)
	vs.round = v.Round
	return true
}

// returns true if the voteSet has +2/3 votes
func (vs *voteSet) hasOverTwoThirds() bool {
	return vs.count > len(vs.msgs)*2/3
}

func (vs *voteSet) getRound() int32 {
	return vs.round
}

// returns true if the voteSet has +2/3 for nil or a block
func (vs *voteSet) getOverTwoThirdsPartSetID() (*PartSetID, bool) {
	var max int
	if vs.maxIndex < 0 {
		max = 0
		for i, c := range vs.counters {
			if c.count > max {
				vs.maxIndex = i
				max = c.count
			}
		}
	} else {
		max = vs.counters[vs.maxIndex].count
	}
	if max > len(vs.msgs)*2/3 {
		return vs.counters[vs.maxIndex].partsID, true
	} else {
		return nil, false
	}
}

func (vs *voteSet) commitVoteListForOverTwoThirds() *commitVoteList {
	if len(vs.msgs) == 0 {
		return newCommitVoteList(nil)
	}

	partSetID, ok := vs.getOverTwoThirdsPartSetID()
	if !ok {
		return nil
	}
	var msgs []*voteMessage
	for _, msg := range vs.msgs {
		if msg != nil && msg.BlockPartSetID.Equal(partSetID) {
			msgs = append(msgs, msg)
		}
	}
	return newCommitVoteList(msgs)
}

func (vs *voteSet) voteListForOverTwoThirds() *voteList {
	partSetID, ok := vs.getOverTwoThirdsPartSetID()
	if !ok {
		return nil
	}
	rvl := newVoteList()
	for _, msg := range vs.msgs {
		if msg != nil && msg.BlockPartSetID.Equal(partSetID) {
			rvl.AddVote(msg)
		}
	}
	return rvl
}

func (vs *voteSet) voteSetForOverTwoThird() *voteSet {
	partSetID, ok := vs.getOverTwoThirdsPartSetID()
	if !ok {
		return nil
	}
	rvs := newVoteSet(len(vs.msgs))
	for i, msg := range vs.msgs {
		if msg != nil && msg.BlockPartSetID.Equal(partSetID) {
			rvs.add(i, msg)
		}
	}
	return rvs
}

func (vs *voteSet) voteList() *voteList {
	rvl := newVoteList()
	for _, msg := range vs.msgs {
		if msg != nil {
			rvl.AddVote(msg)
		}
	}
	return rvl
}

func (vs *voteSet) getRoundEvidences(minRound int32, nid []byte) *voteList {
	rvl := newVoteList()
	l := len(vs.msgs)
	f := l / 3
	for _, msg := range vs.msgs {
		evidence := msg != nil &&
			msg.Round >= minRound &&
			msg.BlockPartSetID == nil &&
			bytes.Equal(nid, msg.BlockID)
		if evidence {
			rvl.AddVote(msg)
		}
	}
	if rvl.Len() > f {
		return rvl
	}
	return nil
}

// shall not modify returned array. invalidated if a vote is added.
func (vs *voteSet) getMask() *bitArray {
	return vs.mask
}

func (vs *voteSet) CommitVoteSet() module.CommitVoteSet {
	return vs.commitVoteListForOverTwoThirds()
}

func (vs *voteSet) checkAndAdd(idx int, msg *voteMessage) bool {
	psid, ok := vs.getOverTwoThirdsPartSetID()
	if !ok || msg.Round != vs.getRound() || !msg.BlockPartSetID.Equal(psid) {
		return false
	}
	return vs.add(idx, msg)
}

func (vs *voteSet) Add(idx int, msg interface{}) bool {
	return vs.checkAndAdd(idx, msg.(*voteMessage))
}

func newVoteSet(nValidators int) *voteSet {
	return &voteSet{
		msgs:     make([]*voteMessage, nValidators),
		maxIndex: -1,
		mask:     newBitArray(nValidators),
		round:    -1,
	}
}

type heightVoteSet struct {
	_nValidators int
	_votes       map[int32][numberOfVoteTypes]*voteSet
}

func (hvs *heightVoteSet) add(index int, v *voteMessage) (bool, *voteSet) {
	vs := hvs.votesFor(v.Round, v.Type)
	return vs.add(index, v), vs
}

func (hvs *heightVoteSet) votesFor(round int32, voteType VoteType) *voteSet {
	rvs := hvs._votes[round]
	if rvs[voteType] == nil {
		rvs[voteType] = newVoteSet(hvs._nValidators)
		hvs._votes[round] = rvs
	}
	vs := rvs[voteType]
	return vs
}

func (hvs *heightVoteSet) reset(nValidators int) {
	hvs._nValidators = nValidators
	hvs._votes = make(map[int32][numberOfVoteTypes]*voteSet)
}

func (hvs *heightVoteSet) getVoteListForMask(round int32, prevotesMask *bitArray, precommitsMask *bitArray) *voteList {
	rvl := newVoteList()
	prevotes := hvs.votesFor(round, VoteTypePrevote)
	for i, msg := range prevotes.msgs {
		if !prevotesMask.Get(i) && msg != nil {
			rvl.AddVote(msg)
		}
	}
	precommits := hvs.votesFor(round, VoteTypePrecommit)
	for i, msg := range precommits.msgs {
		if !precommitsMask.Get(i) && msg != nil {
			rvl.AddVote(msg)
		}
	}
	return rvl
}

func (hvs *heightVoteSet) getRoundEvidences(minRound int32, nid []byte) *voteList {
	for round := range hvs._votes {
		if round >= minRound {
			evidences := hvs.votesFor(round, VoteTypePrevote).getRoundEvidences(minRound, nid)
			if evidences != nil {
				return evidences
			}
		}
	}
	return nil
}

// remove votes.
func (hvs *heightVoteSet) removeLowerRoundExcept(lower int32, except int32) {
	for round := range hvs._votes {
		if round < lower && round != except {
			// safe to delete map entry in range iteration
			delete(hvs._votes, round)
		}
	}
}
