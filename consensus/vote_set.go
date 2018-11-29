package consensus

type voteSet struct {
	msgs     []*voteMessage
	bidCount map[string]int
}

// return true if added
func (vs *voteSet) add(index int, v *voteMessage) bool {
	if vs.msgs[index] != nil {
		return false
	}
	vs.msgs[index] = v
	vs.bidCount[string(v.BlockID)] = vs.bidCount[string(v.BlockID)] + 1
	return true
}

// returns true if has +2/3 votes
func (vs *voteSet) hasOverTwoThirds() bool {
	// TODO
	return true
}

// returns true if has +2/3 for nil or a block
func (vs *voteSet) getOverTwoThirdsPartSetID() (*PartSetID, bool) {
	// TODO
	return nil, true
}

func (vs *voteSet) voteList() *voteList {
	return nil
}

type roundVoteSet = [numberOfVoteTypes]*voteSet

type heightVoteSet struct {
	_nValidators int
	_votes       map[int32][numberOfVoteTypes]*voteSet
}

func (hvs *heightVoteSet) add(index int, v *voteMessage) (bool, *voteSet) {
	rvs := hvs._votes[v.Round]
	if rvs[v.Type] == nil {
		rvs[v.Type] = &voteSet{
			msgs:     make([]*voteMessage, hvs._nValidators),
			bidCount: make(map[string]int),
		}
		hvs._votes[v.Round] = rvs
	}
	vs := rvs[v.Type]
	return vs.add(index, v), vs
}

func (hvs *heightVoteSet) votesFor(round int32, voteType voteType) *voteSet {
	return nil
}

func (hvs *heightVoteSet) reset(_nValidators int) {
}
