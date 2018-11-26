package network

import (
	"testing"
	"time"

	"github.com/ugorji/go/codec"

	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
)

const (
	testSeedAddress    = "127.0.0.1:8081"
	testCitizenAddress = "127.0.0.1:8082"
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
}

func newTestReactor(name string, ms module.Membership, t *testing.T) *testReactor {
	r := &testReactor{name, ms, &codec.MsgpackHandle{}, newLogger("TestReactor", name), t}
	ms.RegistReactor(name, r, testSubProtocols)
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

		re = true
	case ProtoTestNetworkMulticast:
		rm := &testNetworkMulticast{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkMulticast", rm, id)

		re = true
	case ProtoTestNetworkRequest:
		rm := &testNetworkRequest{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkRequest", rm, id)

		r.Response(id)
	case ProtoTestNetworkResponse:
		rm := &testNetworkResponse{}
		r.decode(b, rm)
		r.log.Println("handleProtoTestNetworkResponse", rm, id)
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

func (r *testReactor) Broadcast() {
	m := &testNetworkBroadcast{Message: "TestBroasdcast"}
	r.ms.Broadcast(ProtoTestNetworkBroadcast, r.encode(m), module.BROADCAST_ALL)
	r.log.Println("Broadcast", m)
}

func (r *testReactor) Multicast() {
	m := &testNetworkMulticast{Message: "TestMulticast"}
	r.ms.Multicast(ProtoTestNetworkMulticast, r.encode(m), module.ROLE_VALIDATOR)
	r.log.Println("Multicast", m)
}

func (r *testReactor) Request(id module.PeerID) {
	m := &testNetworkRequest{Message: "Hello"}
	r.ms.Unicast(ProtoTestNetworkRequest, r.encode(m), id)
	r.log.Println("Request", m, id)
}

func (r *testReactor) Response(id module.PeerID) {
	m := &testNetworkResponse{Message: "World"}
	r.ms.Unicast(ProtoTestNetworkRequest, r.encode(m), id)
	r.log.Println("Response", m, id)
}

func Test_network(t *testing.T) {
	cnt := GetTransport()
	snt := NewTransport(testSeedAddress, walletFromGeneratedPrivateKey())
	vnt := NewTransport(testCitizenAddress, walletFromGeneratedPrivateKey())

	t.Logf("c:%v,s:%v,v:%v", cnt.PeerID(), snt.PeerID(), vnt.PeerID())

	cnm := GetManager(testChannel)
	snm := NewManager(testChannel, snt, module.ROLE_SEED)
	vnm := NewManager(testChannel, vnt, module.ROLE_SEED, module.ROLE_VALIDATOR)

	cms := cnm.GetMembership(DefaultMembershipName)
	sms := snm.GetMembership(DefaultMembershipName)
	vms := vnm.GetMembership(DefaultMembershipName)

	cr := newTestReactor("TestCitizen", cms, t)
	sr := newTestReactor("TestSeed", sms, t)
	vr := newTestReactor("TestValidator", vms, t)

	srp := []module.PeerID{snt.PeerID(), vnt.PeerID()}
	vrp := []module.PeerID{vnt.PeerID()}
	cms.AddRole(module.ROLE_VALIDATOR, vrp...)
	cms.AddRole(module.ROLE_SEED, srp...)
	sms.AddRole(module.ROLE_VALIDATOR, vrp...)
	sms.AddRole(module.ROLE_SEED, srp...)
	vms.AddRole(module.ROLE_VALIDATOR, vrp...)
	vms.AddRole(module.ROLE_SEED, srp...)

	cnt.Listen()
	snt.Listen()
	vnt.Listen()

	// snal := []NetAddress{NetAddress(sl.address)}
	// pd.peerToPeers[testChannel].seeds.Merge(snal...)
	// spd.peerToPeers[testChannel].seeds.Merge(snal...)
	// cpd.peerToPeers[testChannel].seeds.Merge(snal...)
	cnt.Dial(snt.Address(), testChannel)
	vnt.Dial(snt.Address(), testChannel)

	time.Sleep(5 * time.Second)
	vr.Broadcast()
	sr.Multicast()
	// sp := cnm.peerToPeer.getPeer(spd.self)
	// err := sp.conn.Close()
	// if err != nil {
	// 	t.Logf("sp.conn.Close error:%v", err)
	// }
	cr.Multicast()
	cr.Request(snt.PeerID())
	time.Sleep(5 * time.Second)
}
