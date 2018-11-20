package network

import (
	"fmt"
	"sync"

	"github.com/icon-project/goloop/module"
)

//TODO KeyEqual()bool, ScoreCompare()int
type Set struct {
	m   map[interface{}]int
	mtx sync.RWMutex
}

func NewSet() *Set {
	return &Set{m: make(map[interface{}]int)}
}

func (s *Set) Add(v interface{}) bool {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	if _, ok := s.m[v]; !ok {
		s.m[v] = 1
		return true
	}
	return false
}

func (s *Set) Remove(v interface{}) bool {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	if _, ok := s.m[v]; ok {
		delete(s.m, v)
		return true
	}
	return false
}

func (s *Set) Contains(v interface{}) bool {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	_, ok := s.m[v]
	return ok
}
func (s *Set) Clear() {
	s.m = make(map[interface{}]int)
}
func (s *Set) IsEmpty() bool {
	return s.Len() == 0
}
func (s *Set) Len() int {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	return len(s.m)
}
func (s *Set) Merge(args ...interface{}) {
	for _, v := range args {
		s.Add(v)
	}
}
func (s *Set) Array() interface{} {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	arr := make([]interface{}, 0)
	for k := range s.m {
		arr = append(arr, k)
	}
	return arr
}
func (s *Set) String() string {
	arr := s.Array()
	return fmt.Sprintf("%v", arr)
}

type PeerSet struct {
	*Set
	addrs *NetAddressSet
}

func NewPeerSet() *PeerSet {
	return &PeerSet{Set: NewSet(), addrs: NewNetAddressSet()}
}

func (s *PeerSet) Add(p *Peer) bool {
	s.addrs.Add(p.netAddress)
	return s.Set.Add(p)
}

func (s *PeerSet) Remove(p *Peer) bool {
	s.addrs.Remove(p.netAddress)
	return s.Set.Remove(p)
}
func (s *PeerSet) Merge(args ...*Peer) {
	for _, p := range args {
		s.addrs.Add(p.netAddress)
		s.Set.Add(p)
	}
}
func (s *PeerSet) Array() []*Peer {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	arr := make([]*Peer, 0)
	for k := range s.Set.m {
		arr = append(arr, k.(*Peer))
	}
	return arr
}

func (s *PeerSet) GetByID(id module.PeerID) *Peer {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	for k := range s.Set.m {
		if p := k.(*Peer); p.id.Equal(id) {
			return p
		}
	}
	return nil
}
func (s *PeerSet) getByRole(role PeerRoleFlag) []*Peer {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	l := make([]*Peer, 0, len(s.Set.m))
	for k := range s.Set.m {
		if p := k.(*Peer); p.role.Has(role) {
			l = append(l, p)
		}
	}
	return l
}
func (s *PeerSet) RemoveByRole(role PeerRoleFlag) []*Peer {
	l := s.getByRole(role)
	for _, p := range s.getByRole(role) {
		s.Remove(p)
	}
	return l
}
func (s *PeerSet) GetByRoleAndIncomming(role PeerRoleFlag, in bool) *Peer {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	for k := range s.Set.m {
		if p := k.(*Peer); p.incomming == in && p.role.Has(role) {
			return p
		}
	}
	return nil
}

func (s *PeerSet) NetAddresses() []NetAddress {
	return s.addrs.Array()
}
func (s *PeerSet) HasNetAddresse(a NetAddress) bool {
	return s.addrs.Contains(a)
}

type NetAddressSet struct {
	*Set
}

func NewNetAddressSet() *NetAddressSet {
	return &NetAddressSet{Set: NewSet()}
}
func (s *NetAddressSet) Merge(args ...NetAddress) {
	for _, a := range args {
		s.Set.Add(a)
	}
}

func (s *NetAddressSet) Array() []NetAddress {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	arr := make([]NetAddress, 0)
	for k := range s.Set.m {
		arr = append(arr, k.(NetAddress))
	}
	return arr
}

type PeerIdSet struct {
	*Set
}

func NewPeerIdSet() *PeerIdSet {
	s := &PeerIdSet{Set: NewSet()}
	return s
}

func (s *PeerIdSet) _contains(v interface{}) bool {
	for k := range s.Set.m {
		if k.(module.PeerID).Equal(v.(module.PeerID)) {
			return true
		}
	}
	return false
}

func (s *PeerIdSet) Add(id module.PeerID) bool {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()
	if !s._contains(id) {
		s.Set.m[id] = 1
		return true
	}
	return false
}
func (s *PeerIdSet) Remove(id module.PeerID) bool {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()
	if s._contains(id) {
		delete(s.Set.m, id)
		return true
	}
	return false
}
func (s *PeerIdSet) Contains(id module.PeerID) bool {
	return s._contains(id)
}
func (s *PeerIdSet) Merge(args ...module.PeerID) {
	for _, id := range args {
		s.Add(id)
	}
}

type RoleSet struct {
	*Set
}

func NewRoleSet() *RoleSet {
	return &RoleSet{Set: NewSet()}
}
func (s *RoleSet) Merge(args ...module.Role) {
	for _, r := range args {
		s.Set.Add(r)
	}
}
