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
	"github.com/icon-project/goloop/service/contract"
)

var methodsAllowingExtraParams = map[string]bool  {
	"registerPRep": true,
	"setPRep": true,
}

func needAllowExtra(method string) bool {
	yn, _ :=  methodsAllowingExtraParams[method]
	return yn
}

type CallHandlerAllowingExtra struct {
	*contract.CallHandler
}

func (h *CallHandlerAllowingExtra) ExecuteAsync(cc contract.CallContext) (err error) {
	if cc.Revision().Value() < icmodule.Revision9 {
		h.AllowExtra()
	}
	return h.CallHandler.ExecuteAsync(cc)
}

type TransferAndCallHandlerAllowingExtra struct {
	*contract.TransferAndCallHandler
}

func (h *TransferAndCallHandlerAllowingExtra) ExecuteAsync(cc contract.CallContext) (err error) {
	if cc.Revision().Value() < icmodule.Revision9 {
		h.AllowExtra()
	}
	return h.TransferAndCallHandler.ExecuteAsync(cc)
}
