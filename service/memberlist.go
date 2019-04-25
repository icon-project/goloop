package service

import (
	"log"
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type memberList struct {
	lock     sync.Mutex
	snapshot state.AccountSnapshot
	updated  bool
	members  []module.Address
}

type memberIterator struct {
	index   int
	members []module.Address
}

func (i *memberIterator) Has() bool {
	return i.index < len(i.members)
}

func (i *memberIterator) Next() error {
	if i.index < len(i.members) {
		i.index += 1
		return nil
	} else {
		return errors.ErrInvalidState
	}
}

func (i *memberIterator) Get() (module.Address, error) {
	if i.index < len(i.members) {
		return i.members[i.index], nil
	} else {
		return nil, errors.ErrInvalidState
	}
}

func (m *memberList) Equal(m2 module.MemberList) bool {
	if ptr, ok := m2.(*memberList); ok {
		return m.equal(ptr)
	}

	if m == nil && m2 == nil {
		return true
	}
	if m == nil || m2 == nil {
		return false
	}

	members, err := m.getMembers()
	if err != nil {
		log.Printf("Fail to get members() err=%+v", err)
		return false
	}

	var index = int(0)
	for itr := m2.Iterator(); itr.Has(); itr.Next() {
		if addr, err := itr.Get(); err != nil {
			log.Printf("Fail to iterating members err=%+v", err)
			return false
		} else {
			if index >= len(members) {
				return false
			}
			if !addr.Equal(members[index]) {
				return false
			}
			index += 1
		}
	}
	return index == len(members)
}

func (m *memberList) Iterator() module.MemberIterator {
	if m == nil {
		return nil
	}
	if members, err := m.getMembers(); err == nil {
		return &memberIterator{
			index:   0,
			members: members,
		}
	}
	return nil
}

func (m *memberList) equal(m2 *memberList) bool {
	if m == nil && m2 == nil {
		return true
	}
	if m == nil || m2 == nil {
		return false
	}

	m1List, err := m.getMembers()
	if err != nil {
		log.Printf("Fail to get member list err=%+v", err)
		return false
	}
	m2List, err := m2.getMembers()
	if err != nil {
		log.Printf("Fail to get member list err=%+v", err)
		return false
	}
	if len(m1List) != len(m2List) {
		return false
	}

	for i := 0; i < len(m1List); i++ {
		if !m1List[i].Equal(m2List[i]) {
			return false
		}
	}
	return true
}

func (m *memberList) getMembers() ([]module.Address, error) {
	if m == nil {
		return nil, nil
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.updated {
		return m.members, nil
	}

	if m.snapshot == nil {
		m.updated = true
		return nil, nil
	}

	as := scoredb.NewStateStoreWith(m.snapshot)
	varMembers := scoredb.NewArrayDB(as, state.VarMembers)
	size := varMembers.Size()
	members := make([]module.Address, size)
	for i := 0; i < size; i++ {
		members[i] = varMembers.Get(i).Address()
	}
	m.updated = true
	m.members = members
	return members, nil
}

func (m *memberList) IsEmpty() bool {
	if l, err := m.getMembers(); err != nil {
		return true
	} else {
		return len(l) == 0
	}
}

func newMemberList(snapshot state.AccountSnapshot) *memberList {
	return &memberList{
		snapshot: snapshot,
	}
}
