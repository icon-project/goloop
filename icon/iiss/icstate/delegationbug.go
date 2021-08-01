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

package icstate

import (
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var DelegationBugPrefix = containerdb.ToKey(
	containerdb.HashBuilder,
	scoredb.DictDBPrefix,
	"delegation_bug",
)

type DelegationBug struct {
	icobject.NoDatabase

	address     *common.Address
	blockHeight int64
	delegations Delegations
}

func NewDelegationBugWithTag(_ icobject.Tag) *DelegationBug {
	return new(DelegationBug)
}

func NewDelegationBug(addr module.Address, height int64, ds Delegations) *DelegationBug {
	return &DelegationBug{
		address:     common.AddressToPtr(addr),
		blockHeight: height,
		delegations: ds,
	}
}

func (d *DelegationBug) Version() int {
	return 1
}

func (d *DelegationBug) Address() module.Address {
	return d.address
}

func (d *DelegationBug) BlockHeight() int64 {
	return d.blockHeight
}

func (d *DelegationBug) Delegations() Delegations {
	return d.delegations
}

func (d *DelegationBug) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
		&d.address,
		&d.blockHeight,
		&d.delegations,
	)
}

func (d *DelegationBug) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		d.address,
		d.blockHeight,
		d.delegations,
	)
}

func (d *DelegationBug) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*DelegationBug); ok {
		return d.address.Equal(d2.address) &&
			d.blockHeight == d2.blockHeight &&
			d.delegations.Equal(d2.delegations)
	} else {
		return false
	}
}


func (d *DelegationBug) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "DelegationBug{address=%s blockHeight=%d delegations=%+v}",
				d.address, d.blockHeight, d.delegations)
		} else {
			fmt.Fprintf(f, "DelegationBug{%s %d %v}", d.address, d.blockHeight, d.delegations)
		}
	}
}
