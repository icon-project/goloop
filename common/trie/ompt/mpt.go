package ompt

import (
	"errors"
	"log"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
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
	}
	mpt struct {
		mptBase
		root  node
		mutex sync.Mutex
	}
)

func (m *mpt) get(n node, keys []byte) (node, trie.Object, error) {
	if n == nil {
		return nil, nil, nil
	}
	return n.get(m, keys)
}

func (m *mpt) set(n node, keys []byte, o trie.Object) (node, bool, error) {
	if n == nil {
		return &leaf{
			keys:  keys,
			value: o,
		}, true, nil
	}
	return n.set(m, keys, o)
}

func (m *mpt) delete(n node, keys []byte) (node, bool, error) {
	if n == nil {
		return nil, false, nil
	}
	return n.delete(m, keys)
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

func (m *mpt) Get(k []byte) (trie.Object, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	root, obj, err := m.get(m.root, bytesToKeys(k))
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
		return m.root.flush(m)
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
	root, dirty, err := m.set(m.root, bytesToKeys(k), o)
	if dirty {
		m.root = root
		if debugDump && root != nil {
			root.dump()
		}
	}
	return err
}

func (m *mpt) Delete(k []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if debugPrint {
		log.Printf("mpt%p.Delete(%x)", m, k)
	}
	root, dirty, err := m.delete(m.root, bytesToKeys(k))
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
	m2, ok := s.(*mpt)
	if (!ok) || !reflect.DeepEqual(m2.mptBase, m.mptBase) {
		log.Panicln("Supplied ImmutableForObject isn't usable in here", s)
	}
	m.root = m2.root
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
			return i.error
		}
		if i.value != nil {
			i.key = string(keysToBytes(i.key))
			return nil
		}
	}
	i.key = ""
	i.value = nil
	i.error = nil
	return errors.New("NoMoreItem")
}

func (i *iterator) Has() bool {
	return i.value != nil
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

	nibbles := bytesToKeys(k)
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
	nibbles := bytesToKeys(k)
	root, obj, err := m.root.prove(m, nibbles, proofs)
	if root != m.root {
		m.root = root
	}
	return obj, err
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
