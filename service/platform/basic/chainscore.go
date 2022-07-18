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

package basic

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
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
type ChainScore struct {
	from  module.Address
	value *big.Int
	gov   bool
	cc    contract.CallContext
	log   log.Logger
}

func NewChainScore(cc contract.CallContext, from module.Address, value *big.Int) (contract.SystemScore, error) {
	return &ChainScore{from, value, cc.Governance().Equal(from), cc, cc.Logger()}, nil
}

const (
	StatusIllegalArgument = module.StatusReverted + iota
	StatusNotFound
)

var chainMethods = []*chainMethod{
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
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "rejectScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"txHash", scoreapi.Bytes, nil, nil},
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
	{scoreapi.Method{scoreapi.Function, "grantValidator",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "grantValidator",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "revokeValidator",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "revokeValidator",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "addMember",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "addMember",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "removeMember",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "removeMember",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
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
	{scoreapi.Method{scoreapi.Function, "addLicense",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"contentId", scoreapi.String, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "addLicense",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"contentId", scoreapi.String, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{scoreapi.Function, "removeLicense",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"contentId", scoreapi.String, nil, nil},
		},
		nil,
	}, 0, Revision4},
	{scoreapi.Method{scoreapi.Function, "removeLicense",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"contentId", scoreapi.String, nil, nil},
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
	{scoreapi.Method{scoreapi.Function, "getMembers",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0},
	{scoreapi.Method{scoreapi.Function, "getValidators",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, 0, 0},
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
	{scoreapi.Method{
		scoreapi.Function, "setTimestampThreshold",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"threshold", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "getTimestampThreshold",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "setRoundLimitFactor",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"factor", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "getRoundLimitFactor",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "setMinimizeBlockGen",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"yn", scoreapi.Bool, nil, nil},
		},
		nil,
	}, Revision8, 0},
	{scoreapi.Method{
		scoreapi.Function, "getMinimizeBlockGen",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, Revision8, 0},
	{scoreapi.Method{
		scoreapi.Function, "setUseSystemDeposit",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
			{"yn", scoreapi.Bool, nil, nil},
		},
		nil,
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getUseSystemDeposit",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getSystemDepositUsage",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getBTPNetworkTypeID",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"name", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getBTPPublicKey",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
			{"name", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bytes,
		},
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "openBTPNetwork",
		scoreapi.FlagExternal, 3,
		[]scoreapi.Parameter{
			{"networkTypeName", scoreapi.String, nil, nil},
			{"name", scoreapi.String, nil, nil},
			{"owner", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "closeBTPNetwork",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"id", scoreapi.Integer, nil, nil},
		},
		nil,
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "sendBTPMessage",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"networkId", scoreapi.Integer, nil, nil},
			{"message", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "setBTPPublicKey",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"name", scoreapi.String, nil, nil},
			{"pubKey", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, Revision9, 0},
}

func (s *ChainScore) GetAPI() *scoreapi.Info {
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

type chain struct {
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

func (s *ChainScore) Install(param []byte) error {
	chain := chain{}
	if param != nil {
		if err := json.Unmarshal(param, &chain); err != nil {
			return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to parse parameter for chainScore. err(%+v)\n", err)
		}
	}

	as := s.cc.GetAccountState(state.SystemID)
	revision := int(DefaultRevision)
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

	confValue := 0
	if chain.AuditEnabled.Value != 0 {
		confValue |= state.SysConfigAudit
	}
	if chain.DeployerWhiteListEnabled.Value != 0 {
		confValue |= state.SysConfigDeployerWhiteList
	}
	if len(chain.MemberList) > 0 {
		confValue |= state.SysConfigMembership
	}
	if chain.FeeSharingEnabled != nil {
		if chain.FeeSharingEnabled.Value != 0 {
			confValue |= state.SysConfigFeeSharing
		}
	}
	if err := scoredb.NewVarDB(as, state.VarServiceConfig).Set(confValue); err != nil {
		return err
	}

	if chain.BlockInterval != nil {
		blockInterval := chain.BlockInterval.Value
		if err := scoredb.NewVarDB(as, state.VarBlockInterval).Set(blockInterval); err != nil {
			return err
		}
	}

	if chain.CommitTimeout != nil {
		timeout := chain.CommitTimeout.Value
		if err := scoredb.NewVarDB(as, state.VarCommitTimeout).Set(timeout); err != nil {
			return err
		}
	}

	if chain.TimestampThreshold != nil {
		tsThreshold := chain.TimestampThreshold.Value
		if err := scoredb.NewVarDB(as, state.VarTimestampThreshold).Set(tsThreshold); err != nil {
			return err
		}
	}

	if chain.RoundLimitFactor != nil {
		factor := chain.RoundLimitFactor.Value
		if err := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Set(factor); err != nil {
			return err
		}
	}

	if chain.MinimizeBlockGen != nil {
		yn := chain.MinimizeBlockGen.Value != 0
		if err := scoredb.NewVarDB(as, state.VarMinimizeBlockGen).Set(yn); err != nil {
			return err
		}
	}

	if chain.DepositTerm != nil {
		if chain.DepositTerm.Value < 0 {
			return scoreresult.IllegalFormatError.Errorf("InvalidDepositTerm(%s)", chain.DepositTerm)
		}
		if err := scoredb.NewVarDB(as, state.VarDepositTerm).Set(chain.DepositTerm.Value); err != nil {
			return err
		}
	}

	if chain.DepositIssueRate != nil {
		if chain.DepositIssueRate.Value < 0 {
			return scoreresult.IllegalFormatError.Errorf("InvalidDepositIssueRate(%s)", chain.DepositIssueRate)
		}
		if err := scoredb.NewVarDB(as, state.VarDepositIssueRate).Set(chain.DepositIssueRate.Value); err != nil {
			return err
		}
	}

	price := chain.Fee
	if err := scoredb.NewVarDB(as, state.VarStepPrice).Set(&price.StepPrice.Int); err != nil {
		return err
	}
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

	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if price.StepCosts != nil {
		stepTypesMap := make(map[string]common.HexInt64)
		if err := json.Unmarshal(*price.StepCosts, &stepTypesMap); err != nil {
			return scoreresult.Errorf(module.StatusIllegalFormat, "Failed to unmarshal. err(%+v)\n", err)
		}
		for k, _ := range stepTypesMap {
			if !state.IsValidStepType(k) {
				return scoreresult.IllegalFormatError.Errorf("InvalidStepType(%s)", k)
			}
		}
		for _, k := range state.AllStepTypes {
			cost, ok := stepTypesMap[k]
			if !ok {
				continue
			}
			if err := stepTypes.Put(k); err != nil {
				return err
			}
			if err := stepCostDB.Set(k, cost.Value); err != nil {
				return err
			}
		}
	} else {
		for _, k := range state.InitialStepTypes {
			if err := stepTypes.Put(k); err != nil {
				return err
			}
			if err := stepCostDB.Set(k, 0); err != nil {
				return err
			}
		}
	}
	validators := make([]module.Validator, len(chain.ValidatorList))
	for i, validator := range chain.ValidatorList {
		validators[i], _ = state.ValidatorFromAddress(validator)
	}
	if err := s.cc.GetValidatorState().Set(validators); err != nil {
		return errors.CriticalUnknownError.Wrap(err, "FailToSetValidators")
	}

	if len(chain.MemberList) > 0 {
		members := scoredb.NewArrayDB(as, state.VarMembers)

		vs := s.cc.GetValidatorState()
		vc := 0
		m := make(map[string]bool)
		for i, member := range chain.MemberList {
			if member == nil {
				return errors.IllegalArgumentError.Errorf(
					"Member[%d] is null", i)
			}
			if member.IsContract() {
				return errors.IllegalArgumentError.Errorf(
					"Member must be EOA(%s)", member.String())
			}
			mn := member.String()
			if _, ok := m[mn]; ok {
				return errors.IllegalArgumentError.Errorf(
					"Duplicated Member(%s)", member.String())
			}
			m[mn] = true
			if idx := vs.IndexOf(member); idx >= 0 {
				vc += 1
			}
			members.Put(member)
		}
		if vc != vs.Len() {
			return errors.IllegalArgumentError.New(
				"All Validators must be included in the members")
		}
	}
	s.handleRevisionChange(as, Revision1, revision)
	return nil
}

func (s *ChainScore) Update(param []byte) error {
	log.Panicf("Implement me")
	return nil
}

func (s *ChainScore) tryChargeCall() error {
	if !s.gov {
		if err := s.cc.ApplyCallSteps(); err != nil {
			return err
		}
	}
	return nil
}

func (s *ChainScore) checkGovernance(charge bool) error {
	if !s.gov {
		if charge {
			if err := s.cc.ApplyCallSteps(); err != nil {
				return err
			}
		}
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	return nil
}

// Destroy : Allowed from score owner
func (s *ChainScore) Ex_disableScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.New(StatusNotFound, "NoContract")
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(true)
	return nil
}

func (s *ChainScore) Ex_enableScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() == false {
		return scoreresult.New(StatusNotFound, "NoContract")
	}
	if as.IsContractOwner(s.from) == false {
		return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
	}
	as.SetDisable(false)
	return nil
}

func (s *ChainScore) fromGovernance() bool {
	return s.cc.Governance().Equal(s.from)
}

func (s *ChainScore) handleRevisionChange(as state.AccountState, r1, r2 int) error {
	if r1 >= r2 {
		return nil
	}
	if r1 < Revision7 && r2 >= Revision7 {
		if err := scoredb.NewVarDB(as, state.VarChainID).Set(s.cc.ChainID()); err != nil {
			return err
		}
	}
	return nil
}

// Governance functions : Functions which can be called by governance SCORE.
func (s *ChainScore) Ex_setRevision(code *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if MaxRevision < code.Int64() {
		return scoreresult.Errorf(StatusIllegalArgument,
			"IllegalArgument(max=%#x,new=%s)", MaxRevision, code)
	}

	as := s.cc.GetAccountState(state.SystemID)
	r := scoredb.NewVarDB(as, state.VarRevision).Int64()
	if code.Int64() < r {
		return scoreresult.Errorf(StatusIllegalArgument,
			"IllegalArgument(current=%#x,new=%s)", r, code)
	}

	if err := scoredb.NewVarDB(as, state.VarRevision).Set(code); err != nil {
		return err
	}
	if err := s.handleRevisionChange(as, int(r), int(code.Int64())); err != nil {
		return nil
	}
	apiInfo := s.GetAPI()
	if err := contract.CheckMethod(s, apiInfo); err != nil {
		return scoreresult.Wrap(err, module.StatusIllegalFormat, "InvalidChainScoreImplementation")
	}
	as.MigrateForRevision(s.cc.ToRevision(int(code.Int64())))
	as.SetAPIInfo(apiInfo)
	return nil
}

func (s *ChainScore) Ex_acceptScore(txHash []byte) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if len(txHash) == 0 {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	auditTxHash := s.cc.TransactionID()

	ch := contract.NewCommonHandler(s.from, state.SystemAddress, big.NewInt(0), false, s.log)
	ah := contract.NewAcceptHandler(ch, txHash, auditTxHash)
	status, steps, _, _ := s.cc.Call(ah, s.cc.StepAvailable())
	s.cc.DeductSteps(steps)
	return status
}

func (s *ChainScore) Ex_rejectScore(txHash []byte) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if len(txHash) == 0 {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}

	sysAs := s.cc.GetAccountState(state.SystemID)
	h2a := scoredb.NewDictDB(sysAs, state.VarTxHashToAddress, 1)
	scoreAddr := h2a.Get(txHash).Address()
	if scoreAddr == nil {
		return scoreresult.Errorf(StatusNotFound, "NoPendingTx")
	}
	scoreAs := s.cc.GetAccountState(scoreAddr.ID())
	// NOTE : cannot change from reject to accept state because data with address mapped txHash is deleted from DB
	auditTxHash := s.cc.TransactionID()
	if err := h2a.Delete(txHash); err != nil {
		return err
	}
	return scoreAs.RejectContract(txHash, auditTxHash)
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_blockScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == false && as.IsContract() {
		as.SetBlock(true)
	}
	return nil
}

// Governance score would check the verification of the address
func (s *ChainScore) Ex_unblockScore(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsBlocked() == true && as.IsContract() {
		as.SetBlock(false)
	}
	return nil
}

func (s *ChainScore) Ex_setStepPrice(price *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

func (s *ChainScore) Ex_setStepCost(costType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if stepCostDB.Get(costType) == nil {
		stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
		if err := stepTypes.Put(costType); err != nil {
			return err
		}
	}
	return stepCostDB.Set(costType, cost)
}

func (s *ChainScore) Ex_setMaxStepLimit(contextType string, cost *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if stepLimitDB.Get(contextType) == nil {
		stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
		if err := stepLimitTypes.Put(contextType); err != nil {
			return err
		}
	}
	return stepLimitDB.Set(contextType, cost)
}

func (s *ChainScore) Ex_grantValidator(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	if address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "address should be EOA")
	}

	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		if err = bs.CheckPublicKey(s.newBTPContext(), s.from); err != nil {
			return err
		}
	}

	if s.cc.MembershipEnabled() {
		found := false
		as := s.cc.GetAccountState(state.SystemID)
		db := scoredb.NewArrayDB(as, state.VarMembers)
		for i := 0; i < db.Size(); i++ {
			if db.Get(i).Address().Equal(address) {
				found = true
				break
			}
		}
		if !found {
			return scoreresult.New(StatusIllegalArgument, "NotInMembers")
		}
	}

	if v, err := state.ValidatorFromAddress(address); err == nil {
		return s.cc.GetValidatorState().Add(v)
	} else {
		return err
	}
}

func (s *ChainScore) Ex_revokeValidator(address module.Address) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if address == nil {
		return scoreresult.ErrInvalidParameter
	}
	if err := s.checkGovernance(false); err != nil {
		return err
	}
	if address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "AddressIsContract")
	}
	if v, err := state.ValidatorFromAddress(address); err == nil {
		vl := s.cc.GetValidatorState()
		if ok := vl.Remove(v); !ok {
			return scoreresult.New(StatusNotFound, "NotFound")
		}
		if vl.Len() == 0 {
			return scoreresult.New(StatusIllegalArgument, "OnlyValidator")
		}
		return nil
	} else {
		return err
	}
}

