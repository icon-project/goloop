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
	"strconv"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
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
	{scoreapi.Method{scoreapi.Function, "getNetworkValue",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getIISSInfo",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "setIRep",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{
		scoreapi.Function, "getIRep",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{
		scoreapi.Function, "getRRep",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0}, // TODO change minVer to Revision11
	{scoreapi.Method{scoreapi.Function, "setStake",
		scoreapi.FlagExternal | scoreapi.FlagPayable, 0,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "getStake",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "setDelegation",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"delegations", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"address", scoreapi.String, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "getDelegation",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "claimIScore",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "queryIScore",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "registerPRep",
		scoreapi.FlagPayable | scoreapi.FlagExternal, 0,
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
	{scoreapi.Method{scoreapi.Function, "getPRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "unregisterPRep",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "setPRep",
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
	{scoreapi.Method{scoreapi.Function, "getPRepManager",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"startRanking", scoreapi.Integer, nil, nil},
			{"endRanking", scoreapi.Integer, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "getMainPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "getSubPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "setBond",
		scoreapi.FlagExternal, 0,
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
	{scoreapi.Method{scoreapi.Function, "getBond",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "setBonderList",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"bonderList", scoreapi.ListTypeOf(1, scoreapi.Address), nil, nil},
		},
		nil,
	}, 0, 0}, // TODO change minVer to Revision5
	{scoreapi.Method{scoreapi.Function, "getBonderList",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0}, // TODO change minVer to Revision5
	{
		scoreapi.Method{
			scoreapi.Function, "estimateUnstakeLockPeriod",
			scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
			nil,
			[]scoreapi.DataType{
				scoreapi.Dict,
			},
		},
		0,
		0}, // TODO change minVer to Revision5
	{
		scoreapi.Method{
			scoreapi.Function,
			"getPRepTerm",
			scoreapi.FlagReadOnly | scoreapi.FlagExternal,
			0,
			nil,
			[]scoreapi.DataType{scoreapi.Dict},
		},
		0,
		0,
	},
	{scoreapi.Method{scoreapi.Function, "disableScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "disableScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "enableScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "enableScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "setRevision",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"code", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "setRevision",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"code", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "acceptScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "acceptScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "rejectScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
			{"reason", scoreapi.String, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "rejectScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
			{"reason", scoreapi.String, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "blockScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "blockScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "unblockScore",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "unblockScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "setStepPrice",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"price", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "setStepPrice",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"price", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "setStepCost",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
			{"cost", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "setStepCost",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
			{"cost", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "setMaxStepLimit",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
			{"limit", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "setMaxStepLimit",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
			{"limit", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "addDeployer",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "addDeployer",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "removeDeployer",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "removeDeployer",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "getRevision",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getStepPrice",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getStepCost",
		scoreapi.FlagReadOnly, 0,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "getStepCost",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "getStepCosts",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getMaxStepLimit",
		scoreapi.FlagReadOnly, 0,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "getMaxStepLimit",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "getScoreStatus",
		scoreapi.FlagReadOnly, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "getScoreStatus",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "isDeployer",
		scoreapi.FlagReadOnly, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "isDeployer",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "getDeployers",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, Revision7, 0},
	{scoreapi.Method{scoreapi.Function, "setDeployerWhiteListEnabled",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"yn", scoreapi.Bool, nil, nil},
		},
		nil,
	}, Revision7, 0},
	{scoreapi.Method{scoreapi.Function, "getServiceConfig",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
}

func applyStepLimits(c Chain, as state.AccountState) error {
	price := c.Fee
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if price.StepLimit != nil {
		stepLimitsMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepLimit, &stepLimitsMap); err != nil {
			return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to unmarshal. err(%+v)\n", err)
		}
		for _, k := range state.AllStepLimitTypes {
			cost := stepLimitsMap[k]
			if err := stepLimitTypes.Put(k); err != nil {
				return err
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					return scoreresult.InvalidParameterError.Errorf(
						"Failed to parse %s to integer. err(%+v)\n", cost, err)
				}
			}
			if err := stepLimitDB.Set(k, icost); err != nil {
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

func applyStepCosts(c Chain, as state.AccountState) error {
	price := c.Fee
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if price.StepCosts != nil {
		stepTypesMap := make(map[string]string)
		if err := json.Unmarshal(*price.StepCosts, &stepTypesMap); err != nil {
			return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to unmarshal. err(%+v)\n", err)
		}
		for _, k := range state.AllStepTypes {
			cost := stepTypesMap[k]
			if err := stepTypes.Put(k); err != nil {
				return err
			}
			var icost int64
			if cost != "" {
				var err error
				icost, err = strconv.ParseInt(cost, 0, 64)
				if err != nil {
					return err
				}
			}
			if err := stepCostDB.Set(k, icost); err != nil {
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
	defaultMainPRepCount   = 22
	defaultSubPRepCount    = 78
	defaultIRep            = iiss.MonthBlock * iiss.IScoreICXRatio
	defaultRRep            = iiss.MonthBlock * iiss.IScoreICXRatio
	defaultBondRequirement = 5
	defaultLockMin         = defaultTermPeriod * 5
	defaultLockMax         = defaultTermPeriod * 20
	rewardPoint            = 0.7
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
	LockMin         *common.HexInt `json:"lockMin,omitempty"`
	LockMax         *common.HexInt `json:"lockMax,omitempty"`
	RewardFund      struct {
		Iglobal common.HexInt `json:"Iglobal"`
		Iprep   common.HexInt `json:"Iprep"`
		Icps    common.HexInt `json:"Icps"`
		Irelay  common.HexInt `json:"Irelay"`
		Ivoter  common.HexInt `json:"Ivoter"`
	} `json:"rewardFund"`
}

func applyRewardFund(iconConfig *config, s *icstate.State) error {
	rf := &icstate.RewardFund{
		Iglobl: new(big.Int).Set(iconConfig.RewardFund.Iglobal.Value()),
		Iprep:  new(big.Int).Set(iconConfig.RewardFund.Iprep.Value()),
		Icps:   new(big.Int).Set(iconConfig.RewardFund.Icps.Value()),
		Irelay: new(big.Int).Set(iconConfig.RewardFund.Irelay.Value()),
		Ivoter: new(big.Int).Set(iconConfig.RewardFund.Ivoter.Value()),
	}
	if err := s.SetRewardFund(rf); err != nil {
		return err
	}
	return nil
}

type Chain struct {
	Revision                 common.HexInt32 `json:"revision"`
	AuditEnabled             common.HexInt16 `json:"auditEnabled"`
	DeployerWhiteListEnabled common.HexInt16 `json:"deployerWhiteListEnabled"`
	Fee                      struct {
		StepPrice common.HexInt    `json:"stepPrice"`
		StepLimit *json.RawMessage `json:"stepLimit"`
		StepCosts *json.RawMessage `json:"stepCosts"`
	} `json:"fee"`
	ValidatorList      []*common.Address `json:"validatorList"`
	MemberList         []*common.Address `json:"memberList"`
	BlockInterval      *common.HexInt64  `json:"blockInterval"`
	CommitTimeout      *common.HexInt64  `json:"commitTimeout"`
	TimestampThreshold *common.HexInt64  `json:"timestampThreshold"`
	RoundLimitFactor   *common.HexInt64  `json:"roundLimitFactor"`
	MinimizeBlockGen   *common.HexInt16  `json:"minimizeBlockGen"`
	DepositTerm        *common.HexInt64  `json:"depositTerm"`
	DepositIssueRate   *common.HexInt64  `json:"depositIssueRate"`
	FeeSharingEnabled  *common.HexInt16  `json:"feeSharingEnabled"`
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

	chain := Chain{}
	if param != nil {
		if err := json.Unmarshal(param, &chain); err != nil {
			return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to parse parameter for chainScore. err(%+v)\n", err)
		}
	}

	iconConfig := s.loadIconConfig()

	as := s.cc.GetAccountState(state.SystemID)
	revision := DefaultRevision
	if chain.Revision.Value != 0 {
		revision = int(chain.Revision.Value)
		if revision > MaxRevision {
			return scoreresult.IllegalFormatError.Errorf(
				"RevisionIsHigherMax(%d > %d)", revision, MaxRevision)
		} else if revision > LatestRevision {
			s.log.Warnf("Revision in genesis is higher than latest(%d > %d)",
				revision, LatestRevision)
		}
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

	stepPrice := big.NewInt(0)

	price := chain.Fee
	if err := scoredb.NewVarDB(as, state.VarStepPrice).Set(&price.StepPrice.Int); err != nil {
		return err
	}

	switch s.cc.ChainID() {
	case CIDForMainNet:
		// initialize for main network
		s.cc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(s.cc.Database(), nil))
	default:

		validators := make([]module.Validator, len(chain.ValidatorList))
		for i, validator := range chain.ValidatorList {
			validators[i], _ = state.ValidatorFromAddress(validator)
			s.log.Debugf("add validator %d: %v", i, validator)
		}
		if err := s.cc.GetValidatorState().Set(validators); err != nil {
			return errors.CriticalUnknownError.Wrap(err, "FailToSetValidators")
		}

		s.cc.GetExtensionState().Reset(iiss.NewExtensionSnapshot(s.cc.Database(), nil))
	}
	if err := scoredb.NewVarDB(as, state.VarChainID).Set(s.cc.ChainID()); err != nil {
		return err
	}

	if err = applyStepLimits(chain, as); err != nil {
		return err
	}
	if err = applyStepCosts(chain, as); err != nil {
		return err
	}
	if err = applyStepPrice(as, stepPrice); err != nil {
		return err
	}

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
	if err = applyRewardFund(iconConfig, es.State); err != nil {
		return err
	}

	s.handleRevisionChange(as, Revision1, revision)
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
