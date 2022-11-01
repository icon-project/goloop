/*
 * Copyright 2020 ICON Foundation
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
	"bytes"
	"fmt"
	"reflect"

	"github.com/icon-project/goloop/module"
)

type consensusInfo struct {
	proposer module.Address
	voters   module.ValidatorList
	voted    []bool
}

func (c *consensusInfo) Proposer() module.Address {
	return c.proposer
}

func (c *consensusInfo) Voters() module.ValidatorList {
	return c.voters
}

func (c *consensusInfo) Voted() []bool {
	return c.voted
}

func (c *consensusInfo) String() string {
	return fmt.Sprintf("ConsensusInfo(proposer=%v,voters=%v,voted=%v)",
		c.proposer, c.voters, c.voted)
}

func NewConsensusInfo(
	proposer module.Address,
	voters module.ValidatorList,
	voted []bool,
) module.ConsensusInfo {
	return &consensusInfo{proposer, voters, voted}
}

func validatorListEqual(vl1, vl2 module.ValidatorList) bool {
	if vl1 == nil && vl2 == nil {
		return true
	}
	if vl1 == nil || vl2 == nil {
		return false
	}
	if reflect.TypeOf(vl1).Comparable() && vl1 == vl2 {
		return true
	}
	return bytes.Equal(vl1.Hash(), vl2.Hash())
}

func votedEqual(voted1, voted2 []bool) bool {
	if len(voted1) != len(voted2) {
		return false
	}
	for i, v := range voted1 {
		if v != voted2[i] {
			return false
		}
	}
	return true
}

func ConsensusInfoEqual(csi1, csi2 module.ConsensusInfo) bool {
	if csi1 == nil && csi2 == nil {
		return true
	}
	if csi1 == nil || csi2 == nil {
		return false
	}
	if reflect.TypeOf(csi1).Comparable() && csi1 == csi2 {
		return true
	}
	return AddressEqual(csi1.Proposer(), csi2.Proposer()) &&
		validatorListEqual(csi1.Voters(), csi2.Voters()) &&
		votedEqual(csi1.Voted(), csi2.Voted())
}
