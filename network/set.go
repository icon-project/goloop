package network

import (
	"fmt"
	"log"
	"sync"

	"github.com/icon-project/goloop/module"
)

//TODO KeyEqual()bool, ScoreCompare()int
type Set struct {
	m   map[interface{}]interface{}
	mtx sync.RWMutex
}

func NewSet() *Set {
	return &Set{m: make(map[interface{}]interface{})}
}

func (s *Set) _add(v interface{}, d interface{}) bool {
	if _, ok := s.m[v]; !ok {
		s.m[v] = d
		return true
	}
	return false
}

func (s *Set) _set(v interface{}, d interface{}) interface{} {
	old, ok := s.m[v]
	if ok && old == d {
		return nil
	}
	s.m[v] = d
	return old
}
func (s *Set) _remove(v interface{}) bool {
	if _, ok := s.m[v]; ok {
		delete(s.m, v)
		return true
	}
	return false
}
func (s *Set) _clear() {
	s.m = make(map[interface{}]interface{})
}
func (s *Set) _merge(args ...interface{}) {
	for _, v := range args {
		s._set(v, true)
	}
}
func (s *Set) Add(v interface{}) bool {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	return s._add(v, true)
}
func (s *Set) Set(v interface{}, d interface{}) interface{} {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	return s._set(v, d)
}
func (s *Set) Remove(v interface{}) bool {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	return s._remove(v)
}
func (s *Set) Clear() {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	s._clear()
}
func (s *Set) Merge(args ...interface{}) {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	s._merge(args...)
}
func (s *Set) _contains(v interface{}) bool {
	_, ok := s.m[v]
	return ok
}
func (s *Set) _len() int {
	return len(s.m)
}
func (s *Set) Contains(v interface{}) bool {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	return s._contains(v)
}
func (s *Set) Len() int {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	return s._len()
}
func (s *Set) IsEmpty() bool {
	return s.Len() == 0
}

//Not ordered array
func (s *Set) Array() interface{} {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	arr := make([]interface{}, 0)
	for k := range s.m {
		arr = append(arr, k)
	}
	return arr
}
func (s *Set) Map() map[interface{}]interface{} {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	m := make(map[interface{}]interface{})
	for k, v := range s.m {
		m[k] = v
	}
	return m
}

func (s *Set) String() string {
	m := s.Map()
	return fmt.Sprintf("%v", m)
}

//TODO peer.Equal
type PeerSet struct {
	*Set
	incomming *PeerIDSet
	outgoing  *PeerIDSet
	addrs     *NetAddressSet
}

func NewPeerSet() *PeerSet {
	return &PeerSet{Set: NewSet(), incomming: NewPeerIDSet(), outgoing: NewPeerIDSet(), addrs: NewNetAddressSet()}
}

func (s *PeerSet) _contains(p *Peer) bool {
	if p.incomming {
		return s.incomming.Contains(p.id)
	} else {
		return s.outgoing.Contains(p.id)
	}
}

func (s *PeerSet) Add(p *Peer) bool {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()

	if !s._contains(p) {
		if p.incomming {
			s.incomming.Add(p.id)
		} else {
			s.outgoing.Add(p.id)
		}
		s.addrs.Add(p.netAddress)
		s.Set.m[p] = 1
		return true
	}
	return false
}

