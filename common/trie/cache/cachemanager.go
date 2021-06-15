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
	defaultStoreDepth   = 5
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
	sid := string(id)
	if c, ok := m.store[sid]; ok {
		return c
	} else {
		if m.depth[0] == 0 {
			return nil
		}
		m.store[sid] = m.newAccountNodeCache(id, m.depth[0], m.depth[1])
		return c
	}
}

func (m *cacheManager) newAccountNodeCache(id []byte, mem, file int) *NodeCache {
	path := path.Join(m.path, hex.EncodeToString(id))
	return NewNodeCache(mem, file, path)
}

func (m *cacheManager) enableAccountNodeCache(id []byte, mem, file int) {
	sid := string(id)
	if _, ok := m.store[sid]; !ok {
		m.store[sid] = m.newAccountNodeCache(id, mem, file)
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

// EnableAccountNodeCacheByForce enable AccountNodeCache ignoring default setting.
// Default setting for account node cache is specified by call in AttachManager.
func EnableAccountNodeCacheByForce(database db.Database, id []byte) bool {
	if cm := cacheManagerOf(database); cm != nil {
		cm.enableAccountNodeCache(id, defaultStoreDepth, 0)
		return true
	} else {
		return false
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
