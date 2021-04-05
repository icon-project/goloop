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

package lcimporter

import (
	"reflect"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

func FeePaymentEqual(p1, p2 module.FeePayment) bool {
	return common.AddressEqual(p1.Payer(), p2.Payer()) &&
		p1.Amount().Cmp(p2.Amount()) == 0
}

func EventLogEqual(e1, e2 module.EventLog) bool {
	return common.AddressEqual(e1.Address(), e2.Address()) &&
		reflect.DeepEqual(e1.Indexed(), e2.Indexed()) &&
		reflect.DeepEqual(e1.Data(), e2.Data())
}

func CheckStatus(logger log.Logger, s1, s2 module.Status) error {
	if s1 == s2 {
		return nil
	}
	if s1 == module.StatusUnknownFailure && s2 == module.StatusInvalidParameter {
		logger.Warnf("Ignore status difference(e=%s,r=%s)", s1, s2)
		return nil
	}
	return errors.InvalidStateError.Errorf("InvalidStatus(e=%s,r=%s)", s1, s2)
}

func CheckReceipt(logger log.Logger, r1, r2 module.Receipt) error {
	if err := CheckStatus(logger, r1.Status(), r2.Status()); err != nil {
		return err
	}

	if !(r1.To().Equal(r2.To()) &&
		r1.CumulativeStepUsed().Cmp(r2.CumulativeStepUsed()) == 0 &&
		r1.StepUsed().Cmp(r2.StepUsed()) == 0 &&
		r1.StepPrice().Cmp(r2.StepPrice()) == 0 &&
		common.AddressEqual(r1.SCOREAddress(), r2.SCOREAddress()) &&
		r1.LogsBloom().Equal(r2.LogsBloom())) {
		return errors.InvalidStateError.New("DifferentResultValue")
	}

	idx := 0
	for itr1, itr2 := r1.FeePaymentIterator(), r2.FeePaymentIterator(); itr1.Has() || itr2.Has(); _, _, idx = itr1.Next(), itr2.Next(), idx+1 {
		p1, err := itr1.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfPayments")
		}
		p2, err := itr2.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfPayments")
		}
		if !FeePaymentEqual(p1, p2) {
			return errors.InvalidStateError.New("DifferentPayment")
		}
	}

	idx = 0
	for itr1, itr2 := r1.EventLogIterator(), r2.EventLogIterator(); itr1.Has() || itr2.Has(); _, _, idx = itr1.Next(), itr2.Next(), idx+1 {
		e1, err := itr1.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfEvents")
		}
		e2, err := itr2.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "EndOfEvents")
		}

		if !EventLogEqual(e1, e2) {
			return errors.InvalidStateError.Errorf("DifferentEvent(idx=%d)", idx)
		}
	}
	return nil
}
