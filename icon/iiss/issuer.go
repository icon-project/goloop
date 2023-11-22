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

package iiss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

type IssuePRepJSON struct {
	IRep            *common.HexInt `json:"irep"`
	RRep            *common.HexInt `json:"rrep"`
	TotalDelegation *common.HexInt `json:"totalDelegation"`
	Value           *common.HexInt `json:"value"`
}

func ParseIssuePRepData(data []byte) (*IssuePRepJSON, error) {
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

func (i *IssuePRepJSON) GetIRep() *big.Int {
	return i.IRep.Value()
}

func (i *IssuePRepJSON) GetRRep() *big.Int {
	return i.RRep.Value()
}

func (i *IssuePRepJSON) GetTotalDelegation() *big.Int {
	return i.TotalDelegation.Value()
}

func (i *IssuePRepJSON) GetValue() *big.Int {
	return i.Value.Value()
}

func (i *IssuePRepJSON) Equal(i2 *IssuePRepJSON) bool {
	return i.IRep.Cmp(i2.IRep.Value()) == 0 &&
		i.RRep.Cmp(i2.RRep.Value()) == 0 &&
		i.TotalDelegation.Cmp(i2.TotalDelegation.Value()) == 0 &&
		i.Value.Cmp(i2.Value.Value()) == 0
}

func (i *IssuePRepJSON) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"IssuePRepJSON{IRep=%s RRep=%s TotalDelegation=%s Value=%s}",
				i.IRep, i.RRep, i.TotalDelegation, i.Value)
		} else {
			fmt.Fprintf(f, "IssuePRepJSON{%s %s %s %s}",
				i.IRep, i.RRep, i.TotalDelegation, i.Value)
		}
	case 's':
		fmt.Fprintf(f, "IssuePRepJSON{%s %s %s %s}",
			i.IRep, i.RRep, i.TotalDelegation, i.Value)
	}
}

type IssueResultJSON struct {
	ByFee           *common.HexInt `json:"coveredByFee"`
	ByOverIssuedICX *common.HexInt `json:"coveredByOverIssuedICX"`
	Issue           *common.HexInt `json:"issue"`
}

func ParseIssueResultData(data []byte) (*IssueResultJSON, error) {
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

func (i *IssueResultJSON) GetByFee() *big.Int {
	return i.ByFee.Value()
}

func (i *IssueResultJSON) GetByOverIssuedICX() *big.Int {
	return i.ByOverIssuedICX.Value()
}

func (i *IssueResultJSON) GetIssue() *big.Int {
	return i.Issue.Value()
}

func (i *IssueResultJSON) Equal(i2 *IssueResultJSON) bool {
	return i.ByFee.Cmp(i2.ByFee.Value()) == 0 &&
		i.ByOverIssuedICX.Cmp(i2.ByOverIssuedICX.Value()) == 0 &&
		i.Issue.Cmp(i2.Issue.Value()) == 0
}

func (i *IssueResultJSON) GetTotalReward() *big.Int {
	total := new(big.Int).Add(i.ByFee.Value(), i.ByOverIssuedICX.Value())
	total.Add(total, i.Issue.Value())
	return total
}

func (i *IssueResultJSON) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"IssueResultJSON{ByFee=%s ByOverIssuedICX=%s Issue=%s}",
				i.ByFee, i.ByOverIssuedICX, i.Issue)
		} else {
			fmt.Fprintf(f, "IssueResultJSON{%s %s %s}",
				i.ByFee, i.ByOverIssuedICX, i.Issue)
		}
	case 's':
		fmt.Fprintf(f, "IssueResultJSON{%s %s %s}",
			i.ByFee, i.ByOverIssuedICX, i.Issue)
	}
}

// RegulateIssueInfo regulate icx issue amount with previous period data.
func RegulateIssueInfo(issue *icstate.Issue, iScore *big.Int) {
	// Do not regulate ICX issue if there is no ICX issuance.
	if issue.PrevTotalReward().Sign() == 0 {
		return
	}
	if iScore == nil || iScore.Sign() == 0 {
		return
	}
	prevTotalIScore := icutils.ICXToIScore(issue.PrevTotalReward())
	prevTotalIScore.Add(prevTotalIScore, issue.OverIssuedIScore())
	overIssuedIScore := new(big.Int).Sub(prevTotalIScore, iScore)
	issue.SetOverIssuedIScore(overIssuedIScore)
}

