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

package main

import (
	"bytes"
	"encoding/json"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ToJSONer interface {
	ToJSON(version module.JSONVersion) (interface{}, error)
}

func JSONMarshalAndCompact(jso interface{}) ([]byte, error) {
	if jsoner, ok := jso.(ToJSONer); ok {
		if obj, err := jsoner.ToJSON(module.JSONVersionLast); err != nil {
			return nil, err
		} else {
			jso = obj
		}
	}
	bs, err := json.Marshal(jso)
	if err != nil {
		log.Warnf("marshalling failure obj=%+v", jso)
		return nil, err
	}
	bf := bytes.NewBuffer(nil)
	if err := json.Compact(bf, bs); err != nil {
		return nil, errors.Wrapf(err, "CompactFailure(bs=%q)", string(bs))
	}
	return bf.Bytes(), nil
}