func (s *ChainScore) Ex_getValidators() ([]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	vs := s.cc.GetValidatorState()
	validators := make([]interface{}, vs.Len())
	for i := 0; i < vs.Len(); i++ {
		if v, ok := vs.Get(i); ok {
			validators[i] = v.Address()
		} else {
			return nil, errors.CriticalUnknownError.New("Unexpected access failure")
		}
	}
	return validators, nil
}

func (s *ChainScore) Ex_addMember(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "AddressIsContract")
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return nil
		}
	}
	return db.Put(address)
}

func (s *ChainScore) Ex_removeMember(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "AddressIsContract")
	}

	// If membership system is on, first check if the member is not a validator
	if s.cc.MembershipEnabled() {
		if s.cc.GetValidatorState().IndexOf(address) >= 0 {
			return scoreresult.New(StatusIllegalArgument, "RevokeValidatorFirst")
		}
	}

	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size() { // addr is not rAddr
				if err := db.Set(i, rAddr); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

func (s *ChainScore) Ex_addDeployer(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return nil
		}
	}
	return db.Put(address)
}

func (s *ChainScore) Ex_removeDeployer(address module.Address) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			rAddr := db.Pop().Address()
			if i < db.Size() { // addr is not rAddr
				if err := db.Set(i, rAddr); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

func (s *ChainScore) Ex_setTimestampThreshold(threshold *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewVarDB(as, state.VarTimestampThreshold)
	return db.Set(threshold)
}

func (s *ChainScore) Ex_getTimestampThreshold() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewVarDB(as, state.VarTimestampThreshold)
	return db.Int64(), nil
}

