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

package consensus

import (
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

func (cs *consensus) ntsdIndexFor(ntid int64, bd module.BTPDigest, prevResult []byte) (int, error) {
	for i, ntd := range bd.NetworkTypeDigests() {
		nt, err := cs.c.ServiceManager().BTPNetworkTypeFromResult(prevResult, ntd.NetworkTypeID())
		if errors.Is(err, errors.ErrNotFound) {
			continue
		}
		if err != nil {
			return -1, err
		}
		if nt.NextProofContext() == nil {
			continue
		}
		if ntid == ntd.NetworkTypeID() {
			return i, nil
		}
	}
	return -1, errors.Wrapf(errors.ErrNotFound, "not found ntid=%d", ntid)
}
