package network

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	DefaultSVPWaitByAuthorizer     = time.Second * 10
	DefaultSVPWaitByValidatorVotes = time.Second * 10
	DefaultSVIssuerExpireDelay     = 10 //height
)

var (
	protoSR  = module.ProtocolInfo(0x0100)
	protoSVR = module.ProtocolInfo(0x0200)
	protoSVP = module.ProtocolInfo(0x0300)

	smProtocols = []module.ProtocolInfo{protoSR, protoSVR, protoSVP}
)

type SRType int

const (
	SRTypeByAuthorizer = iota
	SRTypeByValidatorVotes
)

// SeedRequest is used for authorization request by seed candidate
type SeedRequest struct {
	Type       SRType
	Issuer     []byte
	NetAddress NetAddress
	Height     int64
}

func (sr *SeedRequest) Equal(v *SeedRequest) bool {
	return sr.Type == v.Type &&
		bytes.Equal(sr.Issuer, v.Issuer) &&
		sr.NetAddress == v.NetAddress &&
		sr.Height == v.Height
}

// SeedVerificationRequest is used for short-term-seed authorization request by seed
type SeedVerificationRequest struct {
	SR Signed[SeedRequest]
}

func (svr *SeedVerificationRequest) SRType() SRType {
	return svr.SR.Message.Type
}

func (svr *SeedVerificationRequest) Issuer() []byte {
	return svr.SR.Message.Issuer
}

func (svr *SeedVerificationRequest) Height() int64 {
	return svr.SR.Message.Height
}

func (svr *SeedVerificationRequest) Equal(v *SeedVerificationRequest) bool {
	return bytes.Equal(svr.SR.Signature, v.SR.Signature)
}

// SeedVerificationPart is used for authorization response of SeedVerificationRequest by validator.
type SeedVerificationPart struct {
	SVR    Signed[SeedVerificationRequest]
	Expire int64
}

func (svp *SeedVerificationPart) SRType() SRType {
	return svp.SVR.Message.SRType()
}

func (svp *SeedVerificationPart) Issuer() []byte {
	return svp.SVR.Message.Issuer()
}

func (svp *SeedVerificationPart) Height() int64 {
	return svp.SVR.Message.Height()
}

func (svp *SeedVerificationPart) IsExpire(height int64) bool {
	return svp.Expire < height
}

func (svp *SeedVerificationPart) Equal(v *SeedVerificationPart) bool {
	return bytes.Equal(svp.SVR.Signature, v.SVR.Signature) &&
		svp.Expire == v.Expire
}

type SeedVerification struct {
	MultiSigned[SeedVerificationPart]
	signers []module.PeerID
}

func (sv *SeedVerification) MarshalBinary() (data []byte, err error) {
	return sv.MultiSigned.MarshalBinary()
}

func (sv *SeedVerification) UnmarshalBinary(data []byte) error {
	ms, signers, err := NewMultiSignedFromBytes[SeedVerificationPart](data)
	if err != nil {
		return err
	}
	sv.MultiSigned = *ms
	sv.signers = signers
	return nil
}

// Add verified SVP with signer
func (sv *SeedVerification) Add(svp *Signed[SeedVerificationPart], svpSigner module.PeerID) error {
	if !sv.Message.Equal(&svp.Message) {
		return errors.Errorf("mismatch SVP")
	}
	sv.signers = append(sv.signers, svpSigner)
	sv.Signatures = append(sv.Signatures, svp.Signature)
	return nil
}

func (sv *SeedVerification) Contains(id module.PeerID) bool {
	for _, signer := range sv.signers {
		if signer.Equal(id) {
			return true
		}
	}
	return false
}

func (sv *SeedVerification) SRType() SRType {
	return sv.Message.SRType()
}

func (sv *SeedVerification) Issuer() []byte {
	return sv.Message.Issuer()
}

func (sv *SeedVerification) IsExpire(height int64) bool {
	if sv.Message.Expire == 0 {
		return false
	}
	return sv.Message.IsExpire(height)
}

