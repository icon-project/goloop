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

package icreward

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

const (
	TypeVoted int = iota
	TypeDelegating
	TypeBonding
	TypeIScore
	TypeBugDisabledPRep
	TypeDSA
	TypePublicKey
)

func NewObjectImpl(tag icobject.Tag) (icobject.Impl, error) {
	switch tag.Type() {
	case TypeVoted:
		return newVoted(tag), nil
	case TypeDelegating:
		return newDelegating(tag), nil
	case TypeBonding:
		return newBonding(tag), nil
	case TypeIScore:
		return newIScore(tag), nil
	case TypeBugDisabledPRep:
		return newBugDisabledPRep(tag), nil
	case TypeDSA:
		return newDSA(tag), nil
	case TypePublicKey:
		return newPublicKey(tag), nil
	default:
		return nil, errors.IllegalArgumentError.Errorf(
			"UnknownTypeTag(tag=%#x)", tag)
	}
}

func ToIScore(obj trie.Object) *IScore {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*IScore)
}

func ToVoted(obj trie.Object) *Voted {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*Voted)
}

func ToDelegating(obj trie.Object) *Delegating {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*Delegating)
}

func ToBonding(obj trie.Object) *Bonding {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*Bonding)
}

func ToBugDisabledPRep(obj trie.Object) *BugDisabledPRep {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*BugDisabledPRep)
}

func ToDSA(obj trie.Object) *DSA {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*DSA)
}

func ToPublicKey(obj trie.Object) *PublicKey {
	if obj == nil {
		return nil
	}
	return obj.(*icobject.Object).Real().(*PublicKey)
}
