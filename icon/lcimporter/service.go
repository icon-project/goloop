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

package lcimporter

import (
	"path"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/sync2"
)

type Service interface {
	NewInitTransition(
		result []byte, vl module.ValidatorList, logger log.Logger,
	) (module.Transition, error)
	NewTransition(
		parent module.Transition, patchtxs module.TransactionList,
		normaltxs module.TransactionList, bi module.BlockInfo,
		csi module.ConsensusInfo, alreadyValidated bool,
	) module.Transition
	NewSyncTransition(
		tr module.Transition, result []byte, vl []byte,
	) module.Transition
	FinalizeTransition(tr module.Transition, opt int, noFlush bool) error
	GetNextBlockVersion(result []byte, vl module.ValidatorList) int
}

type BasicService struct {
	Chain   module.Chain
	Plt     base.Platform
	BaseDir string
	cm      contract.ContractManager
	pm      eeproxy.Manager
}

const ContractPath = "contract"

func (s *BasicService) NewInitTransition(
	result []byte, vl module.ValidatorList, logger log.Logger,
) (module.Transition, error) {
	if logger == nil {
		logger = s.Chain.Logger()
	}
	if s.cm == nil {
		cm, err := s.Plt.NewContractManager(s.Chain.Database(), path.Join(s.BaseDir, ContractPath), s.Chain.Logger())
		if err != nil {
			return nil, err
		}
		s.cm = cm
	}
	tr, err := service.NewInitTransition(s.Chain.Database(), result, vl, s.cm, s.pm,
		s.Chain, logger, s.Plt, service.NewTimestampChecker())
	if err != nil {
		return nil, err
	}
	err = service.FinalizeTransition(tr, module.FinalizeResult, true)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (s *BasicService) NewTransition(
	parent module.Transition, patchtxs module.TransactionList,
	normaltxs module.TransactionList, bi module.BlockInfo,
	csi module.ConsensusInfo, alreadyValidated bool,
) module.Transition {
	return service.NewTransition(parent, patchtxs, normaltxs, bi, csi, alreadyValidated)
}

func (s *BasicService) NewSyncTransition(tr module.Transition, result []byte,
	vl []byte,
) module.Transition {
	return nil
}

func (s *BasicService) FinalizeTransition(tr module.Transition, opt int, noFlush bool) error {
	return service.FinalizeTransition(tr, opt, noFlush)
}

const CIDOfMainNet = 1

func (s *BasicService) GetNextBlockVersion(result []byte, vl module.ValidatorList) int {
	if result == nil {
		return s.Plt.DefaultBlockVersionFor(CIDOfMainNet)
	}
	wss, err := service.NewWorldSnapshot(s.Chain.Database(), s.Plt, result, vl)
	if err != nil {
		return -1
	}
	var bss containerdb.BytesStoreState
	ass := wss.GetAccountSnapshot(state.SystemID)
	if ass == nil {
		bss = containerdb.EmptyBytesStoreState
	} else {
		bss = scoredb.NewStateStoreWith(ass)
	}
	v := int(scoredb.NewVarDB(bss, state.VarNextBlockVersion).Int64())
	if v == 0 {
		return s.Plt.DefaultBlockVersionFor(CIDOfMainNet)
	}
	return v
}

type defaultService struct {
	BasicService
	syncMan service.SyncManager
}

func NewService(c module.Chain, plt base.Platform, pm eeproxy.Manager, baseDir string) (Service, error) {
	return &defaultService{
		BasicService: BasicService{
			Chain:   c,
			Plt:     plt,
			BaseDir: baseDir,
			pm:      pm,
		},
		syncMan: sync2.NewSyncManager(c.Database(), c.NetworkManager(), plt, c.Logger()),
	}, nil
}

func (s *defaultService) NewSyncTransition(tr module.Transition, result []byte,
	vl []byte,
) module.Transition {
	return service.NewSyncTransition(tr, s.syncMan, result, vl, false)
}
