package ompt

import (
	"bytes"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type byteObject []byte

func (o byteObject) Bytes() []byte {
	return o
}

func (o byteObject) Flush() error {
	return nil
}

func (o byteObject) Reset(s db.Store, k []byte) error {
	o = []byte(string(k))
	return nil
}

func (o byteObject) Equal(o2 trie.Object) bool {
	return bytes.Equal(o, o2.Bytes())
}
