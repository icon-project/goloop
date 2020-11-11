/*
 * Copyright 2020 ICON Foundation
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

package basic

import (
	"math/big"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

type platform struct{}

var Platform service.Platform = &platform{}

type basicContractManager struct {
	contract.ContractManager
}

func (b basicContractManager) GetSystemScore(contentID string, cc contract.CallContext, from module.Address, value *big.Int) (contract.SystemScore, error) {
	if contentID == contract.CID_CHAIN {
		return NewChainScore(cc, from, value)
	}
	return b.ContractManager.GetSystemScore(contentID, cc, from, value)
}

func (t *platform) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	return nil
}

func (t *platform) NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	cm, err := contract.NewContractManager(dbase, dir, logger)
	if err != nil {
		return nil, err
	}
	return basicContractManager{cm}, nil
}

func (t *platform) NewExtensionSnapshot(database db.Database, raw []byte) state.ExtensionSnapshot {
	return nil
}

func (t *platform) ToRevision(value int) module.Revision {
	return valueToRevision(value)
}

func (t *platform) NewBaseTransaction(wc state.WorldContext) (module.Transaction, error) {
	return nil, nil
}

func (t *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot) {
	// do nothing
}

func (t *platform) OnExecutionEnd(wc state.WorldContext) error {
	return nil
}

func (t *platform) Term() {
	// do nothing
}
