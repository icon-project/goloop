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

package contract

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type DSContext struct {
	Height int64
	Hash   []byte
}

const DSContextHistoryLimit = 20
type DSContextHistory []DSContext

func (h *DSContextHistory) Get(height int64) []byte {
	hh := *h
	sz := len(hh)
	if sz == 0 || height < hh[0].Height {
		return nil
	}
	for idx := sz-1 ; idx>=0 ; idx-- {
		if hh[idx].Height <= height {
			return hh[idx].Hash
		}
	}
	return nil
}

func (h *DSContextHistory) Push(height int64, hash []byte) (bool, error) {
	hh := *h
	sz := len(hh)
	if sz > 0 {
		// new height must be higher than previous
		if height <= hh[sz-1].Height {
			return false, errors.IllegalArgumentError.New("InvalidHeight")
		}

		// if it has same hash with previous, then ignore.
		if bytes.Equal(hh[sz-1].Hash, hash) {
			return false, nil
		}
	}
	item := DSContext{height, hash}
	if sz+1 > DSContextHistoryLimit {
		copy(hh[0:], hh[sz-DSContextHistoryLimit+1:])
		hh = hh[0:DSContextHistoryLimit-1]
	}
	*h = append(hh, item)
	return true, nil
}

func (h *DSContextHistory) Bytes() []byte {
	if len(*h) == 0 {
		return nil
	}
	return codec.BC.MustMarshalToBytes(*h)
}

func (h *DSContextHistory) FirstHeight() (int64, bool) {
	hh := *h
	if len(hh) < 1 {
		return 0, false
	} else {
		return hh[0].Height, true
	}
}

func DSContextHistoryFromBytes(bs []byte) (DSContextHistory, error) {
	var history DSContextHistory
	if len(bs) > 0 {
		_, err := codec.BC.UnmarshalFromBytes(bs, &history)
		if err != nil {
			return nil, err
		}
	}
	return history, nil
}

type DSContextHistoryDB struct {
	DSContextHistory
	store   *containerdb.VarDB
}

func (h *DSContextHistoryDB) Push(height int64, hash[]byte) error {
	if ok, err := h.DSContextHistory.Push(height, hash); err != nil {
		return err
	} else {
		if ok {
			return h.store.Set(h.DSContextHistory.Bytes())
		}
		return nil
	}
}

func NewDSContextHistoryDB(as containerdb.BytesStoreState) (*DSContextHistoryDB, error) {
	store := scoredb.NewVarDB(as, state.VarDSRContextHistory)
	history, err := DSContextHistoryFromBytes(store.Bytes())
	if err != nil {
		return nil, errors.CriticalFormatError.Wrap(err, "InvalidDSContextHistory")
	}
	return &DSContextHistoryDB{history, store}, nil
}