type SeedManager struct {
	w  module.Wallet
	n  module.NetworkManager
	ph module.ProtocolHandler
	rr module.RouteResolver

	cancel context.CancelFunc
	mtx    sync.Mutex

	id    module.PeerID
	na    NetAddress
	cr    PeerRoleFlag
	csrap module.SeedRoleAuthorizationPolicy

	peers *PeerIDSet

	vsp *ValidatorSetCache

	seedsByState *PeerIDSet
	seedsBySV    *PeerIDSet

	svpMap map[module.PeerID]*Signed[SeedVerificationPart]
	svpMtx sync.RWMutex

	svMap   map[module.PeerID]*SeedVerification
	svBuf   map[int64]*SeedVerification
	srTimer *time.Timer
	svMtx   sync.RWMutex

	ss    module.SeedState
	srap  module.SeedRoleAuthorizationPolicy
	ssMtx sync.RWMutex

	l log.Logger
}

func newSeedManager(c module.Chain, self *Peer, srap module.SeedRoleAuthorizationPolicy, l log.Logger) *SeedManager {
	//TODO module.RouteResolver : ProtocolHandler or PeerToPeer
	return &SeedManager{
		w:            c.Wallet(),
		n:            c.NetworkManager(),
		id:           self.ID(),
		na:           self.NetAddress(),
		cr:           self.Role(),
		csrap:        srap,
		peers:        NewPeerIDSet(),
		vsp:          NewValidatorSetCache(DefaultSVIssuerExpireDelay),
		seedsByState: NewPeerIDSet(),
		seedsBySV:    NewPeerIDSet(),
		svpMap:       make(map[module.PeerID]*Signed[SeedVerificationPart]),
		svMap:        make(map[module.PeerID]*SeedVerification),
		svBuf:        make(map[int64]*SeedVerification),
		l:            l.WithFields(log.Fields{LoggerFieldKeySubModule: "seedmgr"}),
	}
}

func (s *SeedManager) setSeedState(ss module.SeedState) {
	s.ssMtx.Lock()
	defer s.ssMtx.Unlock()
	s.ss = ss
	srap := ss.AuthorizationPolicy()
	//resolve SeedRoleAuthorizationPolicy
	srap = srap & s.csrap
	if srap.Enabled(module.SeedRoleAuthorizationPolicyByAuthorizer) {
		s.srap = module.SeedRoleAuthorizationPolicyByAuthorizer
	} else if srap.Enabled(module.SeedRoleAuthorizationPolicyByValidatorVotes) {
		s.srap = module.SeedRoleAuthorizationPolicyByValidatorVotes
	}
}

func (s *SeedManager) seedState() module.SeedState {
	s.ssMtx.RLock()
	defer s.ssMtx.RUnlock()
	return s.ss
}

func (s *SeedManager) seedRoleAuthorizationPolicy() module.SeedRoleAuthorizationPolicy {
	s.ssMtx.RLock()
	defer s.ssMtx.RUnlock()
	return s.srap
}

func (s *SeedManager) selectAuthorizer() module.PeerID {
	for _, id := range s.peers.Array() {
		if s.ss.IsAuthorizer(id) {
			return id
		}
	}
	return s.vsp.Last().Get(0)
}

func (s *SeedManager) upstream() module.PeerID {
	if s.isSeed(s.id) {
		for _, id := range s.peers.Array() {
			if s.isValidator(id) {
				return id
			}
		}
	} else {
		for _, id := range s.peers.Array() {
			if s.isSeed(id) {
				return id
			}
		}
	}
	return nil
}

func (s *SeedManager) isValidator(id module.PeerID) bool {
	return s.vsp.Last().Contains(id)
}

func (s *SeedManager) isSeed(id module.PeerID) bool {
	return s.seedsByState.Contains(id) || s.seedsBySV.Contains(id)
}

func (s *SeedManager) getSV(id module.PeerID) *SeedVerification {
	s.svMtx.RLock()
	defer s.svMtx.RUnlock()
	sv, ok := s.svMap[id]
	if !ok {
		return nil
	}
	return sv
}

