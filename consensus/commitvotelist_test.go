package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitVoteList_Timestamp(t *testing.T) {
	var cases = []struct {
		in  []int64
		out int64
	}{
		{[]int64{}, 0},
		{[]int64{1}, 1},
		{[]int64{0, 3}, 1},
		{[]int64{0, 1, 10}, 1},
		{[]int64{2, 3, 6, 100}, 4},
	}
	for _, c := range cases {
		msgs := make([]*VoteMessage, len(c.in))
		for i, t := range c.in {
			v := newVoteMessage()
			v.Timestamp = t
			msgs[i] = v
		}
		cvl, err := newCommitVoteList(nil, msgs)
		assert.NoError(t, err)
		assert.Equal(t, cvl.Timestamp(), c.out)
	}
}

func TestCommitVoteList_enoughVote(t *testing.T) {
	assert.True(t, enoughVote(0, 0))
	assert.False(t, enoughVote(0, 1))
	assert.True(t, enoughVote(1, 1))
	assert.False(t, enoughVote(1, 2))
	assert.True(t, enoughVote(2, 2))
	assert.False(t, enoughVote(2, 3))
	assert.True(t, enoughVote(3, 3))
	assert.False(t, enoughVote(2, 4))
	assert.True(t, enoughVote(3, 4))
	assert.False(t, enoughVote(3, 5))
	assert.True(t, enoughVote(4, 5))
	assert.False(t, enoughVote(4, 6))
	assert.True(t, enoughVote(5, 6))
	assert.False(t, enoughVote(4, 7))
	assert.True(t, enoughVote(5, 7))
}
