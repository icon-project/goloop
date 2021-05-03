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

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/sync"
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
}

type BasicService struct {
	Chain   module.Chain
	Plt     service.Platform
	BaseDir string
	cm      contract.ContractManager
}

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
	return service.NewInitTransition(s.Chain.Database(), result, vl, s.cm, nil,
		s.Chain, logger, s.Plt, service.NewTimestampChecker())
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

const (
	ContractPath = "contract"
	EESocketPath = "ee.sock"
)

type defaultService struct {
	BasicService
	syncMan service.SyncManager
	cm      contract.ContractManager
	em      eeproxy.Manager
}

func NewService(c module.Chain, plt service.Platform, baseDir string) (Service, error) {
	return &defaultService{
		BasicService: BasicService{
			Chain:   c,
			Plt:     plt,
			BaseDir: baseDir,
		},
		syncMan: sync.NewSyncManager(c.Database(), c.NetworkManager(), plt, c.Logger()),
	}, nil
}

func (s *defaultService) setupEE() error {
	cm, err := s.Plt.NewContractManager(s.Chain.Database(), path.Join(s.BaseDir, ContractPath), s.Chain.Logger())
	if err != nil {
		return errors.Wrap(err, "NewContractManagerFailure")
	}
	ee, err := eeproxy.AllocEngines(s.Chain.Logger(), "python")
	if err != nil {
		return errors.Wrap(err, "FailureInAllocEngines")
	}
	em, err := eeproxy.NewManager("unix", path.Join(s.BaseDir, EESocketPath), s.Chain.Logger(), ee...)
	if err != nil {
		return errors.Wrap(err, "FailureInAllocProxyManager")
	}

	go em.Loop()
	em.SetInstances(1, 1, 1)

	s.cm = cm
	s.em = em
	return nil
}

func (s *defaultService) NewInitTransition(
	result []byte, vl module.ValidatorList, logger log.Logger,
) (module.Transition, error) {
	if s.cm == nil || s.em == nil {
		if err := s.setupEE(); err != nil {
			return nil, err
		}
	}
	if logger == nil {
		logger = s.Chain.Logger()
	}
	return service.NewInitTransition(s.Chain.Database(), result, vl, s.cm, s.em,
		s.Chain, logger, s.Plt, service.NewTimestampChecker())
}

func (s *defaultService) NewSyncTransition(tr module.Transition, result []byte,
	vl []byte,
) module.Transition {
	return service.NewSyncTransition(tr, s.syncMan, result, vl)
}
