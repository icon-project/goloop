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
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
)

type RepJSON struct {
	Address common.Address `json:"address"`
	P2P     string         `json:"p2pEndPoint,omitempty"`
}

type RepsList struct {
	json []RepJSON
	hash []byte
}

func (l *RepsList) UnmarshalJSON(bs []byte) error {
	return json.Unmarshal(bs, &l.json)
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
	return &l.json[i].Address
}

func (l *RepsList) ToJSON(version module.JSONVersion) (interface{}, error) {
	if l == nil {
		return nil, nil
	}
	return l.json, nil
}
