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

package icon

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/state"
)

type platform struct{}

func (p *platform) NewContractManager(dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	return newContractManager(p, dbase, dir, logger)
}

func (p *platform) NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot {
	// TODO return valid ExtensionSnapshot(not nil) which can return valid ExtensionState.
	//  with that state, we may change state of extension.
	//  For initial state, the snapshot returns nil for Bytes() method.
	return nil
}

func (p *platform) NewExtensionWithBuilder(builder merkle.Builder, raw []byte) state.ExtensionSnapshot {
	// TODO return ExtensionSnapshot instance after requesting required data to
	//  the builder.
	return nil
}

func (p *platform) ToRevision(value int) module.Revision {
	return valueToRevision(value)
}

func (p *platform) NewBaseTransaction(wc state.WorldContext) (module.Transaction, error) {
	// TODO calculate issued i-score and amount balance. No changes on world context.
	return nil, nil
}

func (p *platform) OnExtensionSnapshotFinalization(ess state.ExtensionSnapshot) {
	// TODO start background calculator if it's not started.
}

func (p *platform) OnExecutionEnd(wc state.WorldContext) error {
	// TODO implement
	return nil
}

func (p *platform) Term() {
	// TODO implement
}

func NewPlatform(base string, cid int) (service.Platform, error) {
	return &platform{}, nil
}

func init() {
	iiss.RegisterBaseTx()
}
