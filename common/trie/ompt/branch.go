package ompt

import (
	"fmt"
	"log"

	"github.com/icon-project/goloop/common/trie"
)

type branch struct {
	nodeBase
	children [16]node
	value    trie.Object
}

func newBranch(h, s []byte, blist [][]byte) (node, error) {
	br := &branch{
		nodeBase: nodeBase{
			hashValue:  h,
			serialized: s,
			state:      stateFlushed,
		},
	}
	for i, b := range blist {
		if i < 16 {
			child, err := nodeFromLink(b)
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
				br.value = BytesObject(v)
			}
		}
	}
	return br, nil
}

func (n *branch) toString() string {
	return fmt.Sprintf("BRAN[%p](%v,%p)", n, n.children, n.value)
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
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state != stateDirty {
		return
	}
	for _, child := range n.children {
		if child != nil {
			child.freeze()
		}
	}
	n.state = stateFreezed
}

func (n *branch) flush(m *mpt) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.state == stateFlushed {
		return nil
	}
	for _, child := range n.children {
		if child == nil {
			continue
		}
		if err := child.flush(m); err != nil {
			return err
		}
	}
	if n.value != nil {
		if err := n.value.Flush(); err != nil {
			return err
		}
	}
	if err := n.nodeBase.flushBaseInLock(m); err != nil {
		return err
	}
	return nil
}

func (n *branch) getChangable() *branch {
	if n.state == stateDirty {
		return n
	}
	return &branch{children: n.children, value: n.value}
}

func (n *branch) set(m *mpt, keys []byte, o trie.Object) (node, bool, error) {
	if len(keys) == 0 {
		if n.value == nil || !n.value.Equal(o) {
			br := n.getChangable()
			br.value = o
			return br, true, nil
		}
		return n, false, nil
	}
	child, dirty, err := m.set(n.children[keys[0]], keys[1:], o)
	if dirty {
		br := n.getChangable()
		br.children[keys[0]] = child
		return br, true, err
	}
	return n, false, err
}

func (n *branch) delete(m *mpt, keys []byte) (node, bool, error) {
	if len(keys) == 0 {
		if n.value == nil {
			return n, false, nil
		}
		br := n.getChangable()
		br.value = nil
		return br, true, nil
	}
	child := n.children[keys[0]]
	if child == nil {
		return n, false, nil
	}
	nchild, dirty, err := child.delete(m, keys[1:])
	if !dirty {
		return n, false, nil
	}
	if nchild == nil {
		var idx = 16
		for i, c := range n.children {
			if c != nil && c != child {
				if idx != 16 {
					idx = -1
					break
				}
				idx = i
			}
		}
		log.Println("idx", idx)
		if idx != -1 {
			if idx == 16 {
				return &leaf{value: n.value}, true, err
			}
			alive := n.children[idx]
			switch nn := alive.(type) {
			case *extension:
				return nn.getKeyPrepended([]byte{byte(idx)}), true, err
			case *branch:
				return &extension{
					keys: []byte{byte(idx)},
					next: alive,
				}, true, err
			case *leaf:
				return nn.getKeyPrepended([]byte{byte(idx)}), true, err
			}
		}
	}
	br := n.getChangable()
	br.children[keys[0]] = nchild
	return br, true, err
}

func (n *branch) get(m *mpt, keys []byte) (node, trie.Object, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if len(keys) == 0 {
		nv, changed, err := m.getObject(n.value)
		if changed {
			n.value = nv
		}
		return n, nv, err
	}
	child, o, err := n.children[keys[0]].get(m, keys[1:])
	n.children[keys[0]] = child
	return n, o, err
}
