package network

import (
	"sync"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

const (
	seedStateLockIdxSeedPolicy = iota
	seedStateLockIdxSeeds
	seedStateLockIdxAuthorizer
	seedStateLockIdxCandidates
	seedStateLockIdxTerm
	seedStateLockIdxReserved
)

type SeedState struct {
	p  module.SeedRoleAuthorizationPolicy
	t  *int64
	as state.AccountSnapshot
	ss containerdb.BytesStoreState
	vs state.ValidatorSnapshot

	seeds      *PeerIDSet
	candidates *PeerIDSet
	validators *PeerIDSet
	mtxs       []*sync.RWMutex
}

func (s *SeedState) AuthorizationPolicy() module.SeedRoleAuthorizationPolicy {
	mtx := s.mtxs[seedStateLockIdxSeedPolicy]
	mtx.Lock()
	defer mtx.Unlock()
	if s.p == module.SeedRoleAuthorizationPolicyReserved {
		v := scoredb.NewVarDB(s.ss, state.VarSeedPolicy).Int64()
		s.p = module.SeedRoleAuthorizationPolicy(v)
	}
	return s.p
}

func (s *SeedState) Seeds() []module.PeerID {
	mtx := s.mtxs[seedStateLockIdxSeeds]
	mtx.Lock()
	defer mtx.Unlock()
	if s.seeds == nil {
		s.seeds = getPeerIDSetFromStateByKey(s.ss, state.VarSeeds)
	}
	return s.seeds.Array()
}

func (s *SeedState) IsAuthorizer(id module.PeerID) bool {
	mtx := s.mtxs[seedStateLockIdxAuthorizer]
	mtx.Lock()
	defer mtx.Unlock()
	if s.validators == nil {
		vs := NewPeerIDSet()
		size := s.vs.Len()
		for i := 0; i < size; i++ {
			v, _ := s.vs.Get(i)
			vs.Add(NewPeerIDFromAddress(v.Address()))
		}
		s.validators = vs
	}
	return s.validators.Contains(id)
}

func (s *SeedState) IsCandidate(id module.PeerID) bool {
	mtx := s.mtxs[seedStateLockIdxCandidates]
	mtx.Lock()
	defer mtx.Unlock()
	if s.candidates == nil {
		s.candidates = getPeerIDSetFromStateByKey(s.ss, state.VarSeedCandidates)
	}
	if s.candidates.IsEmpty() {
		return !s.validators.Contains(id) && !s.seeds.Contains(id)
	}
	return s.candidates.Contains(id)
}

func (s *SeedState) Term() int64 {
	mtx := s.mtxs[seedStateLockIdxTerm]
	mtx.Lock()
	defer mtx.Unlock()
	if s.t == nil {
		*s.t = scoredb.NewVarDB(s.ss, state.VarSeedTerm).Int64()
	}
	return *s.t
}

func NewSeedState(ws state.WorldSnapshot) (*SeedState, error) {
	as := ws.GetAccountSnapshot(state.SystemID)
	ss := scoredb.NewStateStoreWith(as)
	vs := ws.GetValidatorSnapshot()
	s := &SeedState{
		p:  module.SeedRoleAuthorizationPolicyReserved,
		as: as,
		ss: ss,
		vs: vs,
	}
	for i := 0; i < seedStateLockIdxReserved; i++ {
		mtx := &sync.RWMutex{}
		s.mtxs = append(s.mtxs, mtx)
	}
	return s, nil
}

func getPeerIDSetFromStateByKey(ss containerdb.BytesStoreState, k interface{}) *PeerIDSet {
	ad := scoredb.NewArrayDB(ss, k)
	pis := NewPeerIDSet()
	size := ad.Size()
	for i := 0; i < size; i++ {
		pis.Add(NewPeerID(ad.Get(i).Bytes()))
	}
	return pis
}
