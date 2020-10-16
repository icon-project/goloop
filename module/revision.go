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

package module

type Revision int64

const (
	InputCostingWithJSON Revision = 1 << (8 + iota)
	ExpandErrorCode
	UseChainID
	UseMPTOnEvents
	UseCompactAPIInfo
	LastRevisionBit
)

const (
	NoRevision  Revision = 0
	AllRevision          = LastRevisionBit - 1
)

func (r Revision) Value() int {
	return int(r & 0xff)
}

func (r Revision) InputCostingWithJSON() bool {
	return (r & InputCostingWithJSON) != 0
}

func (r Revision) ExpandErrorCode() bool {
	return (r & ExpandErrorCode) != 0
}

func (r Revision) UseChainID() bool {
	return (r & UseChainID) != 0
}

func (r Revision) UseMPTOnEvents() bool {
	return (r & UseMPTOnEvents) != 0
}

func (r Revision) UseCompactAPIInfo() bool {
	return (r & UseCompactAPIInfo) != 0
}
