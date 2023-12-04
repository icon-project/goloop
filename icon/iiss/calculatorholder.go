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
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/calculator"
	"github.com/icon-project/goloop/service/state"
)

type CalculatorHolder struct {
	lock   sync.Mutex
	runner Calculator
}

func (h *CalculatorHolder) Start(ess state.ExtensionSnapshot, logger log.Logger) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if ess != nil {
		h.runner = updateCalculator(h.runner, ess, logger)
	} else {
		if h.runner != nil {
			h.runner.Stop()
			h.runner = nil
		}
	}
}

func (h *CalculatorHolder) Get() Calculator {
	h.lock.Lock()
	defer h.lock.Unlock()

	return h.runner
}

func updateCalculator(c Calculator, ess state.ExtensionSnapshot, logger log.Logger) Calculator {
	essi := ess.(*ExtensionSnapshotImpl)
	back := essi.Back2()
	reward := essi.Reward()
	if c != nil {
		if c.IsRunningFor(essi.DB(), back.Bytes(), reward.Bytes()) {
			return c
		}
		c.Stop()
	}
	return calculator.New(essi.DB(), back, reward, logger)
}
