package network

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
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
	protoSR  = module.ProtocolInfo(0x0D00)
	protoSVR = module.ProtocolInfo(0x0E00)
	protoSVP = module.ProtocolInfo(0x0F00)

	ctrV1pp = PeerPredicates.Protocol(p2pProtoControlV1)
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

func (sr SeedRequest) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "SR{Type:%v,Issuer:%v,NetAddress:%v,Height:%v}",
			sr.Type, hex.EncodeToString(sr.Issuer), sr.NetAddress, sr.Height)
	case 's':
		fmt.Fprintf(f, "{Type:%v,Issuer:%v,NetAddress:%v,Height:%v}",
			sr.Type, hex.EncodeToString(sr.Issuer), sr.NetAddress, sr.Height)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

// SeedVerificationRequest is used for short-term-seed authorization request by seed
type SeedVerificationRequest struct {
	SR *Signed[SeedRequest]
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

func (svr SeedVerificationRequest) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "SVR{SR:%v}", svr.SR)
	case 's':
		fmt.Fprintf(f, "{SR:%s}", svr.SR)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
}

// SeedVerificationPart is used for authorization response of SeedVerificationRequest by validator.
type SeedVerificationPart struct {
	SVR    *Signed[SeedVerificationRequest]
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

func (svp SeedVerificationPart) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		fmt.Fprintf(f, "SVP{SVR:%v,Expire:%v}", svp.SVR, svp.Expire)
	case 's':
		fmt.Fprintf(f, "{SVR:%s,Expire:%v}", svp.SVR, svp.Expire)
	default:
		panic(fmt.Sprintf("UknownRune(rune=%c)", verb))
	}
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
	c    module.Chain
	w    module.Wallet
	rr   *roleResolver
	pm   *peerManager
	send func(*Packet) error

	cancel context.CancelFunc
	mtx    sync.Mutex

	id    module.PeerID
	na    NetAddress
	cr    PeerRoleFlag
	csrap module.SeedRoleAuthorizationPolicy

	vsp *ValidatorSetCache

	seeds *PeerIDSet

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