func (s *SeedManager) putSV(id module.PeerID, sv *SeedVerification) {
	s.svMtx.Lock()
	defer s.svMtx.Unlock()
	s.svMap[id] = sv
}

func (s *SeedManager) isValidIssuer(srType SRType, issuer []byte, toAuthorize bool) bool {
	switch srType {
	case SRTypeByAuthorizer:
		return s.seedState().IsAuthorizer(NewPeerID(issuer))
	case SRTypeByValidatorVotes:
		if toAuthorize {
			return bytes.Equal(s.vsp.Last().LHash(), issuer)
		} else {
			return s.vsp.Get(issuer) != nil
		}
	}
	return false
}

func (s *SeedManager) newSR(height int64, srap module.SeedRoleAuthorizationPolicy) (*Signed[SeedRequest], error) {
	sr := &Signed[SeedRequest]{}
	sr.Message = SeedRequest{
		NetAddress: s.na,
		Height:     height,
	}
	switch srap {
	case module.SeedRoleAuthorizationPolicyByAuthorizer:
		sr.Message.Type = SRTypeByAuthorizer
		sr.Message.Issuer = s.selectAuthorizer().Bytes()
	case module.SeedRoleAuthorizationPolicyByValidatorVotes:
		sr.Message.Type = SRTypeByValidatorVotes
		sr.Message.Issuer = s.vsp.Last().LHash()
	default:
		return nil, errors.Errorf("not supported SeedRoleAuthorizationPolicy:%v", srap)
	}
	if err := sr.Sign(s.w); err != nil {
		return nil, errors.Wrapf(err, "fail to SR.Sign err:%v", err)
	}
	return sr, nil
}

func (s *SeedManager) svpWait(srap module.SeedRoleAuthorizationPolicy) time.Duration {
	switch srap {
	case module.SeedRoleAuthorizationPolicyByAuthorizer:
		return DefaultSVPWaitByAuthorizer
	case module.SeedRoleAuthorizationPolicyByValidatorVotes:
		return DefaultSVPWaitByValidatorVotes
	default:
		return -1
	}
}

func (s *SeedManager) onBlockUpdate(blk module.Block) error {
	s.svMtx.Lock()
	defer s.svMtx.Unlock()

	vs := NewValidatorSet(blk)
	vsUpdated := s.vsp.Update(vs)
	seedsByState := s.seedState().Seeds()
	seedsUpdated := s.seedsByState.ClearAndAdd(seedsByState...)

	height := blk.Height()
	for k, v := range s.svBuf {
		if !s.isValidIssuer(v.SRType(), v.Issuer(), false) || v.IsExpire(height) {
			delete(s.svBuf, k)
			continue
		}
		if s.isValidSignature(v) {
			s.svMap[s.id] = v
		}
	}

	var seedsBySV []module.PeerID
	for k, v := range s.svMap {
		if _, err := s.verifySV(v); err != nil {
			delete(s.svMap, k)
		} else {
			seedsBySV = append(seedsBySV, k)
		}
	}
	if s.seedsBySV.ClearAndAdd(seedsBySV...) {
		seedsUpdated = true
	}
	if seedsUpdated {
		//TODO NetworkManager.SetRole or PeerToPeer.getAllowed().ClearAndAdd()
		s.n.SetRole(blk.Height(), module.RoleSeed, append(seedsByState, seedsBySV...)...)
	}

	if s.isValidator(s.id) {
		//TODO accumulate tx metric
		//if first block of term : make SWV and multicast to SEED
		return nil
	}
	if s.seedsByState.Contains(s.id) {
		return nil
	}
	if !s.cr.Has(p2pRoleSeed) {
		return nil
	}

	// send SR if validatorSet changed
	if vsUpdated {
		if s.srTimer != nil {
			s.srTimer.Stop()
			s.srTimer = nil
		}
	}
	if s.srTimer == nil {
		srap := s.seedRoleAuthorizationPolicy()
		//TODO [TBD] SR duplication
		sv := &SeedVerification{}
		sr, err := s.newSR(height, srap)
		if err != nil {
			return err
		}
		sv.Message.SVR.Message.SR = *sr
		s.svBuf[height] = sv
		if s.seedsBySV.Contains(s.id) {
			svr := &sv.Message.SVR
			if err = svr.Sign(s.w); err != nil {
				return errors.Wrapf(err, "fail to SVR.Sign err:%v", err)
			}
			if err = s.sendSVR(svr); err != nil {
				return err
			}
		} else {
			if err = s.sendSR(sr); err != nil {
				return err
			}
		}
		s.srTimer = time.AfterFunc(s.svpWait(srap), func() {
			s.svMtx.Lock()
			defer s.svMtx.Unlock()
			s.l.Infof("SR timer expire")
			s.srTimer = nil
		})
	}
	return nil
}

