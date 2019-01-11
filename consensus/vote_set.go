package consensus

type counter struct {
	partsID *PartSetID
	count   int
}

type voteSet struct {
	msgs     []*voteMessage
	maxIndex int
	mask     *bitArray

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
	return true
}

// returns true if has +2/3 votes
func (vs *voteSet) hasOverTwoThirds() bool {
	return vs.count > len(vs.msgs)*2/3
}

// returns true if has +2/3 for nil or a block
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

func (vs *voteSet) voteListForOverTwoThirds() *voteList {
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
	return newVoteList(msgs)
}

func (vs *voteSet) voteList() *roundVoteList {
	rvl := newRoundVoteList()
	for _, msg := range vs.msgs {
		if msg != nil {
			rvl.AddVote(msg)
		}
	}
	return rvl
}

// shall not modify returned array. invalidated if a vote is added.
func (vs *voteSet) getMask() *bitArray {
	return vs.mask
}

func newVoteSet(nValidators int) *voteSet {
	return &voteSet{
		msgs:     make([]*voteMessage, nValidators),
		maxIndex: -1,
		mask:     newBitArray(nValidators),
	}
}

type roundVoteSet = [numberOfVoteTypes]*voteSet

type heightVoteSet struct {
	_nValidators int
	_votes       map[int32][numberOfVoteTypes]*voteSet
}

func (hvs *heightVoteSet) add(index int, v *voteMessage) (bool, *voteSet) {
	vs := hvs.votesFor(v.Round, v.Type)
	return vs.add(index, v), vs
}

func (hvs *heightVoteSet) votesFor(round int32, voteType voteType) *voteSet {
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

func (hvs *heightVoteSet) getVoteListForMask(round int32, prevotesMask *bitArray, precommitsMask *bitArray) *roundVoteList {
	rvl := newRoundVoteList()
	prevotes := hvs.votesFor(round, voteTypePrevote)
	for i, msg := range prevotes.msgs {
		if prevotesMask.Get(i) && msg != nil {
			rvl.AddVote(msg)
		}
	}
	precommits := hvs.votesFor(round, voteTypePrecommit)
	for i, msg := range precommits.msgs {
		if precommitsMask.Get(i) && msg != nil {
			rvl.AddVote(msg)
		}
	}
	return rvl
}