func newSeedManager(
	c module.Chain,
	self *Peer,
	pm *peerManager,
	rr *roleResolver,
	relayableSendFunc func(*Packet) error,
	l log.Logger) *SeedManager {
	return &SeedManager{
		c:      c,
		rr:     rr,
		pm:     pm,
		send:   relayableSendFunc,
		id:     self.ID(),
		na:     self.NetAddress(),
		cr:     self.Role(),
		vsp:    NewValidatorSetCache(DefaultSVIssuerExpireDelay),
		seeds:  NewPeerIDSet(),
		svpMap: make(map[module.PeerID]*Signed[SeedVerificationPart]),
		svMap:  make(map[module.PeerID]*SeedVerification),
		svBuf:  make(map[int64]*SeedVerification),
		l:      l.WithFields(log.Fields{LoggerFieldKeySubModule: "seedmgr"}),
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
	pp := ctrV1pp.And(func(p *Peer) bool {
		return s.seedState().IsAuthorizer(p.ID())
	})
	p := s.pm.findPeer(pp, p2pConnTypeParent, p2pConnTypeUncle)
	if p == nil {
		return s.vsp.Last().Get(0)
	} else {
		return p.ID()
	}
}

func (s *SeedManager) upstream() (*Peer, error) {
	p := s.pm.findPeer(ctrV1pp, p2pConnTypeParent, p2pConnTypeUncle)
	if p == nil {
		return nil, ErrNotAvailable
	}
	return p, nil
}

func (s *SeedManager) resolve(id module.PeerID) (*Peer, int) {
	p := s.pm.findPeer(PeerPredicates.ID(id), joinPeerConnectionTypes...)
	if p != nil {
		return p, 1
	}
	pap := func(v PeerAddress) bool {
		return id.Equal(v.PeerID())
	}
	pp := func(p *Peer) bool {
		_, found := p.FindConn(pap)
		return found
	}
	p = s.pm.findPeer(pp, joinPeerConnectionTypes...)
	if p == nil {
		return nil, 0
	}
	return p, 2
}

func (s *SeedManager) isValidator(id module.PeerID) bool {
	return s.vsp.Last().Contains(id)
}

func (s *SeedManager) isSeed(id module.PeerID) bool {
	return s.seeds.Contains(id)
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
	s._updateSeeds(id)
}

func (s *SeedManager) _replaceSV(sv *SeedVerification) {
	//TODO SV replacement condition
	s.svMap[s.id] = sv
	s._updateSeeds(s.id)
	if s.srTimer != nil {
		s.srTimer.Stop()
	}
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

func (s *SeedManager) _updateSeeds(id module.PeerID) {
	if s.seeds.Add(id) {
		s.seeds.version++
		s.rr.updateAllowed(s.seeds.version, p2pRoleSeed, s.seeds.Array()...)
	}
}

func (s *SeedManager) onBlockUpdate(blk module.Block) error {
	s.svMtx.Lock()
	defer s.svMtx.Unlock()

	vs := NewValidatorSet(blk)
	vsUpdated := s.vsp.Update(vs)
	seedsByState := s.seedState().Seeds()
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
	if s.seeds.ClearAndAdd(append(seedsByState, seedsBySV...)...) {
		s.seeds.version++
		s.rr.updateAllowed(s.seeds.version, p2pRoleSeed, s.seeds.Array()...)
	}

	if s.isValidator(s.id) {
		//TODO accumulate tx metric
		//if first block of term : make SWV and multicast to SEED
		return nil
	}
	for _, id := range seedsByState {
		if id.Equal(s.id) {
			return nil
		}
	}
	if !s.cr.Has(p2pRoleSeed) {
		return nil
	}
	srap := s.seedRoleAuthorizationPolicy()
	switch srap {
	case module.SeedRoleAuthorizationPolicyByAuthorizer, module.SeedRoleAuthorizationPolicyByValidatorVotes:
	default:
		return nil
	}

	// send SR if validatorSet changed
	if vsUpdated {
		if s.srTimer != nil {
			s.srTimer.Stop()
			s.srTimer = nil
			s.l.Debugf("validatorSet changed and clear srTimer")
		}
	}
	if s.srTimer == nil {
		//TODO [TBD] SR duplication
		sv := &SeedVerification{
			MultiSigned: MultiSigned[SeedVerificationPart]{
				Message: SeedVerificationPart{
					SVR: &Signed[SeedVerificationRequest]{},
				},
			},
		}
		sr, err := s.newSR(height, srap)
		if err != nil {
			return err
		}
		sv.Message.SVR.Message.SR = sr
		s.svBuf[height] = sv
		if _, ok := s.svMap[s.id]; ok {
			svr := sv.Message.SVR
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
				SVR:    svr,
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
		s._replaceSV(sv)
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
		if s.vsp.Get(sv.Issuer()).ContainsTwoThird(sv.Contains) {
			s._replaceSV(sv)
		}
	}
	return nil
}

type seedRoleAuthorizationPolicyConfigure interface {
	SeedRoleAuthorizationPolicy() module.SeedRoleAuthorizationPolicy
}

func (s *SeedManager) Start() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.cancel != nil {
		return nil
	}
	//TODO Chain.SeedRoleAuthorizationPolicy()
	if csrap, ok := s.c.(seedRoleAuthorizationPolicyConfigure); ok {
		s.csrap = csrap.SeedRoleAuthorizationPolicy()
	}
	s.w = s.c.Wallet()
	bm := s.c.BlockManager()
	//TODO module.SeedStateSupply : Chain or ServiceManager
	sss := s.c.ServiceManager().(module.SeedStateSupply)

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
				var ss module.SeedState
				if ss, err = sss.SeedState(blk.Result()); err != nil {
					s.l.Errorf("fail to SeedStateSupply.SeedState err:%+v", err)
					continue
				}
				s.setSeedState(ss)
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

func (s *SeedManager) sendToPeer(pi module.ProtocolInfo, b []byte, p *Peer) error {
	pkt := newPacket(p2pProtoControlV1, pi, b, s.id)
	pkt.destPeer = p.ID()
	return p.sendPacket(pkt)
}

func (s *SeedManager) multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	pkt := NewPacket(p2pProtoControlV1, pi, b)
	pkt.dest = byte(role)
	if err := s.send(pkt); err != nil {
		return newNetworkError(err, "multicast", role)
	}
	return nil
}

func (s *SeedManager) onPacket(pkt *Packet, p *Peer) bool {
	var err error
	switch pkt.subProtocol {
	case protoSR:
		err = s.handleSR(pkt, p)
	case protoSVR:
		err = s.handleSVR(pkt, p)
	case protoSVP:
		err = s.handleSVP(pkt, p)
	default:
		return false
	}
	if err != nil {
		s.l.Debugf("onPacket err:%+v", err)
	}
	return true
}

func (s *SeedManager) sendSR(sr *Signed[SeedRequest]) error {
	b, err := sr.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	p, err := s.upstream()
	if err != nil {
		return err
	}
	s.l.Debugln("sendSR", sr, p)
	return s.sendToPeer(protoSR, b, p)
}

func (s *SeedManager) sendSVR(svr *Signed[SeedVerificationRequest]) error {
	b, err := svr.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	switch svr.Message.SRType() {
	case SRTypeByAuthorizer:
		p, _ := s.resolve(NewPeerID(svr.Message.Issuer()))
		if p == nil {
			if p, err = s.upstream(); err != nil {
				return err
			}
		}
		s.l.Debugln("sendSVR", svr, p)
		return s.sendToPeer(protoSVR, b, p)
	case SRTypeByValidatorVotes:
		return s.multicast(protoSVR, b, module.RoleValidator)
	}
	return nil
}

func (s *SeedManager) sendSVP(svp *Signed[SeedVerificationPart], id module.PeerID) error {
	b, err := svp.MarshalBinary()
	if err != nil {
		return errors.Wrapf(err, "fail to MarshalBinary err:%v", err)
	}
	s.l.Debugln("sendSVP", svp, id)
	return s.sendWithResolver(protoSVP, b, id, true)
}

func (s *SeedManager) sendWithResolver(pi module.ProtocolInfo, b []byte, id module.PeerID, broadcastIfResolveFailure bool) error {
	to, hop := s.resolve(id)
	s.l.Traceln("sendWithResolver", "id:", id, "to:", to, hop)
	if to == nil {
		if broadcastIfResolveFailure {
			ps := s.pm.findPeers(ctrV1pp, p2pConnTypeFriend)
			if len(ps) == 0 {
				return ErrNotAvailable
			}
			for _, p := range ps {
				if err := s.sendToPeer(pi, b, p); err != nil {
					s.l.Infof("fail to broadcast err:%v pi:%s peer:%s", err, pi, p.ID())
				}
			}
			return nil
		}
		return ErrNotAvailable
	} else {
		if err := s.sendToPeer(pi, b, to); err != nil {
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

func (s *SeedManager) handleSR(pkt *Packet, p *Peer) error {
	if !s.isSeed(s.id) {
		return errors.Errorf("not applicable SR")
	}
	sr, srSigner, err := NewSignedFromBytes[SeedRequest](pkt.payload)
	if err != nil {
		return errors.Wrapf(err, "fail to NewSignedFromBytes err:%v", err)
	}
	s.l.Traceln("handleSR", sr, p)
	if err = s.verifySR(sr, srSigner, true); err != nil {
		return err
	}

	svr := &Signed[SeedVerificationRequest]{}
	svr.Message.SR = sr
	if err = svr.Sign(s.w); err != nil {
		return errors.Wrapf(err, "fail to SVR.Sign err:%v", err)
	}

	switch sr.Message.Type {
	case SRTypeByAuthorizer:
		issuer := NewPeerID(sr.Message.Issuer)
		if s.id.Equal(issuer) {
			var svp *Signed[SeedVerificationPart]
			if svp, err = s.authorize(svr, srSigner); err != nil {
				return err
			}
			var b []byte
			if b, err = svp.MarshalBinary(); err != nil {
				return err
			}
			return s.sendToPeer(protoSVP, b, p)
		}
	}
	return s.sendSVR(svr)
}

func (s *SeedManager) verifySVR(svr *Signed[SeedVerificationRequest], svrSigner module.PeerID, toAuthorize bool) (module.PeerID, error) {
	srSigner, err := svr.Message.SR.Recover()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to SVR.SR.Recover err:%v", err)
	}
	if err = s.verifySR(svr.Message.SR, srSigner, toAuthorize); err != nil {
		return nil, err
	}
	if !s.isSeed(svrSigner) {
		return nil, errors.Errorf("invalid SVR signer:%v", svrSigner)
	}
	return srSigner, nil
}

func (s *SeedManager) handleSVR(pkt *Packet, p *Peer) error {
	if !(s.isValidator(s.id) || s.seedState().IsAuthorizer(s.id)) {
		return errors.Errorf("not applicable SVR")
	}
	svr, svrSigner, err := NewSignedFromBytes[SeedVerificationRequest](pkt.payload)
	if err != nil {
		return errors.Wrapf(err, "fail to network.NewSignedFromBytes err:%v", err)
	}
	s.l.Traceln("handleSVR", svr, p)
	srSigner, err := s.verifySVR(svr, svrSigner, true)
	if err != nil {
		return err
	}
	var svp *Signed[SeedVerificationPart]
	switch svr.Message.SRType() {
	case SRTypeByAuthorizer:
		issuer := NewPeerID(svr.Message.Issuer())
		if !s.id.Equal(issuer) {
			if err = s.sendWithResolver(protoSVR, pkt.payload, issuer, svrSigner.Equal(p.ID())); err != nil {
				return errors.Wrapf(err, "fail to sendWithResolver err:%v", err)
			}
			return nil
		}
	}
	if svp, err = s.authorize(svr, srSigner); err != nil {
		return err
	}
	if err = s.sendSVP(svp, svrSigner); err != nil {
		return err
	}
	if svr.Message.SRType() == SRTypeByValidatorVotes {
		return s.send(pkt)
	}
	return nil
}

func (s *SeedManager) verifySVP(svp *Signed[SeedVerificationPart], svpSigner module.PeerID) (module.PeerID, module.PeerID, error) {
	svrSigner, err := svp.Message.SVR.Recover()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to SVP.SVR.Recover err:%v", err)
	}
	srSigner, err := s.verifySVR(svp.Message.SVR, svrSigner, false)
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

func (s *SeedManager) handleSVP(pkt *Packet, p *Peer) error {
	// TODO [TBD] ignore SVP if citizen
	//if !(s.isValidator(s.id) || s.ss.IsAuthorizer(s.id) || s.isSeed(s.id) || s.cr.Has(p2pRoleSeed)) {
	//	return false, errors.Errorf("not applicable SVP")
	//}
	svp, svpSigner, err := NewSignedFromBytes[SeedVerificationPart](pkt.payload)
	if err != nil {
		return errors.Wrapf(err, "fail to NewSignedFromBytes err:%v", err)
	}
	s.l.Traceln("handleSVP", svp, p)
	svrSigner, srSigner, err := s.verifySVP(svp, svpSigner)
	if err != nil {
		return nil
	}
	if s.id.Equal(srSigner) {
		return s.updateSVBySVP(svp, svpSigner)
	}
	to := svrSigner
	if s.id.Equal(svrSigner) {
		to = srSigner
	}
	return s.sendWithResolver(protoSVP, pkt.payload, to, false)
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
	srSigner, err := s.verifySVR(sv.Message.SVR, svrSigner, false)
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

func (s *SeedManager) handleSV(b []byte, id module.PeerID) error {
	sv := &SeedVerification{}
	if err := sv.UnmarshalBinary(b); err != nil {
		return err
	}
	s.l.Traceln("handleSV", sv, id)
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
