// +build ompt

package trie_manager

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/ompt"
)

func New(database db.Database) trie.Manager {
	return ompt.NewManager(database)
}
