package ompt

import (
	"log"
	"sync"

	"github.com/icon-project/goloop/common/trie"
	"golang.org/x/crypto/sha3"
)

const (
	stateDirty   = 0
	stateFreezed = 1
	stateFlushed = 2
	hashSize     = 32
)

type (
	node interface {
		getLink(forceHash bool) []byte
		freeze()
		flush(m *mpt) error
		toString() string
		dump()
		set(m *mpt, keys []byte, o trie.Object) (node, bool, error)
		delete(m *mpt, keys []byte) (node, bool, error)
		get(m *mpt, keys []byte) (node, trie.Object, error)
	}
)

type nodeBase struct {
	hashValue  []byte
	serialized []byte
	state      int
	mutex      sync.Mutex
}

func (n *nodeBase) getLink(n2 node, forceHash bool) []byte {
	s := n.serialize(n2)
	if len(s) <= hashSize && !forceHash {
		return s
	}
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.hashValue == nil {
		n.hashValue = calcHash(s)
	}
	if forceHash {
		return n.hashValue
	} else {
		return rlpEncodeBytes(n.hashValue)
	}
}

func (n *nodeBase) flushBaseInLock(m *mpt) error {
	if n.hashValue != nil {
		if err := m.bucket.Set(n.hashValue, n.serialized); err != nil {
			return err
		}
	}
	return nil
}

func (n *nodeBase) serialize(n2 node) []byte {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if n.serialized == nil {
		bytes, err := rlpEncode(n2)
		if err != nil {
			log.Panicln("FAIL to serialize", n, err)
		}
		if n.state == stateDirty {
			n.state = stateFreezed
		}
		n.serialized = bytes
	}
	return n.serialized
}

func bytesToKeys(b []byte) []byte {
	nibbles := make([]byte, len(b)*2)
	for i, v := range b {
		nibbles[i*2] = v >> 4 & 0x0F
		nibbles[i*2+1] = v & 0x0F
	}
	return nibbles
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
	sha := sha3.NewLegacyKeccak256()
	for _, d := range data {
		sha.Write(d)
	}
	sum := sha.Sum([]byte{})
	return sum[:]
}

func nodeFromLink(b []byte) (node, error) {
	if b[0] >= 0xC0 {
		node, err := deserialize(nil, b)
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
		return hash(h)
	}
}

func deserialize(h, serialized []byte) (node, error) {
	blist, err := rlpParseList(serialized)
	if err != nil {
		log.Panicln("FAIL to parse bytes from hash", err)
	}
	switch len(blist) {
	case 2:
		keyheader, err := rlpParseBytes(blist[0])
		if err != nil {
			log.Panicln("Illegal data to decode")
		}
		if (keyheader[0] & 0x20) == 0 {
			// extension
			return newExtension(h, serialized, blist)
		}
		// leaf
		return newLeaf(h, serialized, blist)
	case 17:
		// branch
		return newBranch(h, serialized, blist)
	default:
		log.Panicln("FAIL to parse bytes from hash for MPT")
		return nil, nil
	}
}
