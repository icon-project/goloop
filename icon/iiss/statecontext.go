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
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type stateContext struct {
	icmodule.WorldContext
	*icstate.State
	eventLogger icmodule.EnableEventLogger

	// Cache
	term *icstate.TermSnapshot
}

func NewStateContext(wc icmodule.WorldContext, es *ExtensionStateImpl) icmodule.StateContext {
	return &stateContext{
		WorldContext: wc,
		State: es.State,
		eventLogger: es,
	}
}

func (sc *stateContext) Revision() int {
	return sc.WorldContext.Revision().Value()
}

// TermRevision returns revision stored in TermSnapshot
func (sc *stateContext) TermRevision() int {
	if term := sc.getTermSnapshot(); term != nil {
		return term.Revision()
	}
	return 0
}

func (sc *stateContext) IsIISS4Activated() bool {
	return sc.TermRevision() >= icmodule.RevisionIISS4
}

func (sc *stateContext) AddEventEnable(owner module.Address, status icmodule.EnableStatus) error {
	if sc.eventLogger != nil {
		return sc.eventLogger.AddEventEnable(sc.BlockHeight(), owner, status)
	}
	return nil
}

func (sc *stateContext) getTermSnapshot() *icstate.TermSnapshot {
	if sc.term == nil {
		sc.term = sc.State.GetTermSnapshot()
	}
	return sc.term
}