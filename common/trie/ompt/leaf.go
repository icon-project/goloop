package ompt

import (
	"bytes"
	"fmt"
	"github.com/icon-project/goloop/common"
	"log"

	"github.com/icon-project/goloop/common/trie"
)

type leaf struct {
	nodeBase
	keys  []byte
	value trie.Object
}

func newLeaf(hash, serialized []byte, blist [][]byte, state nodeState) (node, error) {
	kbytes, err := rlpParseBytes(blist[0])
	if err != nil {
		return nil, err
	}
	keys := decodeKeys(kbytes)

	vbytes, err := rlpParseBytes(blist[1])
	if err != nil {
		return nil, err
	}
	value := bytesObject(vbytes)

	return &leaf{
		nodeBase: nodeBase{
			hashValue:  hash,
			serialized: serialized,
			state:      state,
		},
		keys:  keys,
		value: value,
	}, nil
}

func (n *leaf) getLink(fh bool) []byte {
	return n.nodeBase.getLink(n, fh)
}

func (n *leaf) toString() string {
	return fmt.Sprintf("L[%p](%v,[%x],%v)", n, n.state, n.keys, n.value)
}

func (n *leaf) dump() {
	log.Println(n.toString())
}

func (n *leaf) freeze() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state != stateDirty {
		return
	}
	n.state = stateFrozen
}

func (n *leaf) flush(m *mpt) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state == stateFlushed {
		return nil
	}
	if n.value == nil {
		return nil
	}
	if err := n.value.Flush(); err != nil {
		return err
	}
	if err := n.nodeBase.flushBaseInLock(m); err != nil {
		return err
	}
	return nil
}

func (n *leaf) RLPListSize() int {
	return 2
}

func (n *leaf) RLPListEncode(e RLPEncoder) error {
	e.RLPEncode(encodeKeys(0x20, n.keys))
	e.RLPEncode(n.value.Bytes())
	return nil
}

func (n *leaf) getChanged(keys []byte, o trie.Object) *leaf {
	if n.state == stateDirty {
		n.keys = keys
		n.value = o
		return n
	}
	return &leaf{keys: keys, value: o}
}

func (n *leaf) set(m *mpt, keys []byte, o trie.Object) (node, bool, error) {
	cnt, match := compareKeys(keys, n.keys)
	// If it matches, no need to break leaf nodes.
	// Buf if
	switch {
	case cnt == 0 && !match:
		br := &branch{}
		if len(keys) == 0 {
			br.value = o
		} else {
			br.children[keys[0]] = &leaf{
				keys:  keys[1:],
				value: o,
			}
		}
		if len(n.keys) == 0 {
			br.value = n.value
		} else {
			idx := n.keys[0]
			br.children[idx] = n.getChanged(n.keys[1:], n.value)
		}
		return br, true, nil
	case cnt < len(n.keys):
		br := &branch{}
		ext := &extension{keys: keys[:cnt], next: br}
		if cnt == len(keys) {
			br.value = o
		} else {
			br.children[keys[cnt]] = &leaf{keys: keys[cnt+1:], value: o}
		}
		idx := n.keys[cnt]
		br.children[idx] = n.getChanged(n.keys[cnt+1:], n.value)
		return ext, true, nil
	case cnt < len(keys):
		br := &branch{}
		ext := &extension{keys: n.keys, next: br}
		br.value = n.value
		br.children[keys[cnt]] = &leaf{keys: keys[cnt+1:], value: o}
		return ext, true, nil
	default:
		if n.value.Equal(o) {
			return n, false, nil
		}
		return n.getChanged(n.keys, o), true, nil
	}
}

func (n *leaf) getKeyPrepended(k []byte) *leaf {
	nk := make([]byte, len(k)+len(n.keys))
	copy(nk, k)
	copy(nk[len(k):], n.keys)
	return n.getChanged(nk, n.value)
}

func (n *leaf) delete(m *mpt, keys []byte) (node, bool, error) {
	_, match := compareKeys(keys, n.keys)
	if match {
		return nil, true, nil
	}
	return n, false, nil
}

func (n *leaf) get(m *mpt, keys []byte) (node, trie.Object, error) {
	_, match := compareKeys(keys, n.keys)
	if !match {
		return n, nil, nil
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	nv, changed, err := m.getObject(n.value)
	if changed {
		n.value = nv
	}
	return n, nv, err
}

func (n *leaf) realize(m *mpt) (node, error) {
	return n, nil
}

func (n *leaf) traverse(m *mpt, v nodeScheduler) (trie.Object, error) {
	return n.value, nil
}

func (n *leaf) getProof(m *mpt, keys []byte, items [][]byte) (node, [][]byte, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.state < stateHashed {
		return n, nil, fmt.Errorf("IllegaState %s", n.toString())
	}
	if _, match := compareKeys(n.keys, keys); !match {
		return n, nil, nil
	}
	if n.hashValue != nil {
		items = append(items, n.serialized)
	}
	return n, items, nil
}

func (n *leaf) prove(m *mpt, keys []byte, proof [][]byte) (node, trie.Object, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.hashValue != nil {
		if !bytes.Equal(proof[0], n.serialized) {
			return n, nil, common.ErrIllegalArgument
		}
		proof = proof[1:]
	}

	_, match := compareKeys(n.keys, keys)
	if match {
		value, changed, err := m.getObject(n.value)
		if err != nil {
			return n, nil, err
		}
		if changed {
			n.value = value
		}
		return n, n.value, nil
	}
	return n, nil, common.ErrNotFound

}
