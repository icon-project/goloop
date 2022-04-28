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

package blockv1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
)

func Test_applyPatch(t *testing.T) {
	dbase := db.NewMapDB()

	// checkNeedPatch returns false for different chain

	need, err := checkNeedPatch(dbase, BH_41385879_RECORD, BH_41385879_MISSED)
	assert.NoError(t, err)
	assert.False(t, need)

	need, err = checkNeedPatch(dbase, BH_41385450_RECORD, BH_41385450_MISSED)
	assert.NoError(t, err)
	assert.False(t, need)

	// record some for specific block then checkNeedPatch returns true

	bk, err := dbase.GetBucket(db.BytesByHash)
	assert.NoError(t, err)

	// for BH 41385450
	err = bk.Set([]byte(BH_41385450_RECORD), []byte{0x01})
	assert.NoError(t, err)

	need, err = checkNeedPatch(dbase, BH_41385450_RECORD, BH_41385450_MISSED)
	assert.NoError(t, err)
	assert.True(t, need)

	// for BH 41385879
	err = bk.Set([]byte(BH_41385879_RECORD), []byte{0x01})
	assert.NoError(t, err)

	need, err = checkNeedPatch(dbase, BH_41385879_RECORD, BH_41385879_MISSED)
	assert.NoError(t, err)
	assert.True(t, need)

	// apply patch and check validity

	// for BH 41385450
	err = applyPatch(dbase, patchFor41385450)
	assert.NoError(t, err)

	value, err := bk.Get([]byte(BH_41385450_RECORD))
	assert.NoError(t, err)
	bh := string(crypto.SHA3Sum256(value))
	assert.Equal(t, BH_41385450_MISSED, bh)

	need, err = checkNeedPatch(dbase, BH_41385450_RECORD, BH_41385450_MISSED)
	assert.NoError(t, err)
	assert.False(t, need)

	// for BH 41385879
	err = applyPatch(dbase, patchFor41385879)
	assert.NoError(t, err)

	value, err = bk.Get([]byte(BH_41385879_RECORD))
	assert.NoError(t, err)
	bh = string(crypto.SHA3Sum256(value))
	assert.Equal(t, BH_41385879_MISSED, bh)

	need, err = checkNeedPatch(dbase, BH_41385879_RECORD, BH_41385879_MISSED)
	assert.NoError(t, err)
	assert.False(t, need)
}
