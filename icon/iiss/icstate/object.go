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
	TypePRep
	TypePRepStatus
	TypeTimer
)

func newObjectImpl(tag icobject.Tag) (icobject.Impl, error) {
	switch tag.Type() {
	case TypeAccount:
		return newAccountSnapshot(tag), nil
	case TypePRep:
		return newPRepSnapshot(tag), nil
	case TypePRepStatus:
		return newPRepStatusSnapshot(tag), nil
	case TypeTimer:
		return newTimerSnapshot(tag), nil
	default:
		return nil, errors.IllegalArgumentError.Errorf(
			"UnknownTypeTag(tag=%#x)", tag)
	}
}

func ToAccountSnapshot(object trie.Object) *AccountSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*AccountSnapshot)
}

func ToPRepStatusSnapshot(object trie.Object) *PRepStatusSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*PRepStatusSnapshot)
}

func ToPRepSnapshot(object trie.Object) *PRepSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*PRepSnapshot)
}

func ToTimerSnapshot(object trie.Object) *TimerSnapshot {
	if object == nil {
		return nil
	}
	return object.(*icobject.Object).Real().(*TimerSnapshot)
}
