package ompt

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

type hash struct {
	value []byte
}

func (h *hash) getLink(fh bool) []byte {
	if fh {
		return h.value
	}
	return rlpEncodeBytes(h.value)
}

func (h *hash) hash() []byte {
	return []byte(h.value)
}

func (h *hash) freeze() {
	return
}

func (h *hash) toString() string {
	return fmt.Sprintf("H[%p](0x%[1]x)", []byte(h.value))
}

func (h *hash) dump() {
	log.Println(h.toString())
}

func (h *hash) flush(m *mpt) error {
	return nil
}

func (h *hash) serialize() []byte {
	log.Panicln("FAIL to serialize hash itself")
	return nil
}

func (h *hash) realize(m *mpt) (node, error) {
	serialized, err := m.bucket.Get(h.value)
	if err != nil {
		return nil, err
	}
	if serialized == nil {
		return nil, fmt.Errorf("ErrorKeyNotFound(key=%x)", h.value)
	}
	return deserialize(h.value, serialized, stateFlushed)
}

func (h *hash) get(m *mpt, keys []byte) (node, trie.Object, error) {
	n, err := h.realize(m)
	if err != nil || n == nil {
		return h, nil, err
	}
	return n.get(m, keys)
}

func (h *hash) set(m *mpt, keys []byte, o trie.Object) (node, bool, error) {
	n, err := h.realize(m)
	if err != nil || n == nil {
		return h, false, err
	}
	return n.set(m, keys, o)
}

func (h *hash) delete(m *mpt, keys []byte) (node, bool, error) {
	n, err := h.realize(m)
	if err != nil || n == nil {
		return h, false, err
	}
	return n.delete(m, keys)
}

func (h *hash) traverse(m *mpt, k string, v nodeScheduler) (string, trie.Object, error) {
	n, err := h.realize(m)
	if err != nil {
		return "", nil, err
	}
	return n.traverse(m, k, v)
}

func (h *hash) getProof(m *mpt, keys []byte, proofs [][]byte) (node, [][]byte, error) {
	n, err := h.realize(m)
	if err != nil {
		return h, nil, err
	}
	return n.getProof(m, keys, proofs)
}

func (h *hash) prove(m *mpt, kb []byte, items [][]byte) (node, trie.Object, error) {
	b := items[0]
	h2 := calcHash(b)
	if !bytes.Equal(h.value, h2) {
		return h, nil, common.ErrIllegalArgument
	}
	n, err := deserialize(h2, b, stateHashed)
	if err != nil {
		return h, nil, err
	}
	return n.prove(m, kb, items)
}

func (h *hash) resolve(m *mpt, bd merkle.Builder) error {
	panic("It should not be called.")
	return nil
}
