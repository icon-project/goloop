package icstate

import (
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

var termVarPrefix = containerdb.ToKey(containerdb.RawBuilder, "term")

type termCache struct {
	dirty bool
	term  *Term
	varDB *containerdb.VarDB
}

func (c *termCache) Get() *Term {
	return c.term
}

func (c *termCache) Set(term *Term) error {
	if c.term == term {
		return nil
	}
	c.term = term
	c.dirty = true
	return nil
}

func (c *termCache) Reset() error {
	if c.IsDirty() {
		c.term = nil
		c.dirty = false
	}
	return nil
}

func (c *termCache) Flush() error {
	if c.IsDirty() {
		c.dirty = false
		c.term.ResetFlag()

		o := icobject.New(TypeTerm, c.term)
		return c.varDB.Set(o)
	}
	return nil
}

func (c *termCache) IsDirty() bool {
	return c.dirty || (c.term != nil && c.term.IsUpdated())
}

func newTermCache(store containerdb.ObjectStoreState) *termCache {
	varDB := containerdb.NewVarDB(store, termVarPrefix)
	tp := GetTermPeriod(store)

	term := ToTerm(varDB.Object())
	if term == nil {
		term = newTerm(tp)
	}

	cache := &termCache{
		varDB: varDB,
		term:  term,
	}

	return cache
}
