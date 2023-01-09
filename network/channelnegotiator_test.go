package network

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

const (
	testNetAddress = "test"
)

func Test_ChannelNegotiator(t *testing.T) {
	c := newChannelNegotiator(testNetAddress, generatePeerID(), testLogger())

	nextProtoP2P := module.NewProtocolInfo(module.ProtoP2P.ID(), module.ProtoP2P.Version()+1)
	noErrorArgs := []struct {
		given            []module.ProtocolInfo
		param            []module.ProtocolInfo
		expected         []module.ProtocolInfo
		isSupportDefault bool
		isLegacy         bool
	}{
		{ //default protocols
			given:            defaultProtocols,
			param:            defaultProtocols,
			expected:         defaultProtocols,
			isSupportDefault: true,
			isLegacy:         false,
		},
		{ //multi-versioned protocols
			given:            []module.ProtocolInfo{module.ProtoP2P, nextProtoP2P},
			param:            []module.ProtocolInfo{module.ProtoP2P, nextProtoP2P},
			expected:         []module.ProtocolInfo{nextProtoP2P},
			isSupportDefault: false, //Peer.GetAttr should return (nil, false)
			isLegacy:         false,
		},
		{ //multi-versioned protocols
			given:            append(defaultProtocols, nextProtoP2P),
			param:            []module.ProtocolInfo{module.ProtoP2P, nextProtoP2P},
			expected:         []module.ProtocolInfo{nextProtoP2P},
			isSupportDefault: false, //Peer.GetAttr should return (bool(false), true)
			isLegacy:         false,
		},
		{ //legacy
			given:            defaultProtocols,
			param:            nil,
			expected:         defaultProtocols,
			isSupportDefault: true,
			isLegacy:         true,
		},
	}
	for _, arg := range noErrorArgs {
		for _, pi := range arg.given {
			c.addProtocol(testChannel, pi)
		}
		p, _ := newPeerWithFakeConn(true)
		p.setChannel(testChannel)
		err := c.resolveProtocols(p, testChannel, arg.param)
		assert.NoError(t, err)
		for _, pi := range arg.expected {
			assert.True(t, p.ProtocolInfos().Exists(pi))
		}
		//AttrSupportDefaultProtocols for default-protocols supported channel only
		assert.Equal(t, arg.isSupportDefault, p.EqualsAttr(AttrSupportDefaultProtocols, true))
		//AttrP2PLegacy for legacy peer only
		assert.Equal(t, arg.isLegacy, p.EqualsAttr(AttrP2PLegacy, true))

		for _, pi := range c.ProtocolInfos(testChannel).Array() {
			c.removeProtocol(testChannel, pi)
		}
	}

	p, _ := newPeerWithFakeConn(true)
	p.setChannel(testChannel)
	var err error

	//invalid channel
	err = c.resolveProtocols(p, "invalid", defaultProtocols)
	assert.Error(t, err)

	//not exists channel
	err = c.resolveProtocols(p, testChannel, defaultProtocols)
	assert.Error(t, err)

	//not support p2p protocol
	c.addProtocol(testChannel, module.ProtoP2P)
	err = c.resolveProtocols(p, testChannel, []module.ProtocolInfo{nextProtoP2P})
	assert.Error(t, err)
}

func sortProtocols(pis []module.ProtocolInfo) []module.ProtocolInfo {
	sort.Slice(pis, func(i, j int) bool {
		return pis[i].Uint16() < pis[j].Uint16()
	})
	return pis
}

func Test_ChannelNegotiator_Request(t *testing.T) {
	c := newChannelNegotiator(testNetAddress, generatePeerID(), testLogger())
	for _, pi := range defaultProtocols {
		c.addProtocol(testChannel, pi)
	}

	scens := []struct {
		givenJoinRequest   *JoinRequest
		expectJoinResponse *JoinResponse
		expectClose        bool
	}{
		{ //legacy support
			givenJoinRequest: &JoinRequest{
				Channel: testChannel,
				Addr:    testNetAddress,
			},
			expectJoinResponse: &JoinResponse{
				Channel:   testChannel,
				Addr:      testNetAddress,
				Protocols: defaultProtocols,
			},
		},
		{ //invalid channel
			givenJoinRequest: &JoinRequest{
				Channel: "invalid",
			},
			expectClose: true,
		},
	}
	for _, scen := range scens {
		p, conn := newPeerWithFakeConn(true)
		p.setChannel(testChannel)
		c.onPeer(p)
		t.Log(p)

		pkt := newPacket(p2pProtoChan, p2pProtoChanJoinReq,
			codec.MP.MustMarshalToBytes(scen.givenJoinRequest), nil)
		c.handleJoinRequest(pkt, p)

		if scen.expectJoinResponse != nil {
			pkt = conn.Packet()
			assert.NotNil(t, pkt)
			actualJoinResponse := &JoinResponse{}
			if err := c.decode(pkt.payload, actualJoinResponse); err != nil {
				assert.FailNow(t, err.Error())
			}
			assert.Equal(t, scen.givenJoinRequest.Addr, p.NetAddress())
			sortProtocols(actualJoinResponse.Protocols)
			assert.Equal(t, *scen.expectJoinResponse, *actualJoinResponse)
		}

		assert.Equal(t, scen.expectClose, p.IsClosed())
	}
}

