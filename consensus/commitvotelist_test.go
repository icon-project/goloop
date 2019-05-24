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
		msgs := make([]*voteMessage, len(c.in))
		for i, t := range c.in {
			v := newVoteMessage()
			v.Timestamp = t
			msgs[i] = v
		}
		cvl := newCommitVoteList(msgs)
		assert.Equal(t, cvl.Timestamp(), c.out)
	}
}