func (s *ChainScore) Ex_addLicense(contentId string) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarLicenses)
	for i := 0; i < db.Size(); i++ {
		if strings.Compare(db.Get(i).String(), contentId) == 0 {
			return nil
		}
	}
	return db.Put(contentId)
}

func (s *ChainScore) Ex_removeLicense(contentId string) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarLicenses)
	for i := 0; i < db.Size(); i++ {
		if strings.Compare(db.Get(i).String(), contentId) == 0 {
			id := db.Pop().String()
			if i < db.Size() { // id is not contentId
				if err := db.Set(i, id); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

// User calls icx_call : Functions which can be called by anyone.
func (s *ChainScore) Ex_getRevision() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRevision).Int64(), nil
}

func (s *ChainScore) Ex_getStepPrice() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarStepPrice).Int64(), nil
}

func (s *ChainScore) Ex_getStepCost(t string) (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	if v := stepCostDB.Get(t); v != nil {
		return v.Int64(), nil
	}
	return 0, nil
}

func (s *ChainScore) Ex_getStepCosts() (map[string]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)

	stepCosts := make(map[string]interface{})
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	tcount := stepTypes.Size()
	for i := 0; i < tcount; i++ {
		tname := stepTypes.Get(i).String()
		stepCosts[tname] = stepCostDB.Get(tname).Int64()
	}
	return stepCosts, nil
}

