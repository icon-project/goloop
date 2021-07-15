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

package blockv0_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/icon/blockv0"
)

func TestBlockVote_NewBlockVote(t *testing.T) {
	w := wallet.New()
	bv := blockv0.NewBlockVote(w, 1, 0, []byte{0}, 0)
	err := bv.Verify()
	assert.NoError(t, err)
}
