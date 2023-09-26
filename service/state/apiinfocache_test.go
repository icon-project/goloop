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
 *
 */

package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/service/scoreapi"
)

var apis1 = []*scoreapi.Method{
	{
		Type:    scoreapi.Function,
		Name:    "hello",
		Flags:   scoreapi.FlagExternal,
		Indexed: 1,
		Inputs:  []scoreapi.Parameter{
			{
				Name:    "name",
				Type:    scoreapi.String,
			},
		},
		Outputs: nil,
	},
}

var apis2 = []*scoreapi.Method{
	{
		Type:    scoreapi.Function,
		Name:    "say",
		Flags:   scoreapi.FlagExternal,
		Indexed: 2,
		Inputs:  []scoreapi.Parameter{
			{
				Name:    "name",
				Type:    scoreapi.String,
			},
			{
				Name:    "msg",
				Type:    scoreapi.String,
			},
		},
		Outputs: nil,
	},
	{
		Type:    scoreapi.Function,
		Name:    "name",
		Flags:   scoreapi.FlagExternal|scoreapi.FlagReadOnly,
		Indexed: 0,
		Inputs:  []scoreapi.Parameter{},
		Outputs: []scoreapi.DataType{ scoreapi.String },
	},
}

func TestAttachAPIInfoCache(t *testing.T) {
	dbase := db.NewMapDB()
	ldb := db.NewLayerDB(dbase)

	// record first directly.
	info1 := scoreapi.NewInfo(apis1)
	hash, bs := MustEncodeAPIInfo(info1)
	bk, err := dbase.GetBucket(db.BytesByHash)
	assert.NoError(t, err)
	assert.NoError(t, bk.Set(hash, bs))

	// attach cache
	cdb, err := AttachAPIInfoCache(ldb, 1024)
	assert.NoError(t, err)

	// working with caches
	bk1, err := GetAPIInfoBucket(cdb)
	assert.NoError(t, err)
	info, err := bk1.Get(hash)
	assert.NoError(t, err)
	assert.EqualValues(t, info1, info)

	// working without cache
	bk2, err := GetAPIInfoBucket(ldb)
	assert.NoError(t, err)
	info, err = bk2.Get(hash)
	assert.NoError(t, err)
	assert.EqualValues(t, info1, info)

	// add new value
	info2 := scoreapi.NewInfo(apis2)
	hash, bs = MustEncodeAPIInfo(info2)
	assert.NoError(t, bk1.Set(hash, bs, info2))

	// confirm available through database
	info, err = bk2.Get(hash)
	assert.NoError(t, err)
	assert.EqualValues(t, info2, info)

	// clear database
	assert.NoError(t, ldb.Flush(false))

	// cache should still be valid
	info, err = bk1.Get(hash)
	assert.NoError(t, err)
	assert.EqualValues(t, info2, info)

	// no data in database
	info, err = bk2.Get(hash)
	assert.Error(t, err)
}
