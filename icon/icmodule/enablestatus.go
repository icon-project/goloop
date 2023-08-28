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
 */

package icmodule

import "github.com/icon-project/goloop/module"

type EnableStatus int

const (
	ESEnable EnableStatus = iota
	ESDisableTemp
	ESDisablePermanent
	ESJail
	ESUnjail
	ESMax
)

func (ef EnableStatus) IsEnabled() bool {
	return ef == ESEnable
}

func (ef EnableStatus) IsDisabledTemporarily() bool {
	return ef == ESDisableTemp
}

func (ef EnableStatus) IsDisabledPermanently() bool {
	return ef == ESDisablePermanent
}

func (ef EnableStatus) IsJail() bool {
	return ef == ESJail
}

func (ef EnableStatus) IsUnjail() bool {
	return ef == ESUnjail
}

func (ef EnableStatus) String() string {
	switch ef {
	case ESEnable:
		return "Enabled"
	case ESDisableTemp:
		return "DisabledTemporarily"
	case ESDisablePermanent:
		return "DisabledPermanently"
	case ESJail:
		return "Jail"
	case ESUnjail:
		return "Unjail"
	default:
		return "Unknown"
	}
}

type EnableEventLogger interface {
	AddEventEnable(blockHeight int64, owner module.Address, status EnableStatus) error
}
