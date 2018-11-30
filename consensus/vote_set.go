package consensus

type counter struct {
	partsID *PartSetID
	count   int
}

type voteSet struct {
	msgs     []*voteMessage
	counters []counter
	count    int
	maxIndex int
}

// return true if added
func (vs *voteSet) add(index int, v *voteMessage) bool {
	if vs.msgs[index] != nil {
		return false
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
		rvs[voteType] = &voteSet{
			msgs:     make([]*voteMessage, hvs._nValidators),
			maxIndex: -1,
		}
		hvs._votes[round] = rvs
	}
	vs := rvs[voteType]
	return vs
}

func (hvs *heightVoteSet) reset(nValidators int) {
	hvs._nValidators = nValidators
	hvs._votes = make(map[int32][numberOfVoteTypes]*voteSet)
}
