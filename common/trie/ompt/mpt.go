package ompt

import (
	"errors"
	"log"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
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
	root, obj, err := m.get(m.root, k)
	m.root = root
	return obj, err
}

func (m *mpt) Hash() []byte {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.root.getLink(true)
}

func (m *mpt) Flush() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.root.flush(m)
}

func (m *mpt) Dump() {
	log.Printf("mpt[%p] DUMP>>>>>>>>>>>>>>>>>>>>>", m)
	if m.root != nil {
		m.root.dump()
	}
	log.Printf("mpt[%p] DUMP<<<<<<<<<<<<<<<<<<<<<", m)
}

func (m *mpt) GetSnapshot() trie.SnapshotForObject {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.root.freeze()
	return &mpt{
		mptBase: m.mptBase,
		root:    m.root,
	}
}

func (m *mpt) Set(k []byte, o trie.Object) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	root, dirty, err := m.set(m.root, k, o)
	if dirty {
		m.root = root
	}
	return err
}

func (m *mpt) Delete(k []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	root, dirty, err := m.delete(m.root, k)
	if dirty {
		m.root = root
	}
	return err
}

func (m *mpt) Reset(s trie.ImmutableForObject) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m2, ok := s.(*mpt)
	if (!ok) || m2.mptBase != m.mptBase {
		log.Panicln("Supplied ImmutableForObject isn't usable in here", s)
	}
	m.root = m2.root
}

func NewMutableForObject(d db.Database, h []byte, t reflect.Type) trie.MutableForObject {
	bk, err := d.GetBucket(db.StateTrie)
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

func NewImmutableForObject(d db.Database, h []byte, t reflect.Type) trie.ImmutableForObject {
	bk, err := d.GetBucket(db.StateTrie)
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
