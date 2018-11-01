package ompt

import (
	"errors"
	"log"
	"reflect"
	"sync"

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
