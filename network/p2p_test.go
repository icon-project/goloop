package network

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/log"
)

func Test_PeerToPeer_resolveConnection(t *testing.T) {
	p2p := &PeerToPeer{
		self: &Peer{id: generatePeerID()},
	}
	p2p.rr = newRoleResolver(p2p.self, p2p.onAllowedPeerIDSetUpdate, log.GlobalLogger())
	p2p.pm = &peerManager{self: p2p.self}
	p2p.as = newAddressSyncer(nil, p2p.pm, log.GlobalLogger())
	type reqGiven struct {
		r  PeerRoleFlag
		pr PeerRoleFlag
		c  PeerConnectionType
	}
	type reqExpected struct {
		rc         PeerConnectionType
		notAllowed bool
		invalidReq bool
	}
	type reqArg struct {
		given    reqGiven
		expected reqExpected
	}
	reqSucc := func(r, pr PeerRoleFlag, c, rc PeerConnectionType) reqArg {
		return reqArg{
			given:    reqGiven{r, pr, c},
			expected: reqExpected{rc, false, false},
		}
	}
	reqFail := func(r, pr PeerRoleFlag, c PeerConnectionType, notAllowed, invalidReq bool) reqArg {
		return reqArg{
			given:    reqGiven{r, pr, c},
			expected: reqExpected{p2pConnTypeNone, notAllowed, invalidReq},
		}
	}
	reqArgs := []reqArg{
		//self.Role == p2pRoleRoot
		reqSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeFriend),
		reqSucc(p2pRoleRoot, p2pRoleSeed, p2pConnTypeFriend, p2pConnTypeOther),
		reqFail(p2pRoleRoot, p2pRoleNone, p2pConnTypeFriend, true, false),
		reqSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeOther),
		reqSucc(p2pRoleRoot, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeChildren),
		reqFail(p2pRoleRoot, p2pRoleNone, p2pConnTypeParent, true, false),
		reqSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeOther),
		reqSucc(p2pRoleRoot, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeNephew),
		reqFail(p2pRoleRoot, p2pRoleNone, p2pConnTypeUncle, true, false),
		reqSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeNone, p2pConnTypeNone), //
		reqFail(p2pRoleRoot, p2pRoleRoot, p2pConnTypeChildren, false, true),
		//self.Role == p2pRoleSeed
		reqSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeParent),
		reqSucc(p2pRoleSeed, p2pRoleSeed, p2pConnTypeFriend, p2pConnTypeOther),
		reqFail(p2pRoleSeed, p2pRoleNone, p2pConnTypeFriend, false, true),
		reqSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeNone),
		reqSucc(p2pRoleSeed, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeNone),
		reqSucc(p2pRoleSeed, p2pRoleNone, p2pConnTypeParent, p2pConnTypeChildren),
		reqSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeNone),
		reqSucc(p2pRoleSeed, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeNone),
		reqSucc(p2pRoleSeed, p2pRoleNone, p2pConnTypeUncle, p2pConnTypeNephew),
		reqSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeNone, p2pConnTypeNone), //
		reqFail(p2pRoleSeed, p2pRoleRoot, p2pConnTypeChildren, false, true),
		//self.Role == p2pRoleNone
		reqSucc(p2pRoleNone, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeNone),
		reqSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeNone),
		reqSucc(p2pRoleNone, p2pRoleNone, p2pConnTypeParent, p2pConnTypeChildren),
		reqSucc(p2pRoleNone, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeNone),
		reqSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeNone),
		reqSucc(p2pRoleNone, p2pRoleNone, p2pConnTypeUncle, p2pConnTypeNephew),
		reqSucc(p2pRoleNone, p2pRoleRoot, p2pConnTypeNone, p2pConnTypeNone), //
		reqFail(p2pRoleNone, p2pRoleRoot, p2pConnTypeChildren, false, true),
	}
	for _, arg := range reqArgs {
		p2p.setRole(arg.given.r)
		rc, notAllowed, invalidReq := p2p.pm.resolveConnectionRequest(arg.given.pr, arg.given.c)
		assert.Equal(t, arg.expected.rc, rc)
		assert.Equal(t, arg.expected.notAllowed, notAllowed)
		assert.Equal(t, arg.expected.invalidReq, invalidReq)
	}

	type respGiven struct {
		r    PeerRoleFlag
		prr  PeerRoleFlag
		req  PeerConnectionType
		resp PeerConnectionType
	}
	type respExpected struct {
		rc          PeerConnectionType
		rejectResp  bool
		invalidResp bool
	}
	type respArg struct {
		given    respGiven
		expected respExpected
	}
	respSucc := func(r, prr PeerRoleFlag, req, resp, rc PeerConnectionType) respArg {
		return respArg{
			given:    respGiven{r, prr, req, resp},
			expected: respExpected{rc, false, false},
		}
	}
	respFail := func(r, prr PeerRoleFlag, req, resp PeerConnectionType, rejectResp, invalidResp bool) respArg {
		return respArg{
			given:    respGiven{r, prr, req, resp},
			expected: respExpected{p2pConnTypeNone, rejectResp, invalidResp},
		}
	}
	respArgs := []respArg{
		//self.Role == p2pRoleRoot
		////req == p2pConnTypeFriend
		//////resp == p2pConnTypeFriend
		respSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeFriend, p2pConnTypeFriend),
		//////resp == p2pConnTypeOther or resp == p2pConnTypeNone for legacy support
		respSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeFriend),
		respSucc(p2pRoleRoot, p2pRoleSeed, p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeOther),
		respSucc(p2pRoleRoot, p2pRoleNone, p2pConnTypeFriend, p2pConnTypeOther, p2pConnTypeOther),
		respSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeNone, p2pConnTypeFriend),
		respSucc(p2pRoleRoot, p2pRoleSeed, p2pConnTypeFriend, p2pConnTypeNone, p2pConnTypeOther),
		respSucc(p2pRoleRoot, p2pRoleNone, p2pConnTypeFriend, p2pConnTypeNone, p2pConnTypeOther),
		//////resp == p2pConnTypeParent or resp == p2pConnTypeUncle
		respSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeParent, p2pConnTypeOther),
		respSucc(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeUncle, p2pConnTypeOther),
		//////invalidResp
		respFail(p2pRoleRoot, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeChildren, false, true),
		////invalidResp
		respFail(p2pRoleRoot, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeChildren, false, true),
		//self.Role == p2pRoleSeed
		////req == p2pConnTypeParent
		//////resp == p2pConnTypeChildren or resp == p2pConnTypeOther
		respSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeChildren, p2pConnTypeParent),
		respSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeOther, p2pConnTypeParent),
		//////rejectResp
		respFail(p2pRoleSeed, p2pRoleRoot, p2pConnTypeParent, p2pConnTypeNone, true, false),
		////req == p2pConnTypeUncle
		//////resp == p2pConnTypeNephew or resp == p2pConnTypeOther
		respSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeNephew, p2pConnTypeUncle),
		respSucc(p2pRoleSeed, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeOther, p2pConnTypeUncle),
		//////rejectResp
		respFail(p2pRoleSeed, p2pRoleRoot, p2pConnTypeUncle, p2pConnTypeNone, true, false),
		////invalidResp
		respFail(p2pRoleSeed, p2pRoleRoot, p2pConnTypeFriend, p2pConnTypeFriend, false, true),
		//self.Role == p2pRoleNone
		////req == p2pConnTypeParent
		respSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeChildren, p2pConnTypeParent),
		respSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeOther, p2pConnTypeOther),
		respFail(p2pRoleSeed, p2pRoleSeed, p2pConnTypeParent, p2pConnTypeNone, true, false),
		////req == p2pConnTypeUncle
		respSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeNephew, p2pConnTypeUncle),
		respSucc(p2pRoleNone, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeOther, p2pConnTypeOther),
		respFail(p2pRoleNone, p2pRoleSeed, p2pConnTypeUncle, p2pConnTypeNone, true, false),
		////invalidResp
		respFail(p2pRoleNone, p2pRoleSeed, p2pConnTypeFriend, p2pConnTypeFriend, false, true),
	}
	for _, arg := range respArgs {
		p2p.setRole(arg.given.r)
		rc, rejectResp, invalidResp := p2p.pm.resolveConnectionResponse(arg.given.prr, arg.given.req, arg.given.resp)
		assert.Equal(t, arg.expected.rc, rc)
		assert.Equal(t, arg.expected.rejectResp, rejectResp)
		assert.Equal(t, arg.expected.invalidResp, invalidResp)
	}
}
