package ompt

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
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
	if bo, ok := n.(bytesObject); n != nil && !ok {
		return false
	} else {
		return bytes.Equal(bo, o)
	}
}

func (o bytesObject) Resolve(builder merkle.Builder) error {
	return nil
}

func (o bytesObject) ClearCache() {
	// nothing to do, because it doesn't have belonging objects.
}
