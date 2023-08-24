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

package icstate

import (
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

type stateContext struct {
	blockHeight int64
	revision int
	termRevision int
}

func NewStateContext(blockHeight int64, revision, termRevision int) icmodule.StateContext {
	return &stateContext{
		blockHeight,
		revision,
		termRevision,
	}
}

func (sc *stateContext) BlockHeight() int64 {
	return sc.blockHeight
}

func (sc *stateContext) Revision() int {
	return sc.revision
}

// TermRevision returns revision stored in TermSnapshot
func (sc *stateContext) TermRevision() int {
	return sc.termRevision
}

func (sc *stateContext) IsIISS4Activated() bool {
	return sc.termRevision >= icmodule.RevisionIISS4
}

func (sc *stateContext) AddEventEnable(from module.Address, flag icmodule.EnableStatus) error {
	return nil
}