package ompt

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
)

const (
	debugPrint = false
	debugDump  = false
)

type (
	mptBase struct {
		db         db.Database
		bucket     db.Bucket
		objectType reflect.Type
		cache      *NodeCache
	}
	mpt struct {
		mptBase
		root  node
		mutex sync.Mutex
	}
)

func (m *mpt) get(n node, nibs []byte, depth int) (node, trie.Object, error) {
	if n == nil {
		return nil, nil, nil
	}
	return n.get(m, nibs, depth)
}

func (m *mpt) set(n node, nibs []byte, depth int, o trie.Object) (node, bool, error) {
	if n == nil {
		return &leaf{
			keys:  nibs[depth:],
			value: o,
		}, true, nil
	}
	return n.set(m, nibs, depth, o)
}

func (m *mpt) delete(n node, nibs []byte, depth int) (node, bool, error) {
	if n == nil {
		return nil, false, nil
	}
	return n.delete(m, nibs, depth)
}

func (m *mpt) getObject(o trie.Object) (trie.Object, bool, error) {
	if o == nil {
		return nil, false, nil
	}
	if t := reflect.TypeOf(o); t == m.objectType {
		return o, false, nil
	}

	vobj := reflect.New(m.objectType.Elem())
	nobj, ok := vobj.Interface().(trie.Object)
	if !ok {
		return nil, false, errors.New("Illegal type object")
	}
	if err := nobj.Reset(m.db, o.Bytes()); err != nil {
		return o, false, err
	}
	return nobj, true, nil
}

func (m *mpt) realize(h []byte, nibs []byte) (node, error) {
	serialized, cache := m.cache.get(nibs, h)
	if len(serialized) == 0 {
		var err error
		serialized, err = m.bucket.Get(h)
		if err != nil {
			return nil, err
		}
		if serialized == nil {
			return nil, fmt.Errorf("ErrorKeyNotFound(key=%x)", h)
		}
	}

	if cache {
		m.cache.put(nibs, h, serialized)
	}
	return deserialize(h, serialized, stateFlushed)
}

func (m *mpt) Get(k []byte) (trie.Object, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	root, obj, err := m.get(m.root, bytesToNibs(k), 0)
	m.root = root
	return obj, err
}

func (m *mpt) Hash() []byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.root != nil {
		return m.root.getLink(true)
	} else {
		return nil
	}
}

func (m *mpt) Flush() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.root != nil {
		// Before flush node data to Database, We need to make sure that it
		// builds required  data for dumping data.
		m.root.getLink(true)
		return m.root.flush(m, make([]byte, 0, hashSize*2))
	}
	return nil
}

func (m *mpt) Dump() {
	log.Printf("MPT[%p]-------------------------->>", m)
	if m.root != nil {
		m.root.dump()
	}
	log.Printf("<<--------------------------MPT[%p]", m)
}

func (m *mpt) GetSnapshot() trie.SnapshotForObject {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if debugPrint {
		log.Printf("mpt%p.GetSnapshot()", m)
	}
	if m.root != nil {
		m.root.freeze()
		if debugDump {
			m.root.dump()
		}
	}
	return &mpt{
		mptBase: m.mptBase,
		root:    m.root,
	}
}

func (m *mpt) Set(k []byte, o trie.Object) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if debugPrint {
		log.Printf("mpt%p.Set(%x,%v)", m, k, o)
	}
	root, _, err := m.set(m.root, bytesToNibs(k), 0, o)
	m.root = root
	if debugDump && root != nil {
		root.dump()
	}
	return err
}

func (m *mpt) Delete(k []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if debugPrint {
		log.Printf("mpt%p.Delete(%x)", m, k)
	}
	root, dirty, err := m.delete(m.root, bytesToNibs(k), 0)
	if dirty {
		m.root = root
		if debugDump && root != nil {
			root.dump()
		}
	} else {
		if debugPrint {
			log.Printf("mpt%p.Delete(%x) FAILs", m, k)
		}
	}
	return err
}

func (m *mpt) Reset(s trie.ImmutableForObject) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if debugPrint {
		log.Printf("mpt%p.Reset(%p)\n", m, s)
	}

	m2, ok := s.(*mpt)
	if (!ok) || !reflect.DeepEqual(m2.mptBase, m.mptBase) {
		log.Panicln("Supplied ImmutableForObject isn't usable in here", s)
	}
	m.root = m2.root
	if debugDump && m.root != nil {
		m.root.dump()
	}
}

func (m *mpt) ClearCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.root != nil {
		m.root = m.root.compact()
	}
}

type iteratorItem struct {
	k string
	n node
}

