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
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"math/big"
)

type contractManager struct {
	contract.ContractManager
	plt *platform
}

func (cm *contractManager) GetSystemScore(contentID string, cc contract.CallContext, from module.Address, value *big.Int) (contract.SystemScore, error) {
	if contentID == contract.CID_CHAIN {
		return newChainScore(cc, from)
	}
	return cm.ContractManager.GetSystemScore(contentID, cc, from, value)
}

func newContractManager(plt *platform, dbase db.Database, dir string, logger log.Logger) (contract.ContractManager, error) {
	cm, err := contract.NewContractManager(dbase, dir, logger)
	if err != nil {
		return nil, err
	}
	return &contractManager{cm, plt}, nil
}
