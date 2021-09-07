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

package lcstore

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/errors"
)

type testDatabase struct {
	Database
	last int
}

func testBlockForHeight(h int) []byte {
	return []byte(fmt.Sprintf("BlockForHeight(%d)", h))
}

func testIDForHeight(h int) []byte {
	return []byte(fmt.Sprint(h))
}

func (tdb *testDatabase) GetBlockJSONByHeight(h int, unconfirmed bool) ([]byte, error) {
	if h <= tdb.last {
		return testBlockForHeight(h), nil
	} else {
		return nil, nil
	}
}

func (tdb *testDatabase) GetBlockJSONByID(id []byte) ([]byte, error) {
	height, err := strconv.Atoi(string(id))
	if err != nil {
		return nil, err
	}
	if height <= tdb.last {
		return testBlockForHeight(height), nil
	}
	return nil, nil
}

func (tdb *testDatabase) GetLastBlockJSON() ([]byte, error) {
	return testBlockForHeight(tdb.last), nil
}

func Test_layeredDatabase_GetBlockJSONByHeight(t *testing.T) {
	type fields struct {
		lock    sync.Mutex
		dbs     []Database
		current int
	}
	type args struct {
		height int
	}
	db1 := &testDatabase{ last: 10 }
	db2 := &testDatabase{ last: 20 }

	mdb := NewMergedDB([]Database{db1, db2})

	blk, err := mdb.GetLastBlockJSON()
	assert.NoError(t, err)
	assert.Equal(t, blk, testBlockForHeight(20))

	blk, err = mdb.GetBlockJSONByID(testIDForHeight(11))
	assert.NoError(t, err)
	assert.NotNil(t, blk)

	for i := 0 ; i<22 ; i++ {
		blk, err = mdb.GetBlockJSONByHeight(i, false)
		if i <= 20 {
			assert.NoError(t, err)
			assert.NotNil(t, blk)
		} else {
			assert.True(t, errors.NotFoundError.Equals(err))
			assert.Nil(t, blk)
		}
	}

	blk, err = mdb.GetBlockJSONByID(testIDForHeight(11))
	assert.NoError(t, err)
	assert.NotNil(t, blk)
}
