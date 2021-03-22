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

package codec

import (
	"io"
	"sort"
)

type TypedDict struct {
	Keys []string
	Map  map[string]*TypedObj
}

func (m *TypedDict) put(key string, value *TypedObj) {
	m.Keys = append(m.Keys, key)
	m.Map[key] = value
}

func (m *TypedDict) RLPWriteSelf(w Writer) error {
	w2, err := w.WriteMap()
	if err != nil {
		return err
	}
	e2 := NewEncoder(w2)
	keys := m.Keys
	if len(keys) != len(m.Map) {
		keys = make([]string, 0, len(m.Map))
		for k := range m.Map {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	for _, k := range keys {
		if err := e2.Encode(k); err != nil {
			return err
		}
		if err := e2.Encode(m.Map[k]); err != nil {
			return err
		}
	}
	return e2.Close()
}

func (m *TypedDict) RLPReadSelf(r Reader) error {
	m.Keys = []string{}
	m.Map = make(map[string]*TypedObj)
	r2, err := r.ReadMap()
	if err != nil {
		return err
	}
	d2 := NewDecoder(r2)
	for true {
		var key string
		var value *TypedObj
		if err := d2.Decode(&key); err != nil {
			if err == io.EOF {
				return d2.Close()
			}
			return err
		}
		if err := d2.Decode(&value); err != nil {
			return err
		}
		m.put(key, value)
	}
	return nil
}
