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

package test

import (
	"io"
	"path"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus"
)

type record struct {
	bytes []byte
	err   error
}

type WAL struct {
	round  []*record
	lock   []*record
	commit []*record
}

func NewWAL() *WAL {
	return &WAL{}
}

func (w *WAL) data(id string) *[]*record {
	id = path.Base(id)
	switch id {
	case "round":
		return &w.round
	case "lock":
		return &w.lock
	case "commit":
		return &w.commit
	default:
		log.Panicf("invalid wal id %s", id)
		return nil
	}
}

func (w *WAL) OpenForRead(id string) (consensus.WALReader, error) {
	return &WALReader{
		*w.data(id),
	}, nil
}

func (w *WAL) OpenForWrite(id string, cfg *consensus.WALConfig) (consensus.WALWriter, error) {
	return &WALWriter{
		cf:     *cfg,
		synced: w.data(id),
	}, nil
}

type WALReader struct {
	data []*record
}

func (r *WALReader) Read(v interface{}) ([]byte, error) {
	bs, err := r.ReadBytes()
	if err != nil {
		return nil, err
	}
	return codec.UnmarshalFromBytes(bs, v)
}

func (r *WALReader) ReadBytes() ([]byte, error) {
	if len(r.data) == 0 {
		return nil, io.EOF
	}
	m := r.data[0]
	r.data = r.data[1:]
	return m.bytes, nil
}

func (r *WALReader) Close() error {
	return nil
}

func (r *WALReader) CloseAndRepair() error {
	return nil
}

type WALWriter struct {
	cf       consensus.WALConfig
	synced   *[]*record
	buffered []*record
}

func (w *WALWriter) WriteBytes(bytes []byte) (int, error) {
	w.buffered = append(w.buffered, &record{bytes, nil})
	return len(bytes), nil
}

func (w *WALWriter) Sync() error {
	*w.synced = append(*w.synced, w.buffered...)
	return nil
}

func (w *WALWriter) Close() error {
	return w.Sync()
}
