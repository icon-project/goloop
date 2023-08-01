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

package icon

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type rlpBytesStoreState struct {
	id    []byte
	as    state.AccountState
	batch *batchData
}


var markEmpty = []byte{0x00}


func (r *rlpBytesStoreState) RLPSetValue(key []byte, value []byte) ([]byte, error) {
	if len(value) == 0 {
		key2 := crypto.SHA3Sum256(key)
		backup, err := r.as.DeleteValue(key2)
		if err != nil {
			return nil, err
		}
		if len(backup) == 0 {
			return r.as.GetValue(key)
		}
		if backup[0] == 0 {
			return r.as.DeleteValue(key)
		} else {
			return r.as.SetValue(key, backup[1:])
		}
	} else {
		old, err := r.as.SetValue(key, value)
		if err != nil {
			return nil, err
		}
		key2 := crypto.SHA3Sum256(key)
		if len(old) == 0 {
			if _, err := r.as.SetValue(key2, markEmpty); err != nil {
				return nil, err
			} else {
				return old, nil
			}
		} else {
			backup, err := r.as.GetValue(key2)
			if err != nil {
				return nil, err
			}
			if len(backup) == 0 {
				_, err := r.as.SetValue(key2, append([]byte{0x01}, old...))
				if err != nil {
					return nil, err
				}
			}
			return old, nil
		}
	}
}

func (r *rlpBytesStoreState) RLPDeleteValue(key []byte) ([]byte, error) {
	old, err := r.as.DeleteValue(key)
	if err != nil {
		return nil, err
	}
	if len(old) > 0 {
		key2 := crypto.SHA3Sum256(key)
		if _, err := r.as.DeleteValue(key2); err != nil {
			return nil, err
		}
	}
	return old, nil
}

func (r *rlpBytesStoreState) GetValue(key []byte) ([]byte, error) {
	if value, ok := r.batch.GetValue(r.id, key); ok {
		if len(value) == 0 {
			return nil, nil
		}
		return value, nil
	}
	return r.as.GetValue(key)
}

func (r *rlpBytesStoreState) SetValue(key []byte, value []byte) ([]byte, error) {
	old, ok := r.batch.SetValue(r.id, key, value, len(value) != 0)
	old2, err := r.RLPSetValue(key, value)
	if err != nil {
		return nil, err
	} else if ok {
		if len(old) == 0 {
			return nil, nil
		}
		return old, nil
	} else {
		return old2, nil
	}
}

func (r *rlpBytesStoreState) DeleteValue(key []byte) ([]byte, error) {
	old, ok := r.batch.SetValue(r.id, key, nil, true)
	old2, err := r.RLPDeleteValue(key)
	if err != nil {
		return nil, err
	} else if ok  {
		if len(old) == 0 {
			return nil, nil
		}
		return old, nil
	} else {
		return old2, nil
	}
}

func newRLPBytesStore(addr module.Address, as state.AccountState, batch *batchData) *rlpBytesStoreState {
	return &rlpBytesStoreState{addr.ID(), as, batch}
}
