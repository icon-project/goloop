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

package icstate

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	TypeAccount int = iota
	TypePRepBase
	TypePRepStatus
	TypeTimer
	TypeIssue
	TypeTerm
	TypeRewardCalcInfo
	TypeValidators
	TypeBlockVoters
	TypeDelegationBug
)

type StateAndSnapshot struct {
	readonly bool
}

func (s *StateAndSnapshot) IsReadonly() bool {
	return s.readonly
}

func (s *StateAndSnapshot) checkWritable() {
	if s.readonly {
		panic(errors.Errorf("Failed to clear readonly PRepBase: %v", s))
	}
}

func (s *StateAndSnapshot) freeze() {
	s.readonly = true
}

func NewObjectImpl(tag icobject.Tag) (icobject.Impl, error) {
	switch tag.Type() {
	case TypeAccount:
		return newAccountWithTag(tag), nil
	case TypePRepBase:
		return newPRepBaseWithTag(tag), nil
	case TypePRepStatus:
		return newPRepStatusWithTag(tag), nil
	case TypeTimer:
		return newTimerWithTag(tag), nil
	case TypeIssue:
		return newIssue(tag), nil
	case TypeTerm:
		return NewTermWithTag(tag), nil
	case TypeRewardCalcInfo:
		return newRewardCalcInfo(tag), nil
	case TypeValidators:
		return newValidatorsWithTag(tag), nil
	case TypeBlockVoters:
		return NewBlockVotersWithTag(tag), nil
	case TypeDelegationBug:
		return NewDelegationBugWithTag(tag), nil
	default:
		return nil, errors.IllegalArgumentError.Errorf(
			"UnknownTypeTag(tag=%#x)", tag)
	}
}

func ToAccount(object trie.Object) *AccountSnapshot {
	if object == nil {
		return nil
	}
	a := object.(*icobject.Object).Real().(*AccountSnapshot)
	return a
}

func ToPRepStatus(object trie.Object) *PRepStatusSnapshot {
	if object == nil {
		return nil
	}
	ps := object.(*icobject.Object).Real().(*PRepStatusSnapshot)
	return ps
}

func ToPRepBase(object trie.Object) *PRepBaseSnapshot {
	if object == nil {
		return nil
	}
	pbs := object.(*icobject.Object).Real().(*PRepBaseSnapshot)
	return pbs
}

func ToTimer(object trie.Object) *TimerSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*TimerSnapshot)
}

func ToIssue(object trie.Object) *Issue {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*Issue)
}

func ToTerm(object trie.Object) *TermSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*TermSnapshot)
}

func ToRewardCalcInfo(object trie.Object) *RewardCalcInfo {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*RewardCalcInfo)
}

func ToValidators(object trie.Object) *ValidatorsSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*ValidatorsSnapshot)
}

func ToBlockVoters(object trie.Object) *BlockVotersSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*BlockVotersSnapshot)
}

func ToDelegationBug(object trie.Object) *DelegationBug {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*DelegationBug)
}