package icstate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestAllPrepCache(t *testing.T) {
	var err error
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	tree := trie_manager.NewMutableForObject(database, nil, icobject.ObjectType)
	oss := icobject.NewObjectStoreState(tree)
	activePRepCache := NewAllPRepCache(oss)

	size := 5
	for i := 0; i < size; i ++ {
		addr := newDummyAddress(i)
		err = activePRepCache.Add(addr)
		assert.NoError(t, err)
		assert.Equal(t, i + 1, activePRepCache.Size())
	}

	for i := 0; i < size; i++ {
		addr := activePRepCache.Get(i)
		assert.True(t, addr.Equal(newDummyAddress(i)))
	}
}
