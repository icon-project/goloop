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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

type Delegating struct {
	icobject.NoDatabase
	icstate.Delegations
}

func (d *Delegating) Version() int {
	return 0
}

func (d *Delegating) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&d.Delegations)
}

func (d *Delegating) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(d.Delegations)
}

func (d *Delegating) Equal(o icobject.Impl) bool {
	if d2, ok := o.(*Delegating); ok {
		return d.Delegations.Equal(d2.Delegations)
	} else {
		return false
	}
}

func (d *Delegating) Clone() *Delegating {
	if d == nil {
		return nil
	}
	nd := NewDelegating()
	for _, ds := range d.Delegations {
		nd.Delegations = append(nd.Delegations,  ds.Clone())
	}
	return nd
}

func (d *Delegating) IsEmpty() bool {
	return len(d.Delegations) == 0
}

func newDelegating(tag icobject.Tag) *Delegating {
	return NewDelegating()
}

func NewDelegating() *Delegating {
	d := new(Delegating)
	d.Delegations = make([]*icstate.Delegation, 0)
	return d
}
