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
	"fmt"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
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
	flags int
}

const (
	CIDForMainNet         = 0x1
	CIDForTestNet         = 0xca97ec
	StatusIllegalArgument = module.StatusReverted + iota
	StatusNotFound
)

var chainMethods = []*chainMethod{
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
		scoreapi.Function, "getBlockedScores",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.List,
		},
	}, icmodule.Revision9, 0},
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
		scoreapi.Function, "setStepCost",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
			{"cost", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setMaxStepLimit",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
			{"limit", scoreapi.Integer, nil, nil},
		},
		nil,
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getRevision",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStepPrice",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStepCost",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"type", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStepCosts",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getMaxStepLimit",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"contextType", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getScoreStatus",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getScoreDepositInfo",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.Revision4, icmodule.RevisionICON2R0},
	{scoreapi.Method{
		scoreapi.Function, "getServiceConfig",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, 0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getFeeSharingConfig",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.Revision5, 0},
	{scoreapi.Method{
		scoreapi.Function, "getNetworkInfo",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionICON2R0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getIISSInfo",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "setIRep",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getIRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, icmodule.Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "getRRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, icmodule.Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "setStake",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"value", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getStake",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "setDelegation",
		scoreapi.FlagExternal, 0,
		[]scoreapi.Parameter{
			{"delegations", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"address", scoreapi.Address, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, icmodule.RevisionIISS, icmodule.Revision12},
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
	}, icmodule.RevisionFixSetDelegation, 0},
	{scoreapi.Method{
		scoreapi.Function, "getDelegation",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "claimIScore",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "queryIScore",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
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
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "unregisterPRep",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, icmodule.RevisionIISS, 0},
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
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "setGovernanceVariables",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"irep", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionDecentralize, icmodule.Revision8},
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
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getMainPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getSubPReps",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "setBond",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"bonds", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"address", scoreapi.Address, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, icmodule.RevisionEnableBondAPIs, 0},
	{scoreapi.Method{
		scoreapi.Function, "getBond",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionEnableBondAPIs, 0},
	{scoreapi.Method{
		scoreapi.Function, "setBonderList",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"bonderList", scoreapi.ListTypeOf(1, scoreapi.Address), nil, nil},
		},
		nil,
	}, icmodule.RevisionEnableBondAPIs, 0},
	{scoreapi.Method{
		scoreapi.Function, "getBonderList",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionEnableBondAPIs, 0},
	{scoreapi.Method{
		scoreapi.Function, "estimateUnstakeLockPeriod",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRepTerm",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionIISS, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRepStats",
		scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionICON2R0, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRepStatsOf",
		scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionUpdatePRepStats, 0},
	{scoreapi.Method{
		scoreapi.Function, "validateIRep",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"irep", scoreapi.Integer, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, icmodule.Revision9, 0},
	{scoreapi.Method{
		scoreapi.Function, "disqualifyPRep",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, icmodule.Revision6, 0},
	{scoreapi.Method{
		scoreapi.Function, "burn",
		scoreapi.FlagExternal | scoreapi.FlagPayable, 0,
		nil,
		nil,
	}, icmodule.Revision12, 0},
	{scoreapi.Method{
		scoreapi.Function, "validateRewardFund",
		scoreapi.FlagExternal | scoreapi.FlagReadOnly, 1,
		[]scoreapi.Parameter{
			{"iglobal", scoreapi.Integer, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, icmodule.RevisionICON2R0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setRewardFund",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"iglobal", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setRewardFundAllocation",
		scoreapi.FlagExternal, 4,
		[]scoreapi.Parameter{
			{"iprep", scoreapi.Integer, nil, nil},
			{"icps", scoreapi.Integer, nil, nil},
			{"irelay", scoreapi.Integer, nil, nil},
			{"ivoter", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R0, 0},
	{scoreapi.Method{
		scoreapi.Function, "setRewardFundAllocation2",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"values", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"name", scoreapi.String, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "getScoreOwner",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"score", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Address,
		},
	}, icmodule.RevisionEnableSetScoreOwner, 0},
	{scoreapi.Method{
		scoreapi.Function, "setScoreOwner",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"score", scoreapi.Address, nil, nil},
			{"owner", scoreapi.Address, nil, nil},
		},
		nil,
	}, icmodule.RevisionEnableSetScoreOwner, 0},
	{scoreapi.Method{
		scoreapi.Function, "setNetworkScore",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"role", scoreapi.String, nil, nil},
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R2, icmodule.RevisionICON2R3 - 1},
	{scoreapi.Method{
		scoreapi.Function, "setNetworkScore",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"role", scoreapi.String, nil, nil},
			{"address", scoreapi.Address, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R3, 0},
	{scoreapi.Method{
		scoreapi.Function, "getNetworkScores",
		scoreapi.FlagExternal | scoreapi.FlagReadOnly, 0,
		nil,
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionICON2R2, 0},
	{scoreapi.Method{
		scoreapi.Function, "addTimer",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"blockHeight", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R3, 0},
	{scoreapi.Method{
		scoreapi.Function, "removeTimer",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"blockHeight", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R3, 0},
	{scoreapi.Method{
		scoreapi.Function, "penalizeNonvoters",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"preps", scoreapi.ListTypeOf(1, scoreapi.Address), nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R2, 0},
	{scoreapi.Method{
		scoreapi.Function, "setConsistentValidationSlashingRate",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"slashingRate", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R3, icmodule.RevisionPreIISS4 - 1},
	{scoreapi.Method{
		scoreapi.Function, "setNonVoteSlashingRate",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"slashingRate", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionICON2R3, icmodule.RevisionPreIISS4 - 1},
	{scoreapi.Method{
		scoreapi.Function, "setSlashingRates",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"rates", scoreapi.ListTypeOf(1, scoreapi.Struct), nil,
				[]scoreapi.Field{
					{"name", scoreapi.String, nil},
					{"value", scoreapi.Integer, nil},
				},
			},
		},
		nil,
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "getSlashingRates",
		scoreapi.FlagReadOnly, 0,
		[]scoreapi.Parameter{
			{"names", scoreapi.ListTypeOf(1, scoreapi.String), nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Dict,
		},
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "setUseSystemDeposit",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
			{"yn", scoreapi.Bool, nil, nil},
		},
		nil,
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "getUseSystemDeposit",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bool,
		},
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "getBTPNetworkTypeID",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"name", scoreapi.String, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "getPRepNodePublicKey",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
		},
		[]scoreapi.DataType{
			scoreapi.Bytes,
		},
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "setPRepNodePublicKey",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"pubKey", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "registerPRepNodePublicKey",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"address", scoreapi.Address, nil, nil},
			{"pubKey", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, icmodule.RevisionBTP2, 0},
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
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "closeBTPNetwork",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"id", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "sendBTPMessage",
		scoreapi.FlagExternal, 2,
		[]scoreapi.Parameter{
			{"networkId", scoreapi.Integer, nil, nil},
			{"message", scoreapi.Bytes, nil, nil},
		},
		nil,
	}, icmodule.RevisionBTP2, 0},
	{scoreapi.Method{
		scoreapi.Function, "getMinimumBond",
		scoreapi.FlagReadOnly | scoreapi.FlagExternal, 1,
		nil,
		[]scoreapi.DataType{
			scoreapi.Integer,
		},
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "setMinimumBond",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"bond", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "initCommissionRate",
		scoreapi.FlagExternal, 3,
		[]scoreapi.Parameter{
			{"rate", scoreapi.Integer, nil, nil},
			{"maxRate", scoreapi.Integer, nil, nil},
			{"maxChangeRate", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "setCommissionRate",
		scoreapi.FlagExternal, 1,
		[]scoreapi.Parameter{
			{"rate", scoreapi.Integer, nil, nil},
		},
		nil,
	}, icmodule.RevisionPreIISS4, 0},
	{scoreapi.Method{
		scoreapi.Function, "requestUnjail",
		scoreapi.FlagExternal, 0,
		nil,
		nil,
	}, icmodule.RevisionIISS4, 0},
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
		for k, _ := range fee.StepCosts {
			if !state.IsValidStepType(k) {
				return scoreresult.IllegalFormatError.Errorf("InvalidStepType(%s)", k)
			}
		}
		for _, k := range state.AllStepTypes {
			cost, ok := fee.StepCosts[k]
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
	return nil
}

func applyStepPrice(as state.AccountState, price *big.Int) error {
	return scoredb.NewVarDB(as, state.VarStepPrice).Set(price)
}

type config struct {
	TermPeriod                           *common.HexInt `json:"termPeriod"`
	MainPRepCount                        *common.HexInt `json:"mainPRepCount"`
	SubPRepCount                         *common.HexInt `json:"subPRepCount"`
	ExtraMainPRepCount                   *common.HexInt `json:"extraMainPRepCount"`
	Irep                                 *common.HexInt `json:"irep,omitempty"`
	Rrep                                 *common.HexInt `json:"rrep,omitempty"`
	BondRequirement                      *common.HexInt `json:"bondRequirement,omitempty"`
	UnbondingPeriodMultiplier            *common.HexInt `json:"unbondingPeriodMultiplier,omitempty"`
	UnstakeSlotMax                       *common.HexInt `json:"unstakeSlotMax,omitempty"`
	LockMinMultiplier                    *common.HexInt `json:"lockMinMultiplier,omitempty"`
	LockMaxMultiplier                    *common.HexInt `json:"lockMaxMultiplier,omitempty"`
	RewardFund                           rewardFund     `json:"rewardFund"`
	UnbondingMax                         *common.HexInt `json:"unbondingMax"`
	ValidationPenaltyCondition           *common.HexInt `json:"validationPenaltyCondition"`
	ConsistentValidationPenaltyCondition *common.HexInt `json:"consistentValidationPenaltyCondition"`
	ConsistentValidationPenaltyMask      *common.HexInt `json:"consistentValidationPenaltyMask"`
	ConsistentValidationPenaltySlashRate *common.HexInt `json:"consistentValidationPenaltySlashRatio"`
	DelegationSlotMax                    *common.HexInt `json:"delegationSlotMax"`
	NonVotePenaltySlashRate              *common.HexInt `json:"nonVotePenaltySlashRatio"`
}

func (c *config) String() string {
	return fmt.Sprintf(
		"termPeriod=%s mainPReps=%s subPReps=%s extraMainPReps=%s "+
			"irep=%s rrep=%s br=%s upMultiplier=%s unstakeSlotMax=%s unboudingMax=%s "+
			"vpCond=%s cvpCond=%s cvpMask=%s cvpsRatio=%s nvsRatio=%s %s",
		c.TermPeriod,
		c.MainPRepCount,
		c.SubPRepCount,
		c.ExtraMainPRepCount,
		c.Irep,
		c.Rrep,
		c.BondRequirement,
		c.UnbondingPeriodMultiplier,
		c.UnstakeSlotMax,
		c.UnbondingMax,
		c.ValidationPenaltyCondition,
		c.ConsistentValidationPenaltyCondition,
		c.ConsistentValidationPenaltyMask,
		c.ConsistentValidationPenaltySlashRate,
		c.NonVotePenaltySlashRate,
		c.RewardFund,
	)
}

func (c *config) Format(f fmt.State, r rune) {
	switch r {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"Config{termPeriod=%s mainPReps=%s subPReps=%s extraMainPReps=%s "+
					"irep=%s rrep=%s br=%s upMultiplier=%s unstakeSlotMax=%s unboudingMax=%s "+
					"vpCond=%s cvpCond=%s cvpMask=%s cvpsRatio=%s nvsRatio=%s %v}",
				c.TermPeriod,
				c.MainPRepCount,
				c.SubPRepCount,
				c.ExtraMainPRepCount,
				c.Irep,
				c.Rrep,
				c.BondRequirement,
				c.UnbondingPeriodMultiplier,
				c.UnstakeSlotMax,
				c.UnbondingMax,
				c.ValidationPenaltyCondition,
				c.ConsistentValidationPenaltyCondition,
				c.ConsistentValidationPenaltyMask,
				c.ConsistentValidationPenaltySlashRate,
				c.NonVotePenaltySlashRate,
				c.RewardFund,
			)
		} else {
			fmt.Fprintf(
				f,
				"Config{%s %s %s %s %s %s %s %s %s %s %s %s %s %s %v}",
				c.TermPeriod,
				c.MainPRepCount,
				c.SubPRepCount,
				c.Irep,
				c.Rrep,
				c.BondRequirement,
				c.UnbondingPeriodMultiplier,
				c.UnstakeSlotMax,
				c.UnbondingMax,
				c.ValidationPenaltyCondition,
				c.ConsistentValidationPenaltyCondition,
				c.ConsistentValidationPenaltyMask,
				c.ConsistentValidationPenaltySlashRate,
				c.NonVotePenaltySlashRate,
				c.RewardFund,
			)
		}
	case 's':
		fmt.Fprint(f, c.String())
	}
}

type rewardFund struct {
	Iglobal *common.HexInt `json:"Iglobal"`
	Iprep   *common.HexInt `json:"Iprep"`
	Icps    *common.HexInt `json:"Icps"`
	Irelay  *common.HexInt `json:"Irelay"`
	Ivoter  *common.HexInt `json:"Ivoter"`
}

func (r rewardFund) String() string {
	return fmt.Sprintf(
		"Iglobal=%s Iprep=%s Icps=%s Irelay=%s Ivoter=%s",
		r.Iglobal, r.Iprep, r.Icps, r.Irelay, r.Ivoter,
	)
}

func (r rewardFund) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "rewardFund{Iglobal=%s Iprep=%s Icps=%s Irelay=%s Ivoter=%s}",
				r.Iglobal, r.Iprep, r.Icps, r.Irelay, r.Ivoter)
		} else {
			fmt.Fprintf(f, "rewardFund{%s %s %s %s %s}",
				r.Iglobal, r.Iprep, r.Icps, r.Irelay, r.Ivoter)
		}
	case 's':
		fmt.Fprint(f, r.String())
	}
}

func applyRewardFund(iconConfig *config, s *icstate.State) error {
	cfgRewardFund := &iconConfig.RewardFund
	rf, err := icstate.NewSafeRewardFundV1(
		new(big.Int).Set(cfgRewardFund.Iglobal.Value()),
		icmodule.ToRate(cfgRewardFund.Iprep.Int64()),
		icmodule.ToRate(cfgRewardFund.Icps.Int64()),
		icmodule.ToRate(cfgRewardFund.Irelay.Int64()),
		icmodule.ToRate(cfgRewardFund.Ivoter.Int64()),
	)
	if err == nil {
		err = s.SetRewardFund(rf)
	}
	return err
}

type FeeConfig struct {
	StepPrice common.HexInt              `json:"stepPrice"`
	StepLimit map[string]common.HexInt64 `json:"stepLimit,omitempty"`
	StepCosts map[string]common.HexInt64 `json:"stepCosts,omitempty"`
}

type ChainConfig struct {
	Revision           common.HexInt32   `json:"revision"`
	AuditEnabled       common.HexInt16   `json:"auditEnabled"`
	Fee                FeeConfig         `json:"fee"`
	ValidatorList      []*common.Address `json:"validatorList"`
	BlockInterval      *common.HexInt64  `json:"blockInterval"`
	CommitTimeout      *common.HexInt64  `json:"commitTimeout"`
	TimestampThreshold *common.HexInt64  `json:"timestampThreshold"`
	RoundLimitFactor   *common.HexInt64  `json:"roundLimitFactor"`
	DepositTerm        *common.HexInt64  `json:"depositTerm"`
	FeeSharingEnabled  *common.HexInt16  `json:"feeSharingEnabled"`
}

func newIconConfig() *config {
	return &config{
		TermPeriod:                           common.NewHexInt(icmodule.DefaultTermPeriod),
		MainPRepCount:                        common.NewHexInt(icmodule.DefaultMainPRepCount),
		SubPRepCount:                         common.NewHexInt(icmodule.DefaultSubPRepCount),
		ExtraMainPRepCount:                   common.NewHexInt(icmodule.DefaultExtraMainPRepCount),
		Irep:                                 common.NewHexInt(icmodule.DefaultIRep),
		Rrep:                                 common.NewHexInt(icmodule.DefaultRRep),
		BondRequirement:                      common.NewHexInt(icmodule.DefaultBondRequirement),
		LockMinMultiplier:                    common.NewHexInt(icmodule.DefaultLockMinMultiplier),
		LockMaxMultiplier:                    common.NewHexInt(icmodule.DefaultLockMaxMultiplier),
		UnbondingPeriodMultiplier:            common.NewHexInt(icmodule.DefaultUnbondingPeriodMultiplier),
		UnstakeSlotMax:                       common.NewHexInt(icmodule.DefaultUnstakeSlotMax),
		UnbondingMax:                         common.NewHexInt(icmodule.DefaultUnbondingMax),
		ValidationPenaltyCondition:           common.NewHexInt(icmodule.DefaultValidationPenaltyCondition),
		ConsistentValidationPenaltyCondition: common.NewHexInt(icmodule.DefaultConsistentValidationPenaltyCondition),
		ConsistentValidationPenaltyMask:      common.NewHexInt(icmodule.DefaultConsistentValidationPenaltyMask),
		ConsistentValidationPenaltySlashRate: common.NewHexInt(icmodule.DefaultConsistentValidationPenaltySlashRate),
		DelegationSlotMax:                    common.NewHexInt(icmodule.DefaultDelegationSlotMax),
		NonVotePenaltySlashRate:              common.NewHexInt(icmodule.DefaultNonVotePenaltySlashRate),
		RewardFund: rewardFund{
			Iglobal: common.NewHexInt(icmodule.DefaultIglobal),
			Iprep:   common.NewHexInt(icmodule.DefaultIprep),
			Icps:    common.NewHexInt(icmodule.DefaultIcps),
			Irelay:  common.NewHexInt(icmodule.DefaultIrelay),
			Ivoter:  common.NewHexInt(icmodule.DefaultIvoter),
		},
	}
}

func (s *chainScore) loadIconConfig() *config {
	iconConfig := newIconConfig()
	confPath, ok := os.LookupEnv("ICON_CONFIG")
	if !ok {
		confPath = icmodule.ConfigFile
	}
	f, err := os.Open(confPath)
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
	if s.from != nil {
		return scoreresult.AccessDeniedError.New("AccessDeniedToInstallChainSCORE")
	}

	as := s.cc.GetAccountState(state.SystemID)

	var feeConfig *FeeConfig
	var systemConfig int
	var revision int
	var validators []module.Validator
	var handlers []contract.ContractHandler
	blockInterval := int64(2000)
	roundLimitFactor := int64(3)

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
		systemConfig = state.SysConfigAudit | state.SysConfigScorePackageValidator
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

	case CIDForTestNet:
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
		systemConfig = state.SysConfigScorePackageValidator
		revision = icmodule.Revision1
		governance, err := ioutil.ReadFile("icon_governance.zip")
		if err != nil || len(governance) == 0 {
			return transaction.InvalidGenesisError.Wrap(err, "FailOnGovernance")
		}
		params := json.RawMessage("{}")
		handler := contract.NewDeployHandlerForPreInstall(
			common.MustNewAddressFromString("hx6e1dd0d4432620778b54b2bbc21ac3df961adf89"),
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

		if chainConfig.AuditEnabled.Value != 0 {
			systemConfig |= state.SysConfigAudit
		}
		if chainConfig.FeeSharingEnabled != nil {
			if chainConfig.FeeSharingEnabled.Value != 0 {
				systemConfig |= state.SysConfigFeeSharing
			}
		}

		if chainConfig.BlockInterval != nil {
			blockInterval = chainConfig.BlockInterval.Value
		}
		if chainConfig.RoundLimitFactor != nil {
			roundLimitFactor = chainConfig.RoundLimitFactor.Value
		}

		if chainConfig.DepositTerm != nil {
			if chainConfig.DepositTerm.Value < 0 {
				return scoreresult.IllegalFormatError.Errorf("InvalidDepositTerm(%s)", chainConfig.DepositTerm)
			}
			if err := scoredb.NewVarDB(as, state.VarDepositTerm).Set(chainConfig.DepositTerm.Value); err != nil {
				return err
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

	// set block interval 2 seconds
	if err := scoredb.NewVarDB(as, state.VarBlockInterval).Set(blockInterval); err != nil {
		return err
	}

	// skip transaction
	if err := scoredb.NewVarDB(as, state.VarRoundLimitFactor).Set(roundLimitFactor); err != nil {
		return err
	}

	if err := scoredb.NewVarDB(as, state.VarChainID).Set(s.cc.ChainID()); err != nil {
		return err
	}

	if feeConfig != nil {
		systemConfig |= state.SysConfigFee
		if err := applyStepLimits(feeConfig, as); err != nil {
			return err
		}
		if err := applyStepCosts(feeConfig, as); err != nil {
			return err
		}
		if err := applyStepPrice(as, &feeConfig.StepPrice.Int); err != nil {
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

	for _, handler := range handlers {
		status, _, _, _ := s.cc.Call(handler, s.cc.StepAvailable())
		if status != nil {
			return transaction.InvalidGenesisError.Wrap(status,
				"FAIL to install initial governance score.")
		}
	}

	if err := s.handleRevisionChange(icmodule.Revision1, revision); err != nil {
		return err
	}

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
			if err := s.cc.ApplyCallSteps(); err != nil {
				return err
			}
		}
		return scoreresult.New(module.StatusAccessDenied, "NoPermission")
	}
	return nil
}

const (
	SysNoCharge = 1 << iota
	IISSDisabled
	BasicHidden
)

func newChainScore(cc contract.CallContext, from module.Address, value *big.Int) (contract.SystemScore, error) {
	revision := cc.Revision().Value()
	fromGov := cc.Governance().Equal(from)
	flags := 0
	if from != nil && from.IsContract() {
		// Inter-call case
		if revision < icmodule.RevisionSystemSCORE {
			flags |= IISSDisabled
		}
		if revision < icmodule.RevisionIISS && !fromGov {
			flags |= BasicHidden
		}
	} else {
		// External-call case
		if revision < icmodule.RevisionICON2R0 {
			flags |= SysNoCharge
			flags |= BasicHidden
		}
	}
	return &chainScore{
			cc:    cc,
			from:  from,
			value: value,
			log:   icutils.NewIconLogger(cc.Logger()),
			gov:   fromGov,
			flags: flags,
		},
		nil
}
