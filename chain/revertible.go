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

package chain

import (
	"io/fs"
	"os"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const BackupSuffix = ".bk"

type RevertHandler func(revert bool)
type Revertible []RevertHandler

func (h *Revertible) Delete(p string) error {
	p2 := p + BackupSuffix
	err := os.Rename(p, p2)
	if err == nil {
		*h = append(*h, func(revert bool) {
			if revert {
				// log.Tracef("Revertible.Delete os.Rename(%s,%s)", p2, p)
				log.Must(os.Rename(p2, p))
			} else {
				// log.Tracef("Revertible.Delete os.RemoveAll(%s)", p2)
				log.Must(os.RemoveAll(p2))
			}
		})
		return nil
	} else if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

func (h *Revertible) Rename(p1, p2 string) error {
	err := os.Rename(p1, p2)
	if err == nil {
		*h = append(*h, func(revert bool) {
			if revert {
				// log.Tracef("Revertible.Rename os.Rename(%s,%s)", p2, p1)
				log.Must(os.Rename(p2, p1))
			}
		})
		return nil
	} else if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else {
		return err
	}
}

func (h *Revertible) Append(handler RevertHandler) int {
	*h = append(*h, handler)
	return len(*h) - 1
}

func (h *Revertible) RevertOrCommitOne(index int, revert bool) bool {
	items := *h
	if index < 0 || index >= len(items) {
		return false
	}
	if handler := items[index]; handler != nil {
		items[index] = nil
		handler(revert)
		return true
	}
	return false
}

func (h *Revertible) RevertOrCommit(revert bool) {
	rec := *h
	*h = nil
	for idx := len(rec) - 1; idx >= 0; idx -= 1 {
		if handler := rec[idx]; handler != nil {
			handler(revert)
		}
	}
}
