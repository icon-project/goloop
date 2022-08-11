/*
 * Copyright 2022 ICON Foundation
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

package sync2

import (
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
)

const (
	DataRequestEntryLimit    = 20
	DataRequestNodeLimit     = 3
	DataRequestNodeInterval  = time.Millisecond * 300
	DataRequestRoundInterval = time.Second * 3
)

type dataSyncer struct {
	logger   log.Logger
	builder  merkle.Builder
	reactors []SyncReactor
	sp       SyncProcessor
}

type onDataHandler func()

func (r onDataHandler) OnData(value []byte, builder merkle.Builder) error {
	r()
	return nil
}

func (s *dataSyncer) Start() {
	sp := newSyncProcessor(s.builder, s.reactors, s.logger, true)
	sproc := sp.(SyncProcessor)
	sproc.Start(func(err error) {
		if err != nil {
			s.logger.Warnf("DataSyncer finished by error(%v)", err)
		}
	})
	s.sp = sproc
}

func (s *dataSyncer) Term() {
	s.sp.Stop()
}

func (s *dataSyncer) AddRequest(id db.BucketID, key []byte) error {
	return s.sp.AddRequest(id, key)
}

func newDataSyncer(database db.Database, reactors []SyncReactor, logger log.Logger) *dataSyncer {
	s := &dataSyncer{
		logger:   logger,
		builder:  merkle.NewBuilderWithRawDatabase(database),
		reactors: reactors,
	}
	return s
}
