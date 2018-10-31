package ompt

import (
	"bytes"
	"log"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type BytesObject []byte

func (o BytesObject) Bytes() []byte {
	return o
}

func (o BytesObject) Reset(db db.Database, k []byte) error {
	log.Panicln("Bytes object can't RESET!!")
	return nil
}

func (o BytesObject) Flush() error {
	// Nothing to do because it comes from database itself.
	return nil
}

func (o BytesObject) Equal(n trie.Object) bool {
	if bo, ok := n.(BytesObject); ok {
		return bytes.Equal(o, bo)
	}
	return false
}
