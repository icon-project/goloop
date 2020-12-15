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

package icreward

import (
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type Delegated struct {
	icobject.NoDatabase
	Enable   bool     // update via ENABLE event
	Current  *big.Int // update via DELEGATE event
	Snapshot *big.Int // update via PERIOD event. For calculating beta2
}

func (d *Delegated) Version() int {
	return 0
}

func (d *Delegated) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&d.Enable, &d.Current, &d.Snapshot)
	return err
}

func (d *Delegated) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(d.Enable, d.Current, d.Snapshot)
}

func (d *Delegated) Equal(o icobject.Impl) bool {
	if ic2, ok := o.(*Delegated); ok {
		return d.Enable == ic2.Enable && d.Current.Cmp(ic2.Current) == 0 && d.Snapshot.Cmp(ic2.Snapshot) == 0
	} else {
		return false
	}
}

func (d *Delegated) Clone() *Delegated {
	if d == nil {
		return nil
	}
	nd := NewDelegated()
	nd.Enable = d.Enable
	nd.Current.Set(d.Current)
	nd.Snapshot.Set(d.Snapshot)
	return nd
}

func (d *Delegated) IsEmpty() bool {
	return d.Enable == false && d.Current.Sign() == 0 && d.Snapshot.Sign() == 0
}

func newDelegated(tag icobject.Tag) *Delegated {
	return &Delegated{
		Current:  new(big.Int),
		Snapshot: new(big.Int),
	}
}
func NewDelegated() *Delegated {
	return &Delegated{
		Current:  new(big.Int),
		Snapshot: new(big.Int),
	}
}
