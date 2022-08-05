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

package icstage

import (
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type BTPDSA struct {
	icobject.NoDatabase
	index int
}

func (b *BTPDSA) Version() int {
	return 0
}

func (b *BTPDSA) Index() int {
	return b.index
}

func (b *BTPDSA) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&b.index)
	return err
}

func (b *BTPDSA) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(b.index)
}

func (b *BTPDSA) Equal(o icobject.Impl) bool {
	if b2, ok := o.(*BTPDSA); ok {
		return b.index == b2.index
	} else {
		return false
	}
}

func (b *BTPDSA) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "BTPDSA{index=%b}", b.index)
		} else {
			fmt.Fprintf(f, "BTPDSA{%b}", b.index)
		}
	}
}

func newBTPDSA(_ icobject.Tag) *BTPDSA {
	return new(BTPDSA)
}

func NewBTPDSA(index int) *BTPDSA {
	return &BTPDSA{
		index: index,
	}
}

type BTPPublicKey struct {
	icobject.NoDatabase
	from  *common.Address
	index int
}

func (b *BTPPublicKey) Version() int {
	return 0
}

func (b *BTPPublicKey) From() *common.Address {
	return b.from
}

func (b *BTPPublicKey) Index() int {
	return b.index
}

func (b *BTPPublicKey) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(&b.from, &b.index)
	return err
}

func (b *BTPPublicKey) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(b.from, b.index)
}

func (b *BTPPublicKey) Equal(o icobject.Impl) bool {
	if b2, ok := o.(*BTPPublicKey); ok {
		return b.from.Equal(b2.from) && b.index == b2.index
	} else {
		return false
	}
}

func (b *BTPPublicKey) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "BTPPublicKey{address=%s value=%+v}", b.from, b.index)
		} else {
			fmt.Fprintf(f, "BTPPublicKey{%s %v}", b.from, b.index)
		}
	}
}

func newBTPPublicKey(_ icobject.Tag) *BTPPublicKey {
	return new(BTPPublicKey)
}

func NewBTPPublicKey(addr *common.Address, index int) *BTPPublicKey {
	return &BTPPublicKey{
		from:  addr,
		index: index,
	}
}
