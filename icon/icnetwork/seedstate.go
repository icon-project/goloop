package icnetwork

import (
	"sync"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/icon/iiss"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

const (
	seedStateLockIdxSeedPolicy = iota
	seedStateLockIdxSeeds
	seedStateLockIdxTerm
	seedStateLockIdxReserved
)

type SeedState struct {
	p  module.SeedRoleAuthorizationPolicy
	t  *int64
	ss containerdb.BytesStoreState
	es state.ExtensionSnapshot

	seeds *network.PeerIDSet
	mtxs  []*sync.RWMutex
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

func (s *SeedState) getSeeds() *network.PeerIDSet {
	mtx := s.mtxs[seedStateLockIdxSeeds]
	mtx.Lock()
	defer mtx.Unlock()
	if s.seeds == nil {
		seeds := network.NewPeerIDSet()
		preps := s.es.NewState(true).(*iiss.ExtensionStateImpl).State.GetPReps(true)
		for _, prep := range preps {
			switch prep.Grade() {
			case icstate.GradeMain, icstate.GradeSub:
				seeds.Add(network.NewPeerIDFromAddress(prep.NodeAddress()))
			}
		}
		s.seeds = seeds
	}
	return s.seeds
}

func (s *SeedState) Seeds() []module.PeerID {
	return s.getSeeds().Array()
}

func (s *SeedState) IsAuthorizer(id module.PeerID) bool {
	return s.getSeeds().Contains(id)
}

func (s *SeedState) IsCandidate(id module.PeerID) bool {
	return !s.getSeeds().Contains(id)
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
	es := ws.GetExtensionSnapshot()
	s := &SeedState{
		p:  module.SeedRoleAuthorizationPolicyReserved,
		ss: ss,
		es: es,
	}
	for i := 0; i < seedStateLockIdxReserved; i++ {
		mtx := &sync.RWMutex{}
		s.mtxs = append(s.mtxs, mtx)
	}
	return s, nil
}
