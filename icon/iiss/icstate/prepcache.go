package icstate

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
)

var (
	prepBaseDictPrefix = containerdb.ToKey(
		containerdb.HashBuilder,
		scoredb.DictDBPrefix,
		"prep_base",
	)
	prepStatusDictPrefix = containerdb.ToKey(
		containerdb.HashBuilder,
		scoredb.DictDBPrefix,
		"prep_status",
	)
)

type PRepBaseCache struct {
	bases map[string]*PRepBase
	dict  *containerdb.DictDB
}

func (c *PRepBaseCache) Get(owner module.Address, createIfNotExist bool) (*PRepBase, bool) {
	key := icutils.ToKey(owner)
	base := c.bases[key]
	if base != nil {
		return base, false
	}

	created := false
	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			base = NewPRepBase()
			c.bases[key] = base
			created = true
		} else {
			// return nil
		}
	} else {
		base = ToPRepBase(o.Object())
		if base != nil {
			c.bases[key] = base
		}
	}
	return base, created
}

func (c *PRepBaseCache) Clear() {
	c.bases = make(map[string]*PRepBase)
}

func (c *PRepBaseCache) Reset() {
	for key, base := range c.bases {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			panic(errors.Errorf("Address convert error"))
		}
		value := c.dict.Get(addr)

		if value == nil {
			delete(c.bases, key)
		} else {
			base.Set(ToPRepBase(value.Object()))
		}
	}
}

func (c *PRepBaseCache) Flush() {
	for k, base := range c.bases {
		key, err := common.BytesToAddress([]byte(k))
		if err != nil {
			panic(errors.Errorf("PRepBaseCache is broken: %s", k))
		}

		if base.IsEmpty() {
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
			delete(c.bases, k)
		} else {
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

// ====================================

type PRepStatusCache struct {
	statuses map[string]*PRepStatus
	dict     *containerdb.DictDB
}

func (c *PRepStatusCache) Get(owner module.Address, createIfNotExist bool) (*PRepStatus, bool) {
	key := icutils.ToKey(owner)
	status := c.statuses[key]
	if status != nil {
		return status, false
	}

	created := false
	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			status = NewPRepStatus()
			c.statuses[key] = status
			created = true
		} else {
			// return nil
		}
	} else {
		status = ToPRepStatus(o.Object())
		if status != nil {
			c.statuses[key] = status
		}
	}
	return status, created
}

func (c *PRepStatusCache) Clear() {
	c.statuses = make(map[string]*PRepStatus)
}

func (c *PRepStatusCache) Reset() {
	for key, status := range c.statuses {
		addr, err := common.NewAddress([]byte(key))
		if err != nil {
			panic(errors.Errorf("Address convert error"))
		}
		value := c.dict.Get(addr)

		if value == nil {
			delete(c.statuses, key)
		} else {
			status.Set(ToPRepStatus(value.Object()))
		}
	}
}

func (c *PRepStatusCache) Flush() {
	for k, status := range c.statuses {
		key, err := common.BytesToAddress([]byte(k))
		if err != nil {
			panic(errors.Errorf("PRepStatusCache is broken: %s", k))
		}

		if status.IsEmpty() {
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
			delete(c.statuses, k)
		} else {
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
