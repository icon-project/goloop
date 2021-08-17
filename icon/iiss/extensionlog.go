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

package iiss

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type ExtensionLog interface {
	Handle(es *ExtensionStateImpl) error
}

type delegationLog struct {
	from   module.Address
	offset int
	index  int64
	event  *icobject.Object
	ds     icstate.Delegations
}

func (dl *delegationLog) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "delegationLog{from=%s offset=%d index=%d event=%+v}",
				dl.from, dl.offset, dl.index, dl.event)
		} else {
			fmt.Fprintf(f, "delegationLog{%s %d %d %+v}", dl.from, dl.offset, dl.index, dl.event)
		}
	}
}

func (dl *delegationLog) Handle(es *ExtensionStateImpl) error {
	event, err := es.Front.GetEvent(dl.offset, dl.index)
	if err != nil {
		return err
	}
	// setDelegation was failed
	if event == nil || dl.event.Equal(event) == false {
		if err = es.State.SetIllegalDelegation(icstate.NewIllegalDelegation(dl.from, dl.ds)); err != nil {
			return err
		}
		// Add EventDelegationV2 to es.Front
		var delegated, delegating icstage.VoteList
		switch dl.event.Tag().Type() {
		case icstage.TypeEventDelegation:
			e := icstage.ToEventVote(dl.event)
			delegating = e.Votes()
		case icstage.TypeEventDelegationV2:
			e := icstage.ToEventDelegationV2(dl.event)
			delegating = e.Delegating()
		default:
			return errors.IllegalArgumentError.Errorf("Illegal type icstage event object %d", dl.event.Tag().Type())
		}
		_, _, err = es.Front.AddEventDelegationV2(dl.offset, dl.from, delegated, delegating)
		if err != nil {
			return err
		}
	} else {
		if err = es.State.DeleteIllegalDelegation(dl.from); err != nil {
			return err
		}
	}
	return nil
}

func newDelegationLog(from module.Address, offset int, idx int64, obj *icobject.Object, ds icstate.Delegations) *delegationLog {
	return &delegationLog{
		from:   from,
		offset: offset,
		index:  idx,
		event:  obj,
		ds:     ds,
	}
}

type claimIScoreLog struct {
	from   module.Address
	amount *big.Int
	claim  *icstage.IScoreClaim
}

func (cl *claimIScoreLog) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "claimIScoreLog{from=%s amount=%d}", cl.from, cl.amount)
		} else {
			fmt.Fprintf(f, "claimIScoreLog{%s %d}", cl.from, cl.amount)
		}
	}
}

func (cl *claimIScoreLog) Handle(es *ExtensionStateImpl) error {
	claim, err := es.Front.GetIScoreClaim(cl.from)
	if err != nil {
		return err
	}
	// claimIScore was failed
	if claim == nil || cl.claim.Equal(claim) == false {
		// Add IScoreClaim to es.Front
		if _, err = es.Front.AddIScoreClaim(cl.from, cl.amount); err != nil {
			return err
		}
	}
	return nil
}

func NewClaimIScoreLog(from module.Address, amount *big.Int, claim *icstage.IScoreClaim) *claimIScoreLog {
	return &claimIScoreLog{
		from:   from,
		amount: amount,
		claim:  claim,
	}
}
