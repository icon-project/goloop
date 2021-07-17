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
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/trace"
)

var methodsAllowingExtraParams = map[string]bool{
	"registerPRep": true,
	"setPRep":      true,
}

func allowExtraParams(method string) bool {
	yn, _ := methodsAllowingExtraParams[method]
	return yn
}

func doNotChargeContractCallStep(method string, revision int) bool {
	if revision >= icmodule.RevisionICON2 || revision < icmodule.RevisionIISS {
		return false
	}
	if method == scoreapi.FallbackMethodName && revision < icmodule.RevisionSystemSCORE {
		return false
	}
	return true
}

type CallHandler interface {
	contract.AsyncContractHandler
	GetMethodName() string
	AllowExtra()
	DoExecuteAsync(cc contract.CallContext, ch eeproxy.CallContext) error
	TLogStart()
	TLogDone(status error, steps *big.Int, result *codec.TypedObj)
	ApplyCallSteps(cc contract.CallContext) bool
}

type SystemCallHandler struct {
	CallHandler
	cc       contract.CallContext
	log      *trace.Logger
	revision module.Revision
}

func (h *SystemCallHandler) ExecuteAsync(cc contract.CallContext) (err error) {
	h.cc = cc
	h.revision = cc.Revision()
	h.log = trace.LoggerOf(cc.Logger())

	h.TLogStart()
	defer func() {
		if err != nil {
			// do not charge contractCall step for some external methods
			if !doNotChargeContractCallStep(h.GetMethodName(), h.revision.Value()) {
				// charge contractCall step if preprocessing is failed
				if !h.ApplyCallSteps(cc) {
					err = scoreresult.OutOfStepError.Wrap(err, "OutOfStepForCall")
				}
			}
			h.TLogDone(err, cc.StepUsed(), nil)
		}
	}()

	if h.revision.Value() < icmodule.RevisionSystemSCORE {
		if allowExtraParams(h.GetMethodName()) {
			h.log.TSystemf("FRAME[%d] allow extra params", cc.FrameID())
			h.AllowExtra()
		}
		if h.GetMethodName() == scoreapi.FallbackMethodName {
			h.log.TSystemf("FRAME[%d] system contract is unavailable for fallback", cc.FrameID())
			return scoreresult.ContractNotFoundError.New("NoFallback")
		}
		defer func() {
			if scoreresult.MethodNotPayableError.Equals(err) {
				h.log.TSystemf(
					"FRAME[%d] result patch from=%v to=%v",
					cc.FrameID(),
					err,
					scoreresult.ErrInvalidParameter,
				)
				err = errors.WithCode(err, scoreresult.InvalidParameterError)
			}
		}()
	}
	if h.revision.Value() < icmodule.RevisionIISS {
		defer func() {
			if scoreresult.MethodNotFoundError.Equals(err) {
				h.log.TSystemf(
					"FRAME[%d] result patch from=%v to=%v",
					cc.FrameID(),
					err,
					scoreresult.ErrContractNotFound,
				)
				err = errors.WithCode(err, scoreresult.ContractNotFoundError)
			}
		}()
	}
	return h.CallHandler.DoExecuteAsync(cc, h)
}

func (h *SystemCallHandler) OnResult(status error, steps *big.Int, result *codec.TypedObj) {
	if h.revision.Value() < icmodule.RevisionICON2 {
		if icmodule.IllegalArgumentError.Equals(status) {
			status = errors.WithCode(status, scoreresult.IllegalFormatError)
		}
	}
	h.CallHandler.OnResult(status, steps, result)
}

func newSystemHandler(ch CallHandler) contract.ContractHandler {
	return &SystemCallHandler{ CallHandler: ch }
}