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

package basic

import "github.com/icon-project/goloop/module"

const (
	Revision0 = iota
	Revision1
	Revision2
	Revision3
	Revision4
	Revision5
	Revision6
	Revision7
	Revision8
	Revision9
	RevisionReserved
)

// Remove warning for not using the constant.
// It's not used in the platform, but pyee may use it.
var _ = Revision2

const (
	DefaultRevision = Revision4
	MaxRevision     = RevisionReserved - 1
	LatestRevision  = Revision8
)

var revisionFlags []module.Revision

var toggleFlagsOnRevision = []struct {
	value int
	flags module.Revision
}{
	{Revision0, module.FixLostFeeByDeposit | module.PurgeEnumCache | module.FixMapValues},
	{Revision3, module.InputCostingWithJSON},
	{Revision6, module.ExpandErrorCode},
	{Revision7, module.UseChainID | module.UseMPTOnEvents},
	{Revision8, module.UseCompactAPIInfo},
	{Revision9, module.MultipleFeePayers | module.FixJCLSteps | module.ReportConfigureEvents},
}

func init() {
	flags := make([]module.Revision, MaxRevision+1)
	for _, e := range toggleFlagsOnRevision {
		flags[e.value] |= e.flags
	}
	var revSum module.Revision
	for idx, rev := range flags {
		revSum ^= rev
		flags[idx] = revSum
	}
	revisionFlags = flags
}

func valueToRevision(v int) module.Revision {
	if v < Revision1 {
		return 0
	}
	if v >= len(revisionFlags) {
		return module.Revision(v) + revisionFlags[len(revisionFlags)-1]
	} else {
		return module.Revision(v) + revisionFlags[v]
	}
}
