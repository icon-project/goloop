package ompt

import (
	"fmt"
	"log"

	"github.com/icon-project/goloop/common/trie"
)

type hash []byte

func (h hash) getLink(fh bool) []byte {
	if fh {
		return h
	}
	return rlpEncodeBytes(h)
}

func (h hash) freeze() {
	return
}

func (h hash) toString() string {
	return fmt.Sprintf("H[%p](0x%[1]x)", []byte(h))
}

func (h hash) dump() {
	log.Println(h.toString())
}

func (h hash) flush(m *mpt) error {
	return nil
}

func (h hash) serialize() []byte {
	log.Panicln("FAIL to serialize hash itself")
	return nil
}

func (h hash) realize(m *mpt) (node, error) {
	serialized, err := m.bucket.Get([]byte(h))
	if err != nil {
		return nil, err
	}
	if serialized == nil {
		return nil, nil
	}
	return deserialize([]byte(h), serialized)
}

func (h hash) get(m *mpt, keys []byte) (node, trie.Object, error) {
	n, err := h.realize(m)
	if err != nil || n == nil {
		return h, nil, err
	}
	return n.get(m, keys)
}

func (h hash) set(m *mpt, keys []byte, o trie.Object) (node, bool, error) {
	n, err := h.realize(m)
	if err != nil {
		return nil, false, err
	}
	return m.set(n, keys, o)
}

func (h hash) delete(m *mpt, keys []byte) (node, bool, error) {
	log.Panicln("hash class doesn't implement set")
	return nil, false, nil
}
