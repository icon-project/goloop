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
	TypeEventSize
	TypeBlockProduce
	TypeValidator
	TypeGlobal
)

func NewObjectImpl(tag icobject.Tag) (icobject.Impl, error) {
	switch tag.Type() {
	case TypeIScoreClaim:
		return newIScoreClaim(tag), nil
	case TypeEventDelegation:
		return newEventVote(tag), nil
	case TypeEventBond:
		return newEventVote(tag), nil
	case TypeEventEnable:
		return newEventEnable(tag), nil
	case TypeEventSize:
		return newEventSize(tag), nil
	case TypeBlockProduce:
		return newBlockProduce(tag), nil
	case TypeValidator:
		return newValidator(tag), nil
	case TypeGlobal:
		return NewGlobal(tag.Version())
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

func ToEventSize(obj trie.Object) *EventSize {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*EventSize)
}

func ToBlockProduce(obj trie.Object) *BlockProduce {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*BlockProduce)
}

func ToValidator(obj trie.Object) *Validator {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*Validator)
}

func ToGlobal(obj trie.Object) Global {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(Global)
}
