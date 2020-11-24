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

package icstate

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"math/big"
)

const (
	prepStatusVersion1 = iota + 1
	prepStatusVersion  = prepStatusVersion1

	PrepGradeMain = iota
	PrepGradeSub
	PrepGradeCandidate

	StatusActive = iota
	StatusUnregistered
	StatusDisqualified
)

type PRepStatusSnapshot struct {
	icobject.NoDatabase
	grade        int
	penalty      int
	state        int
	delegated    *big.Int
	bonded       *big.Int
	vTotal       int
	vFail        int
	vFailCount   int
	vPenaltyMask int
	lastState    int
	lastHeight   int
}

func (pss *PRepStatusSnapshot) Version() int {
	return 0
}

func (pss *PRepStatusSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&pss.grade,
		&pss.penalty,
		&pss.state,
		&pss.delegated,
		&pss.bonded,
		&pss.vTotal,
		&pss.vFail,
		&pss.vFailCount,
		&pss.vPenaltyMask,
		&pss.lastState,
		&pss.lastHeight,
	)
	return err
}

func (pss *PRepStatusSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		pss.grade,
		pss.penalty,
		pss.state,
		pss.delegated,
		pss.bonded,
		pss.vTotal,
		pss.vFail,
		pss.vFailCount,
		pss.vPenaltyMask,
		pss.lastState,
		pss.lastHeight,
	)
}

func (pss *PRepStatusSnapshot) Equal(o icobject.Impl) bool {
	pss1, ok := o.(*PRepStatusSnapshot)
	if !ok {
		return false
	}
	return pss.grade == pss1.grade &&
		pss.penalty == pss1.penalty &&
		pss.state == pss1.state &&
		pss.delegated.Cmp(pss1.delegated) == 0 &&
		pss.bonded.Cmp(pss.bonded) == 0 &&
		pss.vTotal == pss.vTotal &&
		pss.vFail == pss.vFail &&
		pss.vFailCount == pss.vFailCount &&
		pss.vPenaltyMask == pss.vPenaltyMask &&
		pss.lastState == pss.lastState &&
		pss.lastHeight == pss.lastHeight
}

func newPRepStatusSnapshot(tag icobject.Tag) *PRepStatusSnapshot {
	return &PRepStatusSnapshot{
		delegated: new(big.Int),
		bonded:    new(big.Int),
	}
}

type PRepStatusState struct {
	address      module.Address
	grade        int
	status       int
	penalty      int
	delegated    *big.Int
	bonded       *big.Int
	vTotal       int
	vFail        int
	vFailCount   int
	vPenaltyMask int
	lastState    int
	lastHeight   int
}

func newPRepStatusState(address module.Address) *PRepStatusState {
	return &PRepStatusState{
		address:   address,
		delegated: new(big.Int),
		bonded:    new(big.Int),
	}
}

func (ps *PRepStatusState) Clear() {
	ps.grade = 0
	ps.penalty = 0
	ps.delegated = BigIntZero
	ps.bonded = BigIntZero
	ps.vTotal = 0
	ps.vFail = 0
	ps.vFailCount = 0
	ps.vPenaltyMask = 0
	ps.lastState = 0
	ps.lastHeight = 0
}

func (ps *PRepStatusState) Reset(pss *PRepStatusSnapshot) {
	ps.grade = pss.grade
}

func (ps *PRepStatusState) GetSnapshot() *PRepStatusSnapshot {
	pss := &PRepStatusSnapshot{}
	pss.grade = ps.grade
	pss.penalty = ps.penalty
	pss.delegated = ps.delegated
	pss.bonded = ps.bonded
	pss.vTotal = ps.vTotal
	pss.vFail = ps.vFail
	pss.vFailCount = ps.vFailCount
	pss.vPenaltyMask = ps.vPenaltyMask
	pss.lastState = ps.lastState
	pss.lastHeight = ps.lastHeight
	return pss
}

func (ps PRepStatusState) IsEmpty() bool {
	return ps.status == StatusActive && ps.grade == PrepGradeMain
}

func (ps PRepStatusState) GetAddress() module.Address {
	return ps.address
}

func (ps *PRepStatusState) SetGrade(g int) {
	ps.grade = g
}

func (ps *PRepStatusState) SetStatus(s int) {
	ps.status = s
}

func (ps *PRepStatusState) GetPRepStatusInfo() map[string]interface{} {
	data := make(map[string]interface{})
	data["grade"] = ps.grade
	data["status"] = ps.status
	return data
}

func NewPRepStatusStateWithSnapshot(a module.Address, pss *PRepStatusSnapshot) *PRepStatusState {
	ps := newPRepStatusState(a)
	ps.Reset(pss)
	return ps
}
