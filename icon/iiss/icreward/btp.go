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

package icreward

import (
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type DSA struct {
	icobject.NoDatabase
	mask int64
}

func (d *DSA) Version() int {
	return 0
}

func (d *DSA) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&d.mask)
}

func (d *DSA) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(d.mask)
}

func (d *DSA) Equal(o icobject.Impl) bool {
	if dsa2, ok := o.(*DSA); ok {
		return d.mask == dsa2.mask
	} else {
		return false
	}
}

func (d *DSA) Clone() *DSA {
	if d == nil {
		return nil
	}
	return &DSA{
		mask: d.mask,
	}
}

func (d *DSA) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "DSA{mask=%b}", d.mask)
		} else {
			fmt.Fprintf(f, "DSA{%b}", d.mask)
		}
	}
}

func newDSA(_ icobject.Tag) *DSA {
	return new(DSA)
}

func NewDSA() *DSA {
	return new(DSA)
}

type PublicKey struct {
	icobject.NoDatabase
	mask int64
}

func (p *PublicKey) Version() int {
	return 0
}

func (p *PublicKey) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.Decode(&p.mask)
}

func (p *PublicKey) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.Encode(p.mask)
}

func (p *PublicKey) Equal(o icobject.Impl) bool {
	if p2, ok := o.(*PublicKey); ok {
		return p.mask == p2.mask
	} else {
		return false
	}
}

func (p *PublicKey) Clone() *PublicKey {
	if p == nil {
		return nil
	}
	return &PublicKey{
		mask: p.mask,
	}
}

func (p *PublicKey) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "PublicKey{mask=%b}", p.mask)
		} else {
			fmt.Fprintf(f, "PublicKey{%b}", p.mask)
		}
	}
}

func newPublicKey(_ icobject.Tag) *PublicKey {
	return new(PublicKey)
}

func NewPublicKey() *PublicKey {
	return new(PublicKey)
}
