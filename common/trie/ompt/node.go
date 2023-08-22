package ompt

import (
	"strconv"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

type (
	nodeState int

	// When it traverses nodes, it can push more nodes for next visit.
	// It will visit nodes from last to first.
	nodeScheduler func(string, node) (node, error)

	node interface {
		hash() []byte
		getLink(forceHash bool) []byte
		freeze()
		flush(m *mpt, nibs []byte) error
		toString() string
		dump()
		set(m *mpt, nibs []byte, depth int, o trie.Object) (node, bool, trie.Object, error)
		delete(m *mpt, nibs []byte, depth int) (node, bool, trie.Object, error)
		get(m *mpt, nibs []byte, depth int) (node, trie.Object, error)
		realize(m *mpt) (node, error)
		traverse(m *mpt, nibs string, v nodeScheduler) (string, trie.Object, error)
		getProof(m *mpt, keys []byte, proofs [][]byte) (node, [][]byte, error)
		prove(m *mpt, keys []byte, proofs [][]byte) (node, trie.Object, error)
		resolve(m *mpt, bd merkle.Builder) error
		compact() node
	}
)

const (
	stateDirty   nodeState = 0
	stateFrozen  nodeState = 1
	stateHashed  nodeState = 2
	stateWritten nodeState = 3
	stateFlushed nodeState = 4

	hashSize = 32
)

func (s nodeState) String() string {
	switch s {
	case stateDirty:
		return "Dirty"
	case stateFrozen:
		return "Frozen"
	case stateHashed:
		return "Hashed"
	case stateWritten:
		return "Written"
	case stateFlushed:
		return "Flushed"
	default:
		return strconv.Itoa(int(s))
	}
}

type nodeBase struct {
	hashValue  []byte
	serialized []byte
	state      nodeState
	mutex      sync.RWMutex
}

func (n *nodeBase) hash() []byte {
	return n.hashValue
}

func (n *nodeBase) rlock() AutoRWUnlock {
	return RLock(&n.mutex)
}

func (n *nodeBase) getLink(n2 node, forceHash bool) []byte {
	lock := n.rlock()
	defer lock.Unlock()

	if n.state < stateHashed {
		bytes, err := rlpEncode(n2)
		if err != nil {
			log.Panicln("FAIL to serialize", n, err)
		}
		lock.Migrate()
		n.serialized = bytes
		if len(n.serialized) > hashSize || forceHash {
			n.hashValue = calcHash(n.serialized)
		}
		n.state = stateHashed
	}
	if n.hashValue != nil {
		if forceHash {
			return n.hashValue
		}
		return rlpEncodeBytes(n.hashValue)
	} else {
		return n.serialized
	}
}

func (n *nodeBase) flushBaseInLock(m *mpt, nibs []byte) error {
	if n.state < stateHashed {
		panic("It's not hashed yet.")
	}
	if n.hashValue != nil && n.state < stateWritten {
		if logStatics {
			atomic.AddInt32(&m.s.write, 1)
		}
		if err := m.bucket.Set(n.hashValue, n.serialized); err != nil {
			return err
		}
		m.cache.Put(nibs, n.hashValue, n.serialized)
	}
	return nil
}

func clone(b []byte) []byte {
	return append([]byte(nil), b...)
}

func keysToBytes(s string) []byte {
	k := make([]byte, len(s)/2)
	for i := 0; i < len(k); i++ {
		k[i] = s[i*2]<<4 | s[i*2+1]
	}
	return k
}

func encodeKeys(tag byte, k []byte) []byte {
	keyLen := len(k)
	buf := make([]byte, keyLen/2+1)
	keyIdx := 0
	if keyLen%2 == 1 {
		buf[0] = tag | 0x10 | k[0]
		keyIdx++
	} else {
		buf[0] = tag
	}
	for i := 1; keyIdx < len(k); keyIdx, i = keyIdx+2, i+1 {
		buf[i] = (k[keyIdx] << 4) | (k[keyIdx+1])
	}
	return buf
}

func decodeKeys(bytes []byte) []byte {
	var keys []byte
	kidx := 0
	if (bytes[0] & 0x10) != 0 {
		keys = make([]byte, (len(bytes)-1)*2+1)
		keys[0] = bytes[0] & 0x0f
		kidx++
	} else {
		keys = make([]byte, (len(bytes)-1)*2)
	}
	for _, b := range bytes[1:] {
		keys[kidx] = b >> 4
		kidx++
		keys[kidx] = b & 0xf
		kidx++
	}
	return keys
}

func compareKeys(k1, k2 []byte) (int, bool) {
	klen := len(k1)
	if klen > len(k2) {
		klen = len(k2)
	}
	for i := 0; i < klen; i++ {
		if k1[i] != k2[i] {
			return i, false
		}
	}
	return klen, len(k1) == len(k2)
}

func calcHash(data ...[]byte) []byte {
	if trie.ConfigUseKeccak256 {
		sha := sha3.NewLegacyKeccak256()
		for _, d := range data {
			sha.Write(d)
		}
		sum := sha.Sum([]byte{})
		return sum[:]
	} else {
		sha := sha3.New256()
		for _, d := range data {
			sha.Write(d)
		}
		sum := sha.Sum([]byte{})
		return sum[:]
	}
}

func nodeFromLink(b []byte, state nodeState) (node, error) {
	if b[0] >= 0xC0 {
		node, err := deserialize(nil, b, state)
		if err != nil {
			return nil, err
		}
		return node, nil
	}
	v, err := rlpParseBytes(b)
	if err != nil {
		return nil, err
	}
	return nodeFromHash(v), nil
}

func nodeFromHash(h []byte) node {
	if len(h) == 0 {
		return nil
	} else {
		return &hash{value: h}
	}
}

func deserialize(h, serialized []byte, state nodeState) (node, error) {
	blist, err := rlpParseList(serialized)
	if err != nil {
		return nil, errors.Wrap(err, "fail to deserialize node")
	}
	switch len(blist) {
	case 2:
		keyheader, err := rlpParseBytes(blist[0])
		if err != nil {
			return nil, errors.New("fail to parse header of node")
		}
		if (keyheader[0] & 0x20) == 0 {
			// extension
			return newExtension(h, serialized, blist, state)
		}
		// leaf
		return newLeaf(h, serialized, blist, state)
	case 17:
		// branch
		return newBranch(h, serialized, blist, state)
	default:
		return nil, errors.Errorf("unknown node data items=%d", len(blist))
	}
}
