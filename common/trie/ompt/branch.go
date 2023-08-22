package ompt

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

type branch struct {
	nodeBase
	children [16]node
	value    trie.Object
}

func newBranch(h, s []byte, blist [][]byte, state nodeState) (node, error) {
	br := &branch{
		nodeBase: nodeBase{
			hashValue:  h,
			serialized: s,
			state:      state,
		},
	}
	for i, b := range blist {
		if i < 16 {
			child, err := nodeFromLink(b, state)
			if err != nil {
				return nil, err
			}
			br.children[i] = child
		} else {
			v, err := rlpParseBytes(b)
			if err != nil {
				return nil, err
			}
			if len(v) > 0 {
				br.value = bytesObject(v)
			}
		}
	}
	return br, nil
}

func (n *branch) toString() string {
	return fmt.Sprintf("B[%p](%v,%v,%v)", n, n.state, n.children, n.value)
}

func (n *branch) dump() {
	log.Println(n.toString())
	for _, child := range n.children {
		if child != nil {
			child.dump()
		}
	}
}

func (n *branch) getLink(fh bool) []byte {
	return n.nodeBase.getLink(n, fh)
}

func (n *branch) RLPListSize() int {
	return 17
}

func (n *branch) RLPListEncode(e RLPEncoder) error {
	for _, n := range n.children {
		if n == nil {
			e.RLPEncode(nil)
		} else {
			e.RLPWrite(n.getLink(false))
		}
	}
	if n.value != nil {
		e.RLPEncode(n.value.Bytes())
	} else {
		e.RLPEncode(nil)
	}
	return nil
}

func (n *branch) freeze() {
	lock := n.rlock()
	defer lock.Unlock()
	if n.state != stateDirty {
		return
	}
	for _, child := range n.children {
		if child != nil {
			child.freeze()
		}
	}
	lock.Migrate()
	if n.state == stateDirty {
		n.state = stateFrozen
	}
}

func (n *branch) flush(m *mpt, nibs []byte) error {
	lock := n.rlock()
	defer lock.Unlock()
	if n.state == stateFlushed {
		return nil
	}
	for i, child := range n.children {
		if child == nil {
			continue
		}
		if err := child.flush(m, append(nibs, byte(i))); err != nil {
			return err
		}
	}
	if n.value != nil {
		if err := n.value.Flush(); err != nil {
			return err
		}
	}
	if err := n.nodeBase.flushBaseInLock(m, nibs); err != nil {
		return err
	}
	lock.Migrate()
	n.state = stateFlushed
	return nil
}

func (n *branch) getChangable(lock *AutoRWUnlock) *branch {
	if n.state == stateDirty {
		lock.Migrate()
		return n
	}
	return &branch{children: n.children, value: n.value}
}

func (n *branch) set(m *mpt, nibs []byte, depth int, o trie.Object) (node, bool, trie.Object, error) {
	keys := nibs[depth:]
	lock := n.rlock()
	defer lock.Unlock()

	if len(keys) == 0 {
		old := n.value
		if n.value == nil || !n.value.Equal(o) {
			br := n.getChangable(&lock)
			br.value = o
			return br, true, old, nil
		}
		return n, false, old, nil
	}
	idx := keys[0]
	child := n.children[idx]
	nchild, dirty, old, err := m.set(child, nibs, depth+1, o)
	if dirty {
		br := n.getChangable(&lock)
		br.children[idx] = nchild
		return br, true, old, err
	}
	if child != nchild {
		lock.Migrate()
		n.children[idx] = nchild
	}
	return n, false, old, err
}

func (n *branch) delete(m *mpt, nibs []byte, depth int) (node, bool, trie.Object, error) {
	keys := nibs[depth:]
	lock := n.rlock()
	defer lock.Unlock()

	var br *branch
	var ov trie.Object
	if len(keys) == 0 {
		if n.value == nil {
			return n, false, nil, nil
		}
		ov = n.value
		br = n.getChangable(&lock)
		br.value = nil
	} else {
		idx := keys[0]
		child := n.children[idx]
		if child == nil {
			return n, false, nil, nil
		}
		nchild, dirty, old, err := child.delete(m, nibs, depth+1)
		if !dirty {
			if nchild != child {
				lock.Migrate()
				n.children[idx] = nchild
			}
			return n, false, nil, err
		}
		br = n.getChangable(&lock)
		br.children[idx] = nchild
		ov = old
	}

	var idx = 16
	for i, c := range br.children {
		if c != nil {
			if idx != 16 {
				idx = -1
				break
			}
			idx = i
		}
	}
	if idx != -1 {
		if idx == 16 {
			if br.value == nil {
				log.Panicln("Value is nil")
			}
			return &leaf{value: br.value}, true, ov, nil
		}
		if br.value == nil {
			alive := br.children[idx]
			alive, err := alive.realize(m)
			if err != nil {
				return n, false, nil, err
			}
			switch nn := alive.(type) {
			case *extension:
				return nn.getKeyPrepended([]byte{byte(idx)}), true, ov, nil
			case *branch:
				return &extension{
					keys: []byte{byte(idx)},
					next: alive,
				}, true, ov, nil
			case *leaf:
				return nn.getKeyPrepended([]byte{byte(idx)}), true, ov, nil
			}
		}
	}
	return br, true, ov, nil
}

