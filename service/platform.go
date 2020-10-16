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

package service

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

type Platform interface {
	NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error)
	NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot
	NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot
	OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot)
	ToRevision(value int) module.Revision
	NewBaseTransaction(wc state.WorldContext) (module.Transaction, error)
	OnExecutionEnd(wc state.WorldContext) error
	Term()
}
