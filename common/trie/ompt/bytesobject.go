package ompt

import (
	"bytes"
	"fmt"
	"log"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type bytesObject []byte

func (o bytesObject) Bytes() []byte {
	return o
}

func (o bytesObject) Reset(db db.Database, k []byte) error {
	log.Panicln("Bytes object can't RESET!!")
	return nil
}

func (o bytesObject) Flush() error {
	// Nothing to do because it comes from database itself.
	return nil
}

func (o bytesObject) String() string {
	return fmt.Sprintf("[%x]", []byte(o))
}

func (o bytesObject) Equal(n trie.Object) bool {
	if bo, ok := n.(bytesObject); ok {
		return bytes.Equal(o, bo)
	}
	return false
}
