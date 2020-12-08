package cache

import (
	"encoding/hex"
	"path"
	"sync"

	"github.com/icon-project/goloop/common/db"
)

const (
	nodeCacheManager    = "nodeCM"
	defaultAccountDepth = 5
)

type cacheManager struct {
	lock  sync.Mutex
	path  string
	depth [2]int
	world *NodeCache
	store map[string]*NodeCache
}

func (m *cacheManager) getWorldNodeCache() *NodeCache {
	return m.world
}

func (m *cacheManager) getAccountNodeCache(id []byte) *NodeCache {
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

func cacheManagerOf(database db.Database) *cacheManager {
	value := db.GetFlag(database, nodeCacheManager)
	if cm, ok := value.(*cacheManager); ok {
		return cm
	} else {
		return nil
	}
}

// WorldNodeCacheOf get node cache of the world if it has.
// If node cache for world state is not enabled, it returns nil.
func WorldNodeCacheOf(database db.Database) *NodeCache {
	if cm := cacheManagerOf(database); cm != nil {
		return cm.getWorldNodeCache()
	}
	return nil
}

// AccountNodeCacheOf get node cache of the account specified by *id*.
// If node cache for the account is not enabled, it returns nil.
func AccountNodeCacheOf(database db.Database, id []byte) *NodeCache {
	if cm := cacheManagerOf(database); cm != nil {
		return cm.getAccountNodeCache(id)
	} else {
		return nil
	}
}

// AttachManager attach cache manager to the database, and return it.
// dir is root directory for storing files for cache.
// mem is number of levels of tree items to store in the memory.
// file is number of levels of tree items to store in files.
func AttachManager(database db.Database, dir string, mem, file int) db.Database {
	return db.WithFlags(database, db.Flags{
		nodeCacheManager: &cacheManager{
			path:  dir,
			depth: [2]int{mem, file},
			world: NewNodeCache(defaultAccountDepth, 0, ""),
			store: make(map[string]*NodeCache),
		},
	})
}
