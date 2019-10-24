package cache

import (
	"encoding/hex"
	"path"
	"sync"

	"github.com/icon-project/goloop/common/db"
)

const (
	defaultAccountDepth = 5
)

type databaseWithCacheManager struct {
	db.Database

	lock  sync.Mutex
	path  string
	depth [2]int
	world *NodeCache
	store map[string]*NodeCache
}

func (m *databaseWithCacheManager) getWorldNodeCache() *NodeCache {
	return m.world
}

func (m *databaseWithCacheManager) getAccountNodeCache(id []byte) *NodeCache {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.depth[0] == 0 {
		return nil
	}
	sid := string(id)
	if c, ok := m.store[sid]; ok {
		return c
	} else {
		path := path.Join(m.path, hex.EncodeToString(id))
		c = NewNodeCache(m.depth[0], m.depth[1], path)
		m.store[sid] = c
		return c
	}
}

func WorldNodeCacheOf(database db.Database) *NodeCache {
	if m, ok := database.(*databaseWithCacheManager); ok {
		return m.getWorldNodeCache()
	}
	return nil
}

func AccountNodeCacheOf(database db.Database, id []byte) *NodeCache {
	if m, ok := database.(*databaseWithCacheManager); ok {
		return m.getAccountNodeCache(id)
	}
	return nil
}

func AttachManager(database db.Database, dir string, mem, file int) db.Database {
	return &databaseWithCacheManager{
		Database: database,
		path:     dir,
		depth:    [2]int{mem, file},
		world:    NewNodeCache(defaultAccountDepth, 0, ""),
		store:    make(map[string]*NodeCache),
	}
}
