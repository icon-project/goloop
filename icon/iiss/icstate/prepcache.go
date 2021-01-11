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

func (c *PRepBaseCache) Add(base *PRepBase) {
	key := icutils.ToKey(base.owner)
	c.bases[key] = base
}

func (c *PRepBaseCache) Remove(owner module.Address) error {
	pb := c.Get(owner)
	if pb == nil {
		return errors.Errorf("PRepBase not found: %s", owner)
	}

	pb.Clear()
	return nil
}

func (c *PRepBaseCache) Get(owner module.Address) *PRepBase {
	key := icutils.ToKey(owner)
	base := c.bases[key]
	if base != nil {
		return base
	}

	o := c.dict.Get(owner)
	if o == nil {
		return nil
	}

	base = ToPRepBase(o.Object(), owner)
	if base != nil {
		c.bases[key] = base
	}

	return base
}

func (c *PRepBaseCache) Clear() {
	c.bases = make(map[string]*PRepBase)
}

func (c *PRepBaseCache) Reset() {
	for _, base := range c.bases {
		value := c.dict.Get(base.owner)

		if value == nil {
			base.Clear()
		} else {
			base.Set(ToPRepBase(value.Object(), base.owner))
		}
	}
}

func (c *PRepBaseCache) GetSnapshot() {
	for k, base := range c.bases {
		if base.IsEmpty() {
			key, err := common.BytesToAddress([]byte(k))
			if err != nil {
				panic(errors.Errorf("PRepBaseCache is broken: %s", k))
			}
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
		} else {
			key := base.owner
			o := icobject.New(TypePRepBase, base)
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

func (c *PRepStatusCache) Add(status *PRepStatus) {
	key := icutils.ToKey(status.owner)
	c.statuses[key] = status
}

func (c *PRepStatusCache) Remove(owner module.Address) error {
	ps := c.Get(owner)
	if ps == nil {
		return errors.Errorf("PRepStatus not found: %s", owner)
	}

	ps.Clear()
	return nil
}

func (c *PRepStatusCache) Get(owner module.Address) *PRepStatus {
	key := icutils.ToKey(owner)
	status := c.statuses[key]
	if status != nil {
		return status
	}

	o := c.dict.Get(owner)
	if o == nil {
		return nil
	}

	status = ToPRepStatus(o.Object(), owner)
	if status != nil {
		c.statuses[key] = status
	}

	return status
}

func (c *PRepStatusCache) Clear() {
	c.statuses = make(map[string]*PRepStatus)
}

func (c *PRepStatusCache) Reset() {
	for _, status := range c.statuses {
		value := c.dict.Get(status.owner)

		if value == nil {
			status.Clear()
		} else {
			status.Set(ToPRepStatus(value.Object(), status.owner))
		}
	}
}

func (c *PRepStatusCache) GetSnapshot() {
	for k, status := range c.statuses {
		if status.IsEmpty() {
			key, err := common.BytesToAddress([]byte(k))
			if err != nil {
				panic(errors.Errorf("PRepStatusCache is broken: %s", k))
			}
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
		} else {
			key := status.owner
			o := icobject.New(TypePRepStatus, status)
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
