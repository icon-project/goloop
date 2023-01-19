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

package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	SecondInMicro = 1000000
	MinuteInMicro = SecondInMicro * 60
)

func tidOf(idx int) []byte {
	return []byte(fmt.Sprintf("tx%d", idx))
}

func TestTXIDManager_RecordedLocator(t *testing.T) {
	baseHeight := int64(10)
	baseTS := int64(100)
	tid1 := []byte("tx1")
	tid2 := []byte("tx2")
	tid3 := []byte("tx3")
	tid4 := []byte("tx4")

	dbase := db.NewMapDB()
	bk, err := dbase.GetBucket(db.TransactionLocatorByHash)
	assert.NoError(t, err)

	var locator transactionLocator
	locator.BlockHeight = baseHeight
	err = bk.Set([]byte(tid1), codec.MustMarshalToBytes(&locator))
	assert.NoError(t, err)

	tsc := NewTimestampChecker()
	tsc.SetThreshold(5 * time.Minute)

	t.Run("dummy_logger", func(t *testing.T) {
		mgr, err := NewTXIDManager(dbase, tsc, nil)
		assert.NoError(t, err)

		logger := mgr.NewLogger(module.TransactionGroupNormal, 0, 0)

		logger1 := logger.NewLogger(baseHeight+1, baseTS+MinuteInMicro)

		err = logger1.Add(tid2, false)
		assert.NoError(t, err)

		err = logger.Commit()
		assert.NoError(t, err)

		logger2 := logger1.NewLogger(baseHeight+2, baseTS+2*MinuteInMicro)

		has, err := logger2.Has(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2)
		assert.NoError(t, err)
		assert.True(t, has)

		err = logger2.Add(tid3, false)
		assert.NoError(t, err)

		err = logger2.Add(tid4, false)
		assert.NoError(t, err)

		err = logger1.Commit()
		assert.NoError(t, err)

		has, err = logger2.Has(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(tid2)
		assert.NoError(t, err)
		assert.True(t, has)

		logger3 := logger2.NewLogger(0, 0)

		err = logger2.Commit()
		assert.NoError(t, err)

		err = logger3.Add(tidOf(5), false)
		assert.NoError(t, err)

		err = logger3.Commit()
		assert.NoError(t, err)

		// duplicated commit (should be silently ignored)
		err = logger3.Commit()
		assert.NoError(t, err)

		has, err = mgr.HasRecent(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(tid2)
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("same height same tx", func(t *testing.T) {
		mgr, err := NewTXIDManager(dbase, tsc, nil)
		assert.NoError(t, err)

		has, err := mgr.HasRecent(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(tid2)
		assert.NoError(t, err)
		assert.False(t, has)

		logger := mgr.NewLogger(module.TransactionGroupNormal, baseHeight, baseTS)

		has, err = logger.Has(tid1)
		assert.NoError(t, err)
		assert.False(t, has)

		err = logger.Add(tid1, false)
		assert.NoError(t, err)

		err = logger.Add(tid2, false)
		assert.NoError(t, err)

		logger2 := logger.NewLogger(baseHeight+1, baseTS+5*MinuteInMicro)

		has, err = logger2.Has(tid1)
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = logger2.Has(tid2)
		assert.NoError(t, err)
		assert.True(t, has)

		err = logger2.Add(tid3, false)
		assert.NoError(t, err)

		err = logger2.Add(tid4, false)
		assert.NoError(t, err)

		// first block is committed
		err = logger.Commit()
		assert.NoError(t, err)

		// check first block(tid1:OK, tid2:OK, tid3:NG, tid4:NG)
		has, err = mgr.HasRecent(tid1)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(tid2)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(tid3)
		assert.NoError(t, err)
		assert.False(t, has)
		has, err = mgr.HasRecent(tid4)
		assert.NoError(t, err)
		assert.False(t, has)

		// second block is committed
		err = logger2.Commit()
		assert.NoError(t, err)

		// duplicate commit (should be silently ignored)
		err = logger2.Commit()
		assert.NoError(t, err)

		// check second items (tid3, tid4)
		has, err = mgr.HasRecent(tid3)
		assert.NoError(t, err)
		assert.True(t, has)
		has, err = mgr.HasRecent(tid4)
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("old_block_prunning", func(t *testing.T) {
		mgr, err := NewTXIDManager(dbase, tsc, nil)
		assert.NoError(t, err)

		height := baseHeight
		ts := baseTS

		logger := mgr.NewLogger(module.TransactionGroupNormal, height, ts)
		err = logger.Add(tid1, false)
		assert.NoError(t, err)
		height += 1
		ts += MinuteInMicro

		for i := 2; i <= 20; i++ {
			nlogger := logger.NewLogger(height, ts)

			err := nlogger.Add(tidOf(i), false)
			assert.NoError(t, err)

			err = logger.Commit()
			assert.NoError(t, err)

			logger = nlogger
			height += 1
			ts += MinuteInMicro
		}

		has, err := logger.Has(tid1)
		assert.NoError(t, err)
		assert.False(t, has)
		has, err = mgr.HasRecent(tid1)
		assert.NoError(t, err)
		assert.False(t, has)

		has, err = logger.Has(tidOf(20))
		assert.NoError(t, err)
		assert.True(t, has)

		has, err = mgr.HasRecent(tidOf(20))
		assert.NoError(t, err)
		assert.False(t, has)

		has, err = mgr.HasRecent(tidOf(19))
		assert.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("new height same tx", func(t *testing.T) {
		history, err := NewTXIDManager(dbase, tsc, nil)
		assert.NoError(t, err)
		logger := history.NewLogger(module.TransactionGroupNormal, baseHeight+1, baseTS)
		err = logger.Add(tid1, false)
		assert.Error(t, err)
	})
}
