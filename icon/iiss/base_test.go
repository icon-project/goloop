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
package iiss

import (
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/contract"
)

type dummyPlatformType struct{}

func (d dummyPlatformType) ToRevision(value int) module.Revision {
	return module.LatestRevision
}

type testCallContext struct {
	contract.CallContext
	blockHeight int64
}

func (cc *testCallContext) BlockHeight() int64 {
	return cc.blockHeight
}

func (cc *testCallContext) setBlockHeight(blockHeight int64) {
	cc.blockHeight = blockHeight
}
