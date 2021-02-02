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
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"math/big"
)

type IssuePRepJSON struct {
	IRep            *common.HexInt `json:"irep"`
	RRep            *common.HexInt `json:"rrep"`
	TotalDelegation *common.HexInt `json:"totalDelegation"`
	Value           *common.HexInt `json:"value"`
}

func parseIssuePRepData(data []byte) (*IssuePRepJSON, error) {
	if data == nil {
		return nil, nil
	}
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
	if data == nil {
		return nil, nil
	}
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

func (i *IssueResultJSON) GetTotalReward() *big.Int {
	total := new(big.Int).Add(i.ByFee.Value(), i.ByOverIssuedICX.Value())
	total.Add(total, i.Issue.Value())
	return total
}

func RegulateIssueInfo(es *ExtensionStateImpl, iScore *big.Int) {
	issue, _ := es.State.GetIssue()
	issue = regulateIssueInfo(issue, iScore)
	es.State.SetIssue(issue)
}

// regulateIssueInfo regulate icx issue amount with previous period data.
func regulateIssueInfo(issue *icstate.Issue, iScore *big.Int) *icstate.Issue {
	var icx, remains *big.Int
	if iScore == nil || iScore.Sign() == 0 {
		icx = new(big.Int)
		remains = new(big.Int)
	} else {
		icx, remains = new(big.Int).DivMod(iScore, BigIntIScoreICXRatio, new(big.Int))
	}
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
func calcIssueAmount(reward *big.Int, i *icstate.Issue) (issue *big.Int, byOverIssued *big.Int, byFee *big.Int) {
	issue = new(big.Int).Set(reward)
	byFee = new(big.Int)
	byOverIssued = new(big.Int)

	if issue.Cmp(i.OverIssued) > 0 {
		byOverIssued.Set(i.OverIssued)
		issue.Sub(issue, i.OverIssued)
	} else {
		byOverIssued.Set(issue)
		issue.SetInt64(0)
		return
	}

	if issue.Cmp(i.PrevBlockFee) > 0 {
		byFee.Set(i.PrevBlockFee)
		issue.Sub(issue, i.PrevBlockFee)
	} else {
		byFee.Set(issue)
		issue.SetInt64(0)
	}
	return
}

//GetIssueData return issue information for base TX
func GetIssueData(es *ExtensionStateImpl) (*IssuePRepJSON, *IssueResultJSON) {
	term := es.State.GetTerm()
	if term == nil || !term.IsDecentralized() {
		return nil, nil
	}
	issueInfo, _ := es.State.GetIssue()
	if term.GetIISSVersion() == icstate.IISSVersion1 {
		return getIssueDataV1(es, term)
	} else {
		return nil, getIssueDataV2(issueInfo, term)
	}
}

func getIssueDataV1(es *ExtensionStateImpl, term *icstate.Term) (*IssuePRepJSON, *IssueResultJSON) {
	irep := term.Irep()
	rrep := term.Rrep()
	// TODO read values from Term and replace es to issue
	mainPRepCount := term.MainPRepCount()
	electedPRepCount := term.ElectedPRepCount()
	totalDelegated := term.TotalDelegated()
	reward := calcRewardPerBlock(
		irep,
		rrep,
		new(big.Int).SetInt64(int64(mainPRepCount)),
		new(big.Int).SetInt64(int64(electedPRepCount)),
		totalDelegated,
	)
	prep := &IssuePRepJSON{
		IRep:            icutils.BigInt2HexInt(irep),
		RRep:            icutils.BigInt2HexInt(rrep),
		TotalDelegation: icutils.BigInt2HexInt(totalDelegated),
		Value:           icutils.BigInt2HexInt(reward),
	}

	i, _ := es.State.GetIssue()
	issue, byOverIssued, byFee := calcIssueAmount(reward, i)
	result := &IssueResultJSON{
		ByFee:           icutils.BigInt2HexInt(byFee),
		ByOverIssuedICX: icutils.BigInt2HexInt(byOverIssued),
		Issue:           icutils.BigInt2HexInt(issue),
	}
	return prep, result
}

func getIssueDataV2(issueInfo *icstate.Issue, term *icstate.Term) *IssueResultJSON {
	reward := new(big.Int).Div(term.Iglobal(), big.NewInt(term.Period()))
	issue, byOverIssued, byFee := calcIssueAmount(reward, issueInfo)
	result := &IssueResultJSON{
		ByFee:           icutils.BigInt2HexInt(byFee),
		ByOverIssuedICX: icutils.BigInt2HexInt(byOverIssued),
		Issue:           icutils.BigInt2HexInt(issue),
	}
	return result
}

