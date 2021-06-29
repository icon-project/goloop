package trie_manager

import (
	"bytes"
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
)

type trieManager struct {
	database db.Database
}

func (m *trieManager) NewImmutable(rootHash []byte) trie.Immutable {
	return NewImmutable(m.database, rootHash)
}

func (m *trieManager) NewImmutableForObject(h []byte, t reflect.Type) trie.ImmutableForObject {
	return NewImmutableForObject(m.database, h, t)
}

func (m *trieManager) NewMutable(rootHash []byte) trie.Mutable {
	return NewMutable(m.database, rootHash)
}

func (m *trieManager) NewMutableForObject(h []byte, t reflect.Type) trie.MutableForObject {
	return NewMutableForObject(m.database, h, t)
}

func New(database db.Database) trie.Manager {
	return &trieManager{database}
}

type BytesDifferenceHandler func(diff int, key, expect, real []byte)

func CompareImmutable(exp, real trie.Immutable, handler BytesDifferenceHandler) error {
	for ie, ir := exp.Iterator(), real.Iterator(); ie.Has() || ir.Has(); {
		ve, ke, err := ie.Get()
		if err != nil {
			return err
		}
		vr, kr, err := ir.Get()
		if err != nil {
			return err
		}
		switch bytes.Compare(ke, kr) {
		case -1:
			handler(-1, ke, ve, nil)
			if err := ie.Next(); err != nil {
				return err
			}
		case 0:
			if !bytes.Equal(ve, vr) {
				handler(0, ke, ve, vr)
			}
			if err := ie.Next(); err != nil {
				return err
			}
			if err := ir.Next(); err != nil {
				return err
			}
		case 1:
			handler(1, kr, nil, vr)
			if err := ir.Next(); err != nil {
				return err
			}
		}
	}
	return nil
}

type ObjectDifferenceHandler func(op int, key []byte, expect, real trie.Object)

func CompareImmutableForObject(exp, real trie.ImmutableForObject, handler ObjectDifferenceHandler) error {
	for ie, ir := exp.Iterator(), real.Iterator(); ie.Has() || ir.Has(); {
		ve, ke, err := ie.Get()
		if err != nil {
			return err
		}
		vr, kr, err := ir.Get()
		if err != nil {
			return err
		}
		switch bytes.Compare(ke, kr) {
		case -1:
			handler(-1, ke, ve, nil)
			if err := ie.Next(); err != nil {
				return err
			}
		case 0:
			if !bytes.Equal(ve.Bytes(), vr.Bytes()) {
				handler(0, ke, ve, vr)
			}
			if err := ie.Next(); err != nil {
				return err
			}
			if err := ir.Next(); err != nil {
				return err
			}
		case 1:
			handler(1, kr, nil, vr)
			if err := ir.Next(); err != nil {
				return err
			}
		}
	}
	return nil
}
