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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type PRepStatusSnapshot struct {
	icobject.NoDatabase
	state     int
	grade     int
	delegated common.HexInt
}

func (pss *PRepStatusSnapshot) Version() int {
	return 0
}

func (pss *PRepStatusSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&pss.state,
		&pss.grade,
		&pss.delegated,
	)
	return err
}

func (pss *PRepStatusSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		pss.state,
		pss.grade,
		pss.delegated,
	)
}

func (pss *PRepStatusSnapshot) Equal(o icobject.Impl) bool {
	pss1, ok := o.(*PRepStatusSnapshot)
	if !ok {
		return false
	}
	return pss.state == pss1.state &&
		pss.grade == pss1.grade &&
		pss.delegated.Cmp(pss1.delegated.Value()) == 0
}

func newPRepStatusSnapshot(tag icobject.Tag) *PRepStatusSnapshot {
	return new(PRepStatusSnapshot)
}
