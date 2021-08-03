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

package cache

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/icon-project/goloop/common/log"
)

const (
	hashSize          = 32
	dataMaxSize       = 532
	cacheItemSize     = hashSize + dataMaxSize
	fileCacheItemSize = cacheItemSize + 2
)

type BranchCache struct {
	nodes  [][2][]byte
	offset int
	depth  int
	size   int
	f      *os.File
	missed int
}

func (c *BranchCache) Get(nibs []byte, h []byte) ([]byte, bool) {
	if c == nil || nibs == nil || len(nibs) >= c.depth {
		c.missed += 1
		return nil, false
	}
	idx := indexByNibs(nibs)

	var node [][]byte
	if idx < c.offset {
		node = c.nodes[idx][:]
	} else {
		node = c.read(idx)
		if node == nil {
			return nil, true
		}
	}
	if bytes.Equal(node[0], h) {
		return node[1], true
	}
	c.missed += 1
	return nil, true
}

func (c *BranchCache) String() string {
	return fmt.Sprintf("BranchCache{%p depth=%d offset=%d}", c, c.depth, c.offset)
}

func (c *BranchCache) Put(nibs []byte, h []byte, serialized []byte) {
	if nibs == nil || len(serialized) > dataMaxSize || len(nibs) >= c.depth {
		return
	}
	idx := indexByNibs(nibs)

	if idx < c.offset {
		c.nodes[idx] = [2][]byte{h, serialized}
	} else {
		c.write(idx, h, serialized)
	}
}

func (c *BranchCache) read(idx int) [][]byte {
	at, err := c.f.Seek(int64((idx-c.offset)*fileCacheItemSize), 0)
	if err != nil {
		//return nil, fmt.Errorf("fail to seek err:%+v", err)
		return nil
	}
	var node [2][]byte
	b := make([]byte, fileCacheItemSize)
	n, err := c.f.ReadAt(b, at)
	if err != nil {
		//return nil, fmt.Errorf("fail to read err:%+v", err)
		return nil
	}
	if n < 2 {
		//return nil, fmt.Errorf("fail to read prefix")
		return nil
	}
	vl := binary.BigEndian.Uint16(b[:2])
	if vl < hashSize {
		return nil
	}
	l := 2 + int(vl)
	if n < l {
		//return nil, fmt.Errorf("fail to read n:%d, expected:%d",n,l)
		return nil
	}
	b = b[2:]
	node[0] = b[:hashSize]
	node[1] = b[hashSize:vl]
	return node[:]
}

func (c *BranchCache) write(idx int, h []byte, serialized []byte) {
	at := int64((idx - c.offset) * fileCacheItemSize)
	vl := hashSize + len(serialized)
	b := make([]byte, 2, 2+vl)
	binary.BigEndian.PutUint16(b[:2], uint16(vl))
	b = append(b, h...)
	b = append(b, serialized...)
	if _, err := c.f.WriteAt(b, at); err != nil {
		c.size = c.offset
		c.f.Close()
	}
}

func openfile(path string) (f *os.File, err error) {
	dir := filepath.Dir(path)
	if fi, sErr := os.Stat(dir); sErr == nil {
		if !fi.IsDir() {
			err = fmt.Errorf("open %s: not a directory", dir)
			return
		}
	} else if os.IsNotExist(sErr) {
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			err = fmt.Errorf("fail to mkdir err:%+v", mkErr)
			return
		}
	} else {
		err = fmt.Errorf("fail to stat err:%+v", sErr)
		return
	}
	f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		err = fmt.Errorf("fail to open path:%s, err:%+v", path, err)
		return
	}
	return f, nil
}

func (c *BranchCache) OnAttach(id []byte) cacheImpl {
	if c.missed >= fullCacheMigrationThreshold {
		if logCacheEvents {
			log.Warnf("MigrateCacheFor(id=%#x,missed=%d)", id, c.missed)
		}
		return NewFullCacheFromBranch(c)
	}
	c.missed = 0
	return c
}

func NewBranchCache(depth int, fdepth int, path string) *BranchCache {
	offset := sizeByDepth(depth)
	size := sizeByDepth(depth + fdepth)
	var f *os.File
	if fdepth > 0 {
		var err error
		if f, err = openfile(path); err != nil {
			log.Infof("BranchCache fdepth:%d will be ignored, err:%+v", fdepth, err)
			size = offset
		} else {
			abs, _ := filepath.Abs(f.Name())
			log.Debugf("BranchCache using fdepth:%d path:%s", fdepth, abs)
		}
	}
	return &BranchCache{
		nodes:  make([][2][]byte, offset),
		offset: offset,
		depth:  depth + fdepth,
		size:   size,
		f:      f,
	}
}
