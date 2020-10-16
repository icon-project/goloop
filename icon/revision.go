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

package icon

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
	RevisionReserved
)

var revisionFlags = []module.Revision{
	module.UseChainID | module.UseMPTOnEvents | module.UseCompactAPIInfo,
	0,
	0,
	module.InputCostingWithJSON,
	0,
	0,
	0,
	0,
	0,
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
		return revisionFlags[0]
	}
	if v >= len(revisionFlags) {
		return module.Revision(v) + revisionFlags[len(revisionFlags)-1]
	} else {
		return module.Revision(v) + revisionFlags[v]
	}
}
