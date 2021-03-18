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
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/transaction"
)

type chainMethod struct {
	scoreapi.Method
	minVer, maxVer int
}

type chainScore struct {
	cc    contract.CallContext
	log   log.Logger
	from  module.Address
	value *big.Int
	gov   bool
}

const (
	CIDForMainNet         = 0x1
	StatusIllegalArgument = module.StatusReverted + iota
	StatusNotFound
)

var chainMethods = []*chainMethod{
	{scoreapi.Method{
		scoreapi.Function, "getIISSInfo",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setIRep",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{
		scoreapi.Function, "getIRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{
		scoreapi.Function, "getRRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{
		scoreapi.Function, "setStake",
		scoreapi.FlagExternal | scoreapi.FlagPayable, 1,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getStake",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "setDelegation",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"delegations", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"address", scoreapi.Address, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getDelegation",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "claimIScore",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "queryIScore",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "registerPRep",
		scoreapi.FlagExternal | scoreapi.FlagPayable, 7,
		[]scoreapi.Parameter{
			{"name", scoreapi.String, nil, nil},
			{"email", scoreapi.String, nil, nil},
			{"website", scoreapi.String, nil, nil},
			{"country", scoreapi.String, nil, nil},
			{"city", scoreapi.String, nil, nil},
			{"details", scoreapi.String, nil, nil},
			{"p2pEndpoint", scoreapi.String, nil, nil},
			{"nodeAddress", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getPRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "unregisterPRep",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "setPRep",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"name", scoreapi.String, nil, nil},
			{"email", scoreapi.String, nil, nil},
			{"website", scoreapi.String, nil, nil},
			{"country", scoreapi.String, nil, nil},
			{"city", scoreapi.String, nil, nil},
			{"details", scoreapi.String, nil, nil},
			{"p2pEndpoint", scoreapi.String, nil, nil},
			{"nodeAddress", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "setGovernanceVariables",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"irep", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionDecentralize, icmodule.Revision8},
	{scoreapi.Method{
		scoreapi.Function, "getPRepManager",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"startRanking", scoreapi.Integer, nil, nil},
			{"endRanking", scoreapi.Integer, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getMainPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getSubPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "setBond",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"bondList", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"address", scoreapi.Address, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getBond",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "setBonderList",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"bonderList", scoreapi.ListTypeOf(1, scoreapi.Address), nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getBonderList",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "estimateUnstakeLockPeriod",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{
		scoreapi.Function, "getPRepTerm",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "disableScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "enableScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "txHashToAddress",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Address,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "addressToTxHashes",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "acceptScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "rejectScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "blockScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "unblockScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setRevision",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"code", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setStepPrice",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"price", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getRevision",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStepPrice",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStepCosts",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getMaxStepLimit",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getScoreStatus",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getServiceConfig",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getScoreDenyList",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getImportAllowList",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "removeScoreDenyList",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"score", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "addScoreDenyList",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"score", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRepStats",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "disqualifyPRep",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"prep", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "validateIrep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"newIrep", scoreapi.Integer, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, 0, 0},
}

func applyStepLimits(fee *FeeConfig, as state.AccountState) error {
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if fee.StepLimit != nil {
		for _, k := range state.AllStepLimitTypes {
			if err := stepLimitTypes.Put(k); err != nil {
				return err
			}
			icost := fee.StepLimit[k]
			if err := stepLimitDB.Set(k, icost.Value); err != nil {
				return err
			}
		}
	} else {
		for _, k := range state.AllStepLimitTypes {
			if err := stepLimitTypes.Put(k); err != nil {
				return err
			}
			if err := stepLimitDB.Set(k, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyStepCosts(fee *FeeConfig, as state.AccountState) error {
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if fee.StepCosts != nil {
		for _, k := range state.AllStepTypes {
			if err := stepTypes.Put(k); err != nil {
				return err
			}
			icost := fee.StepCosts[k]
			if err := stepCostDB.Set(k, icost.Value); err != nil {
				return err
			}
		}
	} else {
		for _, k := range state.AllStepTypes {
			if err := stepTypes.Put(k); err != nil {
				return err
			}
			if err := stepCostDB.Set(k, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyStepPrice(as state.AccountState, price *big.Int) error {
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

const (
	configFile             = "./icon_config.json"
	defaultIISSVersion     = 1
	defaultIISSBlockHeight = 0
	defaultTermPeriod      = 43120
	defaultUnbondingPeriod = defaultTermPeriod * 7
	defaultUnstakeSlotMax  = 1000
	defaultMainPRepCount   = 22
	defaultSubPRepCount    = 78
	defaultIRep            = iiss.MonthBlock * iiss.IScoreICXRatio
	defaultRRep            = iiss.MonthBlock * iiss.IScoreICXRatio
	defaultBondRequirement = 5
	defaultLockMin         = defaultTermPeriod * 5
	defaultLockMax         = defaultTermPeriod * 20
	rewardPoint            = 0.7
	defaultIglobal         = iiss.YearBlock * iiss.IScoreICXRatio
	defaultIprep           = 50
	defaultIcps            = 0
	defaultIrelay          = 0
	defaultIvoter          = 50
	defaultUnbondingMax    = 1000
)

type config struct {
	TermPeriod      *common.HexInt `json:"termPeriod"`
	IISSVersion     *common.HexInt `json:"iissVersion,omitempty"`
	IISSBlockHeight *common.HexInt `json:"iissBlockHeight,omitempty"`
	MainPRepCount   *common.HexInt `json:"mainPRepCount"`
	SubPRepCount    *common.HexInt `json:"subPRepCount"`
	Irep            *common.HexInt `json:"irep,omitempty"`
	Rrep            *common.HexInt `json:"rrep,omitempty"`
	BondRequirement *common.HexInt `json:"bondRequirement,omitempty"`
	UnbondingPeriod *common.HexInt `json:"unbondingPeriod,omitempty"`
	UnstakeSlotMax  *common.HexInt `json:"unstakeSlotMax,omitempty"`
	LockMin         *common.HexInt `json:"lockMin,omitempty"`
	LockMax         *common.HexInt `json:"lockMax,omitempty"`
	RewardFund      rewardFund     `json:"rewardFund"`
	UnbondingMax    *common.HexInt `json:"unbondingMax"`
}

type rewardFund struct {
	Iglobal *common.HexInt `json:"Iglobal"`
	Iprep   *common.HexInt `json:"Iprep"`
	Icps    *common.HexInt `json:"Icps"`
	Irelay  *common.HexInt `json:"Irelay"`
	Ivoter  *common.HexInt `json:"Ivoter"`
}

func applyRewardFund(iconConfig *config, s *icstate.State) error {
	rf := &icstate.RewardFund{
		Iglobal: new(big.Int).Set(iconConfig.RewardFund.Iglobal.Value()),
		Iprep:   new(big.Int).Set(iconConfig.RewardFund.Iprep.Value()),
		Icps:    new(big.Int).Set(iconConfig.RewardFund.Icps.Value()),
		Irelay:  new(big.Int).Set(iconConfig.RewardFund.Irelay.Value()),
		Ivoter:  new(big.Int).Set(iconConfig.RewardFund.Ivoter.Value()),
	}
	if err := s.SetRewardFund(rf); err != nil {
		return err
	}
	return nil
}

type FeeConfig struct {
	StepPrice common.HexInt              `json:"stepPrice"`
	StepLimit map[string]common.HexInt64 `json:"stepLimit,omitempty"`
	StepCosts map[string]common.HexInt64 `json:"stepCosts,omitempty"`
}

type ChainConfig struct {
	Revision                 common.HexInt32   `json:"revision"`
	AuditEnabled             common.HexInt16   `json:"auditEnabled"`
	DeployerWhiteListEnabled common.HexInt16   `json:"deployerWhiteListEnabled"`
	Fee                      FeeConfig         `json:"fee"`
	ValidatorList            []*common.Address `json:"validatorList"`
	MemberList               []*common.Address `json:"memberList"`
	BlockInterval            *common.HexInt64  `json:"blockInterval"`
	CommitTimeout            *common.HexInt64  `json:"commitTimeout"`
	TimestampThreshold       *common.HexInt64  `json:"timestampThreshold"`
	RoundLimitFactor         *common.HexInt64  `json:"roundLimitFactor"`
	MinimizeBlockGen         *common.HexInt16  `json:"minimizeBlockGen"`
	DepositTerm              *common.HexInt64  `json:"depositTerm"`
	DepositIssueRate         *common.HexInt64  `json:"depositIssueRate"`
	FeeSharingEnabled        *common.HexInt16  `json:"feeSharingEnabled"`
}

func newIconConfig() *config {
	return &config{
		TermPeriod:      common.NewHexInt(defaultTermPeriod),
		IISSVersion:     common.NewHexInt(defaultIISSVersion),
		IISSBlockHeight: common.NewHexInt(defaultIISSBlockHeight),
		MainPRepCount:   common.NewHexInt(defaultMainPRepCount),
		SubPRepCount:    common.NewHexInt(defaultSubPRepCount),
		Irep:            common.NewHexInt(defaultIRep),
		Rrep:            common.NewHexInt(defaultRRep),
		BondRequirement: common.NewHexInt(defaultBondRequirement),
		LockMin:         common.NewHexInt(defaultLockMin),
		LockMax:         common.NewHexInt(defaultLockMax),
		UnbondingPeriod: common.NewHexInt(defaultUnbondingPeriod),
		UnstakeSlotMax:  common.NewHexInt(defaultUnstakeSlotMax),
		UnbondingMax:    common.NewHexInt(defaultUnbondingMax),
		RewardFund: rewardFund{
			Iglobal: common.NewHexInt(defaultIglobal),
			Iprep:   common.NewHexInt(defaultIprep),
			Icps:    common.NewHexInt(defaultIcps),
			Irelay:  common.NewHexInt(defaultIrelay),
			Ivoter:  common.NewHexInt(defaultIvoter),
		},
	}
}

func (s *chainScore) loadIconConfig() *config {
	iconConfig := newIconConfig()
	f, err := os.Open(configFile)
	if err != nil {
		s.log.Infof("Failed to open configuration file %+v. Use default config", err)
		return iconConfig
	}
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		s.log.Infof("Failed to read configuration file %+v. Use default config", err)
		return iconConfig
	}
	if err = json.Unmarshal(bs, &iconConfig); err != nil {
		s.log.Infof("Failed to unmarshal configuration file %+v. Use default config", err)
		return iconConfig
	}

	return iconConfig
}

func (s *chainScore) Install(param []byte) error {
	var err error
	if s.from != nil {
		return scoreresult.AccessDeniedError.New("AccessDeniedToInstallChainSCORE")
	}

	as := s.cc.GetAccountState(state.SystemID)

	iconConfig := s.loadIconConfig()
	var feeConfig *FeeConfig
	var systemConfig int
	var revision int
	var validators []module.Validator
	var handlers []contract.ContractHandler

	switch s.cc.ChainID() {
	case CIDForMainNet:
		// initialize for main network
		feeConfig = new(FeeConfig)
		feeConfig.StepPrice.SetString("10000000000", 10)
		feeConfig.StepLimit = map[string]common.HexInt64{
			state.StepLimitTypeInvoke: {0x78000000},
			state.StepLimitTypeQuery:  {0x780000},
		}
		feeConfig.StepCosts = map[string]common.HexInt64{
			state.StepTypeDefault:          {1_000_000},
			state.StepTypeContractCall:     {15_000},
			state.StepTypeContractCreate:   {200_000},
			state.StepTypeContractUpdate:   {80_000},
			state.StepTypeContractDestruct: {-70_000},
			state.StepTypeContractSet:      {30_000},
			state.StepTypeGet:              {0},
			state.StepTypeSet:              {200},
			state.StepTypeReplace:          {50},
			state.StepTypeDelete:           {-150},
			state.StepTypeInput:            {200},
			state.StepTypeEventLog:         {100},
			state.StepTypeApiCall:          {0},
		}
		systemConfig = state.SysConfigAudit
		revision = icmodule.Revision1

		// prepare Governance SCORE
		governance, err := ioutil.ReadFile("icon_governance.zip")
		if err != nil || len(governance) == 0 {
			return transaction.InvalidGenesisError.Wrap(err, "FailOnGovernance")
		}
		params := json.RawMessage("{}")
		handler := contract.NewDeployHandlerForPreInstall(
			common.MustNewAddressFromString("hx677133298ed5319607a321a38169031a8867085c"),
			s.cc.Governance(),
			"application/zip",
			governance,
			&params,
			s.cc.Logger(),
		)
		handlers = append(handlers, handler)
	default:
		var chainConfig ChainConfig
		if param != nil {
			if err := json.Unmarshal(param, &chainConfig); err != nil {
				return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to parse parameter for chainScore. err(%+v)\n", err)
			}
		}

		if chainConfig.Revision.Value != 0 {
			revision = int(chainConfig.Revision.Value)
			if revision > icmodule.MaxRevision {
				return scoreresult.IllegalFormatError.Errorf(
					"RevisionIsHigherMax(%d > %d)", revision, icmodule.MaxRevision)
			} else if revision > icmodule.LatestRevision {
				s.log.Warnf("Revision in genesis is higher than latest(%d > %d)",
					revision, icmodule.LatestRevision)
			}
		}

		validators = make([]module.Validator, len(chainConfig.ValidatorList))
		for i, validator := range chainConfig.ValidatorList {
			validators[i], _ = state.ValidatorFromAddress(validator)
			s.log.Debugf("add validator %d: %v", i, validator)
		}
		feeConfig = &chainConfig.Fee
	}

	if err := scoredb.NewVarDB(as, state.VarRevision).Set(revision); err != nil {
		return err
	}

	// load validatorList
	// set block interval 2 seconds
	if err := scoredb.NewVarDB(as, state.VarBlockInterval).Set(2000); err != nil {
		return err
	}

	// skip transaction
	if err := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Set(3); err != nil {
		return err
	}

	if err := scoredb.NewVarDB(as, state.VarChainID).Set(s.cc.ChainID()); err != nil {
		return err
	}

	if feeConfig != nil {
		if err = applyStepLimits(feeConfig, as); err != nil {
			return err
		}
		if err = applyStepCosts(feeConfig, as); err != nil {
			return err
		}
		if err = applyStepPrice(as, &feeConfig.StepPrice.Int); err != nil {
			return err
		}
	}

	if len(validators) > 0 {
		if err := s.cc.GetValidatorState().Set(validators); err != nil {
			return errors.CriticalUnknownError.Wrap(err, "FailToSetValidators")
		}
	}

	if err := scoredb.NewVarDB(as, state.VarServiceConfig).Set(systemConfig); err != nil {
		return err
	}

	s.cc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(s.cc.Database(), nil))
	es := s.cc.GetExtensionState().(*iiss.ExtensionStateImpl)
	if err = es.State.SetIISSVersion(int(iconConfig.IISSVersion.Int64())); err != nil {
		return err
	}
	if err = es.State.SetIISSBlockHeight(iconConfig.IISSBlockHeight.Int64()); err != nil {
		return err
	}
	if err = es.State.SetTermPeriod(iconConfig.TermPeriod.Int64()); err != nil {
		return err
	}
	if err = es.State.SetIRep(iconConfig.Irep.Value()); err != nil {
		return err
	}
	if err = es.State.SetRRep(iconConfig.Rrep.Value()); err != nil {
		return err
	}
	if err = es.State.SetMainPRepCount(iconConfig.MainPRepCount.Int64()); err != nil {
		return err
	}
	if err = es.State.SetSubPRepCount(iconConfig.SubPRepCount.Int64()); err != nil {
		return err
	}
	if err = es.State.SetBondRequirement(iconConfig.BondRequirement.Int64()); err != nil {
		return err
	}
	if err = es.State.SetLockVariables(iconConfig.LockMin.Value(), iconConfig.LockMax.Value()); err != nil {
		return err
	}
	if err = es.State.SetUnbondingPeriod(iconConfig.UnbondingPeriod.Int64()); err != nil {
		return err
	}
	if err = applyRewardFund(iconConfig, es.State); err != nil {
		return err
	}
	if err = es.State.SetUnstakeSlotMax(iconConfig.UnstakeSlotMax.Int64()); err != nil {
		return err
	}
	if err = es.State.SetUnbondingMax(iconConfig.UnbondingMax.Value()); err != nil {
		return err
	}

	for _, handler := range handlers {
		status, _, _, _ := s.cc.Call(handler, s.cc.StepAvailable())
		if status != nil {
			return transaction.InvalidGenesisError.Wrap(status,
				"FAIL to install initial governance score.")
		}
	}

	s.handleRevisionChange(as, icmodule.Revision1, revision)

	return nil
}

func (s *chainScore) Update(param []byte) error {
	return nil
}

func (s *chainScore) GetAPI() *scoreapi.Info {
	ass := s.cc.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	revision := int(scoredb.NewVarDB(as, state.VarRevision).Int64())
	mLen := len(chainMethods)
	methods := make([]*scoreapi.Method, mLen)
	j := 0
	for _, m := range chainMethods {
		if m.minVer <= revision && (m.maxVer == 0 || revision <= m.maxVer) {
			methods[j] = &m.Method
			j += 1
		}
	}

	return scoreapi.NewInfo(methods[:j])
}

func (s *chainScore) checkGovernance(charge bool) error {
	if !s.gov {
		if charge {
			if !s.cc.ApplySteps(state.StepTypeContractCall, 1) {
				return scoreresult.OutOfStepError.New("UserCodeError")
			}
		}
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	return nil
}

func newChainScore(cc contract.CallContext, from module.Address, value *big.Int) (contract.SystemScore, error) {
	return &chainScore{
			cc:    cc,
			from:  from,
			value: value,
			log:   cc.Logger(),
			gov:   cc.Governance().Equal(from),
		},
		nil
}
