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
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/trace"
)

var methodsAllowingExtraParams = map[string]bool  {
	"registerPRep": true,
	"setPRep": true,
}

func allowExtraParams(method string) bool {
	yn, _ :=  methodsAllowingExtraParams[method]
	return yn
}

type CallHandler interface {
	contract.AsyncContractHandler
	GetMethodName() string
	AllowExtra()
}

type SystemCallHandler struct {
	CallHandler
}

func (h *SystemCallHandler) ExecuteAsync(cc contract.CallContext) (err error) {
	logger := trace.LoggerOf(cc.Logger())
	revision := cc.Revision()
	if revision.Value() < icmodule.Revision9 {
		if allowExtraParams(h.GetMethodName()) {
			logger.TSystemf("FRAME[%d] allow extra params", cc.FrameID())
			h.AllowExtra()
		}
		defer func() {
			if scoreresult.MethodNotFoundError.Equals(err) {
				logger.TSystemf(
					"FRAME[%d] result patch from=%v to=%v",
					cc.FrameID(),
					err,
					scoreresult.ErrContractNotFound,
				)
				err = errors.WithCode(err, scoreresult.ContractNotFoundError)
			}
		}()
	}
	return h.CallHandler.ExecuteAsync(cc)
}

func newSystemHandler(ch CallHandler) contract.ContractHandler {
	return &SystemCallHandler{ ch }
}