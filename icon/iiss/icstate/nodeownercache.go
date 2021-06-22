package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var nodeOwnerDictPrefix = containerdb.ToKey(
	containerdb.HashBuilder,
	scoredb.DictDBPrefix,
	"node_owner",
)

// TODO: Remove old nodes which is not used anymore
type NodeOwnerCache struct {
	dict        *containerdb.DictDB
	nodeToOwner map[string]module.Address
}

// Add adds alias, node-to-owner.
// It doesn't record alias to self.
// If there is same alias, it ignores the request.
func (c *NodeOwnerCache) Add(node, owner module.Address) error {
	if node == nil || owner == nil {
		// No need to add
		return nil
	}
	if cur := c.get(node, nil); cur == nil {
		if node.Equal(owner) {
			return nil
		}
	} else {
		if cur.Equal(owner) {
			return nil
		} else {
			return errors.Errorf("AlreadyExist(owner=%s,node=%s)", cur, node)
		}
	}
	c.nodeToOwner[icutils.ToKey(node)] = owner
	return nil
}

func (c *NodeOwnerCache) Get(node module.Address) module.Address {
	return c.get(node, node)
}

func (c *NodeOwnerCache) get(node module.Address, fallback module.Address) module.Address {
	key := icutils.ToKey(node)
	owner := c.nodeToOwner[key]
	if owner != nil {
		return owner
	}

	o := c.dict.Get(node)
	if o == nil {
		// owner address is equal to node address
		return fallback
	}
	return o.Address()
}

func (c *NodeOwnerCache) Contains(node module.Address) bool {
	return c.get(node, nil) != nil
}

func (c *NodeOwnerCache) Clear() {
	c.Flush()
}

func (c *NodeOwnerCache) Reset() {
	c.nodeToOwner = make(map[string]module.Address)
}

func (c *NodeOwnerCache) Flush() {
	for node, owner := range c.nodeToOwner {
		if err := c.dict.Set(node, owner); err != nil {
			panic(errors.Errorf("DictDB.Set(%s, %s) is failed", node, owner))
		}
	}
	c.nodeToOwner = make(map[string]module.Address)
}

func newNodeOwnerCache(store containerdb.ObjectStoreState) *NodeOwnerCache {
	return &NodeOwnerCache{
		dict:        containerdb.NewDictDB(store, 1, nodeOwnerDictPrefix),
		nodeToOwner: make(map[string]module.Address),
	}
}
