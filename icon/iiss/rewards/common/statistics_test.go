/*
 * Copyright 2023 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

type sValue struct {
	rType RewardType
	value int64
}

func TestStats_increase(t *testing.T) {
	stats := NewStats()

	tests := []sValue{
		{RTBlockProduce, 1000},
		{RTBlockProduce, 5},
		{RTPRep, 321},
		{RTPRep, 44444},
		{RTVoter, 100000000},
		{RTVoter, 123},
	}

	for _, tt := range tests {
		var old, current *big.Int
		value := big.NewInt(tt.value)
		switch tt.rType {
		case RTBlockProduce:
			old = stats.BlockProduce()
			stats.IncreaseBlockProduce(value)
			current = stats.BlockProduce()
		case RTPRep:
			old = stats.Voted()
			stats.IncreaseVoted(value)
			current = stats.Voted()
		case RTVoter:
			old = stats.Voting()
			stats.IncreaseVoting(value)
			current = stats.Voting()
		}
		assert.Equal(t, value, new(big.Int).Sub(current, old))
	}

	for _, tt := range tests {
		old := stats.GetValue(tt.rType)
		value := big.NewInt(tt.value)
		stats.IncreaseValue(tt.rType, value)
		current := stats.GetValue(tt.rType)
		assert.Equal(t, value, new(big.Int).Sub(current, old))
	}
}

func TestStats_Total(t *testing.T) {
	tests := []struct {
		values []sValue
		want   int64
	}{
		{
			[]sValue{
				sValue{RTBlockProduce, 1000},
			},
			1000,
		},
		{
			[]sValue{
				sValue{RTPRep, 2000},
			},
			2000,
		},
		{
			[]sValue{
				sValue{RTVoter, 4000},
			},
			4000,
		},
		{
			[]sValue{
				sValue{RTBlockProduce, 1000},
				sValue{RTPRep, 2000},
			},
			3000,
		},
		{
			[]sValue{
				sValue{RTBlockProduce, 1000},
				sValue{RTVoter, 4000},
			},
			5000,
		},
		{
			[]sValue{
				sValue{RTPRep, 2000},
				sValue{RTVoter, 4000},
			},
			6000,
		},
		{
			[]sValue{
				sValue{RTBlockProduce, 1000},
				sValue{RTPRep, 2000},
				sValue{RTVoter, 4000},
			},
			7000,
		},
	}

	for i, tt := range tests {
		stats := NewStats()
		for _, v := range tt.values {
			stats.IncreaseValue(v.rType, big.NewInt(v.value))
		}
		assert.Equal(t, tt.want, stats.Total().Int64(), fmt.Sprintf("Index: %d", i))
	}
}
