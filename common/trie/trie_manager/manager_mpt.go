// +build !ompt

package trie_manager

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
)

func New(database db.Database) trie.Manager {
	return mpt.NewManager(database)
}
