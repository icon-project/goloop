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

package transaction

import (
	"encoding/json"
	"sort"

	"github.com/icon-project/goloop/common/errors"
)

type Factory struct {
	Priority    int
	CheckJSON   func(jso map[string]interface{}) bool
	ParseJSON   func(js []byte, jsm map[string]interface{}, raw bool) (Transaction, error)
	CheckBinary func(bs []byte) bool
	ParseBinary func(bs []byte) (Transaction, error)
}

var factories []*Factory

func RegisterFactory(f *Factory) {
	if f != nil {
		factories = append(factories, f)
		sort.SliceStable(factories, func(i, j int) bool {
			return factories[i].Priority < factories[j].Priority
		})
	}
}

func newTransactionFromJSON(js []byte, raw bool) (Transaction, error) {
	if !raw {
		if bs, err := jsonCompact(js); err != nil {
			return nil, InvalidFormat.Wrap(err, "fail to compact json")
		} else {
			js = bs
		}
	}
	var jso map[string]interface{}
	if err := json.Unmarshal(js, &jso); err != nil {
		return nil, InvalidFormat.Wrap(err, "fail to parse json")
	}
	for _, factory := range factories {
		if factory.CheckJSON != nil && factory.CheckJSON(jso) {
			return factory.ParseJSON(js, jso, raw)
		}
	}
	return nil, InvalidFormat.New("UnknownJSON")
}

func newTransaction(b []byte) (Transaction, error) {
	if len(b) < 1 {
		return nil, errors.New("IllegalTransactionData")
	}
	if b[0] == '{' {
		return newTransactionFromJSON(b, true)
	}
	for _, factory := range factories {
		if factory.CheckBinary != nil && factory.CheckBinary(b) {
			return factory.ParseBinary(b)
		}
	}
	return nil, InvalidFormat.New("UnknownBinary")
}
