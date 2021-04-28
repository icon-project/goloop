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
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/eeproxy"
	"github.com/icon-project/goloop/service/state"
)

// NewInitTransition creates initial transition based on the last result.
// It's only for development purpose. So, normally it should not be used.
func NewInitTransition(
	db db.Database,
	result []byte,
	vl module.ValidatorList,
	cm contract.ContractManager,
	em eeproxy.Manager, chain module.Chain,
	logger log.Logger, plt Platform,
	tsc *TxTimestampChecker,
) (module.Transition, error) {
	var stateHash []byte
	var es state.ExtensionSnapshot
	if len(result) > 0 {
		if tsr, err := newTransitionResultFromBytes(result); err != nil {
			return nil, err
		} else {
			stateHash = tsr.StateHash
			es = plt.NewExtensionSnapshot(db, tsr.ExtensionData)
		}
	}
	if tr, err := newInitTransition(
		db,
		stateHash,
		vl,
		es,
		cm,
		em,
		chain,
		logger,
		plt,
		tsc,
	); err != nil {
		return nil, err
	} else {
		return tr, nil
	}
}

// NewTransition creates new transition based on the parent to execute
// given transactions under given environments.
// It's only for development purpose. So, normally it should not be used.
func NewTransition(
	parent module.Transition,
	patchtxs module.TransactionList,
	normaltxs module.TransactionList,
	bi module.BlockInfo,
	csi module.ConsensusInfo,
	plt Platform,
	alreadyValidated bool,
) module.Transition {
	return newTransition(
		parent.(*transition),
		patchtxs,
		normaltxs,
		bi,
		csi,
		alreadyValidated,
	)
}

// FinalizeTransition finalize parts of transition result without
// updating other information of service manager.
// It's only for development purpose. So, normally it should not be used.
func FinalizeTransition(tr module.Transition, opt int, noFlush bool) error {
	tst := tr.(*transition)
	if opt&module.FinalizeNormalTransaction == module.FinalizeNormalTransaction && !noFlush {
		if err := tst.finalizeNormalTransaction(); err != nil {
			return err
		}
	}
	if opt&module.FinalizePatchTransaction == module.FinalizePatchTransaction && !noFlush {
		if err := tst.finalizePatchTransaction(); err != nil {
			return err
		}
	}
	if opt&module.FinalizeResult == module.FinalizeResult {
		if err := tst.finalizeResult(noFlush); err != nil {
			return err
		}
	}
	return nil
}
