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

package icstage

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	TypeIScoreClaim int = iota
	TypeEventDelegation
	TypeEventBond
	TypeEventEnable
	TypeBlockProduce
	TypeGlobal
	TypeEventVotedReward
	TypeEventDelegationV2
	TypeEventDelegated
	TypeBTPDSA
	TypeBTPPublicKey
	TypeCommissionRate
)

func NewObjectImpl(tag icobject.Tag) (icobject.Impl, error) {
	switch tag.Type() {
	case TypeIScoreClaim:
		return newIScoreClaim(tag), nil
	case TypeEventDelegation:
		return newEventVote(tag), nil
	case TypeEventDelegationV2:
		return newEventDelegationV2(tag), nil
	case TypeEventDelegated:
		return newEventVote(tag), nil
	case TypeEventBond:
		return newEventVote(tag), nil
	case TypeEventEnable:
		return newEventEnable(tag), nil
	case TypeBlockProduce:
		return newBlockProduce(tag), nil
	case TypeGlobal:
		return newGlobal(tag)
	case TypeEventVotedReward:
		return newEventVotedReward(tag), nil
	case TypeBTPDSA:
		return newBTPDSA(tag), nil
	case TypeBTPPublicKey:
		return newBTPPublicKey(tag), nil
	case TypeCommissionRate:
		return newCommissionRate(tag), nil
	default:
		return nil, errors.IllegalArgumentError.Errorf(
			"UnknownTypeTag(tag=%#x)", tag)
	}
}

func ToIScoreClaim(obj trie.Object) *IScoreClaim {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*IScoreClaim)
}

func ToEventVote(obj trie.Object) *EventVote {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*EventVote)
}

func ToEventEnable(obj trie.Object) *EventEnable {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*EventEnable)
}

func ToEventVotedReward(obj trie.Object) *EventVotedReward {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*EventVotedReward)
}

func ToCommissionRate(obj trie.Object) *CommissionRate {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*CommissionRate)
}

func ToBlockProduce(obj trie.Object) *BlockProduce {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*BlockProduce)
}

func ToGlobal(obj trie.Object) Global {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(Global)
}

func ToEventDelegationV2(obj trie.Object) *EventDelegationV2 {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*EventDelegationV2)
}

func ToBTPDSA(obj trie.Object) *BTPDSA {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*BTPDSA)
}

func ToBTPPublicKey(obj trie.Object) *BTPPublicKey {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*BTPPublicKey)
}
