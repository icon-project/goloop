package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

var (
	prepBaseDictPrefix   = containerdb.ToKey(containerdb.RawBuilder, "prep_base")
	prepStatusDictPrefix = containerdb.ToKey(containerdb.RawBuilder, "prep_status")
)

type PRepBaseCache struct {
	bases map[string]*PRepBase
	dict  *containerdb.DictDB
}

func (c *PRepBaseCache) Get(owner module.Address, createIfNotExist bool) *PRepBase {
	key := icutils.ToKey(owner)
	base := c.bases[key]
	if base != nil {
		return base
	}
	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			base = NewPRepBase(owner)
			c.bases[key] = base
		} else {
			// return nil
		}
	} else {
		base = ToPRepBase(o.Object(), owner)
		if base != nil {
			c.bases[key] = base
		}
	}
	return base
}

func (c *PRepBaseCache) Clear() {
	c.bases = make(map[string]*PRepBase)
}

func (c *PRepBaseCache) Reset() {
	for key , base := range c.bases {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			panic(errors.Errorf("Address convert error"))
		}
		value := c.dict.Get(addr)

		if value == nil {
			delete(c.bases, key)
		} else {
			base.Set(ToPRepBase(value.Object(), addr))
		}
	}
}

func (c *PRepBaseCache) Flush() {
	for k, base := range c.bases {
		if base.IsEmpty() {
			key, err := common.BytesToAddress([]byte(k))
			if err != nil {
				panic(errors.Errorf("PRepBaseCache is broken: %s", k))
			}
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
			delete(c.bases, k)
		} else {
			key := base.owner
			o := icobject.New(TypePRepBase, base.Clone())
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set snapshotMap for %x, err+%+v", key, err)
			}
		}
	}
}

func newPRepBaseCache(store containerdb.ObjectStoreState) *PRepBaseCache {
	return &PRepBaseCache{
		bases: make(map[string]*PRepBase),
		dict:  containerdb.NewDictDB(store, 1, prepBaseDictPrefix),
	}
}

type PRepStatusCache struct {
	statuses map[string]*PRepStatus
	dict     *containerdb.DictDB
}

func (c *PRepStatusCache) Get(owner module.Address, createIfNotExist bool) *PRepStatus {
	key := icutils.ToKey(owner)
	status := c.statuses[key]
	if status != nil {
		return status
	}
	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			status = NewPRepStatus(owner)
			c.statuses[key] = status
		} else {
			// return nil
		}
	} else {
		status = ToPRepStatus(o.Object(), owner)
		if status != nil {
			c.statuses[key] = status
		}
	}
	return status
}

func (c *PRepStatusCache) Clear() {
	c.statuses = make(map[string]*PRepStatus)
}

func (c *PRepStatusCache) Reset() {
	for key , status := range c.statuses {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			panic(errors.Errorf("Address convert error"))
		}
		value := c.dict.Get(addr)

		if value == nil {
			delete(c.statuses, key)
		} else {
			status.Set(ToPRepStatus(value.Object(), addr))
		}
	}
}

func (c *PRepStatusCache) Flush() {
	for k, status := range c.statuses {
		if status.IsEmpty() {
			key, err := common.BytesToAddress([]byte(k))
			if err != nil {
				panic(errors.Errorf("PRepStatusCache is broken: %s", k))
			}
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
			delete(c.statuses, k)
		} else {
			key := status.owner
			o := icobject.New(TypePRepStatus, status.Clone())
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set snapshotMap for %x, err+%+v", key, err)
			}
		}
	}
}

func newPRepStatusCache(store containerdb.ObjectStoreState) *PRepStatusCache {
	return &PRepStatusCache{
		statuses: make(map[string]*PRepStatus),
		dict:     containerdb.NewDictDB(store, 1, prepStatusDictPrefix),
	}
}
