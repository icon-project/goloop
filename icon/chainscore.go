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
	"math/big"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

type chainScore struct {
	cc   contract.CallContext
	from module.Address
}

const (
	CIDForMainNet = 0xaf4e97
)

func applyStepLimits(as state.AccountState, limits map[string]int64) error {
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	for _, k := range state.AllStepLimitTypes {
		cost, _ := limits[k]
		if err := stepLimitTypes.Put(k); err != nil {
			return err
		}
		if err := stepLimitDB.Set(k, cost); err != nil {
			return err
		}
	}
	return nil
}

func applyStepCosts(as state.AccountState, costs map[string]int64) error {
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	for _, k := range state.AllStepTypes {
		cost, _ := costs[k]
		if err := stepTypes.Put(k); err != nil {
			return err
		}
		if err := stepCostDB.Set(k, cost); err != nil {
			return err
		}
	}
	return nil
}

func applyStepPrice(as state.AccountState, price *big.Int) error {
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *chainScore) Install(param []byte) error {
	if s.from != nil {
		return scoreresult.AccessDeniedError.New("AccessDeniedToInstallChainSCORE")
	}

	// set block interval 2 seconds
	as := s.cc.GetAccountState(state.SystemID)
	if err := scoredb.NewVarDB(as, state.VarBlockInterval).Set(2000); err != nil {
		return err
	}

	// skip transaction
	if err := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Set(3); err != nil {
		return err
	}

	stepLimitsMap := map[string]int64{}
	stepTypesMap := map[string]int64{}
	stepPrice := big.NewInt(0)

	switch s.cc.ChainID() {
	case CIDForMainNet:
		// initialize for main network
	default:
		stepLimitsMap = map[string]int64{
			state.StepLimitTypeInvoke: 0x9502f900,
			state.StepLimitTypeQuery:  0x2faf080,
		}
		stepTypesMap = map[string]int64{
			state.StepTypeDefault:          0x186a0,
			state.StepTypeContractCall:     0x61a8,
			state.StepTypeContractCreate:   0x3b9aca00,
			state.StepTypeContractUpdate:   0x5f5e1000,
			state.StepTypeContractDestruct: -0x11170,
			state.StepTypeContractSet:      0x7530,
			state.StepTypeGet:              0x0,
			state.StepTypeSet:              0x140,
			state.StepTypeReplace:          0x50,
			state.StepTypeDelete:           -0xf0,
			state.StepTypeInput:            0xc8,
			state.StepTypeEventLog:         0x64,
			state.StepTypeApiCall:          0x2710,
		}
		stepPrice = big.NewInt(0x2e90edd00)
	}

	if err := applyStepLimits(as, stepLimitsMap); err != nil {
		return err
	}
	if err := applyStepCosts(as, stepTypesMap); err != nil {
		return err
	}
	if err := applyStepPrice(as, stepPrice); err != nil {
		return err
	}

	return nil
}

func (s *chainScore) Update(param []byte) error {
	return nil
}

func (s *chainScore) GetAPI() *scoreapi.Info {
	return nil
}

func newChainScore(cc contract.CallContext, from module.Address) (contract.SystemScore, error) {
	return &chainScore{cc: cc, from: from}, nil
}
