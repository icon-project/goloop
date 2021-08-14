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

package icon

import (
	"reflect"

	"github.com/icon-project/goloop/service/contract"
)

const BatchKey = "batch"

var BatchType = reflect.TypeOf((*batchData)(nil)).Elem()

type batchRoot struct {
	block batchData
	tx    *batchData
}

func (r *batchRoot) Init(parent contract.CustomLogs) contract.CustomLogs {
	r.block.Init(nil)
	return r
}

func (r *batchRoot) Apply(data contract.CustomLogs) {
	r.block.Apply(data)
	return
}

func (r *batchRoot) handleTxBatch(success bool) {
	if r.tx != nil {
		if success {
			r.block.Apply(r.tx)
		}
		r.tx = nil
	}
}

type batchData struct {
	parent *batchData
	data   map[string][]byte
}

func (b *batchData) Init(parent contract.CustomLogs) contract.CustomLogs {
	if parent != nil {
		if root, ok := parent.(*batchRoot) ; ok {
			root.tx = b
			b.parent = &root.block
		} else {
			b.parent = parent.(*batchData)
		}
	}
	b.data = make(map[string][]byte)
	return b
}

func (b *batchData) Apply(data contract.CustomLogs) {
	child := data.(*batchData)
	for k, v := range child.data {
		b.data[k] = v
	}
}

func (b *batchData) getValue(id string) ([]byte, bool) {
	for p := b ; p != nil ; p = p.parent {
		if v, ok := p.data[id] ; ok {
			return v, true
		}
	}
	return nil, false
}

func makeKey(id, k []byte) string {
	key := make([]byte, 0, len(id)+len(k))
	key = append(key, id...)
	key = append(key, k...)
	return string(key)
}

func (b *batchData) GetValue(id []byte, k []byte) ([]byte, bool) {
	key := makeKey(id, k)
	return b.getValue(key)
}

func (b *batchData) SetValue(id []byte, k, v []byte, nocreate bool) ([]byte, bool){
	key := makeKey(id, k)
	old, ok := b.getValue(key)
	if ok || !nocreate {
		b.data[key] = v
	}
	return old, ok
}
