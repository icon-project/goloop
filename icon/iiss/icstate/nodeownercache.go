package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

var nodeOwnerDictPrefix = containerdb.ToKey(containerdb.RawBuilder, "node_owner")

// TODO: Remove old nodes which is not used anymore
type NodeOwnerCache struct {
	dict        *containerdb.DictDB
	nodeToOwner map[string]module.Address
	ownerToNode map[string]module.Address
}

func (c *NodeOwnerCache) Add(node, owner module.Address) error {
	// TODO: node must not be an owner of other PRep
	oldOwner := c.Get(node)
	if oldOwner != nil {
		return errors.Errorf("Node already exists: %s", node)
	}

	c.nodeToOwner[icutils.ToKey(node)] = owner
	c.ownerToNode[icutils.ToKey(owner)] = node
	return nil
}

func (c *NodeOwnerCache) Get(node module.Address) module.Address {
	key := icutils.ToKey(node)
	owner := c.nodeToOwner[key]
	if owner != nil {
		return owner
	}

	o := c.dict.Get(node)
	if o == nil {
		return nil
	}

	owner = o.Address()
	if owner != nil {
		c.nodeToOwner[key] = owner
	}

	return owner
}

func (c *NodeOwnerCache) Clear() {
	c.nodeToOwner = make(map[string]module.Address)
	c.ownerToNode = make(map[string]module.Address)
}

func (c *NodeOwnerCache) Reset() {
	c.Clear()
}

func (c *NodeOwnerCache) GetSnapshot() {
	for node, owner := range c.nodeToOwner {
		o := icobject.NewBytesObject(owner.Bytes())
		if err := c.dict.Set(node, o); err != nil {
			panic(errors.Errorf("DictDB.Set(%s, %s) is failed", node, owner))
		}
	}
}

func newNodeOwnerCache(store containerdb.ObjectStoreState) *NodeOwnerCache {
	return &NodeOwnerCache{
		dict:        containerdb.NewDictDB(store, 1, nodeOwnerDictPrefix),
		nodeToOwner: make(map[string]module.Address),
		ownerToNode: make(map[string]module.Address),
	}
}
