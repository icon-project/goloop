package ompt

import (
	"fmt"
	"log"

	"github.com/icon-project/goloop/common/trie"
)

type extension struct {
	nodeBase
	keys []byte
	next node
}

func newExtension(h, s []byte, blist [][]byte) (node, error) {
	kb, err := rlpParseBytes(blist[0])
	if err != nil {
		return nil, err
	}
	node, err := nodeFromLink(blist[1])
	if err != nil {
		return nil, err
	}
	return &extension{
		nodeBase: nodeBase{
			hashValue:  h,
			serialized: s,
		},
		keys: decodeKeys(kb),
		next: node,
	}, nil
}

func (n *extension) getLink(fh bool) []byte {
	return n.nodeBase.getLink(n, fh)
}

func (n *extension) toString() string {
	return fmt.Sprintf("EXTN[%p](%x,%p)", n, n.keys, n.next)
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
	e.RLPEncode(encodeKeys(0x00, n.keys))
	e.RLPWrite(n.next.getLink(false))
	return nil
}

func (n *extension) freeze() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state != stateDirty {
		return
	}
	if n.next != nil {
		n.next.freeze()
	}
	n.state = stateFreezed
}

func (n *extension) flush(m *mpt) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state == stateFlushed {
		return nil
	}
	if err := n.next.flush(m); err != nil {
		return err
	}
	if err := n.nodeBase.flushBaseInLock(m); err != nil {
		return err
	}
	return nil
}

func (n *extension) getChanged(keys []byte, next node) *extension {
	if n.state == stateDirty {
		n.keys = keys
		n.next = next
		return n
	}
	return &extension{keys: keys, next: next}
}

func (n *extension) set(m *mpt, keys []byte, o trie.Object) (node, bool, error) {
	cnt, _ := compareKeys(keys, n.keys)
	switch {
	case cnt == 0:
		nb := &branch{}
		if len(keys) == 0 {
			nb.value = o
		} else {
			nb.children[keys[0]] = &leaf{keys: keys[1:], value: o}
		}
		if len(n.keys) == 1 {
			nb.children[n.keys[0]] = n.next
		} else {
			idx := n.keys[0]
			nb.children[idx] = n.getChanged(n.keys[1:], n.next)
		}
		return nb, true, nil
	case cnt < len(n.keys):
		br := &branch{}
		br.children[keys[cnt]] = &leaf{keys: keys[cnt+1:], value: o}
		if cnt+1 == len(n.keys) {
			br.children[n.keys[cnt]] = n.next
		} else {
			br.children[n.keys[cnt]] = &extension{keys: n.keys[cnt+1:], next: n.next}
		}
		if cnt == len(keys) {
			br.value = o
		} else {
			br.children[keys[cnt]] = &leaf{keys: keys[cnt+1:], value: o}
		}
		return n.getChanged(n.keys[:cnt], br), true, nil
	default:
		next, dirty, err := n.next.set(m, keys[cnt:], o)
		if dirty {
			return n.getChanged(n.keys, next), true, err
		}
		return n, false, err
	}
}

func (n *extension) getKeyPrepended(k []byte) *extension {
	nk := make([]byte, len(k)+len(n.keys))
	copy(nk, k)
	copy(nk[len(k):], n.keys)
	return n.getChanged(nk, n.next)
}

func (n *extension) delete(m *mpt, keys []byte) (node, bool, error) {
	cnt, _ := compareKeys(keys, n.keys)
	if cnt < len(n.keys) {
		return n, false, nil
	}
	next, dirty, err := n.next.delete(m, keys[cnt:])
	log.Println("extension next.delete() returns", next, dirty, err)
	if dirty {
		if next == nil {
			log.Println("branch next =", next)
			return nil, true, err
		}
		switch nn := next.(type) {
		case *extension:
			nkeys := make([]byte, len(n.keys)+len(nn.keys))
			copy(nkeys, n.keys)
			copy(nkeys[len(n.keys):], nn.keys)
			return n.getChanged(nkeys, nn.next), true, err
		case *leaf:
			nkeys := make([]byte, len(n.keys)+len(nn.keys))
			copy(nkeys, n.keys)
			copy(nkeys[len(n.keys):], nn.keys)
			log.Println("nkeys", nkeys, "value", nn.value)
			return &leaf{keys: nkeys, value: nn.value}, true, err
		}
		return n.getChanged(n.keys, next), true, err
	}
	return n, false, nil
}

func (n *extension) get(m *mpt, keys []byte) (node, trie.Object, error) {
	cnt, _ := compareKeys(keys, n.keys)
	if cnt < len(n.keys) {
		return n, nil, nil
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	next, obj, err := n.next.get(m, keys[cnt:])
	n.next = next
	return n, obj, err
}
