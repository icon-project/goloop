/*
 * Copyright 2021 ICON Foundation
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

package icon

import (
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
)

type callHandler struct {
	CallHandler
	to module.Address
}

func (h *callHandler) ExecuteAsync(cc contract.CallContext) (err error) {
	h.TLogStart()
	defer func() {
		if err != nil {
			if !h.ApplyCallSteps(cc) {
				err = scoreresult.OutOfStepError.Wrap(err, "OutOfStepForCall")
			}
			h.TLogDone(err, cc.StepUsed(), nil)
		}
	}()

	if cc.Revision().Value() < icmodule.RevisionICON2 {
		ass := cc.GetAccountSnapshot(h.to.ID())
		if ass == nil || ass.IsEmpty() {
			return scoreresult.UnknownFailureError.New("NoAccount")
		}
	}
	return h.DoExecuteAsync(cc, h)
}

func newCallHandler(ch CallHandler, to module.Address) contract.ContractHandler {
	return &callHandler{CallHandler: ch, to: to}
}
