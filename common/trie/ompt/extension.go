package ompt

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

type extension struct {
	nodeBase
	keys []byte
	next node
}

func newExtension(h, s []byte, blist [][]byte, state nodeState) (node, error) {
	kb, err := rlpParseBytes(blist[0])
	if err != nil {
		return nil, err
	}
	node, err := nodeFromLink(blist[1], state)
	if err != nil {
		return nil, err
	}
	return &extension{
		nodeBase: nodeBase{
			hashValue:  h,
			serialized: s,
			state:      state,
		},
		keys: decodeKeys(kb),
		next: node,
	}, nil
}

func (n *extension) getLink(fh bool) []byte {
	return n.nodeBase.getLink(n, fh)
}

func (n *extension) toString() string {
	return fmt.Sprintf("E[%p](%v,[%x],%p)", n, n.state, n.keys, n.next)
}

func (n *extension) dump() {
	log.Println(n.toString())
	if n.next != nil {
		n.next.dump()
	}
}

func (n *extension) RLPListSize() int {
	return 2
}

func (n *extension) RLPListEncode(e RLPEncoder) error {
	if err := e.RLPEncode(encodeKeys(0x00, n.keys)); err != nil {
		return err
	}
	e.RLPWrite(n.next.getLink(false))
	return nil
}

func (n *extension) freeze() {
	lock := n.rlock()
	defer lock.Unlock()
	if n.state != stateDirty {
		return
	}
	if n.next != nil {
		n.next.freeze()
	}
	lock.Migrate()
	if n.state == stateDirty {
		n.state = stateFrozen
	}
}

func (n *extension) flush(m *mpt, nibs []byte) error {
	lock := n.rlock()
	defer lock.Unlock()
	if n.state == stateFlushed {
		return nil
	}
	if err := n.next.flush(m, append(nibs, n.keys...)); err != nil {
		return err
	}
	if err := n.nodeBase.flushBaseInLock(m, nil); err != nil {
		return err
	}
	lock.Migrate()
	n.state = stateFlushed
	return nil
}

func (n *extension) getChanged(lock *AutoRWUnlock, keys []byte, next node) *extension {
	if n.state == stateDirty {
		lock.Migrate()
		n.keys = keys
		n.next = next
		return n
	}
	return &extension{keys: keys, next: next}
}

func (n *extension) set(m *mpt, nibs []byte, depth int, o trie.Object) (node, bool, trie.Object, error) {
	lock := n.rlock()
	defer lock.Unlock()

	keys := nibs[depth:]
	cnt, _ := compareKeys(keys, n.keys)

	switch {
	case cnt == 0:
		nb := &branch{}
		if len(keys) == 0 {
			nb.value = o
		} else {
			nb.children[keys[0]] = &leaf{keys: clone(keys[1:]), value: o}
		}
		if len(n.keys) == 1 {
			nb.children[n.keys[0]] = n.next
		} else {
			idx := n.keys[0]
			nb.children[idx] = n.getChanged(&lock, n.keys[1:], n.next)
		}
		return nb, true, nil, nil
	case cnt < len(n.keys):
		br := &branch{}
		idx := n.keys[cnt]
		if cnt+1 == len(n.keys) {
			br.children[idx] = n.next
		} else {
			br.children[idx] = &extension{keys: n.keys[cnt+1:], next: n.next}
		}
		if cnt == len(keys) {
			br.value = o
		} else {
			br.children[keys[cnt]] = &leaf{keys: clone(keys[cnt+1:]), value: o}
		}
		return n.getChanged(&lock, n.keys[:cnt], br), true, nil, nil
	default:
		next, dirty, old, err := n.next.set(m, nibs, depth+cnt, o)
		if dirty {
			return n.getChanged(&lock, n.keys, next), true, old, err
		} else {
			if n.next != next {
				lock.Migrate()
				n.next = next
			}
		}
		return n, false, old, err
	}
}

