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

package common

import "github.com/icon-project/goloop/module"

type blockInfo struct {
	height    int64
	timestamp int64
}

func (bi *blockInfo) Height() int64 {
	return bi.height
}

func (bi *blockInfo) Timestamp() int64 {
	return bi.timestamp
}

func NewBlockInfo(height, timestamp int64) module.BlockInfo {
	return &blockInfo{
		height:    height,
		timestamp: timestamp,
	}
}

func BlockInfoEqual(bi1 module.BlockInfo, bi2 module.BlockInfo) bool {
	if bi1 == bi2 {
		return true
	}
	if bi1 == nil || bi2 == nil {
		return false
	}
	return bi1.Timestamp() == bi2.Timestamp() && bi1.Height() == bi2.Height()
}