type iterator struct {
	m     *mpt
	stack []iteratorItem
	key   string
	value trie.Object
	error error
}

func (i *iterator) Get() (trie.Object, []byte, error) {
	return i.value, []byte(i.key), i.error
}

func (i *iterator) Next() error {
	if i.error != nil {
		return i.error
	}
	if i.value == nil && len(i.stack) == 0 {
		return errors.New("NoMore")
	}
	for len(i.stack) > 0 {
		l := len(i.stack)
		ii := i.stack[l-1]
		i.stack = i.stack[0 : l-1]

		i.key, i.value, i.error = ii.n.traverse(i.m, ii.k, func(k string, n node) {
			i.stack = append(i.stack, iteratorItem{k: k, n: n})
		})

		if i.error != nil {
			i.key = ""
			i.value = nil
			return nil
		}
		if i.value != nil {
			i.key = string(keysToBytes(i.key))
			return nil
		}
	}
	i.key = ""
	i.value = nil
	i.error = nil
	return nil
}

func (i *iterator) Has() bool {
	return i.value != nil || i.error != nil
}

func (m *mpt) Iterator() trie.IteratorForObject {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	root := m.root
	if root != nil {
		if n, err := root.realize(m); err == nil {
			root = n
			m.root = n
		}
	}
	if root == nil {
		return &iterator{
			m:     m,
			stack: []iteratorItem{},
		}
	}
	i := &iterator{
		m:     m,
		stack: []iteratorItem{{k: "", n: root}},
	}
	i.Next()
	return i
}

func (m *mpt) GetProof(k []byte) [][]byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.root == nil {
		return nil
	}

	// make sure that it's hashed.
	m.root.getLink(true)

	nibbles := bytesToNibs(k)
	proofs := [][]byte(nil)

	root, proofs, err := m.root.getProof(m, nibbles, proofs)
	if root != m.root {
		m.root = root
	}
	if err != nil {
		if debugPrint {
			log.Printf("Fail to get proof for [%x]", k)
		}
		return nil
	}
	return proofs
}

func (m *mpt) Prove(k []byte, proofs [][]byte) (trie.Object, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.root == nil {
		return nil, common.ErrIllegalArgument
	}
	nibbles := bytesToNibs(k)
	root, obj, err := m.root.prove(m, nibbles, proofs)
	if root != m.root {
		m.root = root
	}
	return obj, err
}

func (m *mpt) Equal(object trie.ImmutableForObject, exact bool) bool {
	if m2, ok := object.(*mpt); ok {
		if m == m2 {
			return true
		}
		m.mutex.Lock()
		defer m.mutex.Unlock()
		m2.mutex.Lock()
		defer m2.mutex.Unlock()

		if m2.root == m.root {
			return true
		}
		if m2.root == nil || m.root == nil {
			return false
		}

		h1 := m.root.hash()
		h2 := m2.root.hash()
		if len(h1) > 0 && bytes.Equal(h1, h2) {
			return true
		}
		if exact {
			return bytes.Equal(m.root.getLink(true),
				m2.root.getLink(true))
		}
	} else {
		panic("Equal with invalid object")
	}
	return false
}

func (m *mpt) Empty() bool {
	return m.root == nil
}

func (m *mpt) Resolve(bd merkle.Builder) {
	if m.root != nil {
		m.resolve(bd, &m.root)
	}
}

type nodeRequester struct {
	mpt  *mpt
	node *node
	hash []byte
}

func (r *nodeRequester) OnData(bs []byte, bd merkle.Builder) error {
	node, err := deserialize(r.hash, bs, stateFlushed)
	if err != nil {
		return err
	}
	*r.node = node
	return node.resolve(r.mpt, bd)
}

func (m *mpt) resolve(d merkle.Builder, pNode *node) {
	node := *pNode
	if node == nil {
		return
	}
	_, err := node.realize(m)
	if err != nil {
		hash := node.hash()
		d.RequestData(db.MerkleTrie, hash, &nodeRequester{
			mpt:  m,
			node: pNode,
			hash: hash,
		})
	}
}

func NewMPT(d db.Database, h []byte, t reflect.Type) *mpt {
	bk, err := d.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicln("NewImmutable fail to get bucket")
	}
	return &mpt{
		mptBase: mptBase{
			db:         d,
			bucket:     bk,
			objectType: t,
		},
		root: nodeFromHash(h),
	}
}

func MPTFromImmutable(immutable trie.ImmutableForObject) *mpt {
	if m, ok := immutable.(*mpt); ok {
		return &mpt{
			mptBase: m.mptBase,
			root:    m.root,
		}
	}
	return nil
}
