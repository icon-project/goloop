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

import "sync"

type mergedDatabase struct {
	lock    sync.Mutex
	dbs     []Database
	current int
}

func (m *mergedDatabase) getCurrent() int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.current
}

func (m *mergedDatabase) setCurrent(v int) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.current = v
}

func (m *mergedDatabase) GetTPS() float32 {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.dbs[m.current].GetTPS()
}

func (m *mergedDatabase) GetBlockJSONByHeight(height int, pre bool) ([]byte, error) {
	start := m.getCurrent()
	for idx := start; idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetBlockJSONByHeight(height, pre); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			if start != idx {
				m.setCurrent(idx)
			}
			return bs, nil
		}
	}
	return nil, nil
}

func (m *mergedDatabase) GetBlockJSONByID(id []byte) ([]byte, error) {
	for idx := m.getCurrent(); idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetBlockJSONByID(id); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, nil
}

func (m *mergedDatabase) GetLastBlockJSON() ([]byte, error) {
	return m.dbs[len(m.dbs)-1].GetLastBlockJSON()
}

func (m *mergedDatabase) GetResultJSON(id []byte) ([]byte, error) {
	for idx := m.getCurrent(); idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetResultJSON(id); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, nil
}

func (m *mergedDatabase) GetTransactionJSON(id []byte) ([]byte, error) {
	for idx := m.getCurrent(); idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetTransactionJSON(id); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, nil
}

func (m *mergedDatabase) GetRepsJSONByHash(id []byte) ([]byte, error) {
	for idx := m.getCurrent(); idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetRepsJSONByHash(id); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, nil
}

func (m *mergedDatabase) GetReceiptJSON(id []byte) ([]byte, error) {
	for idx := m.getCurrent(); idx<len(m.dbs) ; idx++ {
		if bs, err := m.dbs[idx].GetReceiptJSON(id); err != nil {
			return nil, err
		} else if len(bs) > 0 {
			return bs, nil
		}
	}
	return nil, nil
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