func (s *SeedManager) authorize(svr *Signed[SeedVerificationRequest], srSigner module.PeerID) (*Signed[SeedVerificationPart], error) {
	//if !s.currentTerm(sr.Message.Height) errors.Errorf("invalid height")
	switch svr.Message.SRType() {
	case SRTypeByAuthorizer:
		if !s.seedState().IsAuthorizer(s.id) {
			return nil, errors.Errorf("not applicable SR")
		}
		issuer := NewPeerID(svr.Message.Issuer())
		if !s.id.Equal(issuer) {
			return nil, errors.Errorf("mismatch issuer:%v", issuer)
		}
		//FIXME condition ByAuthorizer
	case SRTypeByValidatorVotes:
		if !s.isValidator(s.id) {
			return nil, errors.Errorf("not applicable SR")
		}
		//FIXME condition ByValidatorVotes
	}
	s.svpMtx.Lock()
	defer s.svpMtx.Unlock()
	svp, ok := s.svpMap[srSigner]
	if !ok {
		svp = &Signed[SeedVerificationPart]{
			Message: SeedVerificationPart{
				SVR:    *svr,
				Expire: s.vsp.Height() + s.seedState().Term(),
			},
		}
		if err := svp.Sign(s.w); err != nil {
			return nil, errors.Wrapf(err, "fail to SVP.Sign err:%v", err)
		}
		s.svpMap[srSigner] = svp
	}
	return svp, nil
}

func (s *SeedManager) updateSVBySVP(svp *Signed[SeedVerificationPart], signer module.PeerID) error {
	s.svMtx.Lock()
	defer s.svMtx.Unlock()

	var sv *SeedVerification
	for _, v := range s.svBuf {
		//equal SR
		if bytes.Equal(v.Message.SVR.Message.SR.Signature, svp.Message.SVR.Message.SR.Signature) {
			sv = v
			break
		}
	}
	if sv == nil {
		return errors.Errorf("not found SV")
	}

	switch svp.Message.SRType() {
	case SRTypeByAuthorizer:
		if len(sv.Signatures) == 0 {
			sv.Message = svp.Message
		} else {
			return errors.Errorf("duplicated SVP signer:%s", signer)
		}
		if err := sv.Add(svp, signer); err != nil {
			return err
		}
		if s.srTimer != nil {
			s.srTimer.Stop()
		}
	case SRTypeByValidatorVotes:
		if len(sv.Signatures) == 0 {
			sv.Message = svp.Message
		}
		if sv.Contains(signer) {
			return errors.Errorf("duplicated SVP signer:%s", signer)
		}
		if err := sv.Add(svp, signer); err != nil {
			return err
		}
		//FIXME
		if !s.vsp.Get(sv.Issuer()).ContainsTwoThird(sv.Contains) {
			return errors.Errorf("invalid SV")
		}
	}

	return nil
}