func (n *branch) get(m *mpt, nibs []byte, depth int) (node, trie.Object, error) {
	keys := nibs[depth:]
	lock := n.rlock()
	defer lock.Unlock()

	if len(keys) == 0 {
		nv, changed, err := m.getObject(n.value)
		if changed {
			lock.Migrate()
			n.value = nv
		}
		return n, nv, err
	}

	idx := keys[0]
	child := n.children[idx]
	if child == nil {
		return n, nil, nil
	}
	nchild, o, err := child.get(m, nibs, depth+1)
	if nchild != child {
		lock.Migrate()
		n.children[idx] = nchild
	}
	return n, o, err
}

func (n *branch) realize(m *mpt) (node, error) {
	return n, nil
}

func (n *branch) traverse(m *mpt, k string, v nodeScheduler) (string, trie.Object, error) {
	lock := n.rlock()
	defer lock.Unlock()

	for i := 15; i >= 0; i-- {
		child := n.children[i]
		if child == nil {
			continue
		}
		nchild, err := v(k+string([]byte{byte(i)}), child)
		if err != nil {
			return "", nil, err
		}
		if child != nchild {
			lock.Migrate()
			n.children[i] = nchild
		}
	}
	if n.value != nil {
		value, changed, err := m.getObject(n.value)
		if changed {
			lock.Migrate()
			n.value = value
		}
		if err == nil {
			return k, n.value, nil
		}
	}
	return "", nil, nil
}

func (n *branch) getProof(m *mpt, keys []byte, proofs [][]byte) (nn node, proof [][]byte, err error) {
	lock := n.rlock()
	defer lock.Unlock()

	if n.state < stateHashed {
		return n, nil, fmt.Errorf("IllegaState %s", n.toString())
	}
	if n.hashValue != nil {
		proofs = append(proofs, n.serialized)
	}
	if len(keys) == 0 {
		return n, proofs, nil
	}
	child := n.children[keys[0]]
	if child == nil {
		return n, nil, nil
	}
	nchild, proofs, err := child.getProof(m, keys[1:], proofs)
	if nchild != child {
		lock.Migrate()
		n.children[keys[0]] = nchild
	}
	return n, proofs, err
}

func (n *branch) prove(m *mpt, keys []byte, proof [][]byte) (nn node, obj trie.Object, err error) {
	lock := n.rlock()
	defer func() {
		if err == nil && n.state == stateFlushed {
			lock.Migrate()
			n.state = stateWritten
		}
		defer lock.Unlock()
	}()

	if n.hashValue != nil {
		if len(proof) < 1 || !bytes.Equal(proof[0], n.serialized) {
			return n, nil, common.ErrIllegalArgument
		}
		proof = proof[1:]
	}

	if len(keys) == 0 {
		if n.value != nil {
			value, changed, err := m.getObject(n.value)
			if err != nil {
				return n, nil, err
			}
			if changed {
				lock.Migrate()
				n.value = value
			}
		}
		return n, n.value, nil
	}

	child := n.children[keys[0]]
	if child == nil {
		return n, nil, common.ErrNotFound
	}
	nchild, obj, err := child.prove(m, keys[1:], proof)
	if nchild != child {
		lock.Migrate()
		n.children[keys[0]] = nchild
	}
	return n, obj, err
}

func (n *branch) resolve(m *mpt, bd merkle.Builder) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for i := range n.children {
		m.resolve(bd, &n.children[i])
	}
	if n.value != nil {
		value, changed, err := m.getObject(n.value)
		if err != nil {
			return err
		}
		if changed {
			n.value = value
		}
		if err := n.value.Resolve(bd); err != nil {
			return err
		}
	}
	return nil
}

func (n *branch) compact() node {
	lock := n.rlock()
	defer lock.Unlock()

	if n.state < stateFlushed {
		lock.Migrate()
		for i := range n.children {
			node := n.children[i]
			if node != nil {
				n.children[i] = node.compact()
			}
		}
		if n.value != nil {
			n.value.ClearCache()
		}
		return n
	}
	if n.hashValue == nil {
		return n
	}
	return &hash{
		value: n.hashValue,
	}
}
