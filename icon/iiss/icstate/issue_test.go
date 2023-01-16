package icstate

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewIssue(t *testing.T) {
	issue := NewIssue()
	assert.Zero(t, issue.TotalReward().Sign())
	assert.Zero(t, issue.PrevTotalReward().Sign())
	assert.Zero(t, issue.OverIssuedIScore().Sign())
	assert.Zero(t, issue.PrevBlockFee().Sign())
}

func TestIssue_Equal(t *testing.T) {
	i := NewIssue()
	i2 := i.Clone()

	assert.False(t, i.Equal(nil))
	assert.True(t, i.Equal(i))
	assert.True(t, i.Equal(i2))
	assert.True(t, i2.Equal(i))

	i.SetTotalReward(big.NewInt(100))
	assert.False(t, i.Equal(i2))
	assert.False(t, i2.Equal(i))
	i2.SetTotalReward(big.NewInt(100))
	assert.True(t, i.Equal(i2))
	assert.True(t, i2.Equal(i))

	i.SetPrevBlockFee(big.NewInt(100))
	assert.False(t, i.Equal(i2))
	assert.False(t, i2.Equal(i))
	i2.SetPrevBlockFee(big.NewInt(100))
	assert.True(t, i.Equal(i2))
	assert.True(t, i2.Equal(i))

	i.SetOverIssuedIScore(big.NewInt(100))
	assert.False(t, i.Equal(i2))
	assert.False(t, i2.Equal(i))
	i2.SetOverIssuedIScore(big.NewInt(100))
	assert.True(t, i.Equal(i2))
	assert.True(t, i2.Equal(i))

	i.SetPrevTotalReward(big.NewInt(100))
	assert.False(t, i.Equal(i2))
	assert.False(t, i2.Equal(i))
	i2.SetPrevTotalReward(big.NewInt(100))
	assert.True(t, i.Equal(i2))
	assert.True(t, i2.Equal(i))
}

func TestIssue_Update(t *testing.T) {
	issue := NewIssue()
	orgIssue := issue.Clone()

	totalReward := big.NewInt(1_000_000_000_000_000_000)
	byFee := big.NewInt(rand.Int63())
	byOverIssued := big.NewInt(rand.Int63())

	ni := issue.Update(totalReward, byFee, byOverIssued)
	assert.True(t, orgIssue.Equal(issue))
	assert.False(t, orgIssue.Equal(ni))
}

func TestIssue_ResetTotalReward(t *testing.T) {
	totalReward := big.NewInt(1_000_000_000_000_000_000)
	issue := NewIssue()
	issue.SetTotalReward(totalReward)

	issue.ResetTotalReward()
	assert.Zero(t, issue.TotalReward().Sign())
	assert.Zero(t, issue.PrevTotalReward().Cmp(totalReward))
}

func BenchmarkIssue_Update(b *testing.B) {
	issue := NewIssue()
	totalReward := big.NewInt(1_000_000_000_000_000_000)
	byFee := big.NewInt(1_000_000_000_000_000_000)
	byOverIssued := big.NewInt(1_000_000_000_000_000_000)

	for i := 0; i < b.N; i++ {
		issue.Update(totalReward, byFee, byOverIssued)
	}
}

func BenchmarkClone(b *testing.B) {
	issue := NewIssue()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issue.Clone()
	}
}
