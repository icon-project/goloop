/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package db

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/log"
)

type Flusher interface {
	Flush() error
}

type Writer interface {
	Database() Database
	Add(item Flusher)
	Prepare()
	Flush() error
}

type writer struct {
	layerDB LayerDB
	waiters sync.WaitGroup
	items   []Flusher
	results []error
}

func (w *writer) Database() Database {
	return w.layerDB
}

func (w *writer) Add(r Flusher) {
	w.items = append(w.items, r)
	w.waiters.Add(1)
}

func (w *writer) doFlush(idx int) {
	w.results[idx] = w.items[idx].Flush()
	w.waiters.Done()
}

func (w *writer) Prepare() {
	if w.results != nil {
		return
	}
	log.Debugf("db.Writer[%p].Prepare()", w)
	w.results = make([]error, len(w.items))
	for i := 0 ; i<len(w.results) ; i++ {
		go w.doFlush(i)
	}
}

func (w *writer) Flush() error {
	start := time.Now()
	w.Prepare()
	wait := time.Since(start)
	defer func() {
		total := time.Since(start)
		log.Debugf("db.Writer[%p].Flush() wait=%v total=%v", w, wait, total)
	}()
	w.waiters.Wait()
	for i:=0 ; i<len(w.results) ; i++ {
		if err := w.results[i] ; err != nil {
			return err
		}
	}
	return w.layerDB.Flush(true)
}

func NewWriter(dbase Database) Writer {
	return &writer{
		layerDB: NewLayerDB(dbase),
	}
}
