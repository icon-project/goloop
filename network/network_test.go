package network

import (
	"fmt"
	"testing"
	"time"

	"github.com/ugorji/go/codec"

	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
)

const (
	testNumValidator = 4
	testNumSeed      = 4
	testNumCitizen   = 4
)

var (
	ProtoTestNetworkBroadcast module.ProtocolInfo = protocolInfo(0x0100)
	ProtoTestNetworkMulticast module.ProtocolInfo = protocolInfo(0x0200)
	ProtoTestNetworkRequest   module.ProtocolInfo = protocolInfo(0x0300)
	ProtoTestNetworkResponse  module.ProtocolInfo = protocolInfo(0x0400)
)

var (
	testSubProtocols = []module.ProtocolInfo{
		ProtoTestNetworkBroadcast,
		ProtoTestNetworkMulticast,
		ProtoTestNetworkRequest,
		ProtoTestNetworkResponse,
	}
)

type testReactor struct {
	name        string
	ms          module.Membership
	codecHandle codec.Handle
	log         *logger
	t           *testing.T
	nm          module.NetworkManager
	nt          module.NetworkTransport
	p2p         *PeerToPeer
}

func newTestReactor(name string, ms module.Membership, t *testing.T) *testReactor {
	r := &testReactor{name: name, ms: ms, codecHandle: &codec.MsgpackHandle{}, log: newLogger("TestReactor", name), t: t}
	ms.RegistReactor(name, r, testSubProtocols)
	r.p2p = ms.(*membership).p2p
	r.t.Log(time.Now(), r.name, "newTestReactor", r.p2p.self.id)
	return r
}

type testNetworkBroadcast struct {
	Message string
}

type testNetworkMulticast struct {
	Message string
}

type testNetworkRequest struct {
	Message string
}

type testNetworkResponse struct {
	Message string
}

func (r *testReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (re bool, err error) {
	r.log.Println("OnReceive", pi, b, id)
	switch pi {
	case ProtoTestNetworkBroadcast:
		rm := &testNetworkBroadcast{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkBroadcast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkMulticast:
		rm := &testNetworkMulticast{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkMulticast", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm, r.p2pConn())
		re = true
	case ProtoTestNetworkRequest:
		rm := &testNetworkRequest{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkRequest", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm)

		r.Response(rm.Message, id)
	case ProtoTestNetworkResponse:
		rm := &testNetworkResponse{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkResponse", rm, id)
		r.t.Log(time.Now(), r.name, "OnReceive", rm)
	default:
		re = false
	}
	return
}

func (r *testReactor) OnError() {
	assert.Fail(r.t, "TestReactor.onError")
}

func (r *testReactor) encode(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.NewEncoderBytes(&b, r.codecHandle)
	enc.MustEncode(v)
	return b
}

func (r *testReactor) decode(b []byte, v interface{}) {
	dec := codec.NewDecoderBytes(b, r.codecHandle)
	dec.MustDecode(v)
}

func (r *testReactor) p2pConn() string {
	p2p := r.p2p
	var p int
	if p2p.parent != nil {
		p = 1
	}
	return fmt.Sprintf("friends:%d, parent:%d, uncle:%d, children:%d, nephew:%d",
		p2p.friends.Len(),
		p,
		p2p.uncles.Len(),
		p2p.children.Len(),
		p2p.nephews.Len())
}

func (r *testReactor) Broadcast(msg string) {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("Broadcast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Broadcast", m, r.p2pConn())
	r.ms.Broadcast(ProtoTestNetworkBroadcast, r.encode(m), module.BROADCAST_ALL)
	r.log.Println("Broadcast", m)
}

func (r *testReactor) BroadcastNeighbor(msg string) {
	m := &testNetworkBroadcast{Message: fmt.Sprintf("BroadcastNeighbor.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "BroadcastNeighbor", m, r.p2pConn())
	r.ms.Broadcast(ProtoTestNetworkBroadcast, r.encode(m), module.BROADCAST_NEIGHBOR)
	r.log.Println("BroadcastNeighbor", m)
}

func (r *testReactor) Multicast(msg string) {
	m := &testNetworkMulticast{Message: fmt.Sprintf("Multicast.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Multicast", m, r.p2pConn())
	r.ms.Multicast(ProtoTestNetworkMulticast, r.encode(m), module.ROLE_VALIDATOR)
	r.log.Println("Multicast", m)
}

func (r *testReactor) Request(msg string, id module.PeerID) {
	m := &testNetworkRequest{Message: fmt.Sprintf("Request.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Request", m, r.p2pConn())
	r.ms.Unicast(ProtoTestNetworkRequest, r.encode(m), id)
	r.log.Println("Request", m, id)
}

func (r *testReactor) Response(msg string, id module.PeerID) {
	m := &testNetworkResponse{Message: fmt.Sprintf("Response.%s.%s", msg, r.name)}
	r.t.Log(time.Now(), r.name, "Response", m, r.p2pConn())
	r.ms.Unicast(ProtoTestNetworkResponse, r.encode(m), id)
	r.log.Println("Response", m, id)
}

func generateNetwork(name string, port int, n int, t *testing.T, roles ...module.Role) ([]*testReactor, int) {
	arr := make([]*testReactor, n)
	for i := 0; i < n; i++ {
		nt := NewTransport(fmt.Sprintf("127.0.0.1:%d", port+i), walletFromGeneratedPrivateKey())
		nm := NewManager(testChannel, nt, roles...)
		ms := nm.GetMembership(DefaultMembershipName)
		r := newTestReactor(fmt.Sprintf("%s_%d", name, i), ms, t)
		r.nt = nt
		r.nm = nm
		r.nt.Listen()
		arr[i] = r
	}
	return arr, port + n
}

func Test_network(t *testing.T) {
	m := make(map[string][]*testReactor)
	p := 8080
	m["TestCitizen"], p = generateNetwork("TestCitizen", p, testNumCitizen, t)
	m["TestSeed"], p = generateNetwork("TestSeed", p, testNumSeed, t, module.ROLE_SEED)
	m["TestValidator"], p = generateNetwork("TestValidator", p, testNumValidator, t, module.ROLE_VALIDATOR)

	sr := m["TestSeed"][0]
	sna := sr.nt.Address()
	for _, arr := range m {
		for _, r := range arr {
			if r.nt.Address() != sna {
				r.nt.Dial(sna, testChannel)
			}
		}
	}

	time.Sleep(2 * DefaultSeedPeriod)
	t.Log(time.Now(), "Messaging")
	m["TestValidator"][0].Broadcast("Test1")
	time.Sleep(DefaultSendTaskTimeout)
	m["TestValidator"][0].BroadcastNeighbor("Test2")
	time.Sleep(DefaultSendTaskTimeout)
	m["TestValidator"][0].Multicast("Test3")
	time.Sleep(DefaultSendTaskTimeout)
	m["TestSeed"][0].Multicast("Test4")
	time.Sleep(2 * DefaultSendTaskTimeout)
	m["TestCitizen"][0].Multicast("Test5")
	time.Sleep(3 * DefaultSendTaskTimeout)
	m["TestCitizen"][0].Request("Test6", sr.nt.PeerID())
	time.Sleep(DefaultSendTaskTimeout)

	for _, arr := range m {
		for _, r := range arr {
			r.nt.Close()
		}
	}
	t.Log(time.Now(), "Finish")
}
