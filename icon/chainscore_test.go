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
 *
 */

package icon

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

type fakeCallContext struct {
	contract.CallContext
	accounts map[string]*fakeAccountState
	revision module.Revision
}

func (cc *fakeCallContext) GetAccountState(id []byte) state.AccountState {
	if as, ok := cc.accounts[string(id)]; ok {
		return as
	} else {
		as = &fakeAccountState{
			data: make(map[string][]byte),
		}
		cc.accounts[string(id)] = as
		return as
	}
}

func (cc *fakeCallContext) OnEvent(add module.Address, indexed [][]byte, data [][]byte) {
	// do nothing
}

func (cc *fakeCallContext) Revision() module.Revision {
	return cc.revision
}

func newFakeCallContext() *fakeCallContext {
	return &fakeCallContext{
		accounts: make(map[string]*fakeAccountState),
	}
}

type fakeAccountState struct {
	state.AccountState
	data map[string][]byte
}

func (as *fakeAccountState) GetValue(k []byte) ([]byte, error) {
	v, _ := as.data[string(k)]
	return v, nil
}

func (as *fakeAccountState) SetValue(k, v []byte) ([]byte, error) {
	if v == nil {
		return as.DeleteValue(k)
	}
	old, _ := as.data[string(k)]
	as.data[string(k)] = v
	return old, nil
}

func (as *fakeAccountState) DeleteValue(k []byte) ([]byte, error) {
	if old, ok := as.data[string(k)]; ok {
		delete(as.data, string(k))
		return old, nil
	} else {
		return nil, nil
	}
}

func TestChainScore_GetAPI(t *testing.T) {
	cc := newFakeCallContext()
	score := &chainScore{
		cc: cc,
	}
	for i := 1; i <= icmodule.MaxRevision; i++ {
		t.Run(fmt.Sprintf("revision%d", i), func(t *testing.T) {
			cc.revision = icmodule.ValueToRevision(i - 1)
			old, err := contract.SetRevision(cc, i, false)
			assert.NoError(t, err)
			assert.Equal(t, i-1, old)

			apis := score.GetAPI()
			err = contract.CheckMethod(score, apis)
			assert.NoError(t, err)
		})
	}
}
