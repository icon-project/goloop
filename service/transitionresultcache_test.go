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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/state"
)

type testPlatform struct {
	base.Platform
}

func (p *testPlatform) NewExtensionSnapshot(dbase db.Database, raw []byte) state.ExtensionSnapshot {
	return nil
}

func Test_transitionResultCache_GetWorldSnapshot(t *testing.T) {
	mdb := db.NewMapDB()
	logger := log.GlobalLogger()
	trc := newTransitionResultCache(mdb, &testPlatform{}, 10, 10, logger)

	ws, err := trc.GetWorldSnapshot([]byte{}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ws)

	ws, err = trc.GetWorldSnapshot([]byte(nil), nil)
	assert.NoError(t, err)
	assert.NotNil(t, ws)
}
