package mpt

import (
	"errors"
	"github.com/icon-project/goloop/common/trie"
	"log"
)

func (m *mpt) initIterator(iter *iteratorImpl) {
	var data []byte
	if n, ok := m.root.(hash); ok {
		var err error
		data, err = m.bk.Get(n)
		if err != nil {
			log.Fatalf("Failed to get value. key : %x", n)
			return
		} else if len(data) == 0 {
			return
		}
		iter.stack[0].n = deserialize(data, m.objType, m.db)
	} else {
		iter.stack[0].n = m.root
	}
}

func newIterator(m *mpt) *iteratorImpl {
	iter := &iteratorImpl{key: nil, value: nil, top: -1, m: m}
	m.Hash()
	m.initIterator(iter)

	return iter
}

func (iter *iteratorImpl) nextChildNode(m *mpt, n node, key []byte) ([]byte, trie.Object) {
	switch nn := n.(type) {
	case *branch:
		iter.top++
		iter.stack[iter.top].n = n
		iter.stack[iter.top].key = key
		if nn.value != nil {
			return key, nn.value
		}
		for i, nibbleNode := range nn.nibbles {
			if nibbleNode != nil {
				newKey := make([]byte, len(key)+1)
				if len(key) > 0 {
					copy(newKey, key)
				}
				newKey[len(key)] = byte(i)
				return iter.nextChildNode(m, nibbleNode, newKey)
			}
		}
	case *extension:
		newKey := make([]byte, len(key)+len(nn.sharedNibbles))
		if len(key) > 0 {
			copy(newKey, key)
		}
		copy(newKey[len(key):], nn.sharedNibbles)
		return iter.nextChildNode(m, nn.next, newKey)
	case *leaf:
		newKey := make([]byte, len(key)+len(nn.keyEnd))
		if len(key) > 0 {
			copy(newKey, key)
		}
		if len(nn.keyEnd) > 0 {
			copy(newKey[len(key):], nn.keyEnd)
		}
		iter.top++
		iter.stack[iter.top].key = newKey
		iter.stack[iter.top].n = n
		return newKey, nn.value
	case hash:
		serializedValue, err := m.bk.Get(nn)
		if err != nil {
			return nil, nil
		}
		if serializedValue == nil {
			return nil, nil
		}
		return iter.nextChildNode(m, deserialize(serializedValue, m.objType, m.db), key)
	case nil:
		return nil, nil
	}
	panic("Not considered!!!")
}

func (iter *iteratorImpl) Next() error {
	if iter.end == true {
		return errors.New("NoMoreItem")
	}
	if iter.top == -1 && len(iter.key) == 0 {
		iter.key, iter.value = iter.nextChildNode(iter.m, iter.stack[0].n, nil)
	} else {
		n := iter.stack[iter.top]
		switch nn := n.n.(type) {
		case *branch:
			for _, nibbleNode := range nn.nibbles {
				if nibbleNode != nil {
					iter.key, iter.value = iter.nextChildNode(iter.m, nibbleNode, iter.key)
				}
			}
		case *leaf:
			findNext := false
			prevKey := n.key
			for iter.top != 0 && findNext == false {
				iter.top--
				stackNode := iter.stack[iter.top]
				startNibble := byte(0)
				keyIndex := len(stackNode.key)
				startNibble = prevKey[keyIndex] + 1
				branchNode := stackNode.n.(*branch)
				// do not set nil when trie is not flushed yet
				//branchNode.nibbles[prevKey[keyIndex]] = nil
				prevKey = stackNode.key
				for i := startNibble; i < 16; i++ {
					if branchNode.nibbles[i] != nil {
						findNext = true
						newKey := make([]byte, len(stackNode.key)+1)
						copy(newKey, prevKey)
						newKey[len(stackNode.key)] = i
						iter.key, iter.value = iter.nextChildNode(iter.m, branchNode.nibbles[i], newKey)
						break
					}
				}
			}
			if findNext == false {
				iter.key = nil
				iter.value = nil
				iter.end = true
			}
		}
	}
	return nil
}

func (iter *iteratorImpl) Has() bool {
	if iter.end {
		return false
	}
	return iter.value != nil
}

func (iter *iteratorImpl) get() (value trie.Object, key []byte, err error) {
	k := iter.key
	remainder := len(k) % 2
	returnKey := make([]byte, len(k)/2+remainder)
	if remainder > 0 {
		returnKey[0] = k[0]
	}
	for i := remainder; i < len(k)/2+remainder; i++ {
		returnKey[i] = k[i*2-remainder]<<4 | k[i*2+1-remainder]
	}
	return iter.value, returnKey, nil
}

func (iter *iteratorImpl) Get() (value []byte, key []byte, err error) {
	v, k, err := iter.get()
	if err != nil && v == nil {
		return nil, nil, err
	}
	return v.Bytes(), k, nil
}
