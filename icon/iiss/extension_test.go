package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/iiss/icutils"
)

func TestExtension_calculateRRep(t *testing.T) {
	type test struct {
		name           string
		totalSupply    *big.Int
		totalDelegated *big.Int
		rrep           *big.Int
	}

	tests := [...]test{
		{
			"MainNet-10,362,083-Decentralized",
			new(big.Int).Mul(new(big.Int).SetInt64(800326000), icutils.BigIntICX),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(170075049), icutils.BigIntICX),
				new(big.Int).SetInt64(583626807627704134),
			),
			new(big.Int).SetInt64(0x2ac),
		},
		{
			"MainNet-14,717,202",
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(819800188), icutils.BigIntICX),
				new(big.Int).SetInt64(205880949256032856),
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(203901940), icutils.BigIntICX),
				new(big.Int).SetInt64(576265206775030620),
			),
			new(big.Int).SetInt64(0x267),
		},
		{
			"MainNet-17,304,403",
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(831262951), icutils.BigIntICX),
				new(big.Int).SetInt64(723502790728839479),
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(234347234), icutils.BigIntICX),
				new(big.Int).SetInt64(465991733052158079),
			),
			new(big.Int).SetInt64(0x22c),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rrep := calculateRRep(tt.totalSupply, tt.totalDelegated)
			assert.Equal(t, 0, tt.rrep.Cmp(rrep), "%s\n%s", tt.rrep.String(), rrep.String())
		})
	}

	// from ICON1
	// index 56 : replace 239 to 240
	// 		there is some strange result in python
	//			>>> a: float = 56 / 100 * 10000
	//			>>> a
	//			5600.000000000001
	expectedRrepPerDelegatePercentage := [...]int64{
		1200,
		1171, 1143, 1116, 1088, 1062, 1035, 1010, 984, 959, 934,
		910, 886, 863, 840, 817, 795, 773, 751, 730, 710,
		690, 670, 650, 631, 613, 595, 577, 560, 543, 526,
		510, 494, 479, 464, 450, 435, 422, 408, 396, 383,
		371, 360, 348, 337, 327, 317, 307, 298, 290, 281,
		273, 266, 258, 252, 245, 240, 234, 229, 224, 220,
		216, 213, 210, 207, 205, 203, 201, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
	}

	for i := 0; i < 101; i++ {
		name := fmt.Sprintf("delegated percentage: %d", i)
		t.Run(name, func(t *testing.T) {
			rrep := calculateRRep(new(big.Int).SetInt64(100), new(big.Int).SetInt64(int64(i)))
			assert.Equal(t, expectedRrepPerDelegatePercentage[i], rrep.Int64())
		})
	}
}

func TestExtension_validateRewardFund(t *testing.T) {
	type test struct {
		name           string
		iglobal        *big.Int
		totalSupply    *big.Int
		currentIglobal *big.Int
		err            bool
	}

	tests := [...]test{
		{
			"Inflation rate exceed 15%",
			new(big.Int).SetInt64(125),
			new(big.Int).SetInt64(1000),
			new(big.Int).SetInt64(101),
			true,
		},
		{
			"Inflation rate 18%",
			new(big.Int).SetInt64(150),
			new(big.Int).SetInt64(1000),
			new(big.Int).SetInt64(124),
			true,
		},
		{
			"Increase 10%",
			new(big.Int).SetInt64(110),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			false,
		},
		{
			"Decrease 10%",
			new(big.Int).SetInt64(90),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			false,
		},
		{
			"Increase 25%",
			new(big.Int).SetInt64(125),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			true,
		},
		{
			"Decrease 25%",
			new(big.Int).SetInt64(75),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRewardFund(tt.iglobal, tt.currentIglobal, tt.totalSupply)
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
