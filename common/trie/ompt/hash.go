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

func (h *hash) flush(m *mpt, nibs []byte) error {
	return nil
}

func (h *hash) serialize() []byte {
	log.Panicln("FAIL to serialize hash itself")
	return nil
}

func (h *hash) realize(m *mpt) (node, error) {
	return m.realize(h.value, nil)
}

func (h *hash) realizeWithCache(m *mpt, nibs []byte) (node, error) {
	return m.realize(h.value, nibs)
}

func (h *hash) get(m *mpt, nibs []byte, depth int) (node, trie.Object, error) {
	n, err := h.realizeWithCache(m, nibs[:depth])
	if err != nil || n == nil {
		return h, nil, err
	}
	return n.get(m, nibs, depth)
}

func (h *hash) set(m *mpt, nibs []byte, depth int, o trie.Object) (node, bool, trie.Object, error) {
	n, err := h.realizeWithCache(m, nibs[:depth])
	if err != nil || n == nil {
		return h, false, nil, err
	}
	return n.set(m, nibs, depth, o)
}

func (h *hash) delete(m *mpt, nibs []byte, depth int) (node, bool, trie.Object, error) {
	n, err := h.realizeWithCache(m, nibs[:depth])
	if err != nil || n == nil {
		return h, false, nil, err
	}
	return n.delete(m, nibs, depth)
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
	if len(items) < 1 {
		return h, nil, common.ErrIllegalArgument
	}
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

func (h *hash) compact() node {
	return h
}