// calcRewardPerBlock calculate reward per block
func calcRewardPerBlock(
	irep *big.Int,
	rrep *big.Int,
	mainPRepCount *big.Int,
	totalDelegated *big.Int,
) *big.Int {
	// reference ICON1: IssueFormula._handle_icx_issue_formula_for_prep()
	beta1 := new(big.Int)
	beta2 := new(big.Int)
	beta3 := new(big.Int)
	base := new(big.Int).Mul(irep, new(big.Int).SetInt64(icmodule.MonthPerYear))
	base.Div(base, new(big.Int).SetInt64(icmodule.YearBlock*2))

	beta1.Mul(base, mainPRepCount)

	// 100 : Beta2 percentage
	beta2.Mul(base, new(big.Int).SetInt64(100))

	if totalDelegated.Sign() != 0 {
		// real rrep = rrep + eep + dbp = 3 * rrep
		beta3.Mul(rrep, new(big.Int).SetInt64(icmodule.RrepMultiplier))
		beta3.Mul(beta3, totalDelegated)
		beta3.Div(beta3, new(big.Int).SetInt64(icmodule.YearBlock*icmodule.RrepDivider))
	}

	reward := new(big.Int).Add(beta1, beta2)
	reward.Add(reward, beta3)

	return reward
}

func calcIssueAmount(reward *big.Int, i *icstate.Issue) (issue *big.Int, byOverIssued *big.Int, byFee *big.Int) {
	issue = new(big.Int).Set(reward)
	byFee = new(big.Int)
	byOverIssued = new(big.Int)

	oIScore := new(big.Int).Abs(i.OverIssuedIScore())
	overIssuedICX := icutils.IScoreToICX(oIScore)
	if i.OverIssuedIScore().Sign() == -1 {
		overIssuedICX = new(big.Int).Neg(overIssuedICX)
	}

	if issue.Cmp(overIssuedICX) > 0 {
		byOverIssued.Set(overIssuedICX)
		issue.Sub(issue, overIssuedICX)
	} else {
		byOverIssued.Set(issue)
		issue.SetInt64(0)
		return
	}

	if issue.Cmp(i.PrevBlockFee()) > 0 {
		byFee.Set(i.PrevBlockFee())
		issue.Sub(issue, i.PrevBlockFee())
	} else {
		byFee.Set(issue)
		issue.SetInt64(0)
	}
	return
}

// GetIssueData return issue information for base TX
func GetIssueData(es *ExtensionStateImpl) (*IssuePRepJSON, *IssueResultJSON) {
	if !es.IsDecentralized() {
		return nil, nil
	}
	term := es.State.GetTermSnapshot()
	issueInfo, _ := es.State.GetIssue()
	if term.GetIISSVersion() == icstate.IISSVersion2 {
		return getIssueDataV1(es, term, es.State.GetTotalDelegation())
	} else {
		return nil, getIssueDataV2(issueInfo, term)
	}
}

func getIssueDataV1(
	es *ExtensionStateImpl,
	term *icstate.TermSnapshot,
	totalDelegated *big.Int,
) (*IssuePRepJSON, *IssueResultJSON) {
	irep := term.Irep()
	rrep := term.Rrep()
	mainPRepCount := term.MainPRepCount()
	reward := calcRewardPerBlock(
		irep,
		rrep,
		new(big.Int).SetInt64(int64(mainPRepCount)),
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

func getIssueDataV2(issueInfo *icstate.Issue, term *icstate.TermSnapshot) *IssueResultJSON {
	var reward, remains *big.Int
	if term.Revision() < icmodule.RevisionFixIGlobal {
		reward, remains = new(big.Int).DivMod(term.RewardFund().IGlobal(), big.NewInt(term.Period()), new(big.Int))
	} else {
		reward, remains = new(big.Int).DivMod(term.RewardFund().IGlobal(), big.NewInt(icmodule.MonthBlock), new(big.Int))
	}
	if remains.Sign() == 1 {
		reward.Add(reward, intconv.BigIntOne)
	}
	issue, byOverIssued, byFee := calcIssueAmount(reward, issueInfo)
	result := &IssueResultJSON{
		ByFee:           icutils.BigInt2HexInt(byFee),
		ByOverIssuedICX: icutils.BigInt2HexInt(byOverIssued),
		Issue:           icutils.BigInt2HexInt(issue),
	}
	return result
}