func Test_ChannelNegotiator_Response(t *testing.T) {
	c := newChannelNegotiator(testNetAddress, generatePeerID(), testLogger())
	for _, pi := range defaultProtocols {
		c.addProtocol(testChannel, pi)
	}

	expectJoinRequest := &JoinRequest{
		Channel:   testChannel,
		Addr:      testNetAddress,
		Protocols: defaultProtocols,
	}
	scens := []struct {
		givenPeerChannel  string
		expectJoinRequest *JoinRequest
		givenJoinResponse *JoinResponse
		expectClose       bool
	}{
		{ //legacy support
			givenPeerChannel:  testChannel,
			expectJoinRequest: expectJoinRequest,
			givenJoinResponse: &JoinResponse{
				Channel:   testChannel,
				Addr:      testNetAddress,
				Protocols: defaultProtocols,
			},
		},
		{ //invalid channel
			givenPeerChannel: "invalid",
			expectClose:      true,
		},
		{ //invalid channel
			givenPeerChannel:  testChannel,
			expectJoinRequest: expectJoinRequest,
			givenJoinResponse: &JoinResponse{
				Channel: "invalid",
			},
			expectClose: true,
		},
	}
	for _, scen := range scens {
		p, conn := newPeerWithFakeConn(false)
		p.setChannel(scen.givenPeerChannel)
		c.onPeer(p)
		t.Log(p)

		if scen.expectJoinRequest != nil {
			pkt := conn.Packet()
			assert.NotNil(t, pkt)
			if p.IsClosed() {
				assert.FailNow(t, "closed")
			}
			actualJoinRequest := &JoinRequest{}
			if err := c.decode(pkt.payload, actualJoinRequest); err != nil {
				assert.FailNow(t, err.Error())
			}
			assert.Equal(t, *scen.expectJoinRequest, *actualJoinRequest)
		}

		if scen.givenJoinResponse != nil {
			pkt := newPacket(p2pProtoChan, p2pProtoChanJoinResp,
				codec.MP.MustMarshalToBytes(scen.givenJoinResponse), nil)
			c.handleJoinResponse(pkt, p)
			assert.Equal(t, scen.givenJoinResponse.Addr, p.NetAddress())
		}

		assert.Equal(t, scen.expectClose, p.IsClosed())
	}
}

func Test_ChannelNegotiator_Packet(t *testing.T) {
	c := newChannelNegotiator(testNetAddress, generatePeerID(), testLogger())
	for _, pi := range defaultProtocols {
		c.addProtocol(testChannel, pi)
	}

	args := []struct {
		in          bool
		givenPacket *Packet
		wait        module.ProtocolInfo
		invalidWait module.ProtocolInfo
	}{
		{
			in:          true,
			wait:        p2pProtoChanJoinReq,
			invalidWait: p2pProtoChanJoinResp,
		},
		{
			in:          false,
			wait:        p2pProtoChanJoinResp,
			invalidWait: p2pProtoChanJoinReq,
		},
	}
	for _, arg := range args {
		for _, pi := range []module.ProtocolInfo{arg.wait, arg.invalidWait} {
			p, _ := newPeerWithFakeConn(arg.in)
			c.onPeer(p)

			if arg.givenPacket != nil {
				c.onPacket(arg.givenPacket, p)
			}
			pkt := newPacket(p2pProtoChan, pi, []byte{0x00}, nil)
			c.onPacket(pkt, p)
			assert.True(t, p.IsClosed())
		}
	}

	p, _ := newPeerWithFakeConn(true)
	c.onPeer(p)
	pkt := newPacket(p2pProtoChan, module.ProtocolInfo(0xFFFF), []byte{0x00}, nil)
	c.onPacket(pkt, p)
	assert.True(t, p.HasCloseError(ErrNotRegisteredProtocol))
}
