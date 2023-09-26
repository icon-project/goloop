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
	"github.com/icon-project/goloop/common/cache"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/scoreapi"
)

const APIInfoCache = "cache.api_info"


type APIInfoBucket interface {
	Get(hash []byte) (*scoreapi.Info, error)
	Set(hash, bytes []byte, info *scoreapi.Info) error
}

type apiInfoCache struct {
	bk    db.Bucket
	cache *cache.LRUCache
}

func (a *apiInfoCache) Get(hash []byte) (*scoreapi.Info, error) {
	obj, err := a.cache.Get(hash)
	if err != nil {
		return nil, err
	} else {
		return obj.(*scoreapi.Info), nil
	}
}

func (a *apiInfoCache) Set(hash, bytes []byte, info *scoreapi.Info) error {
	if err := a.bk.Set(hash, bytes); err != nil {
		return err
	}
	a.cache.Put(string(hash), info)
	return nil
}

func GetAPIInfoFromBucket(bk db.Bucket, hash []byte) (*scoreapi.Info, error) {
	bs, err := bk.Get(hash)
	if err != nil {
		return nil, errors.CriticalIOError.Wrapf(err, "FailToGetAPIInfo(hash=%x)", hash)

	}
	var info scoreapi.Info
	_, err = codec.BC.UnmarshalFromBytes(bs, &info)
	if err != nil {
		return nil, errors.CriticalFormatError.Wrapf(err, "InvalidAPIInfo(hash=%x)", hash)
	}
	return &info, nil
}

func MustEncodeAPIInfo(info *scoreapi.Info) ([]byte, []byte) {
	bs := codec.BC.MustMarshalToBytes(info)
	return crypto.SHA3Sum256(bs), bs
}

func newAPIInfoCache(dbase db.Database, size int) (*apiInfoCache, error) {
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	}
	return &apiInfoCache{
		bk: bk,
		cache: cache.NewLRUCache(size, func(hash []byte) (interface{}, error) {
			info, err := GetAPIInfoFromBucket(bk, hash)
			return info, err
		}),
	}, nil
}

func AttachAPIInfoCache(dbase db.Database, size int) (db.Database, error) {
	aic, err := newAPIInfoCache(dbase, size)
	if err != nil {
		return nil, err
	}
	return db.WithFlags(dbase, db.Flags{
		APIInfoCache: aic,
	}), nil
}

type apiInfoBucket struct {
	bk db.Bucket
}

func (a apiInfoBucket) Get(hash []byte) (*scoreapi.Info, error) {
	return GetAPIInfoFromBucket(a.bk, hash)
}

func (a apiInfoBucket) Set(hash, bytes []byte, _ *scoreapi.Info) error {
	return a.bk.Set(hash, bytes)
}

func GetAPIInfoBucket(dbase db.Database) (APIInfoBucket, error) {
	aic := db.GetFlag(dbase, APIInfoCache)
	if aic != nil {
		return aic.(*apiInfoCache), nil
	}
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return nil, err
	} else {
		return apiInfoBucket{bk }, nil
	}
}