func (s *ChainScore) Ex_getMaxStepLimit(contextType string) (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	if v := stepLimitDB.Get(contextType); v != nil {
		return v.Int64(), nil
	}
	return 0, nil
}

func (s *ChainScore) Ex_getScoreStatus(address module.Address) (map[string]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	if !address.IsContract() {
		return nil, scoreresult.New(StatusIllegalArgument, "address must be contract")
	}
	as := s.cc.GetAccountState(address.ID())
	if as == nil || !as.IsContract() {
		return nil, scoreresult.New(StatusNotFound, "ContractNotFound")
	}
	scoreStatus := make(map[string]interface{})

	scoreStatus["owner"] = as.ContractOwner()

	if cur := as.Contract(); cur != nil {
		curContract := make(map[string]interface{})
		curContract["status"] = cur.Status().String()
		curContract["deployTxHash"] = fmt.Sprintf("%#x", cur.DeployTxHash())
		curContract["auditTxHash"] = fmt.Sprintf("%#x", cur.AuditTxHash())
		scoreStatus["current"] = curContract
	}

	if next := as.NextContract(); next != nil {
		nextContract := make(map[string]interface{})
		nextContract["status"] = next.Status().String()
		nextContract["deployTxHash"] = fmt.Sprintf("%#x", next.DeployTxHash())
		scoreStatus["next"] = nextContract
	}

	if di, err := as.GetDepositInfo(s.cc, module.JSONVersion3); err != nil {
		return nil, scoreresult.New(module.StatusUnknownFailure, "FailOnDepositInfo")
	} else if di != nil {
		scoreStatus["depositInfo"] = di
	}

	// blocked
	if as.IsBlocked() == true {
		scoreStatus["blocked"] = "0x1"
	} else {
		scoreStatus["blocked"] = "0x0"
	}

	// disabled
	if as.IsDisabled() == true {
		scoreStatus["disabled"] = "0x1"
	} else {
		scoreStatus["disabled"] = "0x0"
	}
	return scoreStatus, nil
}

