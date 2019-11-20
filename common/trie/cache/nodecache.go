package cache

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/icon-project/goloop/common/log"
)

const (
	hashSize          = 32
	dataMaxSize       = 532
	cacheItemSize     = hashSize + dataMaxSize
	fileCacheItemSize = cacheItemSize + 2
)

type NodeCache struct {
	lock   sync.Mutex
	nodes  [][2][]byte
	offset int
	size   int
	f      *os.File
}

func indexByNibs(nibs []byte) int {
	if len(nibs) == 0 {
		return 0
	}
	idx := 0
	for _, nib := range nibs {
		idx = idx*16 + int(nib) + 1
	}
	return idx
}

func sizeByDepth(d int) int {
	return ((1 << uint(4*d)) - 1) / 15
}

func (c *NodeCache) Get(nibs []byte, h []byte) ([]byte, bool) {
	if c == nil || nibs == nil {
		return nil, false
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()
	if idx >= c.size {
		return nil, false
	}

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
	return nil, true
}

func (c *NodeCache) Put(nibs []byte, h []byte, serialized []byte) {
	if c == nil || nibs == nil || len(serialized) > dataMaxSize {
		return
	}
	idx := indexByNibs(nibs)

	c.lock.Lock()
	defer c.lock.Unlock()
	if idx >= c.size {
		return
	}

	if idx < c.offset {
		c.nodes[idx] = [2][]byte{h, serialized}
	} else {
		c.write(idx, h, serialized)
	}
}

func (c *NodeCache) read(idx int) [][]byte {
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

func (c *NodeCache) write(idx int, h []byte, serialized []byte) {
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

func NewNodeCache(depth int, fdepth int, path string) *NodeCache {
	offset := sizeByDepth(depth)
	size := sizeByDepth(depth + fdepth)
	var f *os.File
	if fdepth > 0 {
		var err error
		if f, err = openfile(path); err != nil {
			log.Infof("NodeCache fdepth:%d will be ignored, err:%+v", fdepth, err)
			size = offset
		} else {
			abs, _ := filepath.Abs(f.Name())
			log.Debugf("NodeCache using fdepth:%d path:%s", fdepth, abs)
		}
	}
	return &NodeCache{
		nodes:  make([][2][]byte, offset),
		offset: offset,
		size:   size,
		f:      f,
	}
}