func (s *SeedManager) Start(bm module.BlockManager, sss module.SeedStateSupply) error {
	//TODO module.SeedStateSupply : Chain or ServiceManager

	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.cancel != nil {
		return nil
	}
	//TODO NetworkManager.RegisterReactor or integrate with PeerToPeer protocol
	ph, err := s.n.RegisterReactor("seedmgr", module.ProtoReserved, s, smProtocols, 1, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		return err
	}
	s.ph = ph

	blk, err := bm.GetLastBlock()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go func() {
		var (
			bch <-chan module.Block
			ok  bool
		)
		for {
			if bch, err = bm.WaitForBlock(blk.Height() + 1); err != nil {
				s.l.Panicf("fail to WaitForBlock err:%+v", err)
			}
			select {
			case blk, ok = <-bch:
				if !ok {
					log.Panicf("closed BlockManager.WaitForBlock")
					return
				}
				s.setSeedState(sss.SeedState(blk.Result()))
				if err = s.onBlockUpdate(blk); err != nil {
					s.l.Errorf("fail to onBlockUpdate err:%+v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (s *SeedManager) Stop() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.cancel == nil {
		return
	}
	s.cancel()
	s.cancel = nil
}

func (s *SeedManager) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	switch pi {
	case protoSR:
		return s.handleSR(b, id)
	case protoSVR:
		return s.handleSVR(b, id)
	case protoSVP:
		return s.handleSVP(b, id)
	default:
		return false, errors.Errorf("not supported protocol:%v", pi)
	}
}

func (s *SeedManager) OnJoin(id module.PeerID) {
	s.peers.Add(id)
}

func (s *SeedManager) OnLeave(id module.PeerID) {
	s.peers.Remove(id)
}

func (s *SeedManager) sendSR(sr *Signed[SeedRequest]) error {
	to := s.upstream()
	if to == nil {
		return errors.Errorf("not available")
	}
	b, err := sr.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	if err = s.ph.Unicast(protoSR, b, to); err != nil {
		return errors.Wrapf(err, "fail to Unicast err:%v", err)
	}
	return nil
}

func (s *SeedManager) sendSVR(svr *Signed[SeedVerificationRequest]) error {
	b, err := svr.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	switch svr.Message.SRType() {
	case SRTypeByAuthorizer:
		to := NewPeerID(svr.Message.Issuer())
		if !s.peers.Contains(to) {
			to = s.upstream()
		}
		if to == nil {
			return errors.Errorf("not available")
		}
		if err = s.ph.Unicast(protoSVR, b, to); err != nil {
			return errors.Wrapf(err, "fail to Unicast err:%v", err)
		}
	case SRTypeByValidatorVotes:
		if err = s.ph.Multicast(protoSVR, b, module.RoleValidator); err != nil {
			return errors.Wrapf(err, "fail to Multicast err:%v", err)
		}
	}
	return nil
}

func (s *SeedManager) sendSVP(svp *Signed[SeedVerificationPart], id module.PeerID) error {
	b, err := svp.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	return s.sendWithResolver(protoSVP, b, id, true)
}

func (s *SeedManager) sendWithResolver(pi module.ProtocolInfo, b []byte, id module.PeerID, broadcastIfResolveFailure bool) error {
	to, _ := s.rr.Resolve(id)
	if to == nil {
		n := 0
		if broadcastIfResolveFailure {
			for _, v := range s.peers.Array() {
				if !s.isValidator(v) {
					continue
				}
				if err := s.ph.Unicast(pi, b, v); err != nil {
					s.l.Infof("fail to Unicast err:%v", err)
				} else {
					n++
				}
			}
		}
		if n == 0 {
			return ErrNotAvailable
		}
	} else {
		if err := s.ph.Unicast(pi, b, to); err != nil {
			return errors.Wrapf(err, "fail to Unicast err:%v", err)
		}
	}
	return nil
}

func (s *SeedManager) verifySR(sr *Signed[SeedRequest], srSigner module.PeerID, toAuthorize bool) error {
	if !s.seedState().IsCandidate(srSigner) {
		return errors.Errorf("invalid SR signer:%v", srSigner)
	}
	switch sr.Message.Type {
	case SRTypeByAuthorizer, SRTypeByValidatorVotes:
		if !s.isValidIssuer(sr.Message.Type, sr.Message.Issuer, toAuthorize) {
			return errors.Errorf("invalid SR issuer")
		}
	default:
		return errors.Errorf("not supported SR type:%v", sr.Message.Type)
	}
	return nil
}

func (s *SeedManager) handleSR(b []byte, id module.PeerID) (bool, error) {
	if !s.isSeed(s.id) {
		return false, errors.Errorf("not applicable SR")
	}
	sr, srSigner, err := NewSignedFromBytes[SeedRequest](b)
	if err != nil {
		return false, errors.Wrapf(err, "fail to NewSignedFromBytes err:%v", err)
	}
	if err = s.verifySR(sr, srSigner, true); err != nil {
		return false, err
	}

	svr := &Signed[SeedVerificationRequest]{}
	svr.Message.SR = *sr
	if err = svr.Sign(s.w); err != nil {
		return false, errors.Wrapf(err, "fail to SVR.Sign err:%v", err)
	}

	switch sr.Message.Type {
	case SRTypeByAuthorizer:
		issuer := NewPeerID(sr.Message.Issuer)
		if s.id.Equal(issuer) {
			var svp *Signed[SeedVerificationPart]
			if svp, err = s.authorize(svr, srSigner); err != nil {
				return false, err
			}
			if err = s.sendSVP(svp, id); err != nil {
				return false, err
			}
			return false, nil
		}
	}
	return false, s.sendSVR(svr)
}

func (s *SeedManager) verifySVR(svr *Signed[SeedVerificationRequest], svrSigner module.PeerID, toAuthorize bool) (module.PeerID, error) {
	srSigner, err := svr.Message.SR.Recover()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to SVR.SR.Recover err:%v", err)
	}
	if err = s.verifySR(&svr.Message.SR, srSigner, toAuthorize); err != nil {
		return nil, err
	}
	if !s.isSeed(svrSigner) {
		return nil, errors.Errorf("invalid SVR signer:%v", svrSigner)
	}
	return srSigner, nil
}

func (s *SeedManager) handleSVR(b []byte, id module.PeerID) (bool, error) {
	if !(s.isValidator(s.id) || s.seedState().IsAuthorizer(s.id)) {
		return false, errors.Errorf("not applicable SVR")
	}
	svr, svrSigner, err := NewSignedFromBytes[SeedVerificationRequest](b)
	if err != nil {
		return false, errors.Wrapf(err, "fail to network.NewSignedFromBytes err:%v", err)
	}
	srSigner, err := s.verifySVR(svr, svrSigner, true)
	if err != nil {
		return false, err
	}
	var svp *Signed[SeedVerificationPart]
	switch svr.Message.SRType() {
	case SRTypeByAuthorizer:
		issuer := NewPeerID(svr.Message.Issuer())
		if !s.id.Equal(issuer) {
			if err = s.sendWithResolver(protoSVR, b, issuer, svrSigner.Equal(id)); err != nil {
				return false, errors.Wrapf(err, "fail to sendWithResolver err:%v", err)
			}
			return false, nil
		}
	}
	if svp, err = s.authorize(svr, srSigner); err != nil {
		return false, err
	}
	if err = s.sendSVP(svp, svrSigner); err != nil {
		return false, err
	}
	return svr.Message.SRType() == SRTypeByValidatorVotes, nil
}

func (s *SeedManager) verifySVP(svp *Signed[SeedVerificationPart], svpSigner module.PeerID) (module.PeerID, module.PeerID, error) {
	svrSigner, err := svp.Message.SVR.Recover()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to SVP.SVR.Recover err:%v", err)
	}
	srSigner, err := s.verifySVR(&svp.Message.SVR, svrSigner, false)
	if err != nil {
		return nil, nil, err
	}
	if svp.Message.IsExpire(s.vsp.Height()) {
		return nil, nil, errors.Errorf("invalid SVP expired")
	}
	switch svp.Message.SRType() {
	case SRTypeByAuthorizer:
		issuer := NewPeerID(svp.Message.SVR.Message.Issuer())
		if !svpSigner.Equal(issuer) {
			return nil, nil, errors.Errorf("invalid SVP mismatch issuer:%s signer:%s", issuer, svpSigner)
		}
	case SRTypeByValidatorVotes:
		if !s.vsp.Last().Contains(svpSigner) {
			return nil, nil, errors.Errorf("invalid SVP signer")
		}
	}
	return svrSigner, srSigner, nil
}

func (s *SeedManager) handleSVP(b []byte, id module.PeerID) (bool, error) {
	// TODO [TBD] ignore SVP if citizen
	//if !(s.isValidator(s.id) || s.ss.IsAuthorizer(s.id) || s.isSeed(s.id) || s.cr.Has(p2pRoleSeed)) {
	//	return false, errors.Errorf("not applicable SVP")
	//}
	svp, svpSigner, err := NewSignedFromBytes[SeedVerificationPart](b)
	if err != nil {
		return false, errors.Wrapf(err, "fail to NewSignedFromBytes err:%v", err)
	}
	svrSigner, srSigner, err := s.verifySVP(svp, svpSigner)
	if err != nil {
		return false, nil
	}
	if s.id.Equal(srSigner) {
		return false, s.updateSVBySVP(svp, svpSigner)
	}
	if s.id.Equal(svrSigner) {
		//redundant
		if !s.peers.Contains(srSigner) {
			return false, errors.Errorf("not found candidate connection:%v", srSigner)
		}
		if err = s.ph.Unicast(protoSVP, b, srSigner); err != nil {
			return false, errors.Wrapf(err, "fail to Unicast err:%v", err)
		}
	} else {
		if err = s.sendWithResolver(protoSVP, b, svrSigner, false); err != nil {
			return false, errors.Wrapf(err, "fail to sendWithResolver err:%v", err)
		}
	}
	return false, nil
}

func (s *SeedManager) isValidSignature(sv *SeedVerification) bool {
	switch sv.SRType() {
	case SRTypeByAuthorizer:
		if len(sv.Signatures) != 1 {
			return false
		}
		return sv.Contains(NewPeerID(sv.Issuer()))
	case SRTypeByValidatorVotes:
		return s.vsp.Get(sv.Issuer()).ContainsTwoThird(sv.Contains)
	}
	return false
}

func (s *SeedManager) verifySV(sv *SeedVerification) (module.PeerID, error) {
	svrSigner, err := sv.Message.SVR.Recover()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to SV.SVR.Recover err:%v", err)
	}
	srSigner, err := s.verifySVR(&sv.Message.SVR, svrSigner, false)
	if err != nil {
		return nil, err
	}
	if sv.IsExpire(s.vsp.Height()) {
		return nil, errors.Errorf("invalid SV expired")
	}
	switch sv.SRType() {
	case SRTypeByAuthorizer:
		if len(sv.Signatures) != 1 {
			return nil, errors.Errorf("invalid SV signature")
		}
		issuer := NewPeerID(sv.Issuer())
		if !sv.Contains(issuer) {
			return nil, errors.Errorf("invalid SV mismatch issuer:%s signer:%s", issuer, sv.signers[0])
		}
		if !s.seedState().IsAuthorizer(issuer) {
			return nil, errors.Errorf("invalid SV invalid signer")
		}
	case SRTypeByValidatorVotes:
		if !s.vsp.Get(sv.Issuer()).ContainsTwoThird(sv.Contains) {
			return nil, errors.Errorf("invalid SV not enough signature")
		}
	}
	return srSigner, nil
}

//TODO QueryProtocol use this
func (s *SeedManager) handleSV(b []byte, id module.PeerID) error {
	sv := &SeedVerification{}
	if err := sv.UnmarshalBinary(b); err != nil {
		return err
	}
	srSigner, err := s.verifySV(sv)
	if err != nil {
		return err
	}
	if !srSigner.Equal(id) {
		return errors.Errorf("mismatch SR signer:%s expected:%s", srSigner, id)
	}
	s.putSV(id, sv)
	return nil
}