func (n *extension) getKeyPrepended(k []byte) *extension {
	lock := n.rlock()
	defer lock.Unlock()

	nk := make([]byte, len(k)+len(n.keys))
	copy(nk, k)
	copy(nk[len(k):], n.keys)
	return n.getChanged(&lock, nk, n.next)
}

func (n *extension) delete(m *mpt, nibs []byte, depth int) (node, bool, trie.Object, error) {
	keys := nibs[depth:]

	lock := n.rlock()
	defer lock.Unlock()

	cnt, _ := compareKeys(keys, n.keys)
	if cnt < len(n.keys) {
		return n, false, nil, nil
	}
	next, dirty, old, err := n.next.delete(m, nibs, depth+cnt)
	if dirty {
		if next == nil {
			return nil, true, old, err
		}
		switch nn := next.(type) {
		case *extension:
			return nn.getKeyPrepended(n.keys), true, old, err
		case *leaf:
			return nn.getKeyPrepended(n.keys), true, old, err
		}
		return n.getChanged(&lock, n.keys, next), true, old, err
	} else {
		if n.next != next {
			lock.Migrate()
			n.next = next
		}
	}
	return n, false, nil, nil
}

func (n *extension) get(m *mpt, nibs []byte, depth int) (node, trie.Object, error) {
	keys := nibs[depth:]
	lock := n.rlock()
	defer lock.Unlock()
	cnt, _ := compareKeys(keys, n.keys)
	if cnt < len(n.keys) {
		return n, nil, nil
	}
	nv, obj, err := n.next.get(m, nibs, depth+cnt)
	if nv != n.next {
		lock.Migrate()
		n.next = nv
	}
	return n, obj, err
}

func (n *extension) realize(m *mpt) (node, error) {
	return n, nil
}

func (n *extension) traverse(m *mpt, k string, v nodeScheduler) (string, trie.Object, error) {
	lock := n.rlock()
	defer lock.Unlock()

	next, err := v(k+string(n.keys), n.next)
	if err != nil {
		return "", nil, err
	}
	if next != n.next {
		lock.Migrate()
		n.next = next
	}
	return "", nil, nil
}

func (n *extension) getProof(m *mpt, keys []byte, proofs [][]byte) (node, [][]byte, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.state < stateHashed {
		return n, nil, fmt.Errorf("IllegaState %s", n.toString())
	}

	cnt, _ := compareKeys(n.keys, keys)
	if cnt < len(n.keys) {
		return n, nil, nil
	}
	if n.hashValue != nil {
		proofs = append(proofs, n.serialized)
	}
	next, proofs, err := n.next.getProof(m, keys[cnt:], proofs)
	if next != n.next {
		n.next = next
	}
	return n, proofs, err
}

func (n *extension) prove(m *mpt, keys []byte, proof [][]byte) (nn node, obj trie.Object, err error) {
	n.mutex.Lock()
	defer func() {
		if err == nil && n.state == stateFlushed {
			n.state = stateWritten
		}
		n.mutex.Unlock()
	}()

	if n.hashValue != nil {
		if len(proof) < 1 || !bytes.Equal(proof[0], n.serialized) {
			return n, nil, common.ErrIllegalArgument
		}
		proof = proof[1:]
	}

	cnt, _ := compareKeys(n.keys, keys)
	if cnt < len(n.keys) {
		return n, nil, common.ErrNotFound
	}
	next, obj, err := n.next.prove(m, keys[cnt:], proof)
	if next != n.next {
		n.next = next
	}
	return n, obj, err
}

func (n *extension) resolve(m *mpt, bd merkle.Builder) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	m.resolve(bd, &n.next)
	return nil
}

func (n *extension) compact() node {
	lock :=n.rlock()
	defer lock.Unlock()

	if n.state < stateFlushed {
		lock.Migrate()
		n.next = n.next.compact()
		return n
	}
	if n.hashValue == nil {
		return n
	}
	return &hash{
		value: n.hashValue,
	}
}
