/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iiss

import (
	"bytes"
	"encoding/json"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"math/big"
)

type IssuePRepJSON struct {
	IRep            *common.HexInt `json:"irep"`
	RRep            *common.HexInt `json:"rrep"`
	TotalDelegation *common.HexInt `json:"totalDelegation"`
	Value           *common.HexInt `json:"value"`
}

func parseIssuePRepData(data []byte) (*IssuePRepJSON, error) {
	jso := new(IssuePRepJSON)
	jd := json.NewDecoder(bytes.NewBuffer(data))
	jd.DisallowUnknownFields()
	if err := jd.Decode(jso); err != nil {
		return nil, err
	}
	return jso, nil
}

func (i *IssuePRepJSON) equal(i2 *IssuePRepJSON) bool {
	return i.IRep.Cmp(i2.IRep.Value()) == 0 &&
		i.RRep.Cmp(i2.RRep.Value()) == 0 &&
		i.TotalDelegation.Cmp(i2.TotalDelegation.Value()) == 0 &&
		i.Value.Cmp(i2.Value.Value()) == 0
}

type IssueResultJSON struct {
	ByFee           *common.HexInt `json:"coveredByFee"`
	ByOverIssuedICX *common.HexInt `json:"coveredByOverIssuedICX"`
	Issue           *common.HexInt `json:"issue"`
}

func parseIssueResultData(data []byte) (*IssueResultJSON, error) {
	jso := new(IssueResultJSON)
	jd := json.NewDecoder(bytes.NewBuffer(data))
	jd.DisallowUnknownFields()
	if err := jd.Decode(jso); err != nil {
		return nil, err
	}
	return jso, nil
}

func (i *IssueResultJSON) equal(i2 *IssueResultJSON) bool {
	return i.ByFee.Cmp(i2.ByFee.Value()) == 0 &&
		i.ByOverIssuedICX.Cmp(i2.ByOverIssuedICX.Value()) == 0 &&
		i.Issue.Cmp(i2.Issue.Value()) == 0
}

func RegulateIssueInfo(es *ExtensionStateImpl, iScore *big.Int) {
	issue, _ := es.State.GetIssue()
	issue = regulateIssueInfo(issue, iScore)
	es.State.SetIssue(issue)
}

// regulateIssueInfo regulate icx issue amount with previous period data.
func regulateIssueInfo(issue *icstate.Issue, iScore *big.Int) *icstate.Issue {
	icx, remains := new(big.Int).DivMod(iScore, BigIntIScoreICXRatio, new(big.Int))
	overIssued := new(big.Int).Sub(issue.PrevTotalReward, icx)
	issue.OverIssued.Add(issue.OverIssued, overIssued)
	issue.IScoreRemains.Add(issue.IScoreRemains, remains)
	if BigIntIScoreICXRatio.Cmp(issue.IScoreRemains) < 0 {
		issue.OverIssued.Sub(issue.OverIssued, intconv.BigIntOne)
		issue.IScoreRemains.Sub(issue.IScoreRemains, BigIntIScoreICXRatio)
	}
	issue.PrevTotalReward.Set(issue.TotalReward)
	issue.TotalReward.SetInt64(0)

	return issue
}

// calcRewardPerBlock calculate reward per block
func calcRewardPerBlock(
	irep *big.Int,
	rrep *big.Int,
	mainPRepCount *big.Int,
	pRepCount *big.Int,
	totalDelegated *big.Int,
) *big.Int {
	beta1 := new(big.Int)
	beta2 := new(big.Int)
	beta3 := new(big.Int)

	beta1.Mul(irep, mainPRepCount)
	beta1.Div(beta1, new(big.Int).SetInt64(MonthBlock))
	beta1.Div(beta1, BigIntTwo)

	if totalDelegated.Sign() != 0 {
		beta2.Mul(irep, pRepCount)
		beta2.Div(beta2, new(big.Int).SetInt64(MonthBlock))
		beta2.Div(beta2, BigIntTwo)

		beta3.Mul(rrep, totalDelegated)
		beta3.Div(beta3, new(big.Int).SetInt64(YearBlock))
	}

	reward := new(big.Int).Add(beta1, beta2)
	reward.Add(reward, beta3)

	return reward
}

func calcIssueAmount(reward *big.Int, i *icstate.Issue) (overIssued *big.Int, issue *big.Int) {
	issue = new(big.Int).Sub(reward, i.PrevBlockFee)
	overIssued = new(big.Int).Set(i.OverIssued)
	if issue.Cmp(overIssued) >= 0 {
		issue.Sub(issue, overIssued)
	} else {
		overIssued.Set(issue)
		issue.SetInt64(0)
	}
	return
}

//GetIssueData return issue information for base TX
func GetIssueData(es *ExtensionStateImpl) (*IssuePRepJSON, *IssueResultJSON) {
	// TODO read values from Term
	irep := icstate.GetIRep(es.State)
	rrep := icstate.GetRRep(es.State)
	mainPRepCount := icstate.GetMainPRepCount(es.State)
	pRepCount := icstate.GetPRepCount(es.State)
	totalDelegated := es.GetTotalDelegated()
	// TODO check condition with API from Term
	//if !isDecentralized {
	//	irep.SetInt64(0)
	//}
	reward := calcRewardPerBlock(
		irep,
		rrep,
		new(big.Int).SetInt64(mainPRepCount),
		new(big.Int).SetInt64(pRepCount),
		totalDelegated,
	)
	prep := &IssuePRepJSON{
		IRep:            bigInt2HexInt(irep),
		RRep:            bigInt2HexInt(rrep),
		TotalDelegation: bigInt2HexInt(totalDelegated),
		Value:           bigInt2HexInt(reward),
	}

	i, _ := es.State.GetIssue()
	overIssued, issue := calcIssueAmount(reward, i)
	result := &IssueResultJSON{
		ByFee:           bigInt2HexInt(i.PrevBlockFee),
		ByOverIssuedICX: bigInt2HexInt(overIssued),
		Issue:           bigInt2HexInt(issue),
	}
	return prep, result
}

func bigInt2HexInt(value *big.Int) *common.HexInt {
	h := new(common.HexInt)
	h.Set(value)
	return h
}
