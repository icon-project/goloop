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

package iiss

import (
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

type stateContext struct {
	icmodule.WorldContext
	termRevision int
	eventLogger icmodule.EnableEventLogger
}

func NewStateContext(
	wc icmodule.WorldContext, termRevision int, eventLogger icmodule.EnableEventLogger) icmodule.StateContext {
	return &stateContext{wc, termRevision, eventLogger}
}

func (sc *stateContext) Revision() int {
	return sc.WorldContext.Revision().Value()
}

// TermRevision returns revision stored in TermSnapshot
func (sc *stateContext) TermRevision() int {
	return sc.termRevision
}

func (sc *stateContext) IsIISS4Activated() bool {
	return sc.termRevision >= icmodule.RevisionIISS4
}

func (sc *stateContext) GetActiveDSAMask() int64 {
	return GetActiveDSAMask(sc.WorldContext)
}

func (sc *stateContext) AddEventEnable(owner module.Address, status icmodule.EnableStatus) error {
	if sc.eventLogger != nil {
		return sc.eventLogger.AddEventEnable(sc.BlockHeight(), owner, status)
	}
	return nil
}

func GetActiveDSAMask(cc icmodule.WorldContext) int64 {
	if cc.Revision().Value() >= icmodule.RevisionBTP2 {
		if bc := cc.GetBTPContext(); bc != nil {
			return bc.GetActiveDSAMask()
		}
	}
	return 0
}