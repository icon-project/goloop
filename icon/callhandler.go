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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
)

type callHandler struct {
	CallHandler
	to module.Address

	rlp bool
	ext bool
}

func (h *callHandler) ExecuteAsync(cc contract.CallContext) (err error) {
	h.TLogStart()
	defer func() {
		if err != nil {
			if err2 := h.ApplyCallSteps(cc); err2 != nil {
				err = err2
			}
			h.TLogDone(err, cc.StepUsed(), nil)
		}
	}()

	rev := cc.Revision().Value()
	if rev < icmodule.RevisionICON2 && !h.ext {
		ass := cc.GetAccountSnapshot(h.to.ID())
		if ass == nil || ass.ActiveContract() == nil {
			return scoreresult.UnknownFailureError.New("NoAccount")
		}
	}
	if rev >= icmodule.Revision12 && rev < icmodule.RevisionICON2 {
		h.rlp = true
	}
	return h.DoExecuteAsync(cc, h)
}

func (h *callHandler) SetValue(key []byte, value []byte) ([]byte, error) {
	if len(value) == 0 {
		old, err := h.CallHandler.DeleteValue(key)
		if err == nil && h.rlp {
			if old != nil {
				key2 := crypto.SHA3Sum256(key)
				var backup []byte
				backup, err = h.CallHandler.GetValue(key2)
				if err == nil {
					if backup != nil {
						if backup[0] == 0 {
							_, err = h.CallHandler.DeleteValue(key2)
						} else if backup[0] == 1 {
							_, err = h.CallHandler.SetValue(key, backup[1:])
						}
					} else {
						_, err = h.CallHandler.SetValue(key, old)
					}
				}
			}
		}
		return old, err
	} else {
		old, err := h.CallHandler.SetValue(key, value)
		if err == nil && h.rlp {
			key2 := crypto.SHA3Sum256(key)
			if old == nil {
				_, err = h.CallHandler.SetValue(key2, []byte{0})
			} else {
				var backup []byte
				backup, err = h.CallHandler.GetValue(key2)
				if err == nil && backup == nil {
					_, err = h.CallHandler.SetValue(key2, append([]byte{1}, old...))
				}
			}
		}
		return old, err
	}
}

func (h *callHandler) DeleteValue(key []byte) ([]byte, error) {
	old, err := h.CallHandler.DeleteValue(key)
	if err == nil && h.rlp {
		key2 := crypto.SHA3Sum256(key)
		_, err = h.CallHandler.DeleteValue(key2)
	}
	return old, err
}

func newCallHandler(ch CallHandler, to module.Address, external bool) contract.ContractHandler {
	return &callHandler{CallHandler: ch, to: to, ext: external}
}
