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
	Revision1 = iota + 1
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

const (
	DefaultRevision = Revision4
	MaxRevision     = RevisionReserved - 1
	LatestRevision  = Revision8
)

var revisionFlags = []module.Revision{
	// Revision 0
	module.FixLostFeeByDeposit | module.PurgeEnumCache | module.FixMapValues,
	// Revision 1
	0,
	// Revision 2
	0,
	// Revision 3
	module.InputCostingWithJSON,
	// Revision 4
	0,
	// Revision 5
	0,
	// Revision 6
	module.ExpandErrorCode,
	// Revision 7
	module.UseChainID | module.UseMPTOnEvents,
	// Revision 8
	module.UseCompactAPIInfo,
	// Revision 9
	module.MultipleFeePayers,
}

func init() {
	var revSum module.Revision
	for idx, rev := range revisionFlags {
		revSum |= rev
		revisionFlags[idx] = revSum
	}
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
