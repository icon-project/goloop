/*
 * Copyright 2022 ICON Foundation
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

package contract

import (
	"bytes"
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

type FeePayer struct {
	payer     module.Address
	portion   int
	steps     *big.Int
	paidSteps *big.Int
	feeSteps  *big.Int
}

func (p *FeePayer) PaySteps(ctx CallContext, steps *big.Int) (*big.Int, error) {
	if bytes.Equal(p.payer.ID(), state.SystemID) {
		return p.payWithSystemDeposit(ctx, steps)
	} else {
		return p.payWithAccountDeposit(ctx, steps)
	}
}

func (p *FeePayer) payWithSystemDeposit(ctx CallContext, steps *big.Int) (*big.Int, error) {
	as := ctx.GetAccountState(state.SystemID)
	if p.steps == nil {
		p.steps = p.calcStepsToPay(steps)
	}
	tsVar := scoredb.NewVarDB(as, state.VarSystemDepositUsage)
	usage := addBigInt(tsVar.BigInt(), new(big.Int).Mul(p.steps, ctx.StepPrice()))
	if err := tsVar.Set(usage); err != nil {
		return nil, err
	}
	p.paidSteps = p.steps
	return p.paidSteps, nil
}

func (p *FeePayer) payWithAccountDeposit(ctx CallContext, steps *big.Int) (*big.Int, error) {
	acc := ctx.GetAccountState(p.payer.ID())
	if p.steps == nil {
		p.steps = p.calcStepsToPay(steps)
	}
	if paidSteps, feeSteps, err := acc.PaySteps(ctx, p.steps); err != nil {
		return nil, err
	} else {
		if paidSteps != nil {
			p.paidSteps = paidSteps
			p.feeSteps = feeSteps
		} else {
			p.paidSteps = new(big.Int)
			p.feeSteps = nil
		}
		return p.paidSteps, nil
	}
}

func (p *FeePayer) calcStepsToPay(s *big.Int) *big.Int {
	steps := new(big.Int).Mul(s, big.NewInt(int64(p.portion)))
	return steps.Div(steps, big.NewInt(100))
}

func (p *FeePayer) ApplySteps(s *big.Int) *big.Int {
	if p.steps == nil {
		p.steps = p.calcStepsToPay(s)
	}
	return new(big.Int).Sub(s, p.steps)
}

type FeePayerInfo []*FeePayer

func (p FeePayerInfo) payers() ([]*FeePayer, *FeePayer) {
	if len(p) > 0 {
		last := len(p) - 1
		return p[0:last], p[last]
	} else {
		return nil, nil
	}
}

func (p *FeePayerInfo) SetFeeProportion(payer module.Address, portion int) error {
	if portion < 0 || portion > 100 || payer == nil {
		return errors.IllegalArgumentError.New("InvalidParameter")
	}
	var fp *FeePayer
	if portion != 0 {
		fp = &FeePayer{payer: payer, portion: portion}
	}
	if len(*p) == 0 {
		*p = make([]*FeePayer, 1)
	}
	(*p)[len(*p)-1] = fp
	return nil
}

func (p *FeePayerInfo) Apply(p2 FeePayerInfo, steps *big.Int) {
	payers := make([]*FeePayer, 0, len(*p)+len(p2)+1)
	sub, own := p.payers()
	if len(sub) > 0 {
		for _, payer := range sub {
			payers = append(payers, payer)
		}
	}
	for _, payer := range p2 {
		if payer != nil {
			steps = payer.ApplySteps(steps)
			payers = append(payers, payer)
		}
	}
	payers = append(payers, own)
	*p = payers
}

func (p *FeePayerInfo) PaySteps(ctx CallContext, steps *big.Int) (*big.Int, error) {
	toPay := steps
	for _, payer := range *p {
		if payer == nil {
			continue
		}
		paidSteps, err := payer.PaySteps(ctx, toPay)
		if err != nil {
			return nil, err
		}
		toPay = new(big.Int).Sub(toPay, paidSteps)
	}
	return new(big.Int).Sub(steps, toPay), nil
}

func addBigInt(v1, v2 *big.Int) *big.Int {
	if v1 == nil {
		return v2
	} else if v2 == nil {
		return nil
	} else {
		return new(big.Int).Add(v1, v2)
	}
}

func (p *FeePayerInfo) GetLogs(r txresult.Receipt) bool {
	m := map[string]*FeePayer{}
	for _, p1 := range *p {
		if p1 == nil {
			continue
		}
		key := string(p1.payer.Bytes())
		if p2, ok := m[key]; ok {
			m[key] = &FeePayer{
				payer:     p1.payer,
				paidSteps: new(big.Int).Add(p1.paidSteps, p2.paidSteps),
				feeSteps:  addBigInt(p1.feeSteps, p2.feeSteps),
			}
		} else {
			m[key] = p1
		}
	}
	if len(m) == 0 {
		return false
	}
	for _, p1 := range m {
		r.AddPayment(p1.payer, p1.paidSteps, p1.feeSteps)
	}
	return true
}

func (p *FeePayerInfo) ClearLogs() {
	*p = nil
}
