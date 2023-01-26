package icstate

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type testVoting struct {
	to     module.Address
	amount *big.Int
}

func (v *testVoting) To() module.Address {
	return v.to
}

func (v *testVoting) Amount() *big.Int {
	return v.amount
}

func newTestVotings(size int) []Voting {
	votings := make([]Voting, size)
	for i := 0; i < size; i++ {
		votings[i] = &testVoting{
			to:     common.MustNewAddressFromString(fmt.Sprintf("hx%d", i)),
			amount: big.NewInt(int64(i)),
		}
	}
	return votings
}

func TestNewVotingIterator(t *testing.T) {
	votings := newTestVotings(10)
	assert.Equal(t, 10, len(votings))

	vi := NewVotingIterator(votings)

	i := 0
	for ; vi.Has(); vi.Next() {
		v, err := vi.Get()
		assert.NoError(t, err)

		to := common.MustNewAddressFromString(fmt.Sprintf("hx%d", i))
		assert.True(t, votings[i].To().Equal(v.To()))
		assert.True(t, to.Equal(v.To()))

		amount := big.NewInt(int64(i))
		assert.Equal(t, votings[i].Amount().Int64(), v.Amount().Int64())
		assert.Equal(t, amount.Int64(), v.Amount().Int64())
		i++
	}

	v, err := vi.Get()
	assert.Nil(t, v)
	assert.Error(t, err)
	assert.Error(t, vi.Next())
	assert.Equal(t, i, len(votings))
}