func (s *ChainScore) Ex_isDeployer(address module.Address) (int, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	for i := 0; i < db.Size(); i++ {
		if db.Get(i).Address().Equal(address) == true {
			return 1, nil
		}
	}
	return 0, nil
}

func (s *ChainScore) Ex_getDeployers() ([]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarDeployers)
	deployers := make([]interface{}, db.Size())
	for i := 0; i < db.Size(); i++ {
		deployers[i] = db.Get(i).Address()
	}
	return deployers, nil
}

func (s *ChainScore) Ex_setDeployerWhiteListEnabled(yn bool) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	confValue := scoredb.NewVarDB(as, state.VarServiceConfig).Int64()
	if yn {
		confValue |= state.SysConfigDeployerWhiteList
	} else {
		confValue &^= state.SysConfigDeployerWhiteList
	}
	return scoredb.NewVarDB(as, state.VarServiceConfig).Set(confValue)
}

func (s *ChainScore) Ex_getServiceConfig() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarServiceConfig).Int64(), nil
}

func (s *ChainScore) Ex_getMembers() ([]interface{}, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	db := scoredb.NewArrayDB(as, state.VarMembers)
	members := make([]interface{}, db.Size())
	for i := 0; i < db.Size(); i++ {
		members[i] = db.Get(i).Address()
	}
	return members, nil
}

func (s *ChainScore) Ex_getRoundLimitFactor() (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	return scoredb.NewVarDB(as, state.VarRoundLimitFactor).Int64(), nil
}

func (s *ChainScore) Ex_setRoundLimitFactor(f *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	if f.Sign() < 0 {
		return scoreresult.New(StatusIllegalArgument, "IllegalArgument")
	}
	as := s.cc.GetAccountState(state.SystemID)
	factor := scoredb.NewVarDB(as, state.VarRoundLimitFactor)
	return factor.Set(f)
}

func (s *ChainScore) Ex_getMinimizeBlockGen() (bool, error) {
	if err := s.tryChargeCall(); err != nil {
		return false, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	mbg := scoredb.NewVarDB(as, state.VarMinimizeBlockGen)
	return mbg.Bool(), nil
}

func (s *ChainScore) Ex_setMinimizeBlockGen(b bool) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(state.SystemID)
	mbg := scoredb.NewVarDB(as, state.VarMinimizeBlockGen)
	return mbg.Set(b)
}

func (s *ChainScore) Ex_setUseSystemDeposit(address module.Address, yn bool) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() != address.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "InvalidPrefixForAddress")
	}
	if !as.IsContract() {
		return scoreresult.New(StatusIllegalArgument, "NotContract")
	}
	return as.SetUseSystemDeposit(yn)
}

