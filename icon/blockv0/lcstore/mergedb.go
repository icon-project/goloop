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
	"sync"

	"github.com/icon-project/goloop/common/errors"
)

type mergedDatabase struct {
	lock   sync.Mutex
	dbs    []Database
	offset int
}

func (m *mergedDatabase) getDBSlice() []Database {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.dbs[m.offset:]
}

func (m *mergedDatabase) setFirstDB(current Database) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for idx, item := range m.dbs {
		if item == current {
			m.offset = idx
			return
		}
	}
}

func (m *mergedDatabase) GetTPS() float32 {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.dbs[m.offset].GetTPS()
}

func (m *mergedDatabase) GetBlockJSONByHeight(height int, pre bool) ([]byte, error) {
	for idx, db := range m.getDBSlice() {
		if bs, err := db.GetBlockJSONByHeight(height, pre); err != nil {
			if err == errors.ErrNotFound {
				continue
			}
			return nil, err
		} else if len(bs) > 0 {
			if idx != 0 {
				m.setFirstDB(db)
			}
			return bs, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (m *mergedDatabase) queryDB(yield func(db Database)([]byte, error)) ([]byte, error) {
	for _, db := range m.getDBSlice() {
		if bs, err := yield(db); err != nil {
			if err == errors.ErrNotFound {
				continue
			}
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (m *mergedDatabase) GetBlockJSONByID(id []byte) ([]byte, error) {
	return m.queryDB(func(db Database) ([]byte, error) {
		return db.GetBlockJSONByID(id)
	})
}

func (m *mergedDatabase) GetLastBlockJSON() ([]byte, error) {
	return m.dbs[len(m.dbs)-1].GetLastBlockJSON()
}

func (m *mergedDatabase) GetResultJSON(id []byte) ([]byte, error) {
	return m.queryDB(func(db Database) ([]byte, error) {
		return db.GetResultJSON(id)
	})
}

func (m *mergedDatabase) GetTransactionJSON(id []byte) ([]byte, error) {
	return m.queryDB(func(db Database) ([]byte, error) {
		return db.GetTransactionJSON(id)
	})
}

func (m *mergedDatabase) GetRepsJSONByHash(id []byte) ([]byte, error) {
	return m.queryDB(func(db Database) ([]byte, error) {
		return db.GetRepsJSONByHash(id)
	})
}

func (m *mergedDatabase) GetReceiptJSON(id []byte) ([]byte, error) {
	return m.queryDB(func(db Database) ([]byte, error) {
		return db.GetReceiptJSON(id)
	})
}

func (m *mergedDatabase) Close() error {
	for i := 0 ; i<len(m.dbs) ; i++ {
		if err := m.dbs[i].Close(); err != nil {
			return err
		}
	}
	return nil
}

func NewMergedDB(dbs []Database) Database {
	return &mergedDatabase{ dbs: dbs }
}
