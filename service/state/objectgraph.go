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

package state

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/merkle"
)

const (
	MissingGraphDataError = iota + errors.CodeService + 500
)

type objectGraph struct {
	bk        db.Bucket
	needFlush bool
	nextHash  int
	graphHash []byte
	graphData []byte
}

func (o *objectGraph) flush() error {
	if o == nil || o.graphData == nil {
		return nil
	}
	if o.needFlush {
		if err := o.bk.Set(o.graphHash, o.graphData); err != nil {
			return err
		}
	}
	return nil
}

func (o *objectGraph) Equal(o2 *objectGraph) bool {
	if o == o2 {
		return true
	}
	if o == nil || o2 == nil {
		return false
	}
	if o.nextHash != o2.nextHash {
		return false
	}
	if !bytes.Equal(o.graphHash, o2.graphHash) {
		return false
	}
	return true
}

func (o *objectGraph) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(o.nextHash, o.graphHash)
}

func (o *objectGraph) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(&o.nextHash, &o.graphHash)
}

func (o *objectGraph) Changed(
	dbase db.Database, hasData bool, nextHash int, graphData []byte,
) (*objectGraph, error) {
	n := new(objectGraph)
	if o != nil {
		*n = *o
	} else {
		if bk, err := dbase.GetBucket(db.BytesByHash); err != nil {
			return nil, errors.CriticalIOError.Wrap(err, "FailToGetBucket")
		} else {
			n.bk = bk
		}
	}
	n.nextHash = nextHash
	if hasData {
		if len(graphData) == 0 {
			n.graphData = nil
			n.graphHash = nil
			n.needFlush = false
		} else {
			graphHash := crypto.SHA3Sum256(graphData)
			if !bytes.Equal(graphHash, n.graphHash) {
				n.graphData = graphData
				n.graphHash = graphHash
				n.needFlush = true
			}
		}
	}
	if n.nextHash == 0 && len(n.graphHash) == 0 {
		return nil, nil
	}
	return n, nil
}

func (o *objectGraph) Get(withData bool) (int, []byte, []byte, error) {
	if o == nil {
		return 0, nil, nil, errors.ErrNotFound
	}
	if withData {
		if o.graphData == nil && len(o.graphHash) > 0 {
			v, err := o.bk.Get(o.graphHash)
			if err != nil {
				err = errors.CriticalIOError.Wrap(err, "FailToGetValue")
				return 0, nil, nil, err
			}
			if v == nil {
				err = MissingGraphDataError.Errorf(
					"NoValueInHash(hash=%#x)", o.graphHash)
				return 0, o.graphHash, nil, err
			}
			o.graphData = v
		}
		return o.nextHash, o.graphHash, o.graphData, nil
	} else {
		return o.nextHash, o.graphHash, nil, nil
	}
}

func (o *objectGraph) Resolve(bd merkle.Builder) error {
	if len(o.graphHash) > 0 {
		v, err := o.bk.Get(o.graphHash)
		if err != nil {
			return err
		}
		if v == nil {
			bd.RequestData(db.BytesByHash, o.graphHash, o)
			return nil
		}
		o.graphData = v
	}
	return nil
}

func (o *objectGraph) OnData(data []byte, bd merkle.Builder) error {
	o.graphData = data
	o.needFlush = true
	return nil
}

func (o *objectGraph) ResetDB(dbase db.Database) error {
	if o == nil {
		return nil
	}
	if bk, err := dbase.GetBucket(db.BytesByHash); err != nil {
		return errors.CriticalIOError.Wrap(err, "FailToGetBucket")
	} else {
		o.bk = bk
		return nil
	}
}

func (o *objectGraph) String() string {
	return fmt.Sprintf("ObjectGraph{hash=%#x,next=%d}", o.graphHash, o.nextHash)
}

type objectGraphCache map[string]*objectGraph

func (o objectGraphCache) Clone() objectGraphCache {
	if len(o) == 0 {
		return nil
	}
	n := make(map[string]*objectGraph, len(o))
	for k, v := range o {
		n[k] = v
	}
	return n
}

func (o *objectGraphCache) Set(hash []byte, graph *objectGraph) {
	if *o == nil {
		*o = make(map[string]*objectGraph)
	}
	(*o)[string(hash)] = graph
}

func (o objectGraphCache) Get(hash []byte) *objectGraph {
	if o == nil || len(hash) == 0 {
		return nil
	}
	return o[string(hash)]
}