func (s *ChainScore) Ex_getUseSystemDeposit(address module.Address) (bool, error) {
	if err := s.tryChargeCall(); err != nil {
		return false, err
	}
	as := s.cc.GetAccountState(address.ID())
	if as.IsContract() != address.IsContract() {
		return false, scoreresult.New(StatusIllegalArgument, "InvalidPrefixForAddress")
	}
	return as.UseSystemDeposit(), nil
}

func (s *ChainScore) Ex_getSystemDepositUsage() (*big.Int, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	as := s.cc.GetAccountState(state.SystemID)
	usage := scoredb.NewVarDB(as, state.VarSystemDepositUsage).BigInt()
	if usage == nil {
		usage = new(big.Int)
	}
	return usage, nil
}

func (s *ChainScore) Ex_getBTPNetworkTypeID(name string) (int64, error) {
	if err := s.tryChargeCall(); err != nil {
		return 0, err
	}
	return s.newBTPContext().GetNetworkTypeIDByName(name), nil
}

func (s *ChainScore) Ex_getBTPPublicKey(address module.Address, name string) ([]byte, error) {
	if err := s.tryChargeCall(); err != nil {
		return nil, err
	}
	pubKey, _ := s.newBTPContext().GetPublicKey(address, name, true)
	return pubKey, nil
}

func (s *ChainScore) Ex_openBTPNetwork(networkTypeName string, name string, owner module.Address) (int64, error) {
	if err := s.checkGovernance(true); err != nil {
		return 0, err
	}
	if bs, err := s.getBTPState(); err != nil {
		return 0, err
	} else {
		bc := s.newBTPContext()
		ntActivated := false
		if bc.GetNetworkTypeIDByName(networkTypeName) <= 0 {
			ntActivated = true
		}
		ntid, nid, err := bs.OpenNetwork(bc, networkTypeName, name, owner)
		if err != nil {
			return 0, err
		}
		if ntActivated {
			s.cc.OnEvent(state.SystemAddress,
				[][]byte{
					[]byte("BTPNetworkTypeActivated(str,int)"),
					[]byte(networkTypeName),
					intconv.Int64ToBytes(ntid),
				},
				nil,
			)
		}
		s.cc.OnEvent(state.SystemAddress,
			[][]byte{
				[]byte("BTPNetworkOpened(int,int)"),
				intconv.Int64ToBytes(ntid),
				intconv.Int64ToBytes(nid),
			},
			nil,
		)
		return nid, nil
	}
}

func (s *ChainScore) Ex_closeBTPNetwork(id *common.HexInt) error {
	if err := s.checkGovernance(true); err != nil {
		return err
	}
	nid := id.Int64()
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		if ntid, err := bs.CloseNetwork(s.newBTPContext(), nid); err != nil {
			return err
		} else {
			s.cc.OnEvent(state.SystemAddress,
				[][]byte{
					[]byte("BTPNetworkClosed(int,int)"),
					intconv.Int64ToBytes(ntid),
					intconv.Int64ToBytes(nid),
				},
				nil,
			)
		}
	}
	return nil
}

func (s *ChainScore) Ex_sendBTPMessage(networkId *common.HexInt, message []byte) error {
	if err := s.tryChargeCall(); err != nil {
		return err
	}
	if len(message) == 0 {
		return scoreresult.ErrInvalidParameter
	}
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		nid := networkId.Int64()
		if err = bs.HandleMessage(s.newBTPContext(), s.from, nid); err != nil {
			return err
		}
		s.cc.OnBTPMessage(nid, message)
		return nil
	}
}

func (s *ChainScore) Ex_setBTPPublicKey(name string, pubKey []byte) error {
	if s.from.IsContract() {
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	if bs, err := s.getBTPState(); err != nil {
		return err
	} else {
		if !bs.IsNetworkTypeUID(name) && !bs.IsDSAName(name) {
			return scoreresult.InvalidParameterError.Errorf("Invalid name %s", name)
		}
		if err = bs.SetPublicKey(s.newBTPContext(), s.from, name, pubKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *ChainScore) getBTPState() (*state.BTPStateImpl, error) {
	btpState := s.cc.GetBTPState()
	if btpState == nil {
		return nil, scoreresult.UnknownFailureError.Errorf("BTP state is nil")
	}
	return btpState.(*state.BTPStateImpl), nil
}

func (s *ChainScore) newBTPContext() state.BTPContext {
	store := s.cc.GetAccountState(state.SystemID)
	return state.NewBTPContext(s.cc, store)
}