func (s *PeerSet) Remove(p *Peer) bool {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()

	if s._contains(p) {
		if p.incomming {
			s.incomming.Remove(p.id)
		} else {
			s.outgoing.Remove(p.id)
		}
		s.addrs.Remove(p.netAddress)
		delete(s.Set.m, p)
		return true
	}
	return false
}
func (s *PeerSet) Contains(p *Peer) bool {
	return s._contains(p)
}
func (s *PeerSet) Merge(args ...*Peer) {
	for _, p := range args {
		s.Add(p)
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
		if p := k.(*Peer); p.hasRole(role) {
			l = append(l, p)
		}
	}
	return l
}
func (s *PeerSet) RemoveByRole(role PeerRoleFlag) []*Peer {
	l := s.getByRole(role)
	for _, p := range l {
		s.Remove(p)
	}
	return l
}
func (s *PeerSet) GetByRoleAndIncomming(role PeerRoleFlag, in bool) *Peer {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	for k := range s.Set.m {
		if p := k.(*Peer); p.incomming == in && p.hasRole(role) {
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
	cache map[NetAddress]string
}

func NewNetAddressSet() *NetAddressSet {
	s := &NetAddressSet{Set: NewSet()}
	s.cache = s.Map()
	return s
}
func (s *NetAddressSet) Add(a NetAddress) bool {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()

	return s._add(a, "")
}
func (s *NetAddressSet) PutByPeer(p *Peer) (old string, removed NetAddress) {
	defer s.mtx.Unlock()
	s.mtx.Lock()
	d := p.id.String()
	od := s._set(p.netAddress, d)
	for k, v := range s.Set.m {
		a := k.(NetAddress)
		if a != p.netAddress && v == d {
			s._remove(k)
			removed = k.(NetAddress)
			log.Println("NetAddressSet.PutByPeer remove", removed, v)
		}
	}
	if od != nil {
		old = od.(string)
	}
	return
}
func (s *NetAddressSet) RemoveByPeer(p *Peer) bool {
	return s.Set.Remove(p.netAddress)
}
func (s *NetAddressSet) Contains(a NetAddress) bool {
	return s.Set.Contains(a)
}
func (s *NetAddressSet) ContainsByPeer(p *Peer) bool {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	d := s.Set.m[p.netAddress]
	return d != nil && d == p.id.String()
}
func (s *NetAddressSet) Merge(args ...NetAddress) {
	defer s.Set.mtx.Unlock()
	s.Set.mtx.Lock()

	//Add
	for _, a := range args {
		if _, ok := s.Set.m[a]; !ok {
			s.Set.m[a] = ""
		}
		if _, ok := s.cache[a]; ok {
			delete(s.cache, a)
		}
	}

	//Remove
	for k := range s.cache {
		if d := s.Set.m[k]; d == "" {
			delete(s.Set.m, k)
		}
	}
	s.cache = s._map()
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
func (s *NetAddressSet) _map() map[NetAddress]string {
	m := make(map[NetAddress]string)
	for k, v := range s.Set.m {
		m[k.(NetAddress)] = v.(string)
	}
	return m
}
func (s *NetAddressSet) Map() map[NetAddress]string {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	return s._map()
}

type PeerIDSet struct {
	*Set
	onUpdate func()
}

func NewPeerIDSet() *PeerIDSet {
	s := &PeerIDSet{Set: NewSet(), onUpdate: func() {}}
	return s
}

func (s *PeerIDSet) _contains(v interface{}) bool {
	for k := range s.Set.m {
		if k.(module.PeerID).Equal(v.(module.PeerID)) {
			return true
		}
	}
	return false
}

func (s *PeerIDSet) Add(id module.PeerID) (r bool) {
	defer func() {
		s.Set.mtx.Unlock()
		if r {
			s.onUpdate()
		}
	}()
	s.Set.mtx.Lock()
	if !s._contains(id) {
		s.Set.m[id] = 1
		r = true
	}
	return
}
func (s *PeerIDSet) Remove(id module.PeerID) (r bool) {
	defer func() {
		s.Set.mtx.Unlock()
		if r {
			s.onUpdate()
		}
	}()
	s.Set.mtx.Lock()
	if s._contains(id) {
		delete(s.Set.m, id)
		r = true
	}
	return
}
func (s *PeerIDSet) Contains(id module.PeerID) bool {
	defer s.mtx.RUnlock()
	s.mtx.RLock()
	return s._contains(id)
}
func (s *PeerIDSet) Merge(args ...module.PeerID) {
	var r bool
	defer func() {
		s.Set.mtx.Unlock()
		if r {
			s.onUpdate()
		}
	}()
	s.Set.mtx.Lock()
	for _, id := range args {
		if !s._contains(id) {
			s.Set.m[id] = 1
			r = true
		}
	}
}
func (s *PeerIDSet) Array() []module.PeerID {
	defer s.Set.mtx.RUnlock()
	s.Set.mtx.RLock()
	arr := make([]module.PeerID, 0)
	for k := range s.Set.m {
		arr = append(arr, k.(module.PeerID))
	}
	return arr
}
func (s *PeerIDSet) ClearAndAdd(args ...module.PeerID) {
	s.Clear()
	s.Merge(args...)
}

type RoleSet struct {
	*Set
}

func NewRoleSet() *RoleSet {
	return &RoleSet{Set: NewSet()}
}
