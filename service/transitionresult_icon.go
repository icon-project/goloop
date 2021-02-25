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

package service

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

func NewWorldSnapshot(database db.Database, plt Platform, result []byte, vl module.ValidatorList) (state.WorldSnapshot, error) {
	var stateHash []byte
	var ess state.ExtensionSnapshot
	if len(result) > 0 {
		tr, err := newTransitionResultFromBytes(result)
		if err != nil {
			return nil, err
		}
		stateHash = tr.StateHash
		ess = plt.NewExtensionSnapshot(database, tr.ExtensionData)
	}
	wss := state.NewWorldSnapshot(database, stateHash, vl, ess)
	return wss, nil
}
