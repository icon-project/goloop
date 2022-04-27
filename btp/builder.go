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

package btp

import (
	"github.com/icon-project/goloop/module"
)

type SectionBuilder interface {
	SendMessage(nid int64, msg []byte)
	EnsureSection(nid int64)
	Build() module.BTPSection
}

type StateView interface {
	// GetNetwork returns Network. Requirement for the fields of the Network
	// is different field by field. PrevNetworkSectionHash and
	// LastNetworkSectionHash field shall have initial value before the
	// transactions of a transition is executed. Other fields shall have
	// final value after the transactions of a transition is executed.
	GetNetwork(nid int64) (*Network, error)

	// GetNetworkType returns final value of NetworkType
	GetNetworkType(ntid int64) (*NetworkType, error)
}

func NewBuilder(view StateView) SectionBuilder {
	return nil
}
