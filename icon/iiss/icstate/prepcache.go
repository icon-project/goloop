package icstate

import (
	"math/big"

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
	bases map[string]*PRepBaseState
	dict  *containerdb.DictDB
}

func (c *PRepBaseCache) Get(owner module.Address, createIfNotExist bool) *PRepBaseState {
	key := icutils.ToKey(owner)
	base := c.bases[key]
	if base != nil {
		return base
	}

	o := c.dict.Get(owner)
	if o == nil {
		if createIfNotExist {
			base = NewPRepBaseState()
			c.bases[key] = base
		} else {
			// return nil
		}
	} else {
		ps := ToPRepBase(o.Object())
		if ps != nil {
			base = new(PRepBaseState).Reset(ps)
			c.bases[key] = base
		}
	}
	return base
}

func (c *PRepBaseCache) Clear() {
	c.Flush()
	c.bases = make(map[string]*PRepBaseState)
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
			base.Reset(ToPRepBase(value.Object()))
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
			o := icobject.New(TypePRepBase, base.GetSnapshot())
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set snapshotMap for %x, err+%+v", key, err)
			}
		}
	}
}

func newPRepBaseCache(store containerdb.ObjectStoreState) *PRepBaseCache {
	return &PRepBaseCache{
		bases: make(map[string]*PRepBaseState),
		dict:  containerdb.NewDictDB(store, 1, prepBaseDictPrefix),
	}
}

// ====================================

type PRepStatusCache struct {
	statuses map[string]*PRepStatusState
	lasts    map[string]*PRepStatusSnapshot
	dict     *containerdb.DictDB
	illegal  *containerdb.DictDB
}

func (c *PRepStatusCache) Get(owner module.Address, createIfNotExist bool) *PRepStatusState {
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
		snapshot := ToPRepStatus(o.Object())
		if snapshot != nil {
			status = NewPRepStatusWithSnapshot(owner, snapshot)
			if value := c.illegal.Get(owner); value != nil {
				status.SetEffectiveDelegated(new(big.Int).Add(status.Delegated(), value.BigInt()))
			}
			c.statuses[key] = status
			c.lasts[key] = snapshot
		}
	}
	return status
}

func (c *PRepStatusCache) Clear() {
	c.Flush()
	c.statuses = make(map[string]*PRepStatusState)
	c.lasts = make(map[string]*PRepStatusSnapshot)
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
			delete(c.lasts, key)
		} else {
			snapshot := ToPRepStatus(value.Object())
			status.Reset(snapshot)
			c.lasts[key] = snapshot
		}
	}
}

func (c *PRepStatusCache) Flush() {
	for k, status := range c.statuses {
		key, err := common.BytesToAddress([]byte(k))
		if err != nil {
			panic(errors.Errorf("PRepStatusCache is broken: %s", k))
		}

		snapshot := status.GetSnapshot()
		if last := c.lasts[k]; last == snapshot {
			continue
		} else if last == nil && snapshot.IsEmpty() {
			delete(c.statuses, k)
			continue
		}

		if snapshot.IsEmpty() {
			if err = c.dict.Delete(key); err != nil {
				log.Errorf("Failed to delete PRep key %x, err+%+v", key, err)
			}
			delete(c.statuses, k)
			delete(c.lasts, k)
		} else {
			o := icobject.New(TypePRepStatus, snapshot)
			if err := c.dict.Set(key, o); err != nil {
				log.Errorf("Failed to set snapshotMap for %x, err+%+v", key, err)
			}
			c.lasts[k] = snapshot
		}
	}
}

func newPRepStatusCache(store containerdb.ObjectStoreState) *PRepStatusCache {
	return &PRepStatusCache{
		statuses: make(map[string]*PRepStatusState),
		lasts:    make(map[string]*PRepStatusSnapshot),
		dict:     containerdb.NewDictDB(store, 1, prepStatusDictPrefix),
		illegal:  containerdb.NewDictDB(store, 1, pRepIllegalDelegatedKey),
	}
}
