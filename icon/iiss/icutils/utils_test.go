package icutils

import (
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPow10(t *testing.T) {
	expected := int64(1)
	for i := 0; i < 19; i++ {
		assert.Zero(t, big.NewInt(expected).Cmp(Pow10(i)))
		expected *= 10
	}

	assert.Nil(t, Pow10(-1))
}

func TestToDecimal(t *testing.T) {
	nums := []int{1, -2, 0}

	for _, x := range nums {
		expected := int64(x)
		for i := 0; i < 10; i++ {
			d := ToDecimal(x, i)
			assert.Equal(t, expected, d.Int64())
			expected *= 10
		}
	}

	assert.Nil(t, ToDecimal(1, -10))
}

func TestToLoop(t *testing.T) {
	var expected int64
	nums := []int{0, 1, -1}

	for _, x := range nums {
		expected = int64(x) * 1_000_000_000_000_000_000
		assert.Zero(t, big.NewInt(expected).Cmp(ToDecimal(x, 18)))
	}
}

func TestValidateRange(t *testing.T) {

	type args struct {
		old    *big.Int
		new    *big.Int
		minPct int
		maxPct int
	}

	tests := []struct {
		name string
		in   args
		err  bool
	}{
		{
			"OK",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(110),
				20,
				20,
			},
			false,
		},
		{
			"Same value",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(100),
				20,
				20,
			},
			false,
		},
		{
			"Old is Zero",
			args{
				new(big.Int).SetInt64(0),
				new(big.Int).SetInt64(10),
				20,
				20,
			},
			true,
		},
		{
			"New is Zero",
			args{
				new(big.Int).SetInt64(10),
				new(big.Int).SetInt64(0),
				20,
				20,
			},
			true,
		},
		{
			"too small",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(79),
				20,
				20,
			},
			true,
		},
		{
			"too big",
			args{
				new(big.Int).SetInt64(100),
				new(big.Int).SetInt64(121),
				20,
				20,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			err := ValidateRange(in.old, in.new, in.minPct, in.maxPct)
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
