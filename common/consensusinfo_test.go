/*
 * Copyright 2022 ICON Foundation
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type validatorList []module.Address

func (v validatorList) Hash() []byte {
	if len(v) == 0 {
		return nil
	}
	return crypto.SHA3Sum256(v.Bytes())
}

func (v validatorList) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(v)
}

func (v validatorList) Flush() error {
	panic("implement me")
}

func (v validatorList) IndexOf(address module.Address) int {
	panic("implement me")
}

func (v validatorList) Len() int {
	panic("implement me")
}

func (v validatorList) Get(i int) (module.Validator, bool) {
	panic("implement me")
}

type dummyConsensusInfo int

func (d dummyConsensusInfo) Proposer() module.Address {
	return nil
}

func (d dummyConsensusInfo) Voters() module.ValidatorList {
	return nil
}

func (d dummyConsensusInfo) Voted() []bool {
	return nil
}

func TestConsensusInfo_NewConsensusInfo(t *testing.T) {
	voters := validatorList([]module.Address{
		MustNewAddressFromString("hx5"),
		MustNewAddressFromString("hx6"),
	})
	proposer := MustNewAddressFromString("hx3")
	voted := []bool{false, true}

	csi := NewConsensusInfo(proposer, voters, voted)
	assert.Equal(t, proposer, csi.Proposer())
	assert.Equal(t, voted, csi.Voted())
	assert.Equal(t, voters, csi.Voters())
}

func TestConsensusInfo_ConsensusInfoEqual(t *testing.T) {
	voters := validatorList([]module.Address{
		MustNewAddressFromString("hx5"),
		MustNewAddressFromString("hx6"),
	})
	proposer := MustNewAddressFromString("hx3")
	voted := []bool{false, true}

	csi := NewConsensusInfo(proposer, voters, voted)

	t.Run("Nil", func(t *testing.T) {
		assert.False(t, ConsensusInfoEqual(csi, nil))
		assert.False(t, ConsensusInfoEqual(nil, csi))
		assert.True(t, ConsensusInfoEqual(nil, nil))
	})

	t.Run("Comparable", func(t *testing.T) {
		csi2 := dummyConsensusInfo(0)
		csi3 := dummyConsensusInfo(1)
		assert.False(t, ConsensusInfoEqual(csi, csi2))
		assert.False(t, ConsensusInfoEqual(csi2, csi))
		assert.True(t, ConsensusInfoEqual(csi2, csi2))
		assert.True(t, ConsensusInfoEqual(csi2, csi3))
	})

	t.Run("SameValue", func(t *testing.T) {
		voters0 := validatorList([]module.Address{
			MustNewAddressFromString("hx5"),
			MustNewAddressFromString("hx6"),
		})
		csi0 := NewConsensusInfo(proposer, voters0, voted)
		assert.True(t, ConsensusInfoEqual(csi, csi0))
	})

	t.Run("SameNil", func(t *testing.T) {
		csi0 := NewConsensusInfo(nil, voters, voted)
		csi1 := NewConsensusInfo(nil, voters, voted)
		assert.True(t, ConsensusInfoEqual(csi0, csi1))
		csi0 = NewConsensusInfo(proposer, nil, nil)
		csi1 = NewConsensusInfo(proposer, nil, nil)
		assert.True(t, ConsensusInfoEqual(csi0, csi1))
		csi0 = NewConsensusInfo(nil, nil, nil)
		csi1 = NewConsensusInfo(nil, nil, nil)
		assert.True(t, ConsensusInfoEqual(csi0, csi1))
	})

	t.Run("ProposerDiff", func(t *testing.T) {
		proposer2 := MustNewAddressFromString("hx8")
		csi1 := NewConsensusInfo(proposer2, voters, voted)
		assert.False(t, ConsensusInfoEqual(csi, csi1))
	})
	t.Run("VotersDiffer", func(t *testing.T) {
		voters2 := validatorList([]module.Address{
			MustNewAddressFromString("hx5"),
			MustNewAddressFromString("hx7"),
		})
		csi1 := NewConsensusInfo(proposer, voters2, voted)
		assert.False(t, ConsensusInfoEqual(csi, csi1))
	})
	t.Run("VotersNil", func(t *testing.T) {
		csi1 := NewConsensusInfo(proposer, nil, voted)
		assert.False(t, ConsensusInfoEqual(csi, csi1))
	})
	t.Run("VotedDiff", func(t *testing.T) {
		voted2 := []bool{true, true}
		csi1 := NewConsensusInfo(proposer, voters, voted2)
		assert.False(t, ConsensusInfoEqual(csi, csi1))
		voted3 := []bool{true, true, false}
		csi2 := NewConsensusInfo(proposer, voters, voted3)
		assert.False(t, ConsensusInfoEqual(csi, csi2))
	})
}

func TestConsensusInfo_String(t *testing.T) {
	csi1 := NewConsensusInfo(nil, nil, nil)
	fmt.Println(csi1)
	vl := validatorList([]module.Address{
		MustNewAddressFromString("hx5"),
		MustNewAddressFromString("hx6"),
	})
	csi2 := NewConsensusInfo(MustNewAddressFromString("hx3"), vl, []bool{false, true})
	fmt.Println(csi2)

	assert.False(t, ConsensusInfoEqual(csi1, csi2))
}
