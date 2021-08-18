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

package blockv0

import (
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type RepJSON struct {
	Address *common.Address `json:"address,omitempty"`
	ID      *common.Address `json:"id,omitempty"`
	P2P     string          `json:"p2pEndPoint,omitempty"`
}

func (r *RepJSON) Normalize() {
	if r.Address == nil {
		r.Address = r.ID
		r.ID = nil
	}
}

type RepsList struct {
	json []RepJSON
	hash []byte
}

func NewRepsList(addresses ...*common.Address) *RepsList {
	res := &RepsList{}
	for _, a := range addresses {
		res.json = append(res.json, RepJSON{Address:a})
	}
	return res
}

func (l *RepsList) UnmarshalJSON(bs []byte) error {
	err := json.Unmarshal(bs, &l.json)
	for i, _ := range l.json {
		l.json[i].Normalize()
	}
	return err
}

func (l *RepsList) Size() int {
	return len(l.json)
}

func (l *RepsList) Hash() []byte {
	if l.hash == nil && len(l.json) > 0 {
		items := make([]merkle.Item, len(l.json))
		for i, rep := range l.json {
			items[i] = merkle.ValueItem(rep.Address.Bytes())
		}
		l.hash = merkle.CalcHashOfList(items)
	}
	return l.hash
}

func (l *RepsList) Get(i int) module.Address {
	return l.json[i].Address
}

func (l *RepsList) GetNextOf(addr module.Address) module.Address {
	for i, rep := range l.json {
		if rep.Address.Equal(addr) {
			idx := (i+1) % len(l.json)
			return l.json[idx].Address
		}
	}
	return nil
}

func (l *RepsList) ToJSON(version module.JSONVersion) (interface{}, error) {
	if l == nil {
		return nil, nil
	}
	return l.json, nil
}

func (l *RepsList) GetValidatorList(dbase db.Database) (module.ValidatorList, error) {
	vs := make([]module.Validator, len(l.json))
	for i, r := range l.json {
		v, err := state.ValidatorFromAddress(r.Address)
		if err != nil {
			return nil, err
		}
		vs[i] = v
	}
	return state.ValidatorSnapshotFromSlice(dbase, vs)
